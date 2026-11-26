package telegramruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seanldnet/tg-group/internal/bots"
)

type Poller struct {
	pool *pgxpool.Pool
	botService *bots.Service
	logger *slog.Logger
	client *http.Client
	apiBaseURL string
}

type managedBot struct {
	id string
	tenantID string
}

type botWorker struct {
	cancel context.CancelFunc
	done chan struct{}
}

type chat struct {
	ID int64 `json:"id"`
	Permissions *chatPermissions `json:"permissions"`
	Type string `json:"type"`
	Title string `json:"title"`
	Username string `json:"username"`
	IsForum bool `json:"is_forum"`
}

type telegramSender struct {
	ID int64 `json:"id"`
	Username string `json:"username"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	IsBot bool `json:"is_bot"`
}

type telegramMessage struct {
	MessageID int64 `json:"message_id"`
	Date int64 `json:"date"`
	Chat chat `json:"chat"`
	From *telegramSender `json:"from"`
	Text string `json:"text"`
	Caption string `json:"caption"`
	Photo []json.RawMessage `json:"photo"`
	Document json.RawMessage `json:"document"`
	Video json.RawMessage `json:"video"`
	Sticker json.RawMessage `json:"sticker"`
	Animation json.RawMessage `json:"animation"`
	Voice json.RawMessage `json:"voice"`
	Audio json.RawMessage `json:"audio"`
	NewChatMembers []telegramSender `json:"new_chat_members"`
	LeftChatMember *telegramSender `json:"left_chat_member"`
}

type telegramChatMember struct {
	User telegramSender `json:"user"`
	Status string `json:"status"`
	IsAnonymous bool `json:"is_anonymous"`
	CanManageChat bool `json:"can_manage_chat"`
	CanDeleteMessages bool `json:"can_delete_messages"`
	CanRestrictMembers bool `json:"can_restrict_members"`
	CanInviteUsers bool `json:"can_invite_users"`
	CanPinMessages bool `json:"can_pin_messages"`
	CanManageTopics bool `json:"can_manage_topics"`
}

type telegramChatMemberUpdate struct {
	Chat chat `json:"chat"`
	Date int64 `json:"date"`
	OldChatMember telegramChatMember `json:"old_chat_member"`
	NewChatMember telegramChatMember `json:"new_chat_member"`
}

type update struct {
	UpdateID int64 `json:"update_id"`
	Message *telegramMessage `json:"message"`
	EditedMessage *telegramMessage `json:"edited_message"`
	ChannelPost *telegramMessage `json:"channel_post"`
	MyChatMember *struct {
		Chat chat `json:"chat"`
		NewChatMember struct { Status string `json:"status"` } `json:"new_chat_member"`
	} `json:"my_chat_member"`
	ChatMember *telegramChatMemberUpdate `json:"chat_member"`
	CallbackQuery *telegramCallbackQuery `json:"callback_query"`
}

type updatesResponse struct {
	OK bool `json:"ok"`
	Result []update `json:"result"`
}

func NewPoller(pool *pgxpool.Pool, botService *bots.Service, logger *slog.Logger) *Poller {
	if logger == nil { logger = slog.Default() }
	return &Poller{pool:pool,botService:botService,logger:logger,client:&http.Client{Timeout:40*time.Second},apiBaseURL:"https://api.telegram.org"}
}

// Run continuously reconciles active Bot records and runs one isolated long-poll loop per Bot.
func (p *Poller) Run(ctx context.Context) {
	workers:=make(map[string]botWorker)
	ticker:=time.NewTicker(15*time.Second)
	defer ticker.Stop()
	defer func(){
		for _,worker:=range workers{worker.cancel()}
		for _,worker:=range workers{<-worker.done}
	}()
	for {
		p.reconcile(ctx,workers)
		select {
		case <-ctx.Done(): return
		case <-ticker.C:
		}
	}
}

func (p *Poller) reconcile(ctx context.Context,workers map[string]botWorker) {
	rows,err:=p.pool.Query(ctx,"SELECT id::text,tenant_id::text FROM telegram_bots WHERE status='active'")
	if err!=nil{p.logger.Error("load active Telegram bots for polling","error",err);return}
	active:=make(map[string]string)
	for rows.Next(){var id,tenantID string;if err:=rows.Scan(&id,&tenantID);err!=nil{p.logger.Error("read active Telegram bot","error",err);continue};active[id]=tenantID}
	rows.Close()
	if err:=rows.Err();err!=nil{p.logger.Error("iterate active Telegram bots","error",err);return}
	for id,worker:=range workers{
		if _,ok:=active[id];!ok{worker.cancel();<-worker.done;delete(workers,id);continue}
		select{case <-worker.done:delete(workers,id);default:}
	}
	for id,tenantID:=range active{
		if _,ok:=workers[id];ok{continue}
		workerCtx,cancel:=context.WithCancel(ctx)
		done:=make(chan struct{})
		workers[id]=botWorker{cancel:cancel,done:done}
		go func(bot managedBot){defer close(done);p.runBot(workerCtx,bot)}(managedBot{id:id,tenantID:tenantID})
	}
}

func (p *Poller) runBot(ctx context.Context,bot managedBot) {
	var token string
	for ctx.Err()==nil {
		var err error
		token,err=p.botService.GetToken(ctx,bot.tenantID,bot.id)
		if err==nil { break }
		p.logger.Warn("Telegram Bot worker could not load token; retrying","bot_id",bot.id,"error",err)
		select{case <-ctx.Done():return;case <-time.After(10*time.Second):}
	}
	if ctx.Err()!=nil{return}
	var offset int64
	if err:=p.pool.QueryRow(ctx,"SELECT next_update_offset FROM telegram_bots WHERE tenant_id=$1 AND id=$2 AND status='active'",bot.tenantID,bot.id).Scan(&offset);err!=nil{return}
	for ctx.Err()==nil {
		p.expireJoinVerificationChallenges(ctx,bot,token)
		batch,err:=p.getUpdates(ctx,token,offset)
		if err!=nil {
			if ctx.Err()!=nil{return}
			p.logger.Warn("Telegram long polling request failed","bot_id",bot.id,"error",err)
			select{case <-ctx.Done():return;case <-time.After(3*time.Second):continue}
		}
		for _,item:=range batch {
			if err:=p.persistUpdate(ctx,bot,item,token);err!=nil {
				p.logger.Error("persist Telegram update failed; offset will not advance","bot_id",bot.id,"update_id",item.UpdateID,"error",err)
				break
			}
			if item.UpdateID>=offset{offset=item.UpdateID+1}
			if _,err:=p.pool.Exec(ctx,"UPDATE telegram_bots SET next_update_offset=$3,updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='active'",bot.tenantID,bot.id,offset);err!=nil {
				p.logger.Error("persist Telegram update offset failed","bot_id",bot.id,"error",err)
				return
			}
		}
	}
}

func (p *Poller) getUpdates(ctx context.Context,token string,offset int64)([]update,error) {
	query:=url.Values{"timeout":{"25"},"allowed_updates":{`["message","edited_message","channel_post","my_chat_member","chat_member","callback_query"]`}}
	if offset>0{query.Set("offset",strconv.FormatInt(offset,10))}
	endpoint:="https://api.telegram.org/bot"+token+"/getUpdates?"+query.Encode()
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,endpoint,nil);if err!=nil{return nil,err}
	response,err:=p.client.Do(req);if err!=nil{return nil,fmt.Errorf("Telegram getUpdates request failed")}
	defer response.Body.Close()
	body,err:=io.ReadAll(io.LimitReader(response.Body,4<<20));if err!=nil{return nil,err}
	if response.StatusCode!=http.StatusOK{return nil,fmt.Errorf("Telegram getUpdates returned HTTP %d",response.StatusCode)}
	var result updatesResponse
	if err:=json.Unmarshal(body,&result);err!=nil{return nil,fmt.Errorf("decode Telegram updates: %w",err)}
	if !result.OK{return nil,errors.New("Telegram getUpdates returned ok=false")}
	return result.Result,nil
}

func messageFromUpdate(item update) *telegramMessage {
	switch {
	case item.Message!=nil:return item.Message
	case item.EditedMessage!=nil:return item.EditedMessage
	case item.ChannelPost!=nil:return item.ChannelPost
	default:return nil
	}
}

func chatFromUpdate(item update) *chat {
	if message:=messageFromUpdate(item);message!=nil{return &message.Chat}
	if item.MyChatMember!=nil{return &item.MyChatMember.Chat}
	if item.ChatMember!=nil{return &item.ChatMember.Chat}
	if item.CallbackQuery!=nil && item.CallbackQuery.Message!=nil{return &item.CallbackQuery.Message.Chat}
	return nil
}

func messageType(message telegramMessage) string {
	switch {
	case message.Text!="":return "text"
	case len(message.Photo)>0:return "photo"
	case len(message.Document)>0:return "document"
	case len(message.Video)>0:return "video"
	case len(message.Sticker)>0:return "sticker"
	case len(message.Animation)>0:return "animation"
	case len(message.Voice)>0:return "voice"
	case len(message.Audio)>0:return "audio"
	default:return "other"
	}
}

func senderDisplayName(sender *telegramSender) string {
	if sender==nil{return ""}
	if sender.FirstName!="" && sender.LastName!=""{return sender.FirstName+" "+sender.LastName}
	if sender.FirstName!=""{return sender.FirstName}
	return sender.LastName
}

func isManagedChat(value *chat) bool {
	return value!=nil && value.ID!=0 && (value.Type=="group" || value.Type=="supergroup" || value.Type=="channel")
}

func groupStatus(item update) string {
	if item.MyChatMember==nil{return "active"}
	switch item.MyChatMember.NewChatMember.Status {
	case "left":return "left"
	case "kicked":return "kicked"
	default:return "active"
	}
}


func (p *Poller) upsertGroupMember(ctx context.Context, bot managedBot, groupID string, member telegramChatMember, joinedAt time.Time) error {
	if member.User.ID == 0 { return nil }
	var joined any
	if !joinedAt.IsZero() { joined = joinedAt }
	_, err := p.pool.Exec(ctx, `INSERT INTO telegram_group_members(
		tenant_id,bot_id,group_id,telegram_user_id,username,first_name,last_name,is_bot,status,
		is_anonymous,can_manage_chat,can_delete_messages,can_restrict_members,can_invite_users,
		can_pin_messages,can_manage_topics,joined_at,last_seen_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			CASE WHEN $9 IN ('member','administrator','creator') THEN COALESCE($17,now()) ELSE NULL END,now(),now())
		ON CONFLICT(group_id,telegram_user_id) DO UPDATE SET
		username=EXCLUDED.username,first_name=EXCLUDED.first_name,last_name=EXCLUDED.last_name,
		is_bot=EXCLUDED.is_bot,status=EXCLUDED.status,is_anonymous=EXCLUDED.is_anonymous,
		can_manage_chat=EXCLUDED.can_manage_chat,can_delete_messages=EXCLUDED.can_delete_messages,
		can_restrict_members=EXCLUDED.can_restrict_members,can_invite_users=EXCLUDED.can_invite_users,
		can_pin_messages=EXCLUDED.can_pin_messages,can_manage_topics=EXCLUDED.can_manage_topics,
		joined_at=CASE
			WHEN EXCLUDED.status IN ('member','administrator','creator') AND telegram_group_members.status IN ('left','kicked') THEN EXCLUDED.joined_at
			ELSE COALESCE(telegram_group_members.joined_at,EXCLUDED.joined_at)
		END,
		last_seen_at=now(),updated_at=now()`,
		bot.tenantID,bot.id,groupID,member.User.ID,member.User.Username,member.User.FirstName,member.User.LastName,
		member.User.IsBot,member.Status,member.IsAnonymous,member.CanManageChat,member.CanDeleteMessages,
		member.CanRestrictMembers,member.CanInviteUsers,member.CanPinMessages,member.CanManageTopics,joined)
	if err != nil { return fmt.Errorf("upsert Telegram group member: %w", err) }
	return nil
}

func (p *Poller) persistUpdate(ctx context.Context,bot managedBot,item update,token string)error {
	value:=chatFromUpdate(item)
	if !isManagedChat(value){return nil}
	var groupID string
	err:=p.pool.QueryRow(ctx,`INSERT INTO telegram_groups(tenant_id,bot_id,telegram_chat_id,title,username,chat_type,is_forum,status,last_seen_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,now())
		ON CONFLICT(bot_id,telegram_chat_id) DO UPDATE SET
		title=EXCLUDED.title,username=EXCLUDED.username,chat_type=EXCLUDED.chat_type,
		is_forum=EXCLUDED.is_forum,status=EXCLUDED.status,last_seen_at=now(),updated_at=now()
		RETURNING id::text`,
		bot.tenantID,bot.id,value.ID,value.Title,value.Username,value.Type,value.IsForum,groupStatus(item)).Scan(&groupID)
	if err!=nil{return fmt.Errorf("upsert Telegram group: %w",err)}
	if item.MyChatMember != nil && (item.MyChatMember.NewChatMember.Status == "administrator" || item.MyChatMember.NewChatMember.Status == "creator") {
		// Publish the interactive management entry point as soon as the Bot is
		// promoted in a group; owners never need to type a management command.
		p.sendGroupMenu(ctx,value.ID,0,token)
	}
	if item.CallbackQuery != nil {
		if strings.HasPrefix(item.CallbackQuery.Data,"group:") { p.handleGroupCallback(ctx,bot,groupID,*item.CallbackQuery,token) } else { p.handleVerificationCallback(ctx,bot,*item.CallbackQuery,token) }
		return nil
	}
	if item.ChatMember != nil {
		member := item.ChatMember.NewChatMember
		joinedAt := time.Time{}
		if item.ChatMember.Date > 0 { joinedAt = time.Unix(item.ChatMember.Date, 0).UTC() }
		if err := p.upsertGroupMember(ctx, bot, groupID, member, joinedAt); err != nil { return err }
	}
	message:=messageFromUpdate(item)
	if message==nil { return nil }
	for _, member := range message.NewChatMembers {
		if err := p.upsertGroupMember(ctx, bot, groupID, telegramChatMember{User:member, Status:"member"}, time.Unix(message.Date,0).UTC()); err != nil { return err }
	}
	if message.LeftChatMember != nil {
		if err := p.upsertGroupMember(ctx, bot, groupID, telegramChatMember{User:*message.LeftChatMember, Status:"left"}, time.Time{}); err != nil { return err }
	}
	if message.MessageID==0{return nil}
	var senderID any
	var username,displayName string
	if message.From!=nil{senderID=message.From.ID;username=message.From.Username;displayName=senderDisplayName(message.From)}
	content:=message.Text
	if content==""{content=message.Caption}
	if runeCount:=[]rune(content);len(runeCount)>4096{content=string(runeCount[:4096])}
	var sentAt any
	if message.Date>0{sentAt=time.Unix(message.Date,0).UTC()}
	tag,err:=p.pool.Exec(ctx,`INSERT INTO telegram_message_events(
		tenant_id,bot_id,group_id,telegram_update_id,telegram_message_id,telegram_user_id,
		sender_username,sender_display_name,message_type,message_text,telegram_sent_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(bot_id,telegram_update_id) DO NOTHING`,
		bot.tenantID,bot.id,groupID,item.UpdateID,message.MessageID,senderID,username,displayName,
		messageType(*message),content,sentAt)
	if err!=nil{return fmt.Errorf("persist Telegram message event: %w",err)}
	if tag.RowsAffected()==1 {
		if p.processPendingGroupInput(ctx,bot,groupID,message,token) { return nil }
		p.processMessageRules(ctx,bot,groupID,item,message,token)
	}
	return nil
}
