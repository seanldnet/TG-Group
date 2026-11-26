package telegramruntime

import (
	"errors"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/seanldnet/tg-group/internal/bots"
)

type executableRule struct {
	ID string
	Type string
	Name string
	Config json.RawMessage
}

func (p *Poller) processMessageRules(ctx context.Context, bot managedBot, groupID string, item update, message *telegramMessage, token string) {
	if message == nil { return }
	// Group administration is callback-driven; slash commands never execute actions.
	var senderID any
	if message.From != nil && message.From.ID > 0 { senderID = message.From.ID }
	content := message.Text
	if content == "" { content = message.Caption }
	rows, err := p.pool.Query(ctx, `SELECT id::text,rule_type,name,config FROM telegram_group_rules
		WHERE tenant_id=$1 AND group_id=$2 AND enabled=true
		ORDER BY priority,created_at`, bot.tenantID, groupID)
	if err != nil { p.logger.Error("load Telegram group rules", "group_id", groupID, "error", err); return }
	rules := make([]executableRule, 0)
	for rows.Next() {
		var rule executableRule
		if err := rows.Scan(&rule.ID,&rule.Type,&rule.Name,&rule.Config); err != nil {
			p.logger.Error("read Telegram group rule", "group_id", groupID, "error", err); rows.Close(); return
		}
		rules = append(rules,rule)
	}
	if err:=rows.Err();err!=nil{p.logger.Error("iterate Telegram group rules","group_id",groupID,"error",err);rows.Close();return}
	rows.Close()

	if len(message.NewChatMembers)>0 {
		for _, member := range message.NewChatMembers {
			for _, rule := range rules {
				switch rule.Type {
				case "welcome":
					var cfg struct{Text string `json:"text"`}
					if json.Unmarshal(rule.Config,&cfg)!=nil { continue }
					text:=renderNotice(cfg.Text,member,message.Chat.Title)
					p.executeRuleNotice(ctx,bot,groupID,item.UpdateID,message.MessageID,member.ID,rule,"welcome",text,token)
				case "join_verification":
					p.startJoinVerification(ctx,bot,groupID,member,rule,token)
				}
			}
		}
	}
	if message.LeftChatMember!=nil {
		for _,rule:=range rules {
			if rule.Type!="leave_notice"{continue}
			var cfg struct{Text string `json:"text"`}
			if json.Unmarshal(rule.Config,&cfg)!=nil{continue}
			text:=renderNotice(cfg.Text,*message.LeftChatMember,message.Chat.Title)
			p.executeRuleNotice(ctx,bot,groupID,item.UpdateID,message.MessageID,message.LeftChatMember.ID,rule,"leave_notice",text,token)
		}
	}

	if content=="" || senderID==nil || message.From.IsBot { return }
	var matched *executableRule
	for i:=range rules {
		rule:=&rules[i]
		switch rule.Type {
		case "keyword_filter","sensitive_word":
			var cfg struct{Words []string `json:"words"`;Action string `json:"action"`}
			if json.Unmarshal(rule.Config,&cfg)!=nil || cfg.Action!="delete" {continue}
			lower:=strings.ToLower(content)
			for _,word:=range cfg.Words {if word!="" && strings.Contains(lower,strings.ToLower(word)){matched=rule;break}}
		case "anti_ad":
			var cfg struct{Patterns []string `json:"patterns"`;Action string `json:"action"`}
			if json.Unmarshal(rule.Config,&cfg)!=nil || cfg.Action!="delete"{continue}
			lower:=strings.ToLower(content)
			for _,pattern:=range cfg.Patterns{if pattern!=""&&strings.Contains(lower,strings.ToLower(pattern)){matched=rule;break}}
		case "anti_spam":
			var cfg struct{MaxMessages int `json:"max_messages"`;WindowSeconds int `json:"window_seconds"`;Action string `json:"action"`}
			if json.Unmarshal(rule.Config,&cfg)!=nil || cfg.Action!="delete"{continue}
			var count int
			err:=p.pool.QueryRow(ctx,`SELECT count(*) FROM telegram_message_events
				WHERE tenant_id=$1 AND group_id=$2 AND telegram_user_id=$3 AND received_at >= now()-($4 * interval '1 second')`,
				bot.tenantID,groupID,senderID,cfg.WindowSeconds).Scan(&count)
			if err!=nil{p.logger.Error("count messages for anti-spam rule","rule_id",rule.ID,"error",err);continue}
			if count>=cfg.MaxMessages{matched=rule}
		}
		if matched!=nil{break}
	}
	if matched==nil{return}
	if message.MessageID<=0{return}
	p.executeDeleteRule(ctx,bot,groupID,item.UpdateID,message.MessageID,senderID.(int64),*matched,token)
}

func renderNotice(template string, member telegramSender, groupName string) string {
	name:=senderDisplayName(&member)
	if name=="" {name=strconv.FormatInt(member.ID,10)}
	username:=member.Username
	if username!="" {username="@"+username}
	result:=strings.NewReplacer("{user}",name,"{username}",username,"{group}",groupName).Replace(template)
	if len([]rune(result))>4096 {result=string([]rune(result)[:4096])}
	return result
}

func (p *Poller) reserveRuleEvent(ctx context.Context,tenantID,groupID,ruleID string,updateID int64,messageID *int64,userID any,action string)(int64,bool,error){
	var eventID int64
	err:=p.pool.QueryRow(ctx,`INSERT INTO telegram_group_rule_events(tenant_id,group_id,rule_id,telegram_update_id,telegram_message_id,telegram_user_id,action,status,detail)
		VALUES($1,$2,$3,$4,$5,$6,$7,'skipped','processing')
		ON CONFLICT DO NOTHING RETURNING id`,
		tenantID,groupID,ruleID,updateID,messageID,userID,action).Scan(&eventID)
	if err==pgx.ErrNoRows{return 0,false,nil}
	if err!=nil{return 0,false,err}
	return eventID,true,nil
}

func (p *Poller) finishRuleEvent(ctx context.Context,eventID int64,status,detail string){
	if len([]rune(detail))>500{detail=string([]rune(detail)[:500])}
	if _,err:=p.pool.Exec(ctx,"UPDATE telegram_group_rule_events SET status=$2,detail=$3 WHERE id=$1",eventID,status,detail);err!=nil{
		p.logger.Error("update Telegram rule event status","event_id",eventID,"error",err)
	}
}

func (p *Poller) executeDeleteRule(ctx context.Context,bot managedBot,groupID string,updateID,messageID,userID int64,rule executableRule,token string){
	eventID,claimed,err:=p.reserveRuleEvent(ctx,bot.tenantID,groupID,rule.ID,updateID,&messageID,userID,"delete_message")
	if err!=nil{p.logger.Error("reserve Telegram delete rule event","rule_id",rule.ID,"error",err);return}
	if !claimed{return}
	var chatID,telegramBotID int64
	err=p.pool.QueryRow(ctx,`SELECT g.telegram_chat_id,b.telegram_bot_id FROM telegram_groups g
		JOIN telegram_bots b ON b.id=g.bot_id AND b.tenant_id=g.tenant_id
		WHERE g.tenant_id=$1 AND g.id=$2 AND b.status='active' AND g.status='active'`,bot.tenantID,groupID).Scan(&chatID,&telegramBotID)
	if err!=nil{p.finishRuleEvent(ctx,eventID,"failed","could not load active Bot and group");return}
	var member telegramChatMember
	if err:=p.callTelegram(ctx,token,"getChatMember",map[string]any{"chat_id":chatID,"user_id":telegramBotID},&member);err!=nil{
		p.finishRuleEvent(ctx,eventID,"failed","could not verify Bot delete permission: "+err.Error());return
	}
	canDelete:=member.Status=="creator" || (member.Status=="administrator"&&member.CanDeleteMessages)
	if !canDelete{p.finishRuleEvent(ctx,eventID,"skipped","Bot lacks administrator can_delete_messages permission");return}
	if err:=p.callTelegram(ctx,token,"deleteMessage",map[string]any{"chat_id":chatID,"message_id":messageID},nil);err!=nil{
		p.finishRuleEvent(ctx,eventID,"failed","Telegram deleteMessage failed: "+err.Error());return
	}
	p.finishRuleEvent(ctx,eventID,"succeeded","message deleted")
}

