package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("APP_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("REDIS_DB", "")
	t.Setenv("INITIAL_ADMIN_USERNAME", "")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AdminUsername != "admin" || cfg.AdminPassword != "123456" {
		t.Fatalf("unexpected initial owner defaults: %q / %q", cfg.AdminUsername, cfg.AdminPassword)
	}
	if cfg.RedisDB != 0 || cfg.NeedsSetup {
		t.Fatalf("unexpected RedisDB/setup state: %d / %v", cfg.RedisDB, cfg.NeedsSetup)
	}
}

func TestLoadEntersSetupModeWithoutDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_CONFIG_FILE", filepath.Join(t.TempDir(), "not-installed.json"))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.NeedsSetup {
		t.Fatal("Load() should enter first-run setup mode without a database URL")
	}
}

func TestLoadReadsPersistedInstallConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"databaseURL":"postgres://user:pass@db:5432/tg_group","redisAddr":"redis:6379","redisPassword":"redis-secret","redisDB":3,"botTokenEncryptionKey":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_CONFIG_FILE", path)
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("CAP_SECRET_ENCRYPTION_KEY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.NeedsSetup || cfg.DatabaseURL != "postgres://user:pass@db:5432/tg_group" ||
		cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "redis-secret" || cfg.RedisDB != 3 || cfg.BotTokenEncryptionKey != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || cfg.CapSecretEncryptionKey != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected persisted configuration: %#v", cfg)
	}
}

func TestLoadRejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("APP_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("REDIS_DB", "-1")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected REDIS_DB error")
	}
}
