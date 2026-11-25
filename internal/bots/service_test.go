package bots

import (
	"crypto/rand"
	"context"
	"encoding/base64"
	"testing"
)

func TestNewServiceRequires32ByteBase64Key(t *testing.T) {
	if _,err:=NewService(nil,"bad-key");err==nil{t.Fatal("expected invalid key error")}
	key:=make([]byte,32)
	if _,err:=rand.Read(key);err!=nil{t.Fatal(err)}
	if _,err:=NewService(nil,base64.StdEncoding.EncodeToString(key));err!=nil{t.Fatalf("NewService: %v",err)}
}

func TestTokenEncryptionAuthenticatesCiphertext(t *testing.T) {
	key:=make([]byte,32)
	if _,err:=rand.Read(key);err!=nil{t.Fatal(err)}
	service,err:=NewService(nil,base64.StdEncoding.EncodeToString(key))
	if err!=nil{t.Fatal(err)}
	token:="123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghi"
	ciphertext,nonce,err:=service.encrypt(token)
	if err!=nil{t.Fatal(err)}
	plain,err:=service.aead.Open(nil,nonce,ciphertext,[]byte("tg-group/telegram-bot-token/v1"))
	if err!=nil{t.Fatalf("decrypt token: %v",err)}
	if string(plain)!=token{t.Fatal("decrypted token mismatch")}
	ciphertext[0]^=0xff
	if _,err:=service.aead.Open(nil,nonce,ciphertext,[]byte("tg-group/telegram-bot-token/v1"));err==nil{t.Fatal("tampered ciphertext was accepted")}
}

func TestModerateGroupMemberRejectsInvalidActionsBeforeDatabaseAccess(t *testing.T) {
 service := &Service{}
 cases := []struct{name,action string; userID int64; duration int}{
  {"zero user","ban",0,0},
  {"unknown action","delete",123,0},
  {"mute without duration","mute",123,0},
  {"mute too long","mute",123,525601},
  {"unban with duration","unban",123,5},
 }
 for _,tc:=range cases { t.Run(tc.name,func(t *testing.T){
  err:=service.ModerateGroupMember(context.Background(),"tenant","actor","group",tc.userID,tc.action,tc.duration)
  if err!=ErrInvalidMemberAction { t.Fatalf("error = %v, want ErrInvalidMemberAction",err) }
 })}
}
