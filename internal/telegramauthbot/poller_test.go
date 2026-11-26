package telegramauthbot

import (
	"encoding/json"
	"strings"

	"github.com/seanldnet/tg-group/internal/auth"
	"testing"
	"time"
)

func TestBotTokenPromptIsOneShot(t *testing.T) {
	p := &Poller{awaitingBotToken: make(map[int64]time.Time)}
	p.stateMu.Lock()
	p.awaitingBotToken[123] = time.Now().Add(time.Minute)
	p.stateMu.Unlock()

	if !p.consumeBotTokenPrompt(123) {
		t.Fatal("first token message should consume the prompt")
	}
	if p.consumeBotTokenPrompt(123) {
		t.Fatal("token prompt must be consumed only once")
	}
	if p.consumeBotTokenPrompt(456) {
		t.Fatal("another Telegram user must not consume this prompt")
	}
}

func TestUpdateBatchDecodesInlineKeyboardCallback(t *testing.T) {
	raw := []byte(`{"ok":true,"result":[{"update_id":7,"callback_query":{"id":"callback-1","from":{"id":123,"is_bot":false},"message":{"message_id":9,"from":{"id":999,"is_bot":true},"chat":{"id":123,"type":"private"},"text":"menu"},"data":"menu:addbot"}}]}`)
	var batch updateBatch
	if err := json.Unmarshal(raw, &batch); err != nil {
		t.Fatal(err)
	}
	if len(batch.Result) != 1 || batch.Result[0].CallbackQuery == nil {
		t.Fatal("callback query was not decoded")
	}
	if got := batch.Result[0].CallbackQuery.Data; got != "menu:addbot" {
		t.Fatalf("callback data = %q, want menu:addbot", got)
	}
}

func TestInlineMenuKeyboardIsAttachedToMessages(t *testing.T) {
	// Keep the keyboard schema explicit and machine-checkable: every menu action
	// is an inline callback, not a slash command or a reply-keyboard text shortcut.
	keyboard := map[string]any{"inline_keyboard": [][]map[string]string{
		{{"text": "账号信息", "callback_data": "menu:account"}, {"text": "添加群管 Bot", "callback_data": "menu:addbot"}},
		{{"text": "使用帮助", "callback_data": "menu:help"}, {"text": "取消当前操作", "callback_data": "menu:cancel"}},
	}}
	encoded, err := json.Marshal(keyboard)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"menu:account", "menu:addbot", "menu:help", "menu:cancel"} {
		if !contains(string(encoded), action) {
			t.Fatalf("inline keyboard is missing %s", action)
		}
	}
}

func contains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}

func TestBotTokenPromptExpires(t *testing.T) {
	p := &Poller{awaitingBotToken: map[int64]time.Time{123: time.Now().Add(-time.Second)}}
	if p.consumeBotTokenPrompt(123) {
		t.Fatal("expired token prompt must not accept a message")
	}
}

func TestRegistrationCredentialsUseCopyableCodeEntities(t *testing.T) {
	message:=registrationCredentialsHTML(auth.TelegramRegistration{UserID:"user-123",Password:"pass-456"})
	if !strings.Contains(message,"<code>user-123</code>") || !strings.Contains(message,"<code>pass-456</code>") {
		t.Fatalf("credentials are not wrapped in Telegram copyable code entities: %s",message)
	}
}

func TestIntroMenuOffersExplicitRegistrationInsteadOfRegisteringOnStart(t *testing.T) {
    encoded,err:=json.Marshal(platformIntroKeyboard())
    if err!=nil { t.Fatal(err) }
    for _,action:=range []string{"menu:register","menu:account","menu:addbot","menu:help"} {
        if !strings.Contains(string(encoded),action) { t.Fatalf("intro menu is missing %s",action) }
    }
}

func TestPlatformIntroUsesRealLineBreaksAndExplainsProduct(t *testing.T) {
    message:=platformIntroText()
    if !strings.Contains(message,"\n\n") { t.Fatal("platform intro must contain real line breaks") }
    if strings.Contains(message,"\\n") { t.Fatal("platform intro must not contain literal backslash-n sequences") }
    for _,feature:=range []string{"多机器人、多群组","自动发现群组","群消息记录","关键词/敏感词过滤","入群验证","封禁、解封、禁言"} {
        if !strings.Contains(message,feature) { t.Fatalf("platform intro missing feature description %q",feature) }
    }
}

func TestPlatformReplyKeyboardIsPersistentAndUsesButtonLabels(t *testing.T) {
    encoded,err:=json.Marshal(platformReplyKeyboard())
    if err!=nil { t.Fatal(err) }
    var keyboard map[string]any
    if err:=json.Unmarshal(encoded,&keyboard);err!=nil { t.Fatal(err) }
    if keyboard["resize_keyboard"]!=true || keyboard["is_persistent"]!=true {
        t.Fatalf("reply keyboard must be resized and persistent: %s",encoded)
    }
    for _,label:=range []string{"平台介绍","开始注册","账号信息","添加群管 Bot","使用帮助","取消操作"} {
        if !strings.Contains(string(encoded),label) { t.Fatalf("reply keyboard missing %q",label) }
    }
    if strings.Contains(string(encoded),"inline_keyboard") { t.Fatal("reply keyboard payload must not mix inline_keyboard") }
}