func (p *Poller) executeRuleNotice(ctx context.Context,bot managedBot,groupID string,updateID,messageID,userID int64,rule executableRule,action,text,token string){
	eventID,claimed,err:=p.reserveRuleEvent(ctx,bot.tenantID,groupID,rule.ID,updateID,&messageID,userID,action)
	if err!=nil{p.logger.Error("reserve Telegram notice rule event","rule_id",rule.ID,"error",err);return}
	if !claimed{return}
	var chatID int64
	err=p.pool.QueryRow(ctx,`SELECT g.telegram_chat_id FROM telegram_groups g JOIN telegram_bots b ON b.id=g.bot_id AND b.tenant_id=g.tenant_id
		WHERE g.tenant_id=$1 AND g.id=$2 AND g.status='active' AND b.status='active'`,bot.tenantID,groupID).Scan(&chatID)
	if err!=nil{p.finishRuleEvent(ctx,eventID,"failed","could not load active group");return}
	if strings.TrimSpace(text)==""{p.finishRuleEvent(ctx,eventID,"skipped","notice text is empty");return}
	err=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":text},nil)
	if err!=nil{p.finishRuleEvent(ctx,eventID,"failed","Telegram sendMessage failed: "+err.Error());return}
	p.finishRuleEvent(ctx,eventID,"succeeded","notice sent")
}

func (p *Poller) callTelegram(ctx context.Context,token,method string,payload any,result any)error{
	body,err:=json.Marshal(payload);if err!=nil{return err}
	baseURL:=p.apiBaseURL
	if baseURL=="" { baseURL="https://api.telegram.org" }
	endpoint:=strings.TrimRight(baseURL,"/")+"/bot"+url.PathEscape(token)+"/"+method
	req,err:=http.NewRequestWithContext(ctx,http.MethodPost,endpoint,bytes.NewReader(body));if err!=nil{return err}
	req.Header.Set("Content-Type","application/json")
	response,err:=p.client.Do(req);if err!=nil{return fmt.Errorf("Telegram request failed: %w",err)}
	defer response.Body.Close()
	data,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return err}
	var envelope struct{OK bool `json:"ok"`;Description string `json:"description"`;Result json.RawMessage `json:"result"`}
	if err:=json.Unmarshal(data,&envelope);err!=nil{return fmt.Errorf("decode Telegram response: %w",err)}
	if response.StatusCode!=http.StatusOK||!envelope.OK{return fmt.Errorf("Telegram API rejected %s: %s",method,envelope.Description)}
	if result!=nil{return json.Unmarshal(envelope.Result,result)}
	return nil
}

// Every Bot in a tenant shares the same owner: the active platform user whose
// Telegram identity is linked to the sender. No per-Bot owner configuration exists.
func (p *Poller) requireGroupOwner(ctx context.Context,bot managedBot,telegramUserID,chatID int64,token string) bool {
	var hasLinkedOwner,isOwner bool
	query:= "SELECT EXISTS(SELECT 1 FROM users WHERE tenant_id=$1 AND status='active' AND telegram_id IS NOT NULL), EXISTS(SELECT 1 FROM users WHERE tenant_id=$1 AND status='active' AND telegram_id=$2)"
	if err:=p.pool.QueryRow(ctx,query,bot.tenantID,telegramUserID).Scan(&hasLinkedOwner,&isOwner);err!=nil {
		p.logger.Error("verify unified Telegram group owner","tenant_id",bot.tenantID,"error",err)
		_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"暂时无法验证平台主人身份，请稍后重试。"},nil)
		return false
	}
	switch groupOwnerAccess(hasLinkedOwner,isOwner) {
	case "unbound":
		_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"当前平台账号尚未绑定 Telegram 主人账号。请先登录 TG-Group 网站，进入“账号设置 → 账号关联”完成绑定。绑定后，你的全部群管 Bot 都会自动使用这个账号作为统一主人，无需逐个设置。"},nil)
		return false
	case "not_owner":
		_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"此操作仅限已绑定 TG-Group 的主人账号使用。"},nil)
		return false
	default:
		return true
	}
}

func groupOwnerAccess(hasLinkedOwner,isOwner bool) string {
	if !hasLinkedOwner { return "unbound" }
	if !isOwner { return "not_owner" }
	return "owner"
}

func (p *Poller) sendGroupMenu(ctx context.Context,chatID,messageID int64,token string) {
	keyboard:=map[string]any{"inline_keyboard":[][]map[string]string{
		{{"text":"规则管理","callback_data":"group:rules"},{"text":"成员管理","callback_data":"group:members"}},
		{{"text":"消息管理","callback_data":"group:messages"},{"text":"检查 Bot 权限","callback_data":"group:status"}},
		{{"text":"群公告","callback_data":"group:announce"},{"text":"群组设置","callback_data":"group:settings"}},
		{{"text":"刷新菜单","callback_data":"group:home"}},
	}}
	payload:=map[string]any{"chat_id":chatID,"text":"TG-Group 群管 Bot 已连接。请直接点击下方按钮管理本群规则、成员和 Bot 权限。所有管理操作仅限已绑定的平台主人。"}
	if messageID>0 { payload["reply_to_message_id"]=messageID }
	payload["reply_markup"]=keyboard
	_ = p.callTelegram(ctx,token,"sendMessage",payload,nil)
}

