package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT", "")
	t.Setenv("TG_GROUP_ENV", "")
	t.Setenv("CAP_BASE_URL", "")
	t.Setenv("CAP_SITE_KEY", "")
	t.Setenv("CAP_SECRET", "")
	t.Setenv("CAP_WIDGET_URL", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want %q", got.HTTPAddr, ":8080")
	}
	if got.Environment != "development" {
		t.Fatalf("Environment = %q, want %q", got.Environment, "development")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid port error")
	}
}

func TestLoadRejectsPartialCAPConfiguration(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT", "8080")
	t.Setenv("CAP_BASE_URL", "https://captcha.example.com")
	t.Setenv("CAP_SITE_KEY", "site-key")
	t.Setenv("CAP_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want incomplete CAP configuration error")
	}
}

func TestLoadReadsCompleteCAPConfiguration(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT", "8080")
	t.Setenv("CAP_BASE_URL", "https://captcha.example.com")
	t.Setenv("CAP_SITE_KEY", "site-key")
	t.Setenv("CAP_SECRET", "secret")
	t.Setenv("CAP_WIDGET_URL", "https://captcha.example.com/assets/widget.js")
	got, err := Load()
	if err != nil { t.Fatalf("Load() error = %v", err) }
	if got.CAPBaseURL != "https://captcha.example.com" || got.CAPSiteKey != "site-key" || got.CAPSecret != "secret" || got.CAPWidgetURL != "https://captcha.example.com/assets/widget.js" {
		t.Fatalf("unexpected CAP configuration: %+v", got)
	}
}
