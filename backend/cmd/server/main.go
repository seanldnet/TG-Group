package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/seanldnet/tg-group/backend/internal/config"
	"github.com/seanldnet/tg-group/backend/internal/httpapi"
	"github.com/seanldnet/tg-group/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil { return err }
	startupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := store.OpenPostgres(startupCtx, cfg.DatabaseURL)
	if err != nil { return err }
	defer db.Close()
	if err := store.Migrate(startupCtx, db); err != nil { return err }
	if err := store.SeedOwner(startupCtx, db, cfg.AdminUsername, cfg.AdminPassword); err != nil { return err }

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword, DB: cfg.RedisDB})
	if err := redisClient.Ping(startupCtx).Err(); err != nil {
		_ = redisClient.Close()
		return err
	}
	defer redisClient.Close()
	mux := http.NewServeMux()
	httpapi.HealthHandler{DB: db, Redis: redisClient}.Register(mux)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 5*time.Second, IdleTimeout: 60*time.Second}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) { return err }
		return nil
	case <-signalCtx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		return server.Shutdown(shutdownCtx)
	}
}
