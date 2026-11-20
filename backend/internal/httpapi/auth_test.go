package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestBearerToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer abc123")
	if got := bearerToken(req); got != "abc123" { t.Fatalf("bearerToken() = %q", got) }
}

func TestBearerTokenRejectsOtherSchemes(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	if got := bearerToken(req); got != "" { t.Fatalf("bearerToken() = %q, want empty", got) }
}

func TestHashToken(t *testing.T) {
	if hashToken("same-token") != hashToken("same-token") { t.Fatal("hash must be stable") }
	if hashToken("same-token") == hashToken("other-token") { t.Fatal("different tokens must hash differently") }
}

func TestRemoteIPUsesSocketPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.2")
	if got := remoteIP(req); got != "192.0.2.10" { t.Fatalf("remoteIP() = %q", got) }
}
