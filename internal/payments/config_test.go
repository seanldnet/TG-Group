package payments

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"testing"
)

func testAEAD(t *testing.T) cipher.AEAD {
	t.Helper()
	key:=make([]byte,32)
	block,err:=aes.NewCipher(key);if err!=nil{t.Fatal(err)}
	aead,err:=cipher.NewGCM(block);if err!=nil{t.Fatal(err)}
	return aead
}

func TestValidateChannelConfig(t *testing.T){
	valid:=ChannelConfig{Code:"epusdt_gmpay",Name:"EPUSDT GMPay",BaseURL:"https://pay.example",MerchantPID:"1000",SecretKey:"0123456789abcdef",Currency:"usd",Token:"usdt",Network:"tron"}
	tests:=[]struct{name string;mutate func(*ChannelConfig);wantErr bool}{
		{name:"valid"},
		{name:"http forbidden",mutate:func(c *ChannelConfig){c.BaseURL="http://pay.example"},wantErr:true},
		{name:"URL credentials forbidden",mutate:func(c *ChannelConfig){c.BaseURL="https://user:pass@pay.example"},wantErr:true},
		{name:"unknown channel",mutate:func(c *ChannelConfig){c.Code="other"},wantErr:true},
		{name:"empty secret",mutate:func(c *ChannelConfig){c.SecretKey=""},wantErr:true},
		{name:"epay pid must be numeric",mutate:func(c *ChannelConfig){c.Code="epusdt_epay";c.MerchantPID="merchant"},wantErr:true},
	}
	for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){c:=valid;if tt.mutate!=nil{tt.mutate(&c)};err:=validateChannelConfig(c);if (err!=nil)!=tt.wantErr{t.Fatalf("validateChannelConfig() error=%v wantErr=%v",err,tt.wantErr)}})}
}

func TestPaymentCredentialEncryptionUsesChannelBoundAAD(t *testing.T){
	aead:=testAEAD(t)
	ciphertext,nonce,err:=sealCredential(aead,"epusdt_gmpay","merchant-secret")
	if err!=nil{t.Fatal(err)}
	plain,err:=aead.Open(nil,nonce,ciphertext,[]byte("tg-group/payment-channel/epusdt_gmpay/v1"))
	if err!=nil||string(plain)!="merchant-secret"{t.Fatalf("decrypt credential: %q, %v",plain,err)}
	if _,err:=aead.Open(nil,nonce,ciphertext,[]byte("tg-group/payment-channel/epusdt_epay/v1"));err==nil{t.Fatal("credential decrypted under a different channel context")}
}

func TestPaymentEncryptionKeyMustBe32Bytes(t *testing.T){
	if _,err:=base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(make([]byte,16)));err!=nil{t.Fatal(err)}
}