func (p *Poller) handleGroupCallback(ctx context.Context,bot managedBot,groupID string,query telegramCallbackQuery,token string) {
	_ = p.callTelegram(ctx,token,"answerCallbackQuery",map[string]any{"callback_query_id":query.ID},nil)
	if query.Message==nil || query.From.ID<=0 { return }
	if !p.requireGroupOwner(ctx,bot,query.From.ID,query.Message.Chat.ID,token) { return }
	switch {
	case query.Data=="group:rules": p.sendGroupRules(ctx,bot,groupID,query.Message.Chat.ID,token)
	case query.Data=="group:members" || strings.HasPrefix(query.Data,"group:members:page:"):
		offset:=0
		if query.Data!="group:members" { parsed,err:=strconv.Atoi(strings.TrimPrefix(query.Data,"group:members:page:"));if err!=nil||parsed<0||parsed>100000{return};offset=parsed }
		p.sendGroupMembers(ctx,bot,groupID,query.Message.Chat.ID,token,offset)
	case query.Data=="group:rule-events": p.sendGroupRuleEvents(ctx,bot,groupID,query.Message.Chat.ID,token)
	case query.Data=="group:messages": p.sendGroupMessages(ctx,bot,groupID,query.Message.Chat.ID,token)
	case query.Data=="group:announce": p.beginGroupAnnouncement(ctx,bot,groupID,query.From.ID,query.Message.Chat.ID,token)
	case query.Data=="group:settings": p.sendGroupSettings(ctx,query.Message.Chat.ID,token)
	case strings.HasPrefix(query.Data,"group:settings:"): p.handleGroupSettings(ctx,bot,groupID,query,token)
	case strings.HasPrefix(query.Data,"group:ec:"): p.beginGroupRuleCustomEdit(ctx,bot,groupID,query.From.ID,query.Message.Chat.ID,query.Data,token)
	case strings.HasPrefix(query.Data,"group:msg:"): p.handleGroupMessageAction(ctx,bot,groupID,query,token)
	case strings.HasPrefix(query.Data,"group:create:"): p.createGroupRulePreset(ctx,bot,groupID,query.Message.Chat.ID,strings.TrimPrefix(query.Data,"group:create:"),token)
	case strings.HasPrefix(query.Data,"group:ed:") || strings.HasPrefix(query.Data,"group:es:"): p.handleGroupRuleEdit(ctx,bot,groupID,query.Message.Chat.ID,query.Data,token)
		case strings.HasPrefix(query.Data,"group:delete:"): p.handleGroupRuleDelete(ctx,bot,groupID,query.Message.Chat.ID,query.Data,token)
	case strings.HasPrefix(query.Data,"group:member:detail:"):
		p.sendGroupMemberDetail(ctx,bot,groupID,query.Message.Chat.ID,query.Data,token)
	case query.Data=="group:status": p.sendGroupStatus(ctx,bot,groupID,query.Message.Chat.ID,token)
	case query.Data=="group:help" || query.Data=="group:home": p.sendGroupMenu(ctx,query.Message.Chat.ID,0,token)
	case strings.HasPrefix(query.Data,"group:toggle:"):
		ruleID:=strings.TrimPrefix(query.Data,"group:toggle:")
		if !validRuleUUID(ruleID) { return }
		var enabled bool
		err:=p.pool.QueryRow(ctx,"UPDATE telegram_group_rules SET enabled=NOT enabled,updated_at=now() WHERE tenant_id=$1 AND group_id=$2 AND id=$3::uuid RETURNING enabled",bot.tenantID,groupID,ruleID).Scan(&enabled)
		if err!=nil { p.logger.Warn("toggle Telegram group rule failed","group_id",groupID,"rule_id",ruleID,"error",err);_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":"规则更新失败，请刷新网站确认规则是否仍存在。"},nil);return }
		p.sendGroupRules(ctx,bot,groupID,query.Message.Chat.ID,token)
	default:
		if stage,action,minutes,targetID,ok:=parseGroupMemberMuteConfirm(query.Data);ok {
			if stage!="confirm" { return }
			p.executeGroupMemberAction(ctx,bot,groupID,query.From.ID,query.Message.Chat.ID,targetID,action,minutes,token)
			return
		}
		stage,action,targetID,ok:=parseGroupMemberCallback(query.Data)
		if !ok { return }
		if stage=="ask" {
			if action=="mute" {
				keyboard:=[][]map[string]string{
					{{"text":"禁言 10 分钟","callback_data":fmt.Sprintf("group:member:confirm:mute:10:%d",targetID)},{"text":"禁言 30 分钟","callback_data":fmt.Sprintf("group:member:confirm:mute:30:%d",targetID)}},
					{{"text":"禁言 1 小时","callback_data":fmt.Sprintf("group:member:confirm:mute:60:%d",targetID)},{"text":"禁言 24 小时","callback_data":fmt.Sprintf("group:member:confirm:mute:1440:%d",targetID)}},
					{{"text":"取消","callback_data":"group:members"}},
				}
				_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":fmt.Sprintf("请选择对用户 %d 的禁言时长。",targetID),"reply_markup":map[string]any{"inline_keyboard":keyboard}},nil)
				return
			}
			label:=map[string]string{"ban":"封禁","unmute":"解除禁言","unban":"解除封禁"}[action]
			keyboard:=map[string]any{"inline_keyboard":[][]map[string]string{
				{{"text":"确认"+label,"callback_data":fmt.Sprintf("group:member:confirm:%s:%d",action,targetID)}},
				{{"text":"取消","callback_data":"group:members"}},
			}}
			_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":fmt.Sprintf("请确认对 Telegram 用户 %d 执行：%s。",targetID,label),"reply_markup":keyboard},nil)
			return
		}
		p.executeGroupMemberAction(ctx,bot,groupID,query.From.ID,query.Message.Chat.ID,targetID,action,0,token)
	}
}


func parseGroupMemberMuteConfirm(data string)(stage,action string,minutes int,targetID int64,ok bool) {
	parts:=strings.Split(data,":")
	if len(parts)!=6 || parts[0]!="group" || parts[1]!="member" || parts[2]!="confirm" || parts[3]!="mute" { return "", "", 0, 0, false }
	minutes,err:=strconv.Atoi(parts[4]);if err!=nil || (minutes!=10 && minutes!=30 && minutes!=60 && minutes!=1440) { return "", "", 0, 0, false }
	targetID,err=strconv.ParseInt(parts[5],10,64);if err!=nil || targetID<=0 { return "", "", 0, 0, false }
	return "confirm","mute",minutes,targetID,true
}

func (p *Poller) executeGroupMemberAction(ctx context.Context,bot managedBot,groupID string,telegramUserID,chatID,targetID int64,action string,duration int,token string) {
	if p.botService==nil { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"成员管理服务暂不可用，请稍后重试。"},nil);return }
	var actorID string
	if err:=p.pool.QueryRow(ctx,"SELECT id::text FROM users WHERE tenant_id=$1 AND telegram_id=$2 AND status='active'",bot.tenantID,telegramUserID).Scan(&actorID);err!=nil {
		p.logger.Warn("resolve Telegram group owner for member action","tenant_id",bot.tenantID,"error",err)
		_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"无法确认平台主人账号，请重新绑定 Telegram 身份后重试。"},nil);return
	}
	if err:=p.botService.ModerateGroupMember(ctx,bot.tenantID,actorID,groupID,targetID,action,duration);err!=nil {
		p.logger.Warn("Telegram inline member moderation failed","group_id",groupID,"telegram_user_id",targetID,"action",action,"error",err)
		_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"操作未完成。请检查 Bot 是否拥有管理成员权限、目标成员是否为普通成员，并在网站成员管理页查看详细状态。"},nil);return
	}
	label:=map[string]string{"mute":fmt.Sprintf("禁言 %d 分钟",duration),"ban":"封禁","unmute":"解除禁言","unban":"解除封禁"}[action]
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":fmt.Sprintf("已完成：用户 %d，%s。操作已记录审计日志。",targetID,label)},nil)
	p.sendGroupMembers(ctx,bot,groupID,chatID,token,0)
}

func parseGroupMessageActionCallback(data string)(action,stage string,messageID int64,ok bool) {
	parts:=strings.Split(data,":")
	if len(parts)!=5 || parts[0]!="group" || parts[1]!="msg" { return "", "", 0, false }
	if parts[2]!="delete" && parts[2]!="pin" { return "", "", 0, false }
	if (parts[2]=="delete" && parts[3]!="ask" && parts[3]!="confirm") || (parts[2]=="pin" && parts[3]!="run") { return "", "", 0, false }
	messageID,err:=strconv.ParseInt(parts[4],10,64);if err!=nil || messageID<=0 { return "", "", 0, false }
	return parts[2],parts[3],messageID,true
}

func (p *Poller) sendGroupMessages(ctx context.Context,bot managedBot,groupID string,chatID int64,token string) {
	rows,err:=p.pool.Query(ctx,`SELECT telegram_message_id,sender_display_name,message_type,message_text,received_at
		FROM telegram_message_events WHERE tenant_id=$1 AND group_id=$2 AND telegram_message_id>0
		ORDER BY received_at DESC,id DESC LIMIT 10`,bot.tenantID,groupID)
	if err!=nil { p.logger.Warn("load recent Telegram messages","group_id",groupID,"error",err);return }
	defer rows.Close()
	lines:=[]string{"最近 10 条已采集消息（可删除或置顶）："}
	keyboard:=[][]map[string]string{}
	count:=0
	for rows.Next() {
		var id int64;var sender,kind,content string;var at time.Time
		if err:=rows.Scan(&id,&sender,&kind,&content,&at);err!=nil { p.logger.Warn("read recent Telegram messages","error",err);return }
		count++
		if sender=="" { sender="未知成员" };if content=="" { content="["+kind+"]" }
		if len([]rune(content))>55 { content=string([]rune(content)[:55])+"…" }
		lines=append(lines,fmt.Sprintf("%d. %s：%s（%s）",count,sender,content,at.Format("15:04")))
		keyboard=append(keyboard,[]map[string]string{
			{"text":fmt.Sprintf("删除 #%d",id),"callback_data":fmt.Sprintf("group:msg:delete:ask:%d",id)},
			{"text":fmt.Sprintf("置顶 #%d",id),"callback_data":fmt.Sprintf("group:msg:pin:run:%d",id)},
		})
	}
	if err:=rows.Err();err!=nil { p.logger.Warn("iterate recent Telegram messages","error",err);return }
	if count==0 { lines=append(lines,"暂无已采集消息。") }
	keyboard=append(keyboard,[]map[string]string{{"text":"取消当前置顶","callback_data":"group:msg:unpin:run:1"},{"text":"刷新","callback_data":"group:messages"}})
	keyboard=append(keyboard,[]map[string]string{{"text":"返回菜单","callback_data":"group:home"}})
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":strings.Join(lines,"\n"),"reply_markup":map[string]any{"inline_keyboard":keyboard}},nil)
}

func (p *Poller) handleGroupMessageAction(ctx context.Context,bot managedBot,groupID string,query telegramCallbackQuery,token string) {
	data:=query.Data
	if data=="group:msg:unpin:run:1" { p.performGroupMessageAction(ctx,bot,groupID,query.From.ID,query.Message.Chat.ID,"unpin",0,token);return }
	action,stage,messageID,ok:=parseGroupMessageActionCallback(data);if !ok{return}
	if stage=="ask" {
		keyboard:=map[string]any{"inline_keyboard":[][]map[string]string{
			{{"text":"确认删除","callback_data":fmt.Sprintf("group:msg:delete:confirm:%d",messageID)}},
			{{"text":"取消","callback_data":"group:messages"}},
		}}
		_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":fmt.Sprintf("确定删除消息 #%d 吗？此操作不可撤销。",messageID),"reply_markup":keyboard},nil);return
	}
	p.performGroupMessageAction(ctx,bot,groupID,query.From.ID,query.Message.Chat.ID,action,messageID,token)
}

