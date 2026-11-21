package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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
	ConfigFile string
	NeedsSetup bool
}

type persistedConfig struct {
	DatabaseURL string `json:"databaseURL"`
	RedisAddr string `json:"redisAddr"`
	RedisPassword string `json:"redisPassword"`
	RedisDB int `json:"redisDB"`
}

func Load() (Config, error) {
	configFile := env("APP_CONFIG_FILE", "/var/lib/tg-group/config.json")
	var saved persistedConfig
	if info, err := os.Lstat(configFile); err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return Config{}, fmt.Errorf("application config file %s must be a regular file with permissions 0600", filepath.Clean(configFile))
		}
		data, err := os.ReadFile(configFile)
		if err != nil {
			return Config{}, fmt.Errorf("read application config file: %w", err)
		}
		if err := json.Unmarshal(data, &saved); err != nil {
			return Config{}, fmt.Errorf("parse application config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("inspect application config file: %w", err)
	}

	cfg := Config{
		HTTPAddr: env("HTTP_ADDR", ":8080"),
		WebDir: env("WEB_DIR", "/app/web"),
		DatabaseURL: env("DATABASE_URL", strings.TrimSpace(saved.DatabaseURL)),
		RedisAddr: env("REDIS_ADDR", envOr(saved.RedisAddr, "127.0.0.1:6379")),
		RedisPassword: envOr(os.Getenv("REDIS_PASSWORD"), saved.RedisPassword),
		AdminUsername: env("INITIAL_ADMIN_USERNAME", "admin"),
		AdminPassword: env("INITIAL_ADMIN_PASSWORD", "123456"),
		BotTokenEncryptionKey: strings.TrimSpace(os.Getenv("BOT_TOKEN_ENCRYPTION_KEY")),
		ShutdownTimeout: 10 * time.Second,
		ConfigFile: configFile,
	}
	redisDBValue := env("REDIS_DB", strconv.Itoa(saved.RedisDB))
	parsed, err := strconv.Atoi(redisDBValue)
	if err != nil || parsed < 0 {
		return Config{}, fmt.Errorf("REDIS_DB must be a non-negative integer")
	}
	cfg.RedisDB = parsed
	if cfg.DatabaseURL == "" {
		cfg.NeedsSetup = true
		return cfg, nil
	}
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("initial administrator credentials must not be empty")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envOr(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
