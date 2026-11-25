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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/seanldnet/tg-group/internal/bots"
)

type Poller struct {
	pool *pgxpool.Pool
	botService *bots.Service
	logger *slog.Logger
	client *http.Client
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
	Type string `json:"type"`
	Title string `json:"title"`
	Username string `json:"username"`
	IsForum bool `json:"is_forum"`
}

type update struct {
	UpdateID int64 `json:"update_id"`
	Message *struct { Chat chat `json:"chat"` } `json:"message"`
	EditedMessage *struct { Chat chat `json:"chat"` } `json:"edited_message"`
	ChannelPost *struct { Chat chat `json:"chat"` } `json:"channel_post"`
	MyChatMember *struct { Chat chat `json:"chat"` } `json:"my_chat_member"`
}

type updatesResponse struct {
	OK bool `json:"ok"`
	Result []update `json:"result"`
}

func NewPoller(pool *pgxpool.Pool, botService *bots.Service, logger *slog.Logger) *Poller {
	if logger == nil { logger = slog.Default() }
	return &Poller{pool:pool,botService:botService,logger:logger,client:&http.Client{Timeout:40*time.Second}}
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
		batch,err:=p.getUpdates(ctx,token,offset)
		if err!=nil {
			if ctx.Err()!=nil{return}
			p.logger.Warn("Telegram long polling request failed","bot_id",bot.id,"error",err)
			select{case <-ctx.Done():return;case <-time.After(3*time.Second):continue}
		}
		for _,item:=range batch {
			if err:=p.persistUpdate(ctx,bot,item);err!=nil {
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
	query:=url.Values{"timeout":{"25"},"allowed_updates":{`["message","edited_message","channel_post","my_chat_member"]`}}
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

func chatFromUpdate(item update) *chat {
	switch {
	case item.Message!=nil:return &item.Message.Chat
	case item.EditedMessage!=nil:return &item.EditedMessage.Chat
	case item.ChannelPost!=nil:return &item.ChannelPost.Chat
	case item.MyChatMember!=nil:return &item.MyChatMember.Chat
	default:return nil
	}
}

func isManagedChat(value *chat) bool {
	return value!=nil && value.ID!=0 && (value.Type=="group" || value.Type=="supergroup" || value.Type=="channel")
}

func (p *Poller) persistUpdate(ctx context.Context,bot managedBot,item update)error {
	value:=chatFromUpdate(item)
	if !isManagedChat(value){return nil}
	_,err:=p.pool.Exec(ctx,`INSERT INTO telegram_groups(tenant_id,bot_id,telegram_chat_id,title,username,chat_type,is_forum,last_seen_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,now())
		ON CONFLICT(bot_id,telegram_chat_id) DO UPDATE SET
		title=EXCLUDED.title,username=EXCLUDED.username,chat_type=EXCLUDED.chat_type,
		is_forum=EXCLUDED.is_forum,status='active',last_seen_at=now(),updated_at=now()`,
		bot.tenantID,bot.id,value.ID,value.Title,value.Username,value.Type,value.IsForum)
	if err!=nil{return fmt.Errorf("upsert Telegram group: %w",err)}
	return nil
}

