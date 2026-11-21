package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBotRuntimeGetUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123456:valid_token/getUpdates" { t.Fatalf("unexpected path: %s", r.URL.Path) }
		if r.URL.Query().Get("timeout") != "20" { t.Fatalf("unexpected timeout: %s", r.URL.Query().Get("timeout")) }
		_, _ = w.Write([]byte(`{"ok":true,"result":[{"update_id":7,"message":{"chat":{"id":-1001,"type":"supergroup","title":"Test"},"from":{"id":99,"username":"tester","first_name":"Test"}}}]}`))
	}))
	defer server.Close()

	runtime := BotRuntime{TelegramAPIBaseURL: server.URL, HTTPClient: server.Client()}
	updates, err := runtime.getUpdates(context.Background(), "123456:valid_token", 7)
	if err != nil { t.Fatalf("getUpdates() error = %v", err) }
	if len(updates) != 1 || updates[0].UpdateID != 7 { t.Fatalf("unexpected updates: %#v", updates) }
}

func TestBotRuntimeGetUpdatesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
	}))
	defer server.Close()

	runtime := BotRuntime{TelegramAPIBaseURL: server.URL, HTTPClient: server.Client()}
	_, err := runtime.getUpdates(context.Background(), "123456:invalid_token", 0)
	apiErr, ok := err.(*telegramAPIError)
	if !ok || apiErr.Code != http.StatusUnauthorized { t.Fatalf("expected unauthorized Telegram API error, got %v", err) }
}

func TestBotRuntimeGetChatMemberStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123456:valid_token/getChatMember" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("chat_id") != "-1001" || r.URL.Query().Get("user_id") != "99" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"status":"administrator"}}`))
	}))
	defer server.Close()

	runtime := BotRuntime{TelegramAPIBaseURL: server.URL, HTTPClient: server.Client()}
	status, err := runtime.telegramChatMemberStatus(context.Background(), "123456:valid_token", -1001, 99)
	if err != nil { t.Fatalf("telegramChatMemberStatus() error = %v", err) }
	if status != "administrator" { t.Fatalf("status = %q, want administrator", status) }
}

func TestBotRuntimeGetChatMemberStatusRejectsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ok":false,"description":"not enough rights"}`))
	}))
	defer server.Close()

	runtime := BotRuntime{TelegramAPIBaseURL: server.URL, HTTPClient: server.Client()}
	_, err := runtime.telegramChatMemberStatus(context.Background(), "123456:valid_token", -1001, 99)
	if err == nil || !strings.Contains(err.Error(), "not enough rights") {
		t.Fatalf("expected Telegram API error, got %v", err)
	}
}
