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
	t.Setenv("PAYMENT_CREDENTIALS_ENCRYPTION_KEY", "")
	t.Setenv("PAYMENT_PUBLIC_BASE_URL", "")
	t.Setenv("TELEGRAM_AUTH_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_AUTH_BOT_USERNAME", "")

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

func TestLoadRejectsPartialTelegramAuthBotConfiguration(t *testing.T) {
    t.Setenv("TG_GROUP_HTTP_PORT","8080")
    t.Setenv("CAP_BASE_URL","")
    t.Setenv("CAP_SITE_KEY","")
    t.Setenv("CAP_SECRET","")
    t.Setenv("TELEGRAM_AUTH_BOT_TOKEN","123:token")
    t.Setenv("TELEGRAM_AUTH_BOT_USERNAME","")
    if _,err:=Load();err==nil { t.Fatal("Load() error = nil, want incomplete Telegram auth Bot configuration error") }
}

func TestLoadReadsTelegramAuthBotConfiguration(t *testing.T) {
    t.Setenv("TG_GROUP_HTTP_PORT","8080")
    t.Setenv("CAP_BASE_URL","")
    t.Setenv("CAP_SITE_KEY","")
    t.Setenv("CAP_SECRET","")
    t.Setenv("TELEGRAM_AUTH_BOT_TOKEN","123:token")
    t.Setenv("TELEGRAM_AUTH_BOT_USERNAME","@tg_group_identity_bot")
    got,err:=Load()
    if err!=nil { t.Fatalf("Load() error = %v",err) }
    if got.TelegramAuthBotToken!="123:token" || got.TelegramAuthBotUsername!="tg_group_identity_bot" {
        t.Fatalf("unexpected Telegram auth Bot config: %+v",got)
    }
}

func TestLoadRejectsUnsafeTelegramBotUsername(t *testing.T) {
    t.Setenv("TG_GROUP_HTTP_PORT","8080")
    t.Setenv("CAP_BASE_URL","")
    t.Setenv("CAP_SITE_KEY","")
    t.Setenv("CAP_SECRET","")
    t.Setenv("TELEGRAM_AUTH_BOT_TOKEN","123:token")
    t.Setenv("TELEGRAM_AUTH_BOT_USERNAME","evil.example/bot")
    if _,err:=Load();err==nil { t.Fatal("Load() error = nil, want invalid Telegram Bot username error") }
}


func TestLoadRequiresPaymentPublicURLWithEncryptionKey(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT","8080")
	t.Setenv("CAP_BASE_URL","")
	t.Setenv("CAP_SITE_KEY","")
	t.Setenv("CAP_SECRET","")
	t.Setenv("TELEGRAM_AUTH_BOT_TOKEN","")
	t.Setenv("TELEGRAM_AUTH_BOT_USERNAME","")
	t.Setenv("PAYMENT_CREDENTIALS_ENCRYPTION_KEY","a2V5")
	t.Setenv("PAYMENT_PUBLIC_BASE_URL","")
	if _,err:=Load();err==nil{t.Fatal("expected paired payment configuration validation")}
}

func TestLoadRejectsInsecurePaymentPublicURL(t *testing.T) {
	t.Setenv("TG_GROUP_HTTP_PORT","8080")
	t.Setenv("CAP_BASE_URL","")
	t.Setenv("CAP_SITE_KEY","")
	t.Setenv("CAP_SECRET","")
	t.Setenv("TELEGRAM_AUTH_BOT_TOKEN","")
	t.Setenv("TELEGRAM_AUTH_BOT_USERNAME","")
	t.Setenv("PAYMENT_CREDENTIALS_ENCRYPTION_KEY","a2V5")
	t.Setenv("PAYMENT_PUBLIC_BASE_URL","http://merchant.example")
	if _,err:=Load();err==nil{t.Fatal("expected HTTPS payment public URL validation")}
}
