package telegramruntime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"html"
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
		RequiredChannels []string `json:"required_channels"`
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
	answerText, prompt, options, numericAnswer, err := generateJoinChallenge(cfg.ChallengeType)
	if err != nil { p.logger.Error("generate join challenge", "error", err); return }
	for i:=len(options)-1;i>0;i-- {
		j,err:=randomInt(0,int64(i)); if err!=nil { p.logger.Error("shuffle join challenge", "error", err); return }
		options[i],options[j]=options[j],options[i]
	}
	var challengeID string
	err=p.pool.QueryRow(ctx,`INSERT INTO telegram_join_verification_challenges(
		tenant_id,bot_id,group_id,telegram_user_id,answer,challenge_type,answer_text,challenge_options,failure_action,expires_at,required_channels)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,now()+($10 * interval '1 second'),$11::jsonb)
		ON CONFLICT(group_id,telegram_user_id) DO UPDATE SET
			tenant_id=EXCLUDED.tenant_id,bot_id=EXCLUDED.bot_id,answer=EXCLUDED.answer,
			challenge_type=EXCLUDED.challenge_type,answer_text=EXCLUDED.answer_text,challenge_options=EXCLUDED.challenge_options,
			status='pending',failure_action=EXCLUDED.failure_action,challenge_message_id=NULL,
			expires_at=EXCLUDED.expires_at,required_channels=EXCLUDED.required_channels,updated_at=now()
		RETURNING id::text`,bot.tenantID,bot.id,groupID,member.ID,numericAnswer,cfg.ChallengeType,answerText,mustJSON(options),cfg.FailureAction,cfg.TimeoutSeconds,mustJSON(cfg.RequiredChannels)).Scan(&challengeID)
	if err!=nil { p.logger.Error("save join verification challenge", "group_id", groupID, "error", err); return }
	locked:=chatPermissions{CanSendMessages:cfg.ChallengeType=="letters" || cfg.ChallengeType=="alphanumeric"}
	if err:=p.callTelegram(ctx,token,"restrictChatMember",map[string]any{
		"chat_id":chatID,"user_id":member.ID,"permissions":locked,"use_independent_chat_permissions":true,
	},nil);err!=nil {
		_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='failed',updated_at=now() WHERE id=$1",challengeID)
		p.logger.Warn("restrict member for join verification failed","group_id",groupID,"user_id",member.ID,"error",err);return
	}
	var replyMarkup any
	if cfg.ChallengeType=="math" {
		buttons:=make([][]map[string]string,0,1)
		row:=make([]map[string]string,0,3)
		for index,option:=range options {
			row=append(row,map[string]string{"text":option,"callback_data":fmt.Sprintf("tgverify:%s:%d",challengeID,index)})
		}
		buttons=append(buttons,row)
		replyMarkup=map[string]any{"inline_keyboard":buttons}
	}
	var sent struct { MessageID int64 `json:"message_id"` }
	err=p.callTelegram(ctx,token,"sendMessage",map[string]any{
		"chat_id":chatID,"text":fmt.Sprintf("欢迎 %s！请在 %d 秒内完成验证：%s%s",renderNotice("{user}",member,""),cfg.TimeoutSeconds,prompt,requiredChannelPrompt(cfg.RequiredChannels)),
		"parse_mode":"HTML","reply_markup":replyMarkup,
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

func (p *Poller) handleJoinVerificationText(ctx context.Context,bot managedBot,groupID string,message *telegramMessage,token string) bool {
	if message==nil || message.From==nil || message.From.IsBot || strings.TrimSpace(message.Text)=="" { return false }
	var challengeID,expected string
	var chatID int64
	var channelsJSON []byte
	err:=p.pool.QueryRow(ctx,`SELECT c.id::text,c.answer_text,g.telegram_chat_id,c.required_channels
		FROM telegram_join_verification_challenges c JOIN telegram_groups g ON g.id=c.group_id AND g.tenant_id=c.tenant_id
		WHERE c.tenant_id=$1 AND c.bot_id=$2 AND c.group_id=$3 AND c.telegram_user_id=$4
		AND c.challenge_type IN ('letters','alphanumeric') AND c.status='pending' AND c.expires_at>now()`,
		bot.tenantID,bot.id,groupID,message.From.ID).Scan(&challengeID,&expected,&chatID,&channelsJSON)
	if err!=nil { return false }
	if !strings.EqualFold(strings.TrimSpace(message.Text),expected) {
		if message.MessageID>0 { _=p.callTelegram(ctx,token,"deleteMessage",map[string]any{"chat_id":chatID,"message_id":message.MessageID},nil) }
		_,_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"验证码不正确，请查看验证提示后重新输入。"},nil)
		return true
	}
	var channels []string
	if json.Unmarshal(channelsJSON,&channels)!=nil { return true }
	if missing,checkErr:=p.checkRequiredChannels(ctx,token,channels,message.From.ID);checkErr!=nil || len(missing)>0 {
		notice:="暂时无法核验频道成员状态，请稍后重试。"
		if checkErr==nil { notice="请先加入必需频道："+strings.Join(missing,", ") }
		_,_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":notice},nil)
		return true
	}
	tag,err:=p.pool.Exec(ctx,`UPDATE telegram_join_verification_challenges SET status='verifying',updated_at=now()
		WHERE id=$1 AND status='pending' AND telegram_user_id=$2 AND expires_at>now()`,challengeID,message.From.ID)
	if err!=nil || tag.RowsAffected()!=1 { return true }
	if err:=p.restoreMemberPermissions(ctx,token,chatID,message.From.ID);err!=nil {
		_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='pending',updated_at=now() WHERE id=$1 AND status='verifying'",challengeID)
		p.logger.Error("restore member permissions after text verification","challenge_id",challengeID,"error",err)
		return true
	}
	_,_=p.pool.Exec(ctx,"UPDATE telegram_join_verification_challenges SET status='verified',updated_at=now() WHERE id=$1 AND status='verifying'",challengeID)
	_,_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"验证成功，欢迎加入！"},nil)
	return true
}

