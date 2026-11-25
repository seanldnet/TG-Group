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
