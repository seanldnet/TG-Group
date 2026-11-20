package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("REDIS_DB", "")
	t.Setenv("INITIAL_ADMIN_USERNAME", "")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "")
	cfg, err := Load()
	if err != nil { t.Fatalf("Load() error = %v", err) }
	if cfg.AdminUsername != "admin" || cfg.AdminPassword != "123456" {
		t.Fatalf("unexpected initial owner defaults: %q / %q", cfg.AdminUsername, cfg.AdminPassword)
	}
	if cfg.RedisDB != 0 { t.Fatalf("RedisDB = %d, want 0", cfg.RedisDB) }
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil { t.Fatal("Load() expected DATABASE_URL error") }
}

func TestLoadRejectsInvalidRedisDB(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("REDIS_DB", "-1")
	if _, err := Load(); err == nil { t.Fatal("Load() expected REDIS_DB error") }
}
