package platformsettings

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidConfig = errors.New("invalid Telegram platform Bot configuration")
	ErrTelegramRejected = errors.New("Telegram rejected the platform Bot credentials")
	ErrTelegramUnavailable = errors.New("Telegram verification is unavailable")
	ErrSchemaNotReady = errors.New("platform Telegram Bot settings migration is not applied")
)

type Service struct { pool *pgxpool.Pool; aead cipher.AEAD }

func classifyDatabaseError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return fmt.Errorf("%w: %v", ErrSchemaNotReady, err)
	}
	return err
}


type Status struct {
	Configured bool `json:"configured"`
	Managed bool `json:"managed"`
	Username string `json:"username"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func New(pool *pgxpool.Pool, encodedKey string) (*Service,error) {
	key,err:=base64.StdEncoding.DecodeString(encodedKey)
	if err!=nil { key,err=base64.RawStdEncoding.DecodeString(encodedKey) }
	if err!=nil || len(key)!=32 { return nil,errors.New("BOT_TOKEN_ENCRYPTION_KEY must be a base64-encoded 32-byte key") }
	block,err:=aes.NewCipher(key);if err!=nil{return nil,err}
	aead,err:=cipher.NewGCM(block);if err!=nil{return nil,err}
	return &Service{pool:pool,aead:aead},nil
}

func normalizeUsername(value string)(string,error){
	value=strings.TrimPrefix(strings.TrimSpace(value),"@")
	if len(value)<5||len(value)>32||!strings.HasSuffix(strings.ToLower(value),"bot"){return "",ErrInvalidConfig}
	for _,r:=range value { if !((r>='a'&&r<='z')||(r>='A'&&r<='Z')||(r>='0'&&r<='9')||r=='_') { return "",ErrInvalidConfig } }
	return value,nil
}

func validTokenShape(token string) bool {
	if len(token)<30||len(token)>256||strings.ContainsAny(token," \r\n\t") {return false}
	parts:=strings.Split(token,":")
	if len(parts)!=2||len(parts[0])<5||len(parts[1])<20{return false}
	for _,r:=range parts[0]{if r<'0'||r>'9'{return false}}
	return true
}

func (s *Service) Status(ctx context.Context)(Status,error){
	var status Status
	var updatedAt time.Time
	var enabled bool
	err:=s.pool.QueryRow(ctx,`SELECT username,updated_at,enabled FROM platform_telegram_auth_bot_settings WHERE id=1`).Scan(&status.Username,&updatedAt,&enabled)
	if err!=nil {
		if errors.Is(err,pgx.ErrNoRows) { return Status{Configured:false,Managed:false},nil }
		return Status{},fmt.Errorf("read platform Telegram Bot settings: %w",classifyDatabaseError(err))
	}
	status.Managed=true
	status.Configured=enabled
	if enabled {status.UpdatedAt=&updatedAt} else {status.Username=""}
	return status,nil
}

func (s *Service) VerifyToken(ctx context.Context,token,username string) error {
	username,err:=normalizeUsername(username);if err!=nil||!validTokenShape(strings.TrimSpace(token)){return ErrInvalidConfig}
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,"https://api.telegram.org/bot"+strings.TrimSpace(token)+"/getMe",nil)
	if err!=nil{return ErrInvalidConfig}
	client:=&http.Client{Timeout:10*time.Second}
	response,err:=client.Do(req);if err!=nil{return ErrTelegramUnavailable}
	defer response.Body.Close()
	data,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return ErrTelegramUnavailable}
	var result struct{OK bool `json:"ok"`;Result struct{Username string `json:"username"`} `json:"result"`}
	if json.Unmarshal(data,&result)!=nil{return ErrTelegramUnavailable}
	if response.StatusCode!=http.StatusOK||!result.OK{return ErrTelegramRejected}
	if !strings.EqualFold(result.Result.Username,username){return ErrInvalidConfig}
	return nil
}

func (s *Service) Save(ctx context.Context,token,username,actorID string) error {
	token=strings.TrimSpace(token)
	username,err:=normalizeUsername(username)
	if err!=nil||!validTokenShape(token) { return ErrInvalidConfig }
	nonce:=make([]byte,s.aead.NonceSize())
	if _,err:=rand.Read(nonce);err!=nil{return fmt.Errorf("generate platform Bot nonce: %w",err)}
	ciphertext:=s.aead.Seal(nil,nonce,[]byte(token),[]byte("tg-group/platform-auth-bot/v1"))
	tx,err:=s.pool.Begin(ctx);if err!=nil{return fmt.Errorf("begin platform Bot settings update: %w",err)}
	defer tx.Rollback(ctx)
	_,err=tx.Exec(ctx,`INSERT INTO platform_telegram_auth_bot_settings(id,token_ciphertext,token_nonce,username,enabled,updated_by,updated_at)
		VALUES(1,$1,$2,$3,TRUE,$4,now()) ON CONFLICT(id) DO UPDATE SET token_ciphertext=EXCLUDED.token_ciphertext,
		token_nonce=EXCLUDED.token_nonce,username=EXCLUDED.username,enabled=TRUE,updated_by=EXCLUDED.updated_by,updated_at=now()`,ciphertext,nonce,username,actorID)
	if err!=nil{return fmt.Errorf("save encrypted platform Telegram Bot settings: %w",classifyDatabaseError(err))}
	_,err=tx.Exec(ctx,`INSERT INTO audit_logs(actor_user_id,action,resource_type,resource_id,metadata)
		VALUES($1,'platform.telegram_auth_bot.update','platform_setting','telegram_auth_bot',jsonb_build_object('username',$2))`,actorID,username)
	if err!=nil{return fmt.Errorf("audit platform Telegram Bot update: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit platform Telegram Bot settings: %w",err)}
	return nil
}

func (s *Service) LoadToken(ctx context.Context)(string,string,error){
	var ciphertext,nonce []byte
	var username string
	err:=s.pool.QueryRow(ctx,`SELECT token_ciphertext,token_nonce,username FROM platform_telegram_auth_bot_settings WHERE id=1 AND enabled=TRUE`).Scan(&ciphertext,&nonce,&username)
	if err!=nil{return "","",err}
	token,err:=s.aead.Open(nil,nonce,ciphertext,[]byte("tg-group/platform-auth-bot/v1"))
	if err!=nil{return "","",errors.New("stored platform Telegram Bot token could not be decrypted")}
	return string(token),username,nil
}

func (s *Service) Clear(ctx context.Context,actorID string)error{
	tx,err:=s.pool.Begin(ctx);if err!=nil{return fmt.Errorf("begin platform Bot settings disable: %w",err)}
	defer tx.Rollback(ctx)
	_,err=tx.Exec(ctx,`INSERT INTO platform_telegram_auth_bot_settings(id,token_ciphertext,token_nonce,username,enabled,updated_by,updated_at)
		VALUES(1,''::bytea,''::bytea,'',FALSE,$1,now()) ON CONFLICT(id) DO UPDATE SET token_ciphertext=''::bytea,
		token_nonce=''::bytea,username='',enabled=FALSE,updated_by=EXCLUDED.updated_by,updated_at=now()`,actorID)
	if err!=nil{return fmt.Errorf("disable platform Telegram Bot settings: %w",classifyDatabaseError(err))}
	_,err=tx.Exec(ctx,`INSERT INTO audit_logs(actor_user_id,action,resource_type,resource_id)
		VALUES($1,'platform.telegram_auth_bot.disable','platform_setting','telegram_auth_bot')`,actorID)
	if err!=nil{return fmt.Errorf("audit platform Telegram Bot disable: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit platform Telegram Bot disable: %w",err)}
	return nil
}