func (p *Poller) performGroupMessageAction(ctx context.Context,bot managedBot,groupID string,telegramUserID,chatID int64,action string,messageID int64,token string) {
	var targetChatID,telegramBotID int64
	if err:=p.pool.QueryRow(ctx,`SELECT g.telegram_chat_id,b.telegram_bot_id FROM telegram_groups g JOIN telegram_bots b ON b.id=g.bot_id AND b.tenant_id=g.tenant_id
		WHERE g.tenant_id=$1 AND g.id=$2 AND g.status='active' AND b.status='active'`,bot.tenantID,groupID).Scan(&targetChatID,&telegramBotID);err!=nil{return}
	var botMember telegramChatMember
	if err:=p.callTelegram(ctx,token,"getChatMember",map[string]any{"chat_id":targetChatID,"user_id":telegramBotID},&botMember);err!=nil {
		_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"无法确认 Bot 当前管理权限。"},nil);return
	}
	method:="";payload:=map[string]any{"chat_id":targetChatID}
	switch action {
	case "delete":
		if botMember.Status!="creator" && (botMember.Status!="administrator" || !botMember.CanDeleteMessages) { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"Bot 缺少删除消息权限。"},nil);return }
		var exists bool
		if err:=p.pool.QueryRow(ctx,"SELECT EXISTS(SELECT 1 FROM telegram_message_events WHERE tenant_id=$1 AND group_id=$2 AND telegram_message_id=$3)",bot.tenantID,groupID,messageID).Scan(&exists);err!=nil || !exists{return}
		method="deleteMessage";payload["message_id"]=messageID
	case "pin":
		if botMember.Status!="creator" && (botMember.Status!="administrator" || !botMember.CanPinMessages) { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"Bot 缺少置顶消息权限。"},nil);return }
		var exists bool
		if err:=p.pool.QueryRow(ctx,"SELECT EXISTS(SELECT 1 FROM telegram_message_events WHERE tenant_id=$1 AND group_id=$2 AND telegram_message_id=$3)",bot.tenantID,groupID,messageID).Scan(&exists);err!=nil || !exists{return}
		method="pinChatMessage";payload["message_id"]=messageID;payload["disable_notification"]=true
	case "unpin":
		if botMember.Status!="creator" && (botMember.Status!="administrator" || !botMember.CanPinMessages) { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"Bot 缺少取消置顶权限。"},nil);return }
		method="unpinChatMessage"
	default:return
	}
	err:=p.callTelegram(ctx,token,method,payload,nil)
	if err!=nil { p.logger.Warn("Telegram group message action failed","group_id",groupID,"action",action,"message_id",messageID,"error",err);_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"操作失败，请确认消息仍存在且 Bot 权限充足。"},nil);return }
	var actorID string
	if err:=p.pool.QueryRow(ctx,"SELECT id::text FROM users WHERE tenant_id=$1 AND telegram_id=$2 AND status='active'",bot.tenantID,telegramUserID).Scan(&actorID);err==nil {
		_,auditErr:=p.pool.Exec(ctx,"INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id) VALUES($1,$2,$3,'telegram_group_message',$4)",bot.tenantID,actorID,"group.message."+action,fmt.Sprintf("%s:%d",groupID,messageID))
		if auditErr!=nil { p.logger.Error("write Telegram message action audit","group_id",groupID,"error",auditErr) }
	}
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":map[string]string{"delete":"消息已删除。","pin":"消息已置顶。","unpin":"已取消当前置顶消息。"}[action]},nil)
	p.sendGroupMessages(ctx,bot,groupID,chatID,token)
}

// parseGroupMemberCallback validates the compact callback payload before any action.
func parseGroupMemberCallback(data string) (stage,action string,targetID int64,ok bool) {
	parts:=strings.Split(data,":")
	if len(parts)!=5 || parts[0]!="group" || parts[1]!="member" { return "", "", 0, false }
	if parts[2]!="ask" && parts[2]!="confirm" { return "", "", 0, false }
	if parts[3]!="mute" && parts[3]!="ban" && parts[3]!="unmute" && parts[3]!="unban" { return "", "", 0, false }
	id,err:=strconv.ParseInt(parts[4],10,64)
	if err!=nil || id<=0 { return "", "", 0, false }
	return parts[2],parts[3],id,true
}

func (p *Poller) sendGroupMembers(ctx context.Context,bot managedBot,groupID string,chatID int64,token string,offset int) {
	rows,err:=p.pool.Query(ctx,`SELECT telegram_user_id,username,first_name,last_name,status FROM telegram_group_members
		WHERE tenant_id=$1 AND group_id=$2 AND status IN ('member','restricted','kicked') AND is_bot=false
		ORDER BY last_seen_at DESC,id DESC LIMIT 11 OFFSET $3`,bot.tenantID,groupID,offset)
	if err!=nil { p.logger.Error("load Telegram members for inline menu","group_id",groupID,"error",err);return }
	defer rows.Close()
	lines:=[]string{"群成员管理（最近活跃的普通成员）："}
	keyboard:=[][]map[string]string{}
	count:=0
	hasNext:=false
	for rows.Next() {
		if count==10 { hasNext=true;break }
		var id int64
		var username,firstName,lastName,status string
		if err:=rows.Scan(&id,&username,&firstName,&lastName,&status);err!=nil { p.logger.Error("read Telegram member for inline menu","error",err);return }
		name:=strings.TrimSpace(firstName+" "+lastName)
		if username!="" { name+=" (@"+username+")" }
		if name=="" { name=strconv.FormatInt(id,10) }
		count++
		lines=append(lines,fmt.Sprintf("%d. %s（ID: %d）",count,name,id))
		switch status {
		case "member":
			keyboard=append(keyboard,[]map[string]string{
				{"text":"禁言30分","callback_data":fmt.Sprintf("group:member:ask:mute:%d",id)},
				{"text":"封禁","callback_data":fmt.Sprintf("group:member:ask:ban:%d",id)},
			})
		case "restricted":
			keyboard=append(keyboard,[]map[string]string{
				{"text":"解除禁言","callback_data":fmt.Sprintf("group:member:ask:unmute:%d",id)},
				{"text":"封禁","callback_data":fmt.Sprintf("group:member:ask:ban:%d",id)},
			})
		case "kicked":
			keyboard=append(keyboard,[]map[string]string{{"text":"解除封禁","callback_data":fmt.Sprintf("group:member:ask:unban:%d",id)}})
		}
		keyboard=append(keyboard,[]map[string]string{{"text":"成员详情","callback_data":fmt.Sprintf("group:member:detail:%d",id)}})
	}
	if err:=rows.Err();err!=nil { p.logger.Error("iterate Telegram members for inline menu","group_id",groupID,"error",err);return }
	if count==0 { lines=append(lines,"暂无已采集的普通成员。成员数据会在 Bot 收到群成员更新后逐步建立。") }
	page:=[][]map[string]string{}
	if offset>0 { prev:=offset-10;if prev<0{prev=0};page=append(page,[]map[string]string{{"text":"上一页","callback_data":fmt.Sprintf("group:members:page:%d",prev)}}) }
	if hasNext { page=append(page,[]map[string]string{{"text":"下一页","callback_data":fmt.Sprintf("group:members:page:%d",offset+10)}}) }
	keyboard=append(keyboard,page...)
	keyboard=append(keyboard,[]map[string]string{{"text":"刷新","callback_data":fmt.Sprintf("group:members:page:%d",offset)},{"text":"返回菜单","callback_data":"group:home"}})
	_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":fmt.Sprintf("%s\n第 %d 页",strings.Join(lines,"\n"),offset/10+1),"reply_markup":map[string]any{"inline_keyboard":keyboard}},nil)
}

func parseGroupMemberDetailCallback(data string) (int64,bool) {
	const prefix="group:member:detail:"
	if !strings.HasPrefix(data,prefix) { return 0,false }
	id,err:=strconv.ParseInt(strings.TrimPrefix(data,prefix),10,64)
	if err!=nil || id<=0 { return 0,false }
	return id,true
}

