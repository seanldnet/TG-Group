package httpapi

import (
	"strings"
	"testing"
)

func TestCapSiteVerifyURL(t *testing.T) {
	got, err := capSiteVerifyURL("https://captcha.example.com/", "site-key-123")
	if err != nil { t.Fatalf("capSiteVerifyURL() error = %v", err) }
	want := "https://captcha.example.com/site-key-123/siteverify"
	if got != want { t.Fatalf("capSiteVerifyURL() = %q, want %q", got, want) }
}

func TestCapSiteVerifyURLRejectsInsecureRemoteDomain(t *testing.T) {
	if _, err := capSiteVerifyURL("http://captcha.example.com", "site-key"); err == nil {
		t.Fatal("expected HTTP domain to be rejected")
	}
}

func TestCapSiteVerifyURLRejectsMalformedKey(t *testing.T) {
	if _, err := capSiteVerifyURL("https://captcha.example.com", "../other"); err == nil {
		t.Fatal("expected path traversal key to be rejected")
	}
}

func TestCapSecretEncryptionRoundTrip(t *testing.T) {
	verifier := CapVerifier{EncryptionKey: strings.Repeat("ab", 32)}
	encrypted, err := verifier.encryptSecret("private-cap-secret")
	if err != nil { t.Fatalf("encryptSecret() error = %v", err) }
	if strings.Contains(encrypted, "private-cap-secret") { t.Fatal("encrypted secret contains plaintext") }
	plain, err := verifier.decryptSecret(encrypted)
	if err != nil { t.Fatalf("decryptSecret() error = %v", err) }
	if plain != "private-cap-secret" { t.Fatalf("decrypted secret = %q", plain) }
}

func TestCapSecretEncryptionRejectsInvalidKey(t *testing.T) {
	verifier := CapVerifier{EncryptionKey: "invalid"}
	if _, err := verifier.encryptSecret("secret"); err == nil {
		t.Fatal("expected invalid encryption key error")
	}
}
