package telegramruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckRequiredChannels(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.URL.Path!="/botTEST_TOKEN/getChatMember" { t.Fatalf("unexpected Telegram path %s",r.URL.Path) }
		var payload struct { ChatID string `json:"chat_id"`; UserID int64 `json:"user_id"` }
		if err:=json.NewDecoder(r.Body).Decode(&payload);err!=nil { t.Fatalf("decode request: %v",err) }
		if payload.UserID!=42 { t.Fatalf("user_id=%d, want 42",payload.UserID) }
		status:="member"
		if payload.ChatID=="@private_news" { status="left" }
		_ = json.NewEncoder(w).Encode(map[string]any{"ok":true,"result":map[string]any{"status":status,"user":map[string]any{"id":42,"first_name":"Test"}}})
	}))
	defer server.Close()
	poller:=&Poller{client:server.Client(),apiBaseURL:server.URL}
	missing,err:=poller.checkRequiredChannels(context.Background(),"TEST_TOKEN",[]string{"@public_news","@private_news"},42)
	if err!=nil { t.Fatalf("checkRequiredChannels error: %v",err) }
	if len(missing)!=1 || missing[0]!="@private_news" { t.Fatalf("missing=%v, want [@private_news]",missing) }
}

func TestCheckRequiredChannelsFailsClosedOnTelegramError(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok":false,"description":"bot is not an administrator"})
	}))
	defer server.Close()
	poller:=&Poller{client:server.Client(),apiBaseURL:server.URL}
	if _,err:=poller.checkRequiredChannels(context.Background(),"TEST_TOKEN",[]string{"@private_news"},42);err==nil {
		t.Fatal("expected Telegram permission error")
	}
}

func TestRequiredChannelPrompt(t *testing.T) {
	if got:=requiredChannelPrompt(nil);got!="" { t.Fatalf("empty channel prompt=%q",got) }
	if got:=requiredChannelPrompt([]string{"@news","@updates"});got!="\n请先加入以下频道，再完成验证：\n@news\n@updates" {
		t.Fatalf("channel prompt=%q",got)
	}
}

func TestValidRequiredChannelReferences(t *testing.T) {
	valid:=[]string{"@tg_group_news","-1001234567890"}
	for _,value:=range valid { if !validRequiredChannel(value) { t.Errorf("%q should be valid",value) } }
	invalid:=[]string{"news","@ab","https://t.me/news","-1234567890","@bad-name"}
	for _,value:=range invalid { if validRequiredChannel(value) { t.Errorf("%q should be invalid",value) } }
}