func (p *Poller) sendGroupMemberDetail(ctx context.Context,bot managedBot,groupID string,chatID int64,rawUserID,token string) {
	userID,ok:=parseGroupMemberDetailCallback(rawUserID)
	if !ok { return }
	var username,firstName,lastName,status string
	var joinedAt,lastSeenAt *time.Time
	var isAnonymous,canManageChat,canDeleteMessages,canRestrictMembers,canInviteUsers,canPinMessages,canManageTopics bool
	err:=p.pool.QueryRow(ctx,`SELECT username,first_name,last_name,status,joined_at,last_seen_at,is_anonymous,
		can_manage_chat,can_delete_messages,can_restrict_members,can_invite_users,can_pin_messages,can_manage_topics
		FROM telegram_group_members WHERE tenant_id=$1 AND group_id=$2 AND telegram_user_id=$3 AND is_bot=false`,
		bot.tenantID,groupID,userID).Scan(&username,&firstName,&lastName,&status,&joinedAt,&lastSeenAt,&isAnonymous,
		&canManageChat,&canDeleteMessages,&canRestrictMembers,&canInviteUsers,&canPinMessages,&canManageTopics)
	if err!=nil { p.logger.Warn("load Telegram group member detail","group_id",groupID,"telegram_user_id",userID,"error",err);_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"成员资料暂不可用，可能尚未采集到该成员。"},nil);return }
	name:=strings.TrimSpace(firstName+" "+lastName)
	if name=="" { name=strconv.FormatInt(userID,10) }
	lines:=[]string{"成员详情", "姓名："+name,fmt.Sprintf("Telegram ID：%d",userID),"用户名："+username,"状态："+status}
	if joinedAt!=nil { lines=append(lines,"加入时间："+joinedAt.Format("2006-01-02 15:04:05")) } else { lines=append(lines,"加入时间：暂无记录") }
	if lastSeenAt!=nil { lines=append(lines,"最近活跃："+lastSeenAt.Format("2006-01-02 15:04:05")) } else { lines=append(lines,"最近活跃：暂无记录") }
	lines=append(lines,"管理员权限快照：")
	lines=append(lines,fmt.Sprintf("管理群组：%s｜删除消息：%s｜限制成员：%s",yesNo(canManageChat),yesNo(canDeleteMessages),yesNo(canRestrictMembers)))
	lines=append(lines,fmt.Sprintf("邀请成员：%s｜置顶消息：%s｜管理话题：%s｜匿名管理员：%s",yesNo(canInviteUsers),yesNo(canPinMessages),yesNo(canManageTopics),yesNo(isAnonymous)))
	resourceID:=fmt.Sprintf("%s:%d",groupID,userID)
	rows,err:=p.pool.Query(ctx,`SELECT a.action,COALESCE(u.display_name,u.email,'未知'),a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id=a.actor_user_id
		WHERE a.tenant_id=$1 AND a.resource_type='telegram_group_member' AND a.resource_id=$2
		ORDER BY a.created_at DESC,a.id DESC LIMIT 5`,bot.tenantID,resourceID)
	if err!=nil { p.logger.Warn("load Telegram member moderation history","group_id",groupID,"telegram_user_id",userID,"error",err) } else {
		lines=append(lines,"最近处罚记录：")
		count:=0
		for rows.Next() {
			var action,actor string
			var at time.Time
			if scanErr:=rows.Scan(&action,&actor,&at);scanErr!=nil { p.logger.Warn("read Telegram member moderation history","error",scanErr);break }
			count++
			lines=append(lines,fmt.Sprintf("%d. %s｜操作人：%s｜%s",count,action,actor,at.Format("2006-01-02 15:04")))
		}
		if count==0 { lines=append(lines,"暂无处罚记录。") }
		rows.Close()
	}
	keyboard:=map[string]any{"inline_keyboard":[][]map[string]string{{{"text":"返回成员列表","callback_data":"group:members"}}}}
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":strings.Join(lines,"\n"),"reply_markup":keyboard},nil)
}

func validRuleUUID(value string) bool {
	if len(value)!=36 { return false }
	for i,r:=range value {
		switch i { case 8,13,18,23: if r!='-' { return false }; default: if !(r>='0'&&r<='9'||r>='a'&&r<='f'||r>='A'&&r<='F') { return false } }
	}
	return true
}

func (p *Poller) sendGroupRules(ctx context.Context,bot managedBot,groupID string,chatID int64,token string) {
	rows,err:=p.pool.Query(ctx,"SELECT id::text,name,rule_type,enabled FROM telegram_group_rules WHERE tenant_id=$1 AND group_id=$2 ORDER BY priority,created_at LIMIT 20",bot.tenantID,groupID)
	if err!=nil { p.logger.Error("load rules for Telegram group menu","group_id",groupID,"error",err);return }
	defer rows.Close()
	lines:=[]string{"群组规则管理（最多显示 20 条）："}
	keyboard:=[][]map[string]string{}
	count:=0
	for rows.Next(){
		var id,name,kind string;var enabled bool
		if err:=rows.Scan(&id,&name,&kind,&enabled);err!=nil{p.logger.Error("read rules for Telegram group menu","error",err);return}
		count++;state:="已停用";action:="启用";if enabled{state="运行中";action="停用"}
		lines=append(lines,fmt.Sprintf("%d. %s（%s）- %s",count,name,kind,state))
		keyboard=append(keyboard,[]map[string]string{{"text":action+"："+name,"callback_data":"group:toggle:"+id},{"text":"编辑："+name,"callback_data":"group:ed:"+id}})
		keyboard=append(keyboard,[]map[string]string{{"text":"删除："+name,"callback_data":"group:delete:ask:"+id+":rule"}})
	}
	if err:=rows.Err();err!=nil{p.logger.Error("iterate Telegram group rules for Telegram group menu","group_id",groupID,"error",err);return}
	if count==0 { lines=append(lines,"暂无群组规则。点击下方预设按钮即可快速添加。") }
	keyboard=append(keyboard,[]map[string]string{{"text":"添加欢迎语","callback_data":"group:create:welcome"},{"text":"离群通知","callback_data":"group:create:leave"}})
	keyboard=append(keyboard,[]map[string]string{{"text":"广告拦截","callback_data":"group:create:anti_ad"},{"text":"关键词过滤","callback_data":"group:create:keyword"}})
	keyboard=append(keyboard,[]map[string]string{{"text":"刷屏限制","callback_data":"group:create:anti_spam"},{"text":"数学入群验证","callback_data":"group:create:verify"}})
	keyboard=append(keyboard,[]map[string]string{{"text":"执行记录","callback_data":"group:rule-events"},{"text":"刷新规则","callback_data":"group:rules"}})
	keyboard=append(keyboard,[]map[string]string{{"text":"返回菜单","callback_data":"group:home"}})
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":strings.Join(lines,"\n"),"reply_markup":map[string]any{"inline_keyboard":keyboard}},nil)
}


func (p *Poller) createGroupRulePreset(ctx context.Context,bot managedBot,groupID string,chatID int64,preset,token string) {
	if p.botService==nil { return }
	type presetRule struct { kind,name,config string }
	presets:=map[string]presetRule{
		"welcome":{"welcome","自动欢迎语","{\"text\":\"欢迎 {user} 加入 {group}！请阅读群规，文明交流。\"}"},
		"leave":{"leave_notice","成员离群通知","{\"text\":\"{user} 离开了群组。\"}"},
		"anti_ad":{"anti_ad","基础广告拦截","{\"patterns\":[\"http://\",\"https://\",\"t.me/\"],\"action\":\"delete\"}"},
		"keyword":{"keyword_filter","关键词过滤示例","{\"words\":[\"加我\",\"私聊我\"],\"action\":\"delete\"}"},
		"anti_spam":{"anti_spam","基础刷屏限制","{\"max_messages\":6,\"window_seconds\":10,\"action\":\"delete\"}"},
		"verify":{"join_verification","数学入群验证","{\"challenge_type\":\"math\",\"timeout_seconds\":180,\"failure_action\":\"kick\",\"required_channels\":[]}"},
	}
	item,ok:=presets[preset];if !ok{return}
	_,err:=p.botService.CreateGroupRule(ctx,bot.tenantID,groupID,bots.GroupRuleInput{RuleType:item.kind,Name:item.name,Enabled:true,Priority:100,Config:[]byte(item.config)})
	if err!=nil {
		if errors.Is(err,bots.ErrGroupRuleConflict) { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"该预设规则已经存在。可在规则列表中启用或停用。"},nil) } else { p.logger.Warn("create Telegram group rule preset","group_id",groupID,"preset",preset,"error",err);_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"创建规则失败，请稍后重试。"},nil) }
		return
	}
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"规则已创建并启用："+item.name},nil)
	p.sendGroupRules(ctx,bot,groupID,chatID,token)
}

