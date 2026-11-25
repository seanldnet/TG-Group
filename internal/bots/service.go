package bots

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidToken = errors.New("Telegram rejected the bot token")
	ErrTokenTaken = errors.New("this Telegram bot is already registered")
	ErrNotFound = errors.New("bot not found")
	ErrInvalidStatus = errors.New("invalid bot status")
	ErrIdentityChanged = errors.New("replacement token belongs to a different Telegram bot")
	ErrTelegramUnavailable = errors.New("Telegram verification service is unavailable")
	ErrBotDisabled = errors.New("Telegram bot is disabled")
)

type Service struct {
	pool *pgxpool.Pool
	aead cipher.AEAD
	client *http.Client
	apiBaseURL string
}

type Bot struct {
	ID string `json:"id"`
	TelegramBotID int64 `json:"telegram_bot_id"`
	Username string `json:"username"`
	FirstName string `json:"first_name"`
	Status string `json:"status"`
	VerifiedAt time.Time `json:"verified_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Group struct {
	ID string `json:"id"`
	BotID string `json:"bot_id"`
	TelegramChatID int64 `json:"telegram_chat_id"`
	Title string `json:"title"`
	Username string `json:"username"`
	ChatType string `json:"chat_type"`
	IsForum bool `json:"is_forum"`
	Status string `json:"status"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type GroupMessageEvent struct {
	ID int64 `json:"id"`
	GroupID string `json:"group_id"`
	TelegramMessageID int64 `json:"telegram_message_id"`
	TelegramUserID int64 `json:"telegram_user_id"`
	SenderUsername string `json:"sender_username"`
	SenderDisplayName string `json:"sender_display_name"`
	MessageType string `json:"message_type"`
	MessageText string `json:"message_text"`
	TelegramSentAt *time.Time `json:"telegram_sent_at"`
	ReceivedAt time.Time `json:"received_at"`
}

type telegramUser struct {
	ID int64 `json:"id"`
	IsBot bool `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username string `json:"username"`
}

type telegramResponse struct {
	OK bool `json:"ok"`
	Result telegramUser `json:"result"`
}

func NewService(pool *pgxpool.Pool, encodedKey string) (*Service, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil { key, err = base64.RawStdEncoding.DecodeString(encodedKey) }
	if err != nil || len(key) != 32 { return nil, errors.New("BOT_TOKEN_ENCRYPTION_KEY must be a base64-encoded 32-byte key") }
	block, err := aes.NewCipher(key)
	if err != nil { return nil, fmt.Errorf("create bot token cipher: %w", err) }
	aead, err := cipher.NewGCM(block)
	if err != nil { return nil, fmt.Errorf("create bot token AEAD: %w", err) }
	return &Service{pool:pool,aead:aead,client:&http.Client{Timeout:10*time.Second},apiBaseURL:"https://api.telegram.org"},nil
}

func (s *Service) List(ctx context.Context, tenantID string) ([]Bot,error) {
	rows,err:=s.pool.Query(ctx,`SELECT id::text,telegram_bot_id,username,first_name,status,verified_at,created_at
		FROM telegram_bots WHERE tenant_id=$1 ORDER BY created_at DESC`,tenantID)
	if err!=nil{return nil,fmt.Errorf("list bots: %w",err)}
	defer rows.Close()
	result:=make([]Bot,0)
	for rows.Next(){var b Bot;if err:=rows.Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt);err!=nil{return nil,err};result=append(result,b)}
	return result,rows.Err()
}

func (s *Service) ListGroups(ctx context.Context,tenantID string)([]Group,error){
	rows,err:=s.pool.Query(ctx,`SELECT id::text,bot_id::text,telegram_chat_id,title,username,chat_type,is_forum,status,last_seen_at
		FROM telegram_groups WHERE tenant_id=$1 ORDER BY last_seen_at DESC`,tenantID)
	if err!=nil{return nil,fmt.Errorf("list Telegram groups: %w",err)}
	defer rows.Close()
	result:=make([]Group,0)
	for rows.Next(){var group Group;if err:=rows.Scan(&group.ID,&group.BotID,&group.TelegramChatID,&group.Title,&group.Username,&group.ChatType,&group.IsForum,&group.Status,&group.LastSeenAt);err!=nil{return nil,err};result=append(result,group)}
	return result,rows.Err()
}

func (s *Service) ListGroupEvents(ctx context.Context,tenantID,groupID string,limit int)([]GroupMessageEvent,error){
	if limit<1 || limit>100{limit=50}
	rows,err:=s.pool.Query(ctx,`SELECT id,group_id::text,telegram_message_id,COALESCE(telegram_user_id,0),
		sender_username,sender_display_name,message_type,message_text,telegram_sent_at,received_at
		FROM telegram_message_events WHERE tenant_id=$1 AND group_id=$2
		ORDER BY received_at DESC,id DESC LIMIT $3`,tenantID,groupID,limit)
	if err!=nil{return nil,fmt.Errorf("list Telegram group events: %w",err)}
	defer rows.Close()
	result:=make([]GroupMessageEvent,0)
	for rows.Next(){var event GroupMessageEvent;if err:=rows.Scan(&event.ID,&event.GroupID,&event.TelegramMessageID,&event.TelegramUserID,
		&event.SenderUsername,&event.SenderDisplayName,&event.MessageType,&event.MessageText,&event.TelegramSentAt,&event.ReceivedAt);err!=nil{return nil,err};result=append(result,event)}
	return result,rows.Err()
}

func (s *Service) Add(ctx context.Context,tenantID,userID,token string)(Bot,error){
	token=strings.TrimSpace(token)
	if len(token)<20 || len(token)>256 || strings.ContainsAny(token," \r\n\t"){return Bot{},ErrInvalidToken}
	identity,err:=s.verify(ctx,token);if err!=nil{return Bot{},err}
	ciphertext,nonce,err:=s.encrypt(token);if err!=nil{return Bot{},err}
	fingerprint:=sha256.Sum256([]byte(token))
	tx,err:=s.pool.Begin(ctx);if err!=nil{return Bot{},fmt.Errorf("begin Bot registration: %w",err)}
	defer tx.Rollback(ctx)
	var b Bot
	err=tx.QueryRow(ctx,`INSERT INTO telegram_bots(tenant_id,created_by,telegram_bot_id,username,first_name,token_ciphertext,token_nonce,token_fingerprint)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text,telegram_bot_id,username,first_name,status,verified_at,created_at`,
		tenantID,userID,identity.ID,identity.Username,identity.FirstName,ciphertext,nonce,fingerprint[:]).Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt)
	if err!=nil{if isUniqueViolation(err){return Bot{},ErrTokenTaken};return Bot{},fmt.Errorf("save Telegram bot: %w",err)}
	if err:=recordBotAudit(ctx,tx,tenantID,userID,"bot.add",b.ID);err!=nil{return Bot{},err}
	if err:=tx.Commit(ctx);err!=nil{return Bot{},fmt.Errorf("commit Bot registration: %w",err)}
	return b,nil
}
func (s *Service) SetStatus(ctx context.Context,tenantID,actorUserID,botID,status string)(Bot,error){
	if status!="active" && status!="disabled"{return Bot{},ErrInvalidStatus}
	tx,err:=s.pool.Begin(ctx);if err!=nil{return Bot{},fmt.Errorf("begin Bot status update: %w",err)}
	defer tx.Rollback(ctx)
	var b Bot
	err=tx.QueryRow(ctx,`UPDATE telegram_bots SET status=$3,updated_at=now()
		WHERE tenant_id=$1 AND id=$2 RETURNING id::text,telegram_bot_id,username,first_name,status,verified_at,created_at`,tenantID,botID,status).
		Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt)
	if errors.Is(err,pgx.ErrNoRows){return Bot{},ErrNotFound}
	if err!=nil{return Bot{},fmt.Errorf("update bot status: %w",err)}
	if err:=recordBotAudit(ctx,tx,tenantID,actorUserID,"bot.status."+status,botID);err!=nil{return Bot{},err}
	if err:=tx.Commit(ctx);err!=nil{return Bot{},fmt.Errorf("commit Bot status update: %w",err)}
	return b,nil
}
func (s *Service) Remove(ctx context.Context,tenantID,actorUserID,botID string)error{
	tx,err:=s.pool.Begin(ctx);if err!=nil{return fmt.Errorf("begin Bot removal: %w",err)}
	defer tx.Rollback(ctx)
	tag,err:=tx.Exec(ctx,"DELETE FROM telegram_bots WHERE tenant_id=$1 AND id=$2",tenantID,botID)
	if err!=nil{return fmt.Errorf("remove bot: %w",err)}
	if tag.RowsAffected()==0{return ErrNotFound}
	if err:=recordBotAudit(ctx,tx,tenantID,actorUserID,"bot.remove",botID);err!=nil{return err}
	if err:=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit Bot removal: %w",err)}
	return nil
}
func (s *Service) RotateToken(ctx context.Context,tenantID,actorUserID,botID,token string)(Bot,error){
	token=strings.TrimSpace(token)
	if len(token)<20 || len(token)>256 || strings.ContainsAny(token," \r\n\t"){return Bot{},ErrInvalidToken}
	identity,err:=s.verify(ctx,token);if err!=nil{return Bot{},err}
	ciphertext,nonce,err:=s.encrypt(token);if err!=nil{return Bot{},err}
	fingerprint:=sha256.Sum256([]byte(token))
	tx,err:=s.pool.Begin(ctx);if err!=nil{return Bot{},fmt.Errorf("begin Bot token rotation: %w",err)}
	defer tx.Rollback(ctx)
	var currentID int64
	err=tx.QueryRow(ctx,"SELECT telegram_bot_id FROM telegram_bots WHERE tenant_id=$1 AND id=$2 FOR UPDATE",tenantID,botID).Scan(&currentID)
	if errors.Is(err,pgx.ErrNoRows){return Bot{},ErrNotFound};if err!=nil{return Bot{},err}
	if identity.ID!=currentID{return Bot{},ErrIdentityChanged}
	var b Bot
	err=tx.QueryRow(ctx,`UPDATE telegram_bots SET token_ciphertext=$3,token_nonce=$4,token_fingerprint=$5,
		username=$6,first_name=$7,verified_at=now(),updated_at=now()
		WHERE tenant_id=$1 AND id=$2
		RETURNING id::text,telegram_bot_id,username,first_name,status,verified_at,created_at`,
		tenantID,botID,ciphertext,nonce,fingerprint[:],identity.Username,identity.FirstName).
		Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt)
	if err!=nil{if isUniqueViolation(err){return Bot{},ErrTokenTaken};return Bot{},fmt.Errorf("rotate bot token: %w",err)}
	if err:=recordBotAudit(ctx,tx,tenantID,actorUserID,"bot.token.rotate",botID);err!=nil{return Bot{},err}
	if err:=tx.Commit(ctx);err!=nil{return Bot{},fmt.Errorf("commit Bot token rotation: %w",err)}
	return b,nil
}
func (s *Service) verify(ctx context.Context,token string)(telegramUser,error){
	endpoint:=s.apiBaseURL+"/bot"+url.PathEscape(token)+"/getMe"
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,endpoint,nil);if err!=nil{return telegramUser{},err}
	response,err:=s.client.Do(req);if err!=nil{return telegramUser{},ErrTelegramUnavailable}
	defer response.Body.Close()
	body,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return telegramUser{},err}
	var result telegramResponse
	if response.StatusCode==http.StatusTooManyRequests || response.StatusCode>=http.StatusInternalServerError{return telegramUser{},ErrTelegramUnavailable}
	if response.StatusCode!=http.StatusOK || json.Unmarshal(body,&result)!=nil || !result.OK || !result.Result.IsBot || result.Result.ID==0{return telegramUser{},ErrInvalidToken}
	return result.Result,nil
}

func (s *Service) GetToken(ctx context.Context,tenantID,botID string)(string,error){
	var ciphertext,nonce []byte
	var status string
	err:=s.pool.QueryRow(ctx,"SELECT token_ciphertext,token_nonce,status FROM telegram_bots WHERE tenant_id=$1 AND id=$2",tenantID,botID).Scan(&ciphertext,&nonce,&status)
	if errors.Is(err,pgx.ErrNoRows){return "",ErrNotFound}
	if err!=nil{return "",fmt.Errorf("load encrypted Telegram bot token: %w",err)}
	if status!="active"{return "",ErrBotDisabled}
	plaintext,err:=s.aead.Open(nil,nonce,ciphertext,[]byte("tg-group/telegram-bot-token/v1"))
	if err!=nil{return "",errors.New("stored Telegram bot token could not be decrypted")}
	return string(plaintext),nil
}

func (s *Service) encrypt(token string)([]byte,[]byte,error){
	nonce:=make([]byte,s.aead.NonceSize())
	if _,err:=rand.Read(nonce);err!=nil{return nil,nil,fmt.Errorf("generate token nonce: %w",err)}
	ciphertext:=s.aead.Seal(nil,nonce,[]byte(token),[]byte("tg-group/telegram-bot-token/v1"))
	return ciphertext,nonce,nil
}

func isUniqueViolation(err error)bool{
	var pgErr interface{SQLState()string}
	return errors.As(err,&pgErr)&&pgErr.SQLState()=="23505"
}


func recordBotAudit(ctx context.Context,tx pgx.Tx,tenantID,actorUserID,action,resourceID string)error{
	_,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,$3,'telegram_bot',$4)`,tenantID,actorUserID,action,resourceID)
	if err!=nil{return fmt.Errorf("write Bot audit record: %w",err)}
	return nil
}
