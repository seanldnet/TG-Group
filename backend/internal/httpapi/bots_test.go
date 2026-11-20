package httpapi

import (
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
