package telegramruntime

import (
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
)

type executableRule struct {
	ID string
	Type string
	Name string
	Config json.RawMessage
}

func (p *Poller) processMessageRules(ctx context.Context, bot managedBot, groupID string, item update, message *telegramMessage, token string) {
	if message == nil { return }
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
				if rule.Type!="welcome" { continue }
				var cfg struct{Text string `json:"text"`}
				if json.Unmarshal(rule.Config,&cfg)!=nil { continue }
				text:=renderNotice(cfg.Text,member,message.Chat.Title)
				p.executeRuleNotice(ctx,bot,groupID,item.UpdateID,message.MessageID,member.ID,rule,"welcome",text,token)
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
		ON CONFLICT(group_id,telegram_update_id,rule_id,action) DO NOTHING RETURNING id`,
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
	endpoint:="https://api.telegram.org/bot"+url.PathEscape(token)+"/"+method
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
