package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/seanldnet/tg-group/backend/internal/config"
	"github.com/seanldnet/tg-group/backend/internal/httpapi"
	"github.com/seanldnet/tg-group/backend/internal/store"
)

var (
	version = "dev"
	commit = "unknown"
	builtAt = "unknown"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Printf("TG Group %s (commit %s, built %s)\n", version, commit, builtAt)
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.NeedsSetup {
		return runInstaller(signalCtx, logger, cfg)
	}
	return runApplication(signalCtx, logger, cfg)
}

func runInstaller(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	tokenPath := cfg.ConfigFile + ".install-token"
	token, err := loadOrCreateInstallToken(tokenPath)
	if err != nil {
		return fmt.Errorf("prepare one-time installation token: %w", err)
	}
	logger.Warn("first-run installation token; keep it private and enter it in the setup page", "token", token, "token_file", tokenPath)
	mux := http.NewServeMux()
	(&httpapi.InstallHandler{ConfigFile: cfg.ConfigFile, TokenFile: tokenPath, Token: token}).Register(mux)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: mux,
		ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("first-run installation wizard listening", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func runApplication(signalCtx context.Context, logger *slog.Logger, cfg config.Config) error {
	startupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := store.OpenPostgres(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := store.Migrate(startupCtx, db); err != nil {
		return err
	}
	if err := store.SeedOwner(startupCtx, db, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return err
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	if err := redisClient.Ping(startupCtx).Err(); err != nil {
		_ = redisClient.Close()
		return err
	}
	defer redisClient.Close()
	go (&httpapi.BotRuntime{DB: db, EncryptionKey: cfg.BotTokenEncryptionKey, Logger: logger}).Run(signalCtx)

	apiMux := http.NewServeMux()
	httpapi.HealthHandler{DB: db, Redis: redisClient}.Register(apiMux)
	authHandler := httpapi.AuthHandler{DB: db, Redis: redisClient}
	authHandler.Register(apiMux)
	httpapi.BotHandler{DB: db, EncryptionKey: cfg.BotTokenEncryptionKey}.Register(apiMux, authHandler)

	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	mux.Handle("/health/", apiMux)
	mux.Handle("/", http.FileServer(http.Dir(cfg.WebDir)))
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-signalCtx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		return server.Shutdown(shutdownCtx)
	}
}


func loadOrCreateInstallToken(path string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return "", err
	}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return "", fmt.Errorf("installation token file must be a regular file with permissions 0600")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		token := strings.TrimSpace(string(data))
		if len(token) < 32 {
			return "", fmt.Errorf("installation token file is invalid")
		}
		return token, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			return loadOrCreateInstallToken(path)
		}
		return "", err
	}
	if _, err := file.WriteString(token + "\\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return token, nil
}
