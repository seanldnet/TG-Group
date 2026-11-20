package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBotTokenEncryptionRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key { key[i] = byte(i + 1) }
	encrypted, err := encryptSecret(key, "123456:very-secret-telegram-token")
	if err != nil { t.Fatalf("encryptSecret() error = %v", err) }
	if encrypted == "123456:very-secret-telegram-token" { t.Fatal("token was not encrypted") }
	decrypted, err := decryptSecret(key, encrypted)
	if err != nil { t.Fatalf("decryptSecret() error = %v", err) }
	if decrypted != "123456:very-secret-telegram-token" { t.Fatalf("decrypted token = %q", decrypted) }
}

func TestBotTokenEncryptionRejectsWrongKey(t *testing.T) {
	key := make([]byte, 32)
	encrypted, err := encryptSecret(key, "secret")
	if err != nil { t.Fatal(err) }
	wrongKey := make([]byte, 32)
	wrongKey[0] = 1
	if _, err := decryptSecret(wrongKey, encrypted); err == nil { t.Fatal("expected authentication failure with wrong key") }
}

func TestMaskToken(t *testing.T) {
	if got := maskToken("123456:abcdef1234"); got != "••••1234" { t.Fatalf("maskToken() = %q", got) }
}

func TestVerifyTelegramToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123456:valid_token/getMe" {
			t.Fatalf("unexpected Telegram API path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":123,"username":"sample_bot","first_name":"Sample"}}`))
	}))
	defer server.Close()

	handler := BotHandler{TelegramAPIBaseURL: server.URL, HTTPClient: server.Client()}
	username, err := handler.verifyTelegramToken(t.Context(), "123456:valid_token")
	if err != nil { t.Fatalf("verifyTelegramToken() error = %v", err) }
	if username != "sample_bot" { t.Fatalf("username = %q, want sample_bot", username) }
}

func TestVerifyTelegramTokenRejectsInvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer server.Close()
	handler := BotHandler{TelegramAPIBaseURL: server.URL, HTTPClient: server.Client()}
	if _, err := handler.verifyTelegramToken(t.Context(), "123456:invalid_token"); err == nil {
		t.Fatal("expected invalid Telegram token error")
	}
}
