package auth

import (
	"strings"
	"testing"
)

func TestValidEmail(t *testing.T) {
	for _, value := range []string{"member@example.com", "first.last+tag@example.co"} {
		if !validEmail(value) { t.Errorf("validEmail(%q) = false", value) }
	}
	for _, value := range []string{"", "plain-text", "a@b", "Name <member@example.com>"} {
		if validEmail(value) { t.Errorf("validEmail(%q) = true", value) }
	}
}

func TestNewToken(t *testing.T) {
	first, firstHash, err := newToken()
	if err != nil { t.Fatal(err) }
	second, secondHash, err := newToken()
	if err != nil { t.Fatal(err) }
	if first == second { t.Fatal("generated duplicate session tokens") }
	if len(first) < 40 || len(firstHash) != 32 || len(secondHash) != 32 { t.Fatal("unexpected token/hash length") }
	if strings.Contains(first, "+") || strings.Contains(first, "/") { t.Fatal("token is not URL-safe") }
}

func TestClientIP(t *testing.T) {
	if got := clientIP("192.0.2.10:1234"); got != "192.0.2.10" { t.Fatalf("clientIP = %q", got) }
	if got := clientIP("not-an-ip"); got != "" { t.Fatalf("clientIP invalid = %q", got) }
}


func TestValidateAgentLevel(t *testing.T) {
	tests := []struct {
		name string
		code string
		levelName string
		price int64
		sortOrder int32
		wantErr bool
	}{
		{name:"valid",code:"silver_2",levelName:"白银代理",price:1200,sortOrder:2},
		{name:"zero price",code:"base",levelName:"基础代理",price:0,sortOrder:0},
		{name:"negative price",code:"gold",levelName:"黄金代理",price:-1,sortOrder:1,wantErr:true},
		{name:"negative sort",code:"gold",levelName:"黄金代理",price:1,sortOrder:-1,wantErr:true},
		{name:"short code",code:"a",levelName:"代理",price:1,sortOrder:1,wantErr:true},
		{name:"uppercase code",code:"Gold",levelName:"代理",price:1,sortOrder:1,wantErr:true},
		{name:"invalid code character",code:"gold tier",levelName:"代理",price:1,sortOrder:1,wantErr:true},
		{name:"empty name",code:"gold",levelName:" ",price:1,sortOrder:1,wantErr:true},
		{name:"long code",code:strings.Repeat("a",41),levelName:"代理",price:1,sortOrder:1,wantErr:true},
	}
	for _,tt:=range tests {
		t.Run(tt.name,func(t *testing.T){
			err:=validateAgentLevel(tt.code,tt.levelName,tt.price,tt.sortOrder)
			if (err!=nil)!=tt.wantErr { t.Fatalf("validateAgentLevel() error=%v, wantErr=%v",err,tt.wantErr) }
		})
	}
}
