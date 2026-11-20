package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr string
	WebDir string
	DatabaseURL string
	RedisAddr string
	RedisPassword string
	RedisDB int
	AdminUsername string
	AdminPassword string
	BotTokenEncryptionKey string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr: env("HTTP_ADDR", ":8080"),
		WebDir: env("WEB_DIR", "/app/web"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisAddr: env("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		AdminUsername: env("INITIAL_ADMIN_USERNAME", "admin"),
		AdminPassword: env("INITIAL_ADMIN_PASSWORD", "123456"),
		BotTokenEncryptionKey: strings.TrimSpace(os.Getenv("BOT_TOKEN_ENCRYPTION_KEY")),
		ShutdownTimeout: 10 * time.Second,
	}
	parsed, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil || parsed < 0 { return Config{}, fmt.Errorf("REDIS_DB must be a non-negative integer") }
	cfg.RedisDB = parsed
	if cfg.DatabaseURL == "" { return Config{}, fmt.Errorf("DATABASE_URL is required") }
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" { return Config{}, fmt.Errorf("initial administrator credentials must not be empty") }
	return cfg, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" { return value }
	return fallback
}
