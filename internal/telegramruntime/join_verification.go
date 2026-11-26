package telegramruntime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type chatPermissions struct {
	CanSendMessages bool `json:"can_send_messages"`
	CanSendAudios bool `json:"can_send_audios"`
	CanSendDocuments bool `json:"can_send_documents"`
	CanSendPhotos bool `json:"can_send_photos"`
	CanSendVideos bool `json:"can_send_videos"`
	CanSendVideoNotes bool `json:"can_send_video_notes"`
	CanSendVoiceNotes bool `json:"can_send_voice_notes"`
	CanSendPolls bool `json:"can_send_polls"`
	CanSendOtherMessages bool `json:"can_send_other_messages"`
	CanAddWebPagePreviews bool `json:"can_add_web_page_previews"`
	CanChangeInfo bool `json:"can_change_info"`
	CanInviteUsers bool `json:"can_invite_users"`
	CanPinMessages bool `json:"can_pin_messages"`
	CanManageTopics bool `json:"can_manage_topics"`
}

type telegramChatInfo struct {
	Permissions *chatPermissions `json:"permissions"`
}

type callbackMessage struct {
	MessageID int64 `json:"message_id"`
	Chat chat `json:"chat"`
}

type telegramCallbackQuery struct {
	ID string `json:"id"`
	From telegramSender `json:"from"`
	Message *callbackMessage `json:"message"`
	Data string `json:"data"`
}

func randomInt(minimum, maximum int64) (int64, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(maximum-minimum+1))
	if err != nil { return 0, err }
	return value.Int64()+minimum, nil
}

func (p *Poller) startJoinVerification(ctx context.Context, bot managedBot, groupID string, member telegramSender, rule executableRule, token string) {
	if member.ID <= 0 || member.IsBot { return }
	var cfg struct {
		ChallengeType string `json:"challenge_type"`
		TimeoutSeconds int `json:"timeout_seconds"`
		FailureAction string `json:"failure_action"`
	}
	if err := json.Unmarshal(rule.Config, &cfg); err != nil { return }
	var chatID, telegramBotID int64
	var chatType string
	if err := p.pool.QueryRow(ctx, `SELECT g.telegram_chat_id,b.telegram_bot_id,g.chat_type
		FROM telegram_groups g JOIN telegram_bots b ON b.id=g.bot_id AND b.tenant_id=g.tenant_id
		WHERE g.tenant_id=$1 AND g.id=$2 AND g.status='active' AND b.status='active'`,
		bot.tenantID, groupID).Scan(&chatID, &telegramBotID, &chatType); err != nil {
		p.logger.Error("load group for join verification", "group_id", groupID, "error", err); return
	}
	if chatType != "supergroup" { p.logger.Warn("join verification requires a Telegram supergroup", "group_id", groupID, "chat_type", chatType); return }
	var botMember telegramChatMember
	if err := p.callTelegram(ctx, token, "getChatMember", map[string]any{"chat_id":chatID,"user_id":telegramBotID}, &botMember); err != nil {
		p.logger.Warn("join verification cannot inspect Bot permissions", "group_id", groupID, "error", err); return
	}
	if botMember.Status != "creator" && (botMember.Status != "administrator" || !botMember.CanRestrictMembers) {
		p.logger.Warn("join verification skipped because Bot lacks can_restrict_members", "group_id", groupID); return
	}
	a, err := randomInt(1,9); if err != nil { p.logger.Error("generate join challenge", "error", err); return }
	b, err := randomInt(1,9); if err != nil { p.logger.Error("generate join challenge", "error", err); return }
	answer := a+b
	options := joinChallengeOptions(answer)
	for i:=len(options)-1;i>0;i-- {
		j,err:=randomInt(0,int64(i)); if err!=nil { p.logger.Error("shuffle join challenge", "error", err); return }
		options[i],options[j]=options[j],options[i]
	}
	var challengeID string
	err=p.pool.QueryRow(ctx,`INSERT INTO telegram_join_verification_challenges(
		tenant_id,bot_id,group_id,telegram_user_id,answer,failure_action,expires_at)
		VALUES($1,$2,$3,$4,$5,$6,now()+($7 * interval '1 second'))
		ON CONFLICT(group_id,telegram_user_id) DO UPDATE SET
			tenant_id=EXCLUDED.tenant_id,bot_id=EXCLUDED.bot_id,answer=EXCLUDED.answer,
			status='pending',failure_action=EXCLUDED.failure_action,challenge_message_id=NULL,
			expires_at=EXCLUDED.expires_at,updated_at=now()
		RETURNING id::text`,bot.tenantID,bot.id,groupID,member.ID,answer,cfg.FailureAction,cfg.TimeoutSeconds).Scan(&challengeID)
	if err!=nil { p.logger.Error("save join verification challenge", "group_id", groupID, "error", err); return }
	locked:=chatPermissions{}
	if err:=p.callTelegram(ctx,token,"restrictChatMember",map[string]any{
		"chat_id":chatID,"user_id":member.ID,"permissions":locked,"use_independent_chat_permissions":true,
	},nil);err!=nil {
		_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='failed',updated_at=now() WHERE id=$1",challengeID)
		p.logger.Warn("restrict member for join verification failed","group_id",groupID,"user_id",member.ID,"error",err);return
	}
	buttons:=make([][]map[string]string,0,1)
	row:=make([]map[string]string,0,3)
	for _,option:=range options {
		row=append(row,map[string]string{"text":strconv.FormatInt(option,10),"callback_data":fmt.Sprintf("tgverify:%s:%d",challengeID,option)})
	}
	buttons=append(buttons,row)
	var sent struct { MessageID int64 `json:"message_id"` }
	err=p.callTelegram(ctx,token,"sendMessage",map[string]any{
		"chat_id":chatID,"text":fmt.Sprintf("欢迎 %s！请在 %d 秒内完成验证：%d + %d = ?",renderNotice("{user}",member,""),cfg.TimeoutSeconds,a,b),
		"reply_markup":map[string]any{"inline_keyboard":buttons},
	},&sent)
	if err!=nil {
		_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='failed',updated_at=now() WHERE id=$1",challengeID)
		if restoreErr:=p.restoreMemberPermissions(ctx,token,chatID,member.ID);restoreErr!=nil {
			p.logger.Error("restore member after failed verification prompt","user_id",member.ID,"error",restoreErr)
		}
		p.logger.Warn("send join verification prompt failed","group_id",groupID,"error",err);return
	}
	_,err=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET challenge_message_id=$2,updated_at=now() WHERE id=$1",challengeID,sent.MessageID)
	if err!=nil { p.logger.Error("save join verification prompt message id","challenge_id",challengeID,"error",err) }
}

