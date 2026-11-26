package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCaptchaVerifierUsesOfficialSiteVerifyContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/site-key/siteverify" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil { t.Fatalf("decode body: %v", err) }
		if body["secret"] != "site-secret" || body["response"] != "one-time-token" {
			t.Fatalf("unexpected verification payload: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	verifier, err := NewCaptchaVerifier(server.URL, "site-key", "site-secret")
	if err != nil { t.Fatalf("NewCaptchaVerifier: %v", err) }
	if err := verifier.Verify(context.Background(), "one-time-token"); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestCaptchaVerifierRejectsInvalidOrReusedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false,"error":"already_redeemed"}`))
	}))
	defer server.Close()
	verifier, err := NewCaptchaVerifier(server.URL, "site-key", "site-secret")
	if err != nil { t.Fatal(err) }
	if err := verifier.Verify(context.Background(), "used-token"); !errors.Is(err, ErrCaptchaRejected) {
		t.Fatalf("Verify() error = %v, want ErrCaptchaRejected", err)
	}
	if err := verifier.Verify(context.Background(), " "); !errors.Is(err, ErrCaptchaRejected) {
		t.Fatalf("empty token error = %v, want ErrCaptchaRejected", err)
	}
}

func TestNewCaptchaVerifierRequiresCompleteAndSafeConfiguration(t *testing.T) {
	for _, input := range [][3]string{
		{"https://captcha.example.com", "site", ""},
		{"//captcha.example.com", "site", "secret"},
		{"https://user:pass@captcha.example.com", "site", "secret"},
		{"https://captcha.example.com?x=1", "site", "secret"},
	} {
		if _, err := NewCaptchaVerifier(input[0], input[1], input[2]); err == nil {
			t.Errorf("NewCaptchaVerifier(%q, %q, %q) error = nil", input[0], input[1], input[2])
		}
	}
	if verifier, err := NewCaptchaVerifier("", "", ""); err != nil || verifier != nil {
		t.Fatalf("disabled verifier = %v, error = %v", verifier, err)
	}
}