func (p *Poller) handleVerificationCallback(ctx context.Context, bot managedBot, query telegramCallbackQuery, token string) {
	parts:=strings.Split(query.Data,":")
	if len(parts)!=3 || parts[0]!="tgverify" { return }
	optionIndex,err:=strconv.Atoi(parts[2])
	if err!=nil || optionIndex<0 || optionIndex>2 { return }
	if query.Message==nil { return }
	var challengeID,groupID,expected,challengeType string
	var chatID int64
	var requiredChannelsJSON,optionsJSON []byte
	err=p.pool.QueryRow(ctx,`SELECT c.id::text,c.group_id::text,c.answer_text,c.challenge_type,c.challenge_options,g.telegram_chat_id,c.required_channels
		FROM telegram_join_verification_challenges c JOIN telegram_groups g
		ON g.id=c.group_id AND g.tenant_id=c.tenant_id
		WHERE c.id=$1 AND c.tenant_id=$2 AND c.bot_id=$3 AND c.telegram_user_id=$4
		AND c.status='pending' AND c.expires_at>now() AND g.telegram_chat_id=$5
		AND c.challenge_message_id=$6`,
		parts[1],bot.tenantID,bot.id,query.From.ID,query.Message.Chat.ID,query.Message.MessageID).Scan(&challengeID,&groupID,&expected,&challengeType,&optionsJSON,&chatID,&requiredChannelsJSON)
	if err!=nil {
		_ = p.answerCallback(ctx,token,query.ID,"验证已过期或该按钮不属于你的验证",true); return
	}
	var options []string
	if err:=json.Unmarshal(optionsJSON,&options);err!=nil || optionIndex>=len(options) { _ = p.answerCallback(ctx,token,query.ID,"验证选项无效，请重新加入后再试",true); return }
	if options[optionIndex]!=expected {
		_ = p.answerCallback(ctx,token,query.ID,"答案不正确，请重新选择",true); return
	}
	var requiredChannels []string
	if err:=json.Unmarshal(requiredChannelsJSON,&requiredChannels);err!=nil {
		_ = p.answerCallback(ctx,token,query.ID,"验证配置读取失败，请联系管理员",true);return
	}
	if missing,checkErr:=p.checkRequiredChannels(ctx,token,requiredChannels,query.From.ID);checkErr!=nil {
		_ = p.answerCallback(ctx,token,query.ID,"暂时无法核验频道成员状态，请稍后重试",true)
		p.logger.Warn("verify required Telegram channel membership","challenge_id",challengeID,"error",checkErr);return
	} else if len(missing)>0 {
		_ = p.answerCallback(ctx,token,query.ID,"请先加入必需频道："+strings.Join(missing,", ")+"，再点击正确答案",true);return
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


func generateJoinChallenge(challengeType string) (string,string,[]string,int64,error) {
	const letters="ABCDEFGHJKLMNPQRSTUVWXYZ"
	const alpha="ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	switch challengeType {
	case "math":
		a,err:=randomInt(1,9);if err!=nil{return "","",nil,0,err}
		b,err:=randomInt(1,9);if err!=nil{return "","",nil,0,err}
		answer:=a+b
		values:=joinChallengeOptions(answer)
		options:=make([]string,len(values))
		for i,v:=range values{options[i]=strconv.FormatInt(v,10)}
		return strconv.FormatInt(answer,10),fmt.Sprintf("%d + %d = ?",a,b),options,answer,nil
	case "letters","alphanumeric":
		alphabet:=letters;if challengeType=="alphanumeric"{alphabet=alpha}
		code:=make([]byte,4)
		for i:=range code{n,err:=randomInt(0,int64(len(alphabet)-1));if err!=nil{return "","",nil,0,err};code[i]=alphabet[n]}
		answer:=string(code)
		wrong:=make([]string,0,2)
		for len(wrong)<2{
			candidate:=make([]byte,4)
			for i:=range candidate{n,err:=randomInt(0,int64(len(alphabet)-1));if err!=nil{return "","",nil,0,err};candidate[i]=alphabet[n]}
			value:=string(candidate);if value==answer{continue}
			duplicate:=false;for _,v:=range wrong{if v==value{duplicate=true;break}}
			if !duplicate{wrong=append(wrong,value)}
		}
		return answer,"请在群内发送验证码：<tg-spoiler>"+html.EscapeString(answer)+"</tg-spoiler>",nil,2,nil
	default:
		return "","",nil,0,fmt.Errorf("unsupported join challenge type %q",challengeType)
	}
}

func joinChallengeOptions(answer int64) []int64 {
	if answer < 2 || answer > 18 { return nil }
	wrongOne,wrongTwo:=answer+1,answer-1
	if wrongOne>18 { wrongOne,wrongTwo=answer-1,answer-2 }
	if wrongTwo<2 { wrongOne,wrongTwo=answer+1,answer+2 }
	return []int64{answer,wrongOne,wrongTwo}
}

func mustJSON(value any) []byte {
	data,err:=json.Marshal(value)
	if err!=nil { return []byte("[]") }
	return data
}

func requiredChannelPrompt(channels []string) string {
	if len(channels)==0 { return "" }
	return "\n请先加入以下频道，再完成验证：\n"+strings.Join(channels,"\n")
}

func (p *Poller) checkRequiredChannels(ctx context.Context,token string,channels []string,userID int64)([]string,error) {
	missing:=make([]string,0)
	for _,channel:=range channels {
		var member telegramChatMember
		if err:=p.callTelegram(ctx,token,"getChatMember",map[string]any{"chat_id":channel,"user_id":userID},&member);err!=nil {
			return nil,fmt.Errorf("check channel %s membership: %w",channel,err)
		}
		if member.Status!="member" && member.Status!="administrator" && member.Status!="creator" {
			missing=append(missing,channel)
		}
	}
	return missing,nil
}