func (p *Poller) handleVerificationCallback(ctx context.Context, bot managedBot, query telegramCallbackQuery, token string) {
	parts:=strings.Split(query.Data,":")
	if len(parts)!=3 || parts[0]!="tgverify" { return }
	answer,err:=strconv.ParseInt(parts[2],10,16)
	if err!=nil { return }
	if query.Message==nil { return }
	var challengeID,groupID string
	var expected int64
	var chatID int64
	err=p.pool.QueryRow(ctx,`SELECT c.id::text,c.group_id::text,c.answer,g.telegram_chat_id
		FROM telegram_join_verification_challenges c JOIN telegram_groups g
		ON g.id=c.group_id AND g.tenant_id=c.tenant_id
		WHERE c.id=$1 AND c.tenant_id=$2 AND c.bot_id=$3 AND c.telegram_user_id=$4
		AND c.status='pending' AND c.expires_at>now() AND g.telegram_chat_id=$5`,
		parts[1],bot.tenantID,bot.id,query.From.ID,query.Message.Chat.ID).Scan(&challengeID,&groupID,&expected,&chatID)
	if err!=nil {
		_ = p.answerCallback(ctx,token,query.ID,"验证已过期或该按钮不属于你的验证",true); return
	}
	if answer!=expected {
		_ = p.answerCallback(ctx,token,query.ID,"答案不正确，请重新选择",true); return
	}
	tag,err:=p.pool.Exec(ctx,`UPDATE telegram_join_verification_challenges SET status='verifying',updated_at=now()
		WHERE id=$1 AND status='pending' AND telegram_user_id=$2 AND expires_at>now()`,challengeID,query.From.ID)
	if err!=nil || tag.RowsAffected()!=1 {
		_ = p.answerCallback(ctx,token,query.ID,"验证状态已变化，请重新加入后再试",true); return
	}
	if err=p.restoreMemberPermissions(ctx,token,chatID,query.From.ID);err!=nil {
		_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='pending',updated_at=now() WHERE id=$1 AND status='verifying'",challengeID)
		_ = p.answerCallback(ctx,token,query.ID,"验证通过，但恢复群权限失败，请联系管理员",true)
		p.logger.Error("restore member permissions after verification","challenge_id",challengeID,"error",err);return
	}
	_,err=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='verified',updated_at=now() WHERE id=$1 AND status='verifying'",challengeID)
	if err!=nil { p.logger.Error("mark join verification successful","challenge_id",challengeID,"error",err) }
	_ = p.answerCallback(ctx,token,query.ID,"验证成功，欢迎加入！",false)
	if query.Message.MessageID>0 {
		_ = p.callTelegram(ctx,token,"editMessageText",map[string]any{
			"chat_id":chatID,"message_id":query.Message.MessageID,"text":"验证成功，欢迎加入！","reply_markup":map[string]any{"inline_keyboard":[]any{}},
		},nil)
	}
}

