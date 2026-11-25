package bots

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
)

type Service struct {
	pool *pgxpool.Pool
	aead cipher.AEAD
	client *http.Client
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
	return &Service{pool:pool,aead:aead,client:&http.Client{Timeout:10*time.Second}},nil
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

func (s *Service) Add(ctx context.Context,tenantID,userID,token string)(Bot,error){
	token=strings.TrimSpace(token)
	if len(token)<20 || len(token)>256 || strings.ContainsAny(token," \r\n\t"){return Bot{},ErrInvalidToken}
	identity,err:=s.verify(ctx,token);if err!=nil{return Bot{},err}
	ciphertext,nonce,err:=s.encrypt(token);if err!=nil{return Bot{},err}
	fingerprint:=sha256.Sum256([]byte(token))
	var b Bot
	err=s.pool.QueryRow(ctx,`INSERT INTO telegram_bots(tenant_id,created_by,telegram_bot_id,username,first_name,token_ciphertext,token_nonce,token_fingerprint)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text,telegram_bot_id,username,first_name,status,verified_at,created_at`,
		tenantID,userID,identity.ID,identity.Username,identity.FirstName,ciphertext,nonce,fingerprint[:]).Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt)
	if err!=nil{if isUniqueViolation(err){return Bot{},ErrTokenTaken};return Bot{},fmt.Errorf("save Telegram bot: %w",err)}
	return b,nil
}

func (s *Service) SetStatus(ctx context.Context,tenantID,botID,status string)(Bot,error){
	if status!="active" && status!="disabled"{return Bot{},ErrInvalidStatus}
	var b Bot
	err:=s.pool.QueryRow(ctx,`UPDATE telegram_bots SET status=$3,updated_at=now()
		WHERE tenant_id=$1 AND id=$2 RETURNING id::text,telegram_bot_id,username,first_name,status,verified_at,created_at`,tenantID,botID,status).
		Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt)
	if errors.Is(err,pgx.ErrNoRows){return Bot{},ErrNotFound}
	if err!=nil{return Bot{},fmt.Errorf("update bot status: %w",err)}
	return b,nil
}

func (s *Service) Remove(ctx context.Context,tenantID,botID string)error{
	tag,err:=s.pool.Exec(ctx,"DELETE FROM telegram_bots WHERE tenant_id=$1 AND id=$2",tenantID,botID)
	if err!=nil{return fmt.Errorf("remove bot: %w",err)}
	if tag.RowsAffected()==0{return ErrNotFound}
	return nil
}

func (s *Service) RotateToken(ctx context.Context,tenantID,botID,token string)(Bot,error){
	token=strings.TrimSpace(token)
	if len(token)<20 || len(token)>256 || strings.ContainsAny(token," \r\n\t"){return Bot{},ErrInvalidToken}
	identity,err:=s.verify(ctx,token);if err!=nil{return Bot{},err}
	ciphertext,nonce,err:=s.encrypt(token);if err!=nil{return Bot{},err}
	fingerprint:=sha256.Sum256([]byte(token))
	var currentID int64
	err=s.pool.QueryRow(ctx,"SELECT telegram_bot_id FROM telegram_bots WHERE tenant_id=$1 AND id=$2",tenantID,botID).Scan(&currentID)
	if errors.Is(err,pgx.ErrNoRows){return Bot{},ErrNotFound};if err!=nil{return Bot{},err}
	if identity.ID!=currentID{return Bot{},ErrIdentityChanged}
	var b Bot
	err=s.pool.QueryRow(ctx,`UPDATE telegram_bots SET token_ciphertext=$3,token_nonce=$4,token_fingerprint=$5,
		username=$6,first_name=$7,verified_at=now(),updated_at=now()
		WHERE tenant_id=$1 AND id=$2
		RETURNING id::text,telegram_bot_id,username,first_name,status,verified_at,created_at`,
		tenantID,botID,ciphertext,nonce,fingerprint[:],identity.Username,identity.FirstName).
		Scan(&b.ID,&b.TelegramBotID,&b.Username,&b.FirstName,&b.Status,&b.VerifiedAt,&b.CreatedAt)
	if err!=nil{return Bot{},fmt.Errorf("rotate bot token: %w",err)}
	return b,nil
}

func (s *Service) verify(ctx context.Context,token string)(telegramUser,error){
	endpoint:="https://api.telegram.org/bot"+url.PathEscape(token)+"/getMe"
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,endpoint,nil);if err!=nil{return telegramUser{},err}
	response,err:=s.client.Do(req);if err!=nil{return telegramUser{},fmt.Errorf("verify Telegram bot token: %w",err)}
	defer response.Body.Close()
	body,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return telegramUser{},err}
	var result telegramResponse
	if response.StatusCode!=http.StatusOK || json.Unmarshal(body,&result)!=nil || !result.OK || !result.Result.IsBot || result.Result.ID==0{return telegramUser{},ErrInvalidToken}
	return result.Result,nil
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

// Keep bytes imported in this package's compile-time contract for future
// authenticated token decryption tests without exposing plaintext tokens.
var _ = bytes.Equal
var _ = hex.EncodeToString
