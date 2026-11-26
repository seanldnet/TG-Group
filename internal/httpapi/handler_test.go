package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/seanldnet/tg-group/internal/auth"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var body healthResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want ok", body.Status)
	}
	if body.Timestamp.IsZero() || body.Timestamp.After(time.Now().Add(time.Second)) {
		t.Fatalf("unexpected timestamp: %v", body.Timestamp)
	}
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	handler := NewHandler(slog.Default(), nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestAuthEndpointsRequireDatabase(t *testing.T) {
	handler := NewHandler(slog.Default(), nil, nil, nil)
	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/register"},
		{http.MethodPost, "/api/v1/auth/login"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/auth/password/change"},
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/auth/security-events"},
		{http.MethodGet, "/api/v1/auth/sessions"},
		{http.MethodDelete, "/api/v1/auth/sessions/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/auth/telegram"},
		{http.MethodPost, "/api/v1/auth/telegram/link"},
		{http.MethodDelete, "/api/v1/auth/telegram"},
	} {
		request := httptest.NewRequest(route.method, route.path, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s status = %d, want %d", route.method, route.path, response.Code, http.StatusServiceUnavailable)
		}
	}
}

func TestBotEndpointsRequireConfiguredServices(t *testing.T) {
	handler := NewHandler(slog.Default(), nil, nil, nil)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/bots"},
		{http.MethodGet, "/api/v1/groups"},
		{http.MethodGet, "/api/v1/groups/00000000-0000-0000-0000-000000000000/events"},
		{http.MethodGet, "/api/v1/groups/00000000-0000-0000-0000-000000000000/members"},
		{http.MethodGet, "/api/v1/groups/00000000-0000-0000-0000-000000000000/permissions"},
		{http.MethodGet, "/api/v1/groups/00000000-0000-0000-0000-000000000000/rules"},
		{http.MethodGet, "/api/v1/groups/00000000-0000-0000-0000-000000000000/rule-events"},
		{http.MethodPost, "/api/v1/groups/00000000-0000-0000-0000-000000000000/rules"},
		{http.MethodPut, "/api/v1/groups/00000000-0000-0000-0000-000000000000/rules/00000000-0000-0000-0000-000000000001"},
		{http.MethodDelete, "/api/v1/groups/00000000-0000-0000-0000-000000000000/rules/00000000-0000-0000-0000-000000000001"},
		{http.MethodPost, "/api/v1/groups/00000000-0000-0000-0000-000000000000/members/123/action"},
		{http.MethodGet, "/api/v1/groups/00000000-0000-0000-0000-000000000000/members/123/warnings"},
		{http.MethodPost, "/api/v1/groups/00000000-0000-0000-0000-000000000000/members/123/warnings"},
		{http.MethodDelete, "/api/v1/groups/00000000-0000-0000-0000-000000000000/members/123/warnings"},
		{http.MethodPost, "/api/v1/bots"},
		{http.MethodPatch, "/api/v1/bots/00000000-0000-0000-0000-000000000000/status"},
		{http.MethodPut, "/api/v1/bots/00000000-0000-0000-0000-000000000000/token"},
		{http.MethodDelete, "/api/v1/bots/00000000-0000-0000-0000-000000000000"},
	} {
		request := httptest.NewRequest(route.method, route.path, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s status = %d, want %d", route.method, route.path, response.Code, http.StatusServiceUnavailable)
		}
	}
}

func TestCaptchaPublicConfigDoesNotExposeSecret(t *testing.T) {
	verifier, err := NewCaptchaVerifier("https://captcha.example.com", "public-site-key", "private-secret", "https://captcha.example.com/assets/widget.js")
	if err != nil { t.Fatal(err) }
	handler := NewHandler(slog.Default(), nil, nil, verifier)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/security/captcha", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK { t.Fatalf("status = %d", response.Code) }
	body := response.Body.String()
	if !strings.Contains(body, `"enabled":true`) || !strings.Contains(body, "https://captcha.example.com/public-site-key/") || !strings.Contains(body, "https://captcha.example.com/assets/widget.js") {
		t.Fatalf("unexpected public CAPTCHA config: %s", body)
	}
	if strings.Contains(body, "private-secret") { t.Fatal("CAP secret leaked in public config") }
}

func TestCaptchaPublicConfigDisabled(t *testing.T) {
	handler := NewHandler(slog.Default(), nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/security/captcha", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"enabled":false`) {
		t.Fatalf("unexpected disabled CAPTCHA response: %d %s", response.Code, response.Body.String())
	}
}

func TestLoginRateLimitReturnsTooManyRequests(t *testing.T) {
	handler:=&Handler{logger:slog.Default()}
	response:=httptest.NewRecorder()
	handler.writeAuthError(response,auth.ErrLoginRateLimited)
	if response.Code!=http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d",response.Code,http.StatusTooManyRequests)
	}
	if response.Header().Get("Retry-After")!="900" {
		t.Fatalf("Retry-After = %q, want 900",response.Header().Get("Retry-After"))
	}
	if !strings.Contains(response.Body.String(),"15 分钟") {
		t.Fatalf("unexpected rate limit response: %s",response.Body.String())
	}
}
