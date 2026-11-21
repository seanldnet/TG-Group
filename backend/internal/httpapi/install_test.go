package httpapi

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDatabaseURLEscapesCredentialsAndIPv6(t *testing.T) {
	input := installRequest{
		PostgresHost: "::1", PostgresPort: 5432, PostgresDatabase: "tg_group",
		PostgresUsername: "user@example", PostgresPassword: "p@ss:/?#",
		PostgresSSLMode: "require",
	}
	raw, err := buildDatabaseURL(input)
	if err != nil {
		t.Fatalf("buildDatabaseURL() error = %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	username := parsed.User.Username()
	password, ok := parsed.User.Password()
	if !ok || username != input.PostgresUsername || password != input.PostgresPassword {
		t.Fatalf("credentials were not safely encoded: %q / %q", username, password)
	}
	if parsed.Host != "[::1]:5432" || parsed.Path != "/tg_group" || parsed.Query().Get("sslmode") != "require" {
		t.Fatalf("unexpected PostgreSQL URL: %s", raw)
	}
}

func TestPersistConfigCreatesPrivateInstallLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "config.json")
	handler := &InstallHandler{ConfigFile: path}
	config := persistedInstallConfig{
		DatabaseURL: "postgres://owner:secret@db:5432/tg_group?sslmode=require",
		RedisAddr: "redis:6379", RedisPassword: "redis-secret", RedisDB: 2,
	}
	if err := handler.persistConfig(config); err != nil {
		t.Fatalf("persistConfig() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var saved persistedInstallConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if saved != config {
		t.Fatalf("saved config = %#v, want %#v", saved, config)
	}
	if handler.isInstalled() != true {
		t.Fatal("installation lock should exist after config persistence")
	}
	if err := handler.persistConfig(config); err == nil {
		t.Fatal("persistConfig() should not overwrite an existing installation lock")
	}
}
