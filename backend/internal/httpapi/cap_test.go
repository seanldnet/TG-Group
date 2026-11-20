package httpapi

import (
	"strings"
	"testing"
	"net/http"
	"net/http/httptest"
	"encoding/json"
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

func TestVerifyCapTokenPostsJSONAndAcceptsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { t.Fatalf("method = %s, want POST", r.Method) }
		if got := r.Header.Get("Content-Type"); got != "application/json" { t.Fatalf("Content-Type = %q", got) }
		var payload capVerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil { t.Fatal(err) }
		if payload.Secret != "cap-secret" || payload.Response != "one-time-token" {
			t.Fatalf("unexpected verification payload: %#v", payload)
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	if err := verifyCapToken(t.Context(), server.Client(), server.URL, "cap-secret", "one-time-token"); err != nil {
		t.Fatalf("verifyCapToken() error = %v", err)
	}
}

func TestVerifyCapTokenRejectsFailedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer server.Close()
	if err := verifyCapToken(t.Context(), server.Client(), server.URL, "secret", "token"); err == nil {
		t.Fatal("expected unsuccessful Cap response to be rejected")
	}
}