func (p *Poller) restoreMemberPermissions(ctx context.Context, token string, chatID,userID int64) error {
	var info telegramChatInfo
	if err:=p.callTelegram(ctx,token,"getChat",map[string]any{"chat_id":chatID},&info);err!=nil{return err}
	if info.Permissions==nil{return errors.New("Telegram group default permissions are unavailable; refusing to grant permissions")}
	return p.callTelegram(ctx,token,"restrictChatMember",map[string]any{
		"chat_id":chatID,"user_id":userID,"permissions":info.Permissions,"use_independent_chat_permissions":true,
	},nil)
}

func (p *Poller) answerCallback(ctx context.Context,token,callbackID,text string,alert bool) error {
	return p.callTelegram(ctx,token,"answerCallbackQuery",map[string]any{
		"callback_query_id":callbackID,"text":text,"show_alert":alert,
	},nil)
}

func (p *Poller) expireJoinVerificationChallenges(ctx context.Context, bot managedBot, token string) {
	rows,err:=p.pool.Query(ctx,`SELECT c.id::text,c.group_id::text,c.telegram_user_id,c.failure_action,g.telegram_chat_id
		FROM telegram_join_verification_challenges c JOIN telegram_groups g
		ON g.id=c.group_id AND g.tenant_id=c.tenant_id
		WHERE c.tenant_id=$1 AND c.bot_id=$2 AND c.status='pending' AND c.expires_at<=now()
		ORDER BY c.expires_at LIMIT 50`,bot.tenantID,bot.id)
	if err!=nil { p.logger.Error("load expired join verification challenges","error",err);return }
	type expired struct{id,groupID string;userID,chatID int64;action string}
	items:=make([]expired,0)
	for rows.Next(){var item expired;if err:=rows.Scan(&item.id,&item.groupID,&item.userID,&item.action,&item.chatID);err!=nil{p.logger.Error("read expired join challenge","error",err);rows.Close();return};items=append(items,item)}
	if err:=rows.Err();err!=nil{rows.Close();p.logger.Error("iterate expired join challenges","error",err);return}
	rows.Close()
	for _,item:=range items {
		tag,err:=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='expired',updated_at=now() WHERE id=$1 AND status='pending' AND expires_at<=now()",item.id)
		if err!=nil || tag.RowsAffected()!=1 { continue }
		if item.action=="kick" {
			if err:=p.callTelegram(ctx,token,"banChatMember",map[string]any{"chat_id":item.chatID,"user_id":item.userID},nil);err!=nil {
				_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='failed',updated_at=now() WHERE id=$1",item.id)
				p.logger.Error("kick member after join verification timeout","challenge_id",item.id,"error",err);continue
			}
			if err:=p.callTelegram(ctx,token,"unbanChatMember",map[string]any{"chat_id":item.chatID,"user_id":item.userID,"only_if_banned":true},nil);err!=nil {
				p.logger.Error("release timed-out member ban","challenge_id",item.id,"error",err)
			}
		}
	}
}


func joinChallengeOptions(answer int64) []int64 {
	if answer < 2 || answer > 18 { return nil }
	wrongOne,wrongTwo:=answer+1,answer-1
	if wrongOne>18 { wrongOne,wrongTwo=answer-1,answer-2 }
	if wrongTwo<2 { wrongOne,wrongTwo=answer+1,answer+2 }
	return []int64{answer,wrongOne,wrongTwo}
}
