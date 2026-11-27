package payments

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidChannelConfig = errors.New("invalid payment channel configuration")

type ChannelConfig struct {
	Code string `json:"code"`
	Name string `json:"name"`
	BaseURL string `json:"base_url"`
	MerchantPID string `json:"merchant_pid"`
	SecretKey string `json:"-"`
	Currency string `json:"currency"`
	Token string `json:"token"`
	Network string `json:"network"`
	Enabled bool `json:"enabled"`
}

type ChannelStatus struct {
	Code string `json:"code"`
	Name string `json:"name"`
	BaseURL string `json:"base_url"`
	MerchantPID string `json:"merchant_pid"`
	Currency string `json:"currency"`
	Token string `json:"token"`
	Network string `json:"network"`
	Enabled bool `json:"enabled"`
}

type ConfigStore struct {
	pool *pgxpool.Pool
	aead cipher.AEAD
}

func NewConfigStore(pool *pgxpool.Pool, encodedKey string) (*ConfigStore,error) {
	key,err:=base64.StdEncoding.DecodeString(encodedKey)
	if err!=nil { key,err=base64.RawStdEncoding.DecodeString(encodedKey) }
	if err!=nil || len(key)!=32 { return nil,errors.New("PAYMENT_CREDENTIALS_ENCRYPTION_KEY must be a base64-encoded 32-byte key") }
	block,err:=aes.NewCipher(key);if err!=nil{return nil,err}
	aead,err:=cipher.NewGCM(block);if err!=nil{return nil,err}
	return &ConfigStore{pool:pool,aead:aead},nil
}

func validateChannelConfig(c ChannelConfig) error {
	c.Code=strings.TrimSpace(c.Code)
	c.Name=strings.TrimSpace(c.Name)
	c.MerchantPID=strings.TrimSpace(c.MerchantPID)
	c.SecretKey=strings.TrimSpace(c.SecretKey)
	c.Currency=strings.ToLower(strings.TrimSpace(c.Currency))
	c.Token=strings.ToLower(strings.TrimSpace(c.Token))
	c.Network=strings.ToLower(strings.TrimSpace(c.Network))
	u,err:=url.Parse(strings.TrimSpace(c.BaseURL))
	if err!=nil || u.Scheme!="https" || u.Host=="" || u.User!=nil || u.RawQuery!="" || u.Fragment!="" { return ErrInvalidChannelConfig }
	if c.Code!="epusdt_gmpay" && c.Code!="epusdt_epay" { return ErrInvalidChannelConfig }
	if c.Name=="" || len(c.Name)>80 || c.MerchantPID=="" || len(c.MerchantPID)>64 || len(c.SecretKey)<16 || len(c.SecretKey)>256 { return ErrInvalidChannelConfig }
	if c.Currency=="" || len(c.Currency)>8 || c.Token=="" || len(c.Token)>24 || c.Network=="" || len(c.Network)>32 { return ErrInvalidChannelConfig }
	if c.Code=="epusdt_epay" { if _,err:=strconv.Atoi(c.MerchantPID);err!=nil{return ErrInvalidChannelConfig} }
	return nil
}

func (s *ConfigStore) Save(ctx context.Context,c ChannelConfig,actorID string) error {
	if err:=validateChannelConfig(c);err!=nil{return err}
	nonce:=make([]byte,s.aead.NonceSize())
	if _,err:=rand.Read(nonce);err!=nil{return fmt.Errorf("generate payment credential nonce: %w",err)}
	ciphertext:=s.aead.Seal(nil,nonce,[]byte(strings.TrimSpace(c.SecretKey)),[]byte("tg-group/payment-channel/"+c.Code+"/v1"))
	_,err:=s.pool.Exec(ctx,`INSERT INTO payment_channels(code,name,base_url,merchant_pid,secret_ciphertext,secret_nonce,currency,token,network,enabled,updated_by,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now())
		ON CONFLICT(code) DO UPDATE SET name=EXCLUDED.name,base_url=EXCLUDED.base_url,merchant_pid=EXCLUDED.merchant_pid,
		secret_ciphertext=EXCLUDED.secret_ciphertext,secret_nonce=EXCLUDED.secret_nonce,currency=EXCLUDED.currency,
		token=EXCLUDED.token,network=EXCLUDED.network,enabled=EXCLUDED.enabled,updated_by=EXCLUDED.updated_by,updated_at=now()`,
		c.Code,c.Name,strings.TrimRight(strings.TrimSpace(c.BaseURL),"/"),c.MerchantPID,ciphertext,nonce,c.Currency,c.Token,c.Network,c.Enabled,actorID)
	if err!=nil{return fmt.Errorf("save payment channel: %w",err)}
	return nil
}

func (s *ConfigStore) List(ctx context.Context)([]ChannelStatus,error){
	rows,err:=s.pool.Query(ctx,`SELECT code,name,base_url,merchant_pid,currency,token,network,enabled FROM payment_channels ORDER BY code`)
	if err!=nil{return nil,fmt.Errorf("list payment channels: %w",err)}
	defer rows.Close()
	items:=make([]ChannelStatus,0)
	for rows.Next(){var c ChannelStatus;if err:=rows.Scan(&c.Code,&c.Name,&c.BaseURL,&c.MerchantPID,&c.Currency,&c.Token,&c.Network,&c.Enabled);err!=nil{return nil,fmt.Errorf("scan payment channel: %w",err)};items=append(items,c)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate payment channels: %w",err)}
	return items,nil
}

func (s *ConfigStore) Load(ctx context.Context,code string)(ChannelConfig,error){
	var c ChannelConfig
	var ciphertext,nonce []byte
	err:=s.pool.QueryRow(ctx,`SELECT code,name,base_url,merchant_pid,secret_ciphertext,secret_nonce,currency,token,network,enabled
		FROM payment_channels WHERE code=$1 AND enabled=TRUE`,code).Scan(&c.Code,&c.Name,&c.BaseURL,&c.MerchantPID,&ciphertext,&nonce,&c.Currency,&c.Token,&c.Network,&c.Enabled)
	if err!=nil{return ChannelConfig{},err}
	secret,err:=s.aead.Open(nil,nonce,ciphertext,[]byte("tg-group/payment-channel/"+c.Code+"/v1"))
	if err!=nil{return ChannelConfig{},errors.New("stored payment credential could not be decrypted")}
	c.SecretKey=string(secret)
	return c,nil
}

func sealCredential(aead cipher.AEAD,code,secret string)([]byte,[]byte,error){
	nonce:=make([]byte,aead.NonceSize())
	if _,err:=rand.Read(nonce);err!=nil{return nil,nil,err}
	return aead.Seal(nil,nonce,[]byte(secret),[]byte("tg-group/payment-channel/"+code+"/v1")),nonce,nil
}
