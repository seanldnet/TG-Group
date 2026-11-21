package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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
	handler := &InstallHandler{ConfigFile: path, Token: "integration-token"}
	config := persistedInstallConfig{
		DatabaseURL: "postgres://owner:secret@db:5432/tg_group?sslmode=require",
		RedisAddr: "redis:6379", RedisPassword: "redis-secret", RedisDB: 2,
		BotTokenEncryptionKey: strings.Repeat("a", 64),
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


func TestInstallCompleteIntegration(t *testing.T) {
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	redisAddress := os.Getenv("TEST_REDIS_ADDR")
	if testDatabaseURL == "" || redisAddress == "" {
		t.Skip("TEST_DATABASE_URL and TEST_REDIS_ADDR are required for install integration test")
	}
	parsed, err := url.Parse(testDatabaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse PostgreSQL port: %v", err)
	}
	password, _ := parsed.User.Password()
	input := installRequest{
		PostgresHost: parsed.Hostname(), PostgresPort: port,
		PostgresDatabase: strings.TrimPrefix(parsed.Path, "/"),
		PostgresUsername: parsed.User.Username(), PostgresPassword: password,
		PostgresSSLMode: parsed.Query().Get("sslmode"),
		RedisAddress: redisAddress, OwnerUsername: fmt.Sprintf("owner-%d", time.Now().UnixNano()),
		OwnerPassword: "Strong-Owner-Password-2026",
	}
	configPath := filepath.Join(t.TempDir(), "config.json")
	handler := &InstallHandler{ConfigFile: configPath, Token: "integration-token"}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/install/complete", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Install-Token", "integration-token")
	response := httptest.NewRecorder()
	handler.complete(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("install completion status = %d, body = %s", response.Code, response.Body.String())
	}
	var result struct {
		Code int `json:"code"`
		Data struct {
			Installed bool `json:"installed"`
			RestartRequired bool `json:"restartRequired"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if result.Code != 0 || !result.Data.Installed || !result.Data.RestartRequired {
		t.Fatalf("unexpected install response: %#v", result)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat install config: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("install config mode = %o, want 600", info.Mode().Perm())
	}
	var persisted persistedInstallConfig
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(configBytes, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted.BotTokenEncryptionKey) != 64 {
		t.Fatalf("generated Bot token encryption key length = %d, want 64 hex characters", len(persisted.BotTokenEncryptionKey))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	var passwordHash string
	if err := db.QueryRow(ctx, "SELECT password_hash FROM platform_users WHERE username = $1 AND role = 'owner'", input.OwnerUsername).Scan(&passwordHash); err != nil {
		t.Fatalf("find initialized Owner: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.OwnerPassword)); err != nil {
		t.Fatalf("Owner password was not bcrypt-hashed correctly: %v", err)
	}

	lockedPage := httptest.NewRecorder()
	handler.page(lockedPage, httptest.NewRequest(http.MethodGet, "/", nil))
	if lockedPage.Code != http.StatusGone {
		t.Fatalf("installer page status after completion = %d, want 410", lockedPage.Code)
	}
	lockedAPI := httptest.NewRecorder()
	lockedRequest := httptest.NewRequest(http.MethodPost, "/api/install/test", bytes.NewReader(body))
	lockedRequest.Header.Set("X-Install-Token", "integration-token")
	handler.testConnections(lockedAPI, lockedRequest)
	if lockedAPI.Code != http.StatusGone {
		t.Fatalf("installer test API status after completion = %d, want 410", lockedAPI.Code)
	}

	// A second installation attempt against the same database must not replace
	// the existing Owner's password or username.
	second := input
	second.OwnerPassword = "Different-Owner-Password-2026"
	secondBody, _ := json.Marshal(second)
	secondHandler := &InstallHandler{ConfigFile: filepath.Join(t.TempDir(), "second-config.json"), Token: "integration-token"}
	secondResponse := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/install/complete", bytes.NewReader(secondBody))
	secondRequest.Header.Set("X-Install-Token", "integration-token")
	secondHandler.complete(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusConflict {
		t.Fatalf("second install status = %d, want 409; body = %s", secondResponse.Code, secondResponse.Body.String())
	}
}


func TestInstallerRejectsInvalidBootstrapToken(t *testing.T) {
	handler := &InstallHandler{ConfigFile: filepath.Join(t.TempDir(), "config.json"), Token: "expected-secret-token"}
	request := httptest.NewRequest(http.MethodPost, "/api/install/test", bytes.NewReader([]byte("{}")))
	request.Header.Set("X-Install-Token", "wrong-token")
	response := httptest.NewRecorder()
	handler.testConnections(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("invalid token status = %d, want 403", response.Code)
	}
}