// Group rule edits are callback-only and use validated presets. This avoids
// accepting arbitrary message text as a management command.
func parseGroupRuleEditCallback(data string) (stage,ruleID,preset string,ok bool) {
	parts:=strings.Split(data,":")
	if len(parts)==3 && parts[0]=="group" && parts[1]=="ed" && validRuleUUID(parts[2]) {
		return "open",parts[2],"",true
	}
	if len(parts)==4 && parts[0]=="group" && parts[1]=="es" && validRuleUUID(parts[2]) {
		switch parts[3] {
		case "w1","w2","w3","l1","l2","k1","k2","k3","a1","a2","s1","s2","s3","v1","v2","v3","vr","vk":
			return "set",parts[2],parts[3],true
		}
	}
	return "","","",false
}

func groupRuleEditConfig(kind,preset string) []byte {
	switch kind {
	case "welcome":
		switch preset {
		case "w1": return []byte(`{"text":"欢迎 {user} 加入 {group}！请阅读群规，文明交流。"}`)
		case "w2": return []byte(`{"text":"欢迎 {user}！请先阅读群公告，遵守群组规则。"}`)
		case "w3": return []byte(`{"text":"欢迎 {user}（{username}）加入我们！"}`)
		}
	case "leave_notice":
		switch preset {
		case "l1": return []byte(`{"text":"{user} 离开了群组。"}`)
		case "l2": return []byte(`{"text":"成员 {user} 已离开，感谢曾经的参与。"}`)
		}
	case "keyword_filter", "sensitive_word":
		switch preset {
		case "k1": return []byte(`{"words":["加我","私聊我"],"action":"delete"}`)
		case "k2": return []byte(`{"words":["送彩金","稳赚","刷单"],"action":"delete"}`)
		case "k3": return []byte(`{"words":["扫码","代理","推广"],"action":"delete"}`)
		}
	case "anti_ad":
		switch preset {
		case "a1": return []byte(`{"patterns":["http://","https://","t.me/"],"action":"delete"}`)
		case "a2": return []byte(`{"patterns":["加入频道","点击链接","开户链接","私聊领取"],"action":"delete"}`)
		}
	case "anti_spam":
		switch preset {
		case "s1": return []byte(`{"max_messages":10,"window_seconds":10,"action":"delete"}`)
		case "s2": return []byte(`{"max_messages":6,"window_seconds":10,"action":"delete"}`)
		case "s3": return []byte(`{"max_messages":4,"window_seconds":10,"action":"delete"}`)
		}
	case "join_verification":
		switch preset {
		case "v1": return []byte(`{"challenge_type":"math","timeout_seconds":60,"failure_action":"kick","required_channels":[]}`)
		case "v2": return []byte(`{"challenge_type":"math","timeout_seconds":180,"failure_action":"kick","required_channels":[]}`)
		case "v3": return []byte(`{"challenge_type":"math","timeout_seconds":300,"failure_action":"kick","required_channels":[]}`)
		case "vr": return []byte(`{"challenge_type":"math","timeout_seconds":180,"failure_action":"restrict","required_channels":[]}`)
		case "vk": return []byte(`{"challenge_type":"math","timeout_seconds":180,"failure_action":"kick","required_channels":[]}`)
		}
	}
	return nil
}

func (p *Poller) sendGroupSettings(ctx context.Context,chatID int64,token string) {
 keyboard:=[][]map[string]string{
  {{"text":"修改群名称","callback_data":"group:settings:title"},{"text":"修改群简介","callback_data":"group:settings:description"}},
  {{"text":"生成邀请链接","callback_data":"group:settings:invite"}},
  {{"text":"返回菜单","callback_data":"group:home"}},
 }
 _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"群组设置：请选择要执行的操作。","reply_markup":map[string]any{"inline_keyboard":keyboard}},nil)
}
func (p *Poller) handleGroupSettings(ctx context.Context,bot managedBot,groupID string,query telegramCallbackQuery,token string) {
 switch query.Data {
 case "group:settings:title","group:settings:description":
  action,prompt:="group_title","请回复新的群名称（1–128 个字符）。"
  if query.Data=="group:settings:description" { action="group_description";prompt="请回复新的群简介（最多 255 个字符）；发送 clear 可清空简介。" }
  if err:=p.saveGroupPendingInput(ctx,bot,groupID,query.From.ID,action,"");err!=nil { p.logger.Warn("save Telegram group settings input","group_id",groupID,"error",err);return }
  _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":prompt+"\n输入等待 5 分钟。","reply_markup":map[string]any{"force_reply":true}},nil)
 case "group:settings:invite":
  var chatID int64
  if err:=p.pool.QueryRow(ctx,"SELECT telegram_chat_id FROM telegram_groups WHERE tenant_id=$1 AND id=$2 AND status='active'",bot.tenantID,groupID).Scan(&chatID);err!=nil{return}
  var inviteLink string
  if err:=p.callTelegram(ctx,token,"exportChatInviteLink",map[string]any{"chat_id":chatID},&inviteLink);err!=nil {
   _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":"生成邀请链接失败，请检查 Bot 的邀请成员权限。"},nil);return
  }
  _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":query.Message.Chat.ID,"text":"群组邀请链接：\n"+inviteLink},nil)
 default:return
 }
}

