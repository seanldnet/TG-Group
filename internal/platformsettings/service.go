package platformsettings

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidConfig = errors.New("invalid Telegram platform Bot configuration")

type Service struct { pool *pgxpool.Pool; aead cipher.AEAD }

type Status struct {
	Configured bool `json:"configured"`
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

func (s *Service) Status(ctx context.Context)(Status,error){
	var status Status
	err:=s.pool.QueryRow(ctx,`SELECT username,updated_at FROM platform_telegram_auth_bot_settings WHERE id=1`).Scan(&status.Username,&status.UpdatedAt)
	if err!=nil {
		if errors.Is(err,pgx.ErrNoRows) { return Status{Configured:false},nil }
		return Status{},fmt.Errorf("read platform Telegram Bot settings: %w",err)
	}
	status.Configured=true
	return status,nil
}

func (s *Service) Save(ctx context.Context,token,username,actorID string) error {
	token=strings.TrimSpace(token)
	username,err:=normalizeUsername(username)
	if err!=nil||len(token)<20||len(token)>256||strings.ContainsAny(token," \r\n\t") { return ErrInvalidConfig }
	nonce:=make([]byte,s.aead.NonceSize())
	if _,err:=rand.Read(nonce);err!=nil{return fmt.Errorf("generate platform Bot nonce: %w",err)}
	ciphertext:=s.aead.Seal(nil,nonce,[]byte(token),[]byte("tg-group/platform-auth-bot/v1"))
	_,err=s.pool.Exec(ctx,`INSERT INTO platform_telegram_auth_bot_settings(id,token_ciphertext,token_nonce,username,updated_by,updated_at)
		VALUES(1,$1,$2,$3,$4,now()) ON CONFLICT(id) DO UPDATE SET token_ciphertext=EXCLUDED.token_ciphertext,
		token_nonce=EXCLUDED.token_nonce,username=EXCLUDED.username,updated_by=EXCLUDED.updated_by,updated_at=now()`,ciphertext,nonce,username,actorID)
	if err!=nil{return fmt.Errorf("save encrypted platform Telegram Bot settings: %w",err)}
	return nil
}

func (s *Service) LoadToken(ctx context.Context)(string,string,error){
	var ciphertext,nonce []byte
	var username string
	err:=s.pool.QueryRow(ctx,`SELECT token_ciphertext,token_nonce,username FROM platform_telegram_auth_bot_settings WHERE id=1`).Scan(&ciphertext,&nonce,&username)
	if err!=nil{return "","",err}
	token,err:=s.aead.Open(nil,nonce,ciphertext,[]byte("tg-group/platform-auth-bot/v1"))
	if err!=nil{return "","",errors.New("stored platform Telegram Bot token could not be decrypted")}
	return string(token),username,nil
}

func (s *Service) Clear(ctx context.Context)error{
	_,err:=s.pool.Exec(ctx,`DELETE FROM platform_telegram_auth_bot_settings WHERE id=1`)
	if err!=nil{return fmt.Errorf("clear platform Telegram Bot settings: %w",err)}
	return nil
}