func parseGroupRuleCustomEditCallback(data string)(ruleID,field string,ok bool) {
 parts:=strings.Split(data,":")
 if len(parts)!=4 || parts[0]!="group" || parts[1]!="ec" || !validRuleUUID(parts[2]) { return "", "", false }
 if parts[3]!="text" && parts[3]!="words" && parts[3]!="patterns" && parts[3]!="channels" { return "", "", false }
 return parts[2],parts[3],true
}
func (p *Poller) saveGroupPendingInput(ctx context.Context,bot managedBot,groupID string,userID int64,action,ruleID string) error {
 var rule any
 if ruleID!="" { if !validRuleUUID(ruleID) { return bots.ErrInvalidGroupRule }; rule=ruleID }
 _,err:=p.pool.Exec(ctx,"INSERT INTO telegram_group_pending_inputs(tenant_id,group_id,telegram_user_id,action,rule_id,expires_at) VALUES($1,$2,$3,$4,$5,now()+interval '5 minutes') ON CONFLICT(tenant_id,group_id,telegram_user_id) DO UPDATE SET action=EXCLUDED.action,rule_id=EXCLUDED.rule_id,created_at=now(),expires_at=EXCLUDED.expires_at",bot.tenantID,groupID,userID,action,rule)
 return err
}
func (p *Poller) beginGroupRuleCustomEdit(ctx context.Context,bot managedBot,groupID string,userID,chatID int64,data,token string) {
 ruleID,field,ok:=parseGroupRuleCustomEditCallback(data); if !ok || userID<=0 || p.botService==nil { return }
 rules,err:=p.botService.ListGroupRules(ctx,bot.tenantID,groupID); if err!=nil { return }
 var current *bots.GroupRule
 for i:=range rules { if rules[i].ID==ruleID { current=&rules[i]; break } }
 if current==nil { return }
 action,prompt:="",""
 switch field {
 case "text":
  if current.RuleType!="welcome" && current.RuleType!="leave_notice" { return }
  action="rule_text";prompt="请直接回复新的完整文案（最多 4096 个字符）。可用变量：{user}、{username}、{group}。"
 case "words":
  if current.RuleType!="keyword_filter" && current.RuleType!="sensitive_word" { return }
  action="rule_words";prompt="请发送关键词列表，每行一个，也可以用逗号分隔。最多 100 个词，每个不超过 100 个字符。"
 case "patterns":
  if current.RuleType!="anti_ad" { return }
  action="rule_patterns";prompt="请发送广告匹配内容，每行一条，也可以用逗号分隔。最多 100 条，每条不超过 100 个字符。"
 case "channels":
  if current.RuleType!="join_verification" { return }
  action="rule_channels";prompt="请发送必需加入的频道列表，每行一个。支持 @频道用户名 或 -100 开头的频道 ID；最多 10 个。清空请发送 none。"
 }
 if err:=p.saveGroupPendingInput(ctx,bot,groupID,userID,action,ruleID); err!=nil { p.logger.Warn("save Telegram group edit session","group_id",groupID,"error",err); return }
 _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":prompt+"\n本次输入等待 5 分钟；发送后自动校验并保存。","reply_markup":map[string]any{"force_reply":true}},nil)
}
func (p *Poller) beginGroupAnnouncement(ctx context.Context,bot managedBot,groupID string,userID,chatID int64,token string) {
 if userID<=0 { return }
 if err:=p.saveGroupPendingInput(ctx,bot,groupID,userID,"announcement",""); err!=nil { p.logger.Warn("save Telegram group announcement session","group_id",groupID,"error",err); return }
 _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"请直接回复公告正文（最多 4000 个字符）。提交后 Bot 会在群内发布，并在具备置顶权限时自动置顶。5 分钟内有效。","reply_markup":map[string]any{"force_reply":true,"selective":true}},nil)
}
func splitRuleTerms(value string) []string {
 parts:=strings.FieldsFunc(value,func(r rune)bool{return r=='\n'||r=='\r'||r==','||r=='，'||r==';'||r=='；'})
 result:=make([]string,0,len(parts));seen:=map[string]bool{}
 for _,part:=range parts { term:=strings.TrimSpace(part);key:=strings.ToLower(term);if term!=""&&!seen[key] { seen[key]=true;result=append(result,term) } }
 return result
}
func (p *Poller) processPendingGroupInput(ctx context.Context,bot managedBot,groupID string,message *telegramMessage,token string) bool {
 if message==nil || message.From==nil || message.From.ID<=0 { return false }
 var action,ruleID string
 err:=p.pool.QueryRow(ctx,"SELECT action,COALESCE(rule_id::text,'') FROM telegram_group_pending_inputs WHERE tenant_id=$1 AND group_id=$2 AND telegram_user_id=$3 AND expires_at>now()",bot.tenantID,groupID,message.From.ID).Scan(&action,&ruleID)
 if err==pgx.ErrNoRows { _,_=p.pool.Exec(ctx,"DELETE FROM telegram_group_pending_inputs WHERE tenant_id=$1 AND group_id=$2 AND telegram_user_id=$3 AND expires_at<=now()",bot.tenantID,groupID,message.From.ID); return false }
 if err!=nil { p.logger.Warn("load Telegram pending group input","group_id",groupID,"error",err); return false }
 if !p.requireGroupOwner(ctx,bot,message.From.ID,message.Chat.ID,token) { _,_=p.pool.Exec(ctx,"DELETE FROM telegram_group_pending_inputs WHERE tenant_id=$1 AND group_id=$2 AND telegram_user_id=$3",bot.tenantID,groupID,message.From.ID); return true }
 value:=strings.TrimSpace(message.Text); if value=="" { value=strings.TrimSpace(message.Caption) }
 if value=="" || len([]rune(value))>4000 { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"输入不能为空，且不能超过 4000 个字符。请重新发送。"},nil); return true }
 switch action {
 case "group_title","group_description":
  var targetChatID int64
  if err:=p.pool.QueryRow(ctx,"SELECT telegram_chat_id FROM telegram_groups WHERE tenant_id=$1 AND id=$2 AND status='active'",bot.tenantID,groupID).Scan(&targetChatID);err!=nil{return true}
  method:="editChatTitle";auditAction:="group.settings.title"
  if action=="group_description" { method="editChatDescription";auditAction="group.settings.description";if strings.EqualFold(value,"clear"){value=""};if len([]rune(value))>255 { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"群简介最多 255 个字符，请重新发送。"},nil);return true } } else if len([]rune(value))>128 { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"群名称最多 128 个字符，请重新发送。"},nil);return true }
  if action=="group_title" && value=="" { return true }
  if err:=p.callTelegram(ctx,token,method,map[string]any{"chat_id":targetChatID,map[string]string{"group_title":"title","group_description":"description"}[action]:value},nil);err!=nil {
   _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"修改失败，请检查 Bot 是否具备管理群组权限。"},nil);return true
  }
  var actorID string
  if err:=p.pool.QueryRow(ctx,"SELECT id::text FROM users WHERE tenant_id=$1 AND telegram_id=$2 AND status='active'",bot.tenantID,message.From.ID).Scan(&actorID);err==nil {
   _,_=p.pool.Exec(ctx,"INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id) VALUES($1,$2,$3,'telegram_group',$4)",bot.tenantID,actorID,auditAction,groupID)
  }
  _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"群组资料已更新。"},nil)
 case "announcement":
  var targetChatID,botID int64
  if err:=p.pool.QueryRow(ctx,"SELECT g.telegram_chat_id,b.telegram_bot_id FROM telegram_groups g JOIN telegram_bots b ON b.id=g.bot_id AND b.tenant_id=g.tenant_id WHERE g.tenant_id=$1 AND g.id=$2 AND g.status='active' AND b.status='active'",bot.tenantID,groupID).Scan(&targetChatID,&botID); err!=nil { return true }
  var botMember telegramChatMember
  if err:=p.callTelegram(ctx,token,"getChatMember",map[string]any{"chat_id":targetChatID,"user_id":botID},&botMember); err!=nil { return true }
  var sent struct { MessageID int64 "json:\"message_id\"" }
  if err:=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":targetChatID,"text":"📢 群公告\n\n"+value},&sent); err!=nil { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"公告发布失败，请检查 Bot 是否有发送消息权限。"},nil); return true }
  pinned:=botMember.Status=="creator" || (botMember.Status=="administrator"&&botMember.CanPinMessages)
  if pinned { if err:=p.callTelegram(ctx,token,"pinChatMessage",map[string]any{"chat_id":targetChatID,"message_id":sent.MessageID,"disable_notification":false},nil); err!=nil { pinned=false } }
  var actorID string
  if err:=p.pool.QueryRow(ctx,"SELECT id::text FROM users WHERE tenant_id=$1 AND telegram_id=$2 AND status='active'",bot.tenantID,message.From.ID).Scan(&actorID); err==nil {
   _,_=p.pool.Exec(ctx,"INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id) VALUES($1,$2,'group.announcement.publish','telegram_group_message',$3)",bot.tenantID,actorID,fmt.Sprintf("%s:%d",groupID,sent.MessageID))
  }
  status:="公告已发布并置顶。"; if !pinned { status="公告已发布，但 Bot 缺少置顶权限或置顶失败。" }
  _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":status},nil)
 case "rule_text","rule_words","rule_patterns","rule_channels":
  if p.botService==nil || !validRuleUUID(ruleID) { return true }
  rules,err:=p.botService.ListGroupRules(ctx,bot.tenantID,groupID); if err!=nil { return true }
  var current *bots.GroupRule
  for i:=range rules { if rules[i].ID==ruleID { current=&rules[i]; break } }
  if current==nil { return true }
  config:=map[string]any{}
  switch action {
  case "rule_text":
   if current.RuleType!="welcome" && current.RuleType!="leave_notice" { return true }; config["text"]=value
  case "rule_words":
   if current.RuleType!="keyword_filter" && current.RuleType!="sensitive_word" { return true }; config["words"]=splitRuleTerms(value); config["action"]="delete"
  case "rule_patterns":
   if current.RuleType!="anti_ad" { return true }; config["patterns"]=splitRuleTerms(value); config["action"]="delete"
  case "rule_channels":
   if current.RuleType!="join_verification" { return true }
   var existing struct { ChallengeType string "json:\"challenge_type\""; TimeoutSeconds int "json:\"timeout_seconds\""; FailureAction string "json:\"failure_action\"" }
   if json.Unmarshal(current.Config,&existing)!=nil { return true }
   channels:=[]string{};if strings.ToLower(value)!="none" { channels=splitRuleTerms(value) }
   config["challenge_type"]=existing.ChallengeType;config["timeout_seconds"]=existing.TimeoutSeconds;config["failure_action"]=existing.FailureAction;config["required_channels"]=channels
  }
  encoded,err:=json.Marshal(config); if err!=nil { return true }
  _,err=p.botService.UpdateGroupRule(ctx,bot.tenantID,groupID,ruleID,bots.GroupRuleInput{RuleType:current.RuleType,Name:current.Name,Enabled:current.Enabled,Priority:current.Priority,Config:encoded})
  if err!=nil { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"内容未通过规则校验，原配置保持不变。请检查格式后重新发送。"},nil); return true }
  _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":message.Chat.ID,"text":"自定义内容已校验并保存。"},nil)
  p.sendGroupRules(ctx,bot,groupID,message.Chat.ID,token)
 default: return true
 }
 _,_=p.pool.Exec(ctx,"DELETE FROM telegram_group_pending_inputs WHERE tenant_id=$1 AND group_id=$2 AND telegram_user_id=$3",bot.tenantID,groupID,message.From.ID)
 return true
}

func (p *Poller) handleGroupRuleEdit(ctx context.Context,bot managedBot,groupID string,chatID int64,data,token string) {
	stage,ruleID,preset,ok:=parseGroupRuleEditCallback(data)
	if !ok || p.botService==nil { return }
	rules,err:=p.botService.ListGroupRules(ctx,bot.tenantID,groupID)
	if err!=nil { p.logger.Warn("load Telegram group rule for edit","group_id",groupID,"rule_id",ruleID,"error",err); return }
	var current *bots.GroupRule
	for i:=range rules { if rules[i].ID==ruleID { current=&rules[i]; break } }
	if current==nil { _=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"这条规则已不存在，请刷新规则列表。"},nil); return }
	if stage=="open" {
		rows:=[][]map[string]string{}
		add:=func(label,key string){rows=append(rows,[]map[string]string{{"text":label,"callback_data":"group:es:"+ruleID+":"+key}})}
		switch current.RuleType {
		case "welcome": add("欢迎模板：标准","w1");add("欢迎模板：群规提醒","w2");add("欢迎模板：用户名","w3")
		case "leave_notice": add("离群通知：简洁","l1");add("离群通知：完整","l2")
		case "keyword_filter","sensitive_word": add("关键词：引流词","k1");add("关键词：营销词","k2");add("关键词：推广词","k3")
		case "anti_ad": add("广告：链接类","a1");add("广告：推广话术","a2")
		case "anti_spam": add("刷屏：宽松（10条/10秒）","s1");add("刷屏：标准（6条/10秒）","s2");add("刷屏：严格（4条/10秒）","s3")
		case "join_verification": add("验证超时：60秒","v1");add("验证超时：180秒","v2");add("验证超时：300秒","v3");add("失败后：限制发言","vr");add("失败后：移出群组","vk")
		default: return
		}
		switch current.RuleType { case "welcome","leave_notice": rows=append(rows,[]map[string]string{{"text":"自定义输入文案","callback_data":"group:ec:"+ruleID+":text"}}); case "keyword_filter","sensitive_word": rows=append(rows,[]map[string]string{{"text":"自定义关键词","callback_data":"group:ec:"+ruleID+":words"}}); case "anti_ad": rows=append(rows,[]map[string]string{{"text":"自定义广告匹配内容","callback_data":"group:ec:"+ruleID+":patterns"}}); case "join_verification": rows=append(rows,[]map[string]string{{"text":"设置必需加入的频道","callback_data":"group:ec:"+ruleID+":channels"}}) }
		rows=append(rows,[]map[string]string{{"text":"返回规则列表","callback_data":"group:rules"}})
		_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"编辑规则："+current.Name+"\n请选择要应用的配置预设。保存后立即生效；原配置会被所选预设整体替换。","reply_markup":map[string]any{"inline_keyboard":rows}},nil)
		return
	}
	config:=groupRuleEditConfig(current.RuleType,preset)
	if len(config)==0 { return }
	_,err=p.botService.UpdateGroupRule(ctx,bot.tenantID,groupID,ruleID,bots.GroupRuleInput{
		RuleType:current.RuleType,Name:current.Name,Enabled:current.Enabled,Priority:current.Priority,Config:config,
	})
	if err!=nil { p.logger.Warn("update Telegram group rule preset","group_id",groupID,"rule_id",ruleID,"preset",preset,"error",err);_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"规则更新失败，原配置未变更。请稍后重试。"},nil);return }
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"规则配置已更新并保存。"},nil)
	p.sendGroupRules(ctx,bot,groupID,chatID,token)
}

func parseGroupRuleDeleteCallback(data string) (stage,ruleID string,ok bool) {
	parts:=strings.Split(data,":")
	if len(parts)!=5 || parts[0]!="group" || parts[1]!="delete" || (parts[2]!="ask" && parts[2]!="confirm") || !validRuleUUID(parts[3]) || parts[4]!="rule" { return "","",false }
	return parts[2],parts[3],true
}

func (p *Poller) handleGroupRuleDelete(ctx context.Context,bot managedBot,groupID string,chatID int64,data,token string) {
	stage,ruleID,ok:=parseGroupRuleDeleteCallback(data)
	if !ok { return }
	if stage=="ask" {
		_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"确定删除这条群规吗？历史执行记录会保留。","reply_markup":map[string]any{"inline_keyboard":[][]map[string]string{
			{{"text":"确认删除","callback_data":"group:delete:confirm:"+ruleID+":rule"}},
			{{"text":"取消","callback_data":"group:rules"}},
		}}},nil)
		return
	}
	if p.botService==nil{return}
	if err:=p.botService.DeleteGroupRule(ctx,bot.tenantID,groupID,ruleID);err!=nil { p.logger.Warn("delete Telegram group rule","group_id",groupID,"rule_id",ruleID,"error",err);_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"删除失败，该规则可能已不存在。"},nil);return }
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"群规已删除，历史执行记录仍保留。"},nil)
	p.sendGroupRules(ctx,bot,groupID,chatID,token)
}

func (p *Poller) sendGroupRuleEvents(ctx context.Context,bot managedBot,groupID string,chatID int64,token string) {
	if p.botService==nil{return}
	events,err:=p.botService.ListGroupRuleEvents(ctx,bot.tenantID,groupID,20)
	if err!=nil{p.logger.Warn("list Telegram group rule events","group_id",groupID,"error",err);return}
	lines:=[]string{"最近 20 条群规执行记录："}
	if len(events)==0{lines=append(lines,"暂无执行记录。")}
	for i,event:=range events {
		detail:=strings.TrimSpace(event.Detail);if len([]rune(detail))>100{detail=string([]rune(detail)[:100])+"…"}
		lines=append(lines,fmt.Sprintf("%d. %s | %s | %s",i+1,event.Action,event.Status,detail))
	}
	_=p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":strings.Join(lines,"\n"),"reply_markup":map[string]any{"inline_keyboard":[][]map[string]string{
		{{"text":"刷新记录","callback_data":"group:rule-events"},{"text":"返回规则","callback_data":"group:rules"}},
	}}},nil)
}

func (p *Poller) sendGroupStatus(ctx context.Context,bot managedBot,groupID string,chatID int64,token string) {
	var telegramChatID,telegramBotID int64
	err:=p.pool.QueryRow(ctx,"SELECT g.telegram_chat_id,b.telegram_bot_id FROM telegram_groups g JOIN telegram_bots b ON b.id=g.bot_id AND b.tenant_id=g.tenant_id WHERE g.tenant_id=$1 AND g.id=$2 AND g.status='active' AND b.status='active'",bot.tenantID,groupID).Scan(&telegramChatID,&telegramBotID)
	if err!=nil { p.logger.Error("load Bot status for Telegram group menu","group_id",groupID,"error",err);return }
	var member telegramChatMember
	if err:=p.callTelegram(ctx,token,"getChatMember",map[string]any{"chat_id":telegramChatID,"user_id":telegramBotID},&member);err!=nil {
		_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":"暂时无法读取 Bot 权限，请确认 Bot 仍在群内。"},nil);return
	}
	text:=fmt.Sprintf("Bot 状态：%s\n删除消息：%s\n限制成员：%s\n邀请成员：%s\n置顶消息：%s",
		member.Status,yesNo(member.CanDeleteMessages),yesNo(member.CanRestrictMembers),yesNo(member.CanInviteUsers),yesNo(member.CanPinMessages))
	_ = p.callTelegram(ctx,token,"sendMessage",map[string]any{"chat_id":chatID,"text":text},nil)
}

func yesNo(value bool) string { if value { return "具备" }; return "未具备" }
