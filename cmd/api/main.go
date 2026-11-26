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

	"github.com/seanldnet/tg-group/internal/auth"
	"github.com/seanldnet/tg-group/internal/bots"
	"github.com/seanldnet/tg-group/internal/config"
	"github.com/seanldnet/tg-group/internal/database"
	"github.com/seanldnet/tg-group/internal/httpapi"
	"github.com/seanldnet/tg-group/internal/telegramruntime"
	"github.com/seanldnet/tg-group/internal/telegramauthbot"
	"github.com/seanldnet/tg-group/internal/platformsettings"
)

func main() {
	cfg, err := config.Load()
	if err != nil { slog.Error("load configuration", "error", err); os.Exit(1) }
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	startupCtx, startupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startupCancel()

	var db *database.DB
	if cfg.DatabaseURL != "" {
		db, err = database.Open(startupCtx, cfg.DatabaseURL)
		if err != nil { logger.Error("connect PostgreSQL", "error", err); os.Exit(1) }
		defer db.Close()
		if err := db.Migrate(startupCtx); err != nil { logger.Error("run database migrations", "error", err); os.Exit(1) }
		logger.Info("PostgreSQL connected and migrations applied")
	} else {
		logger.Warn("DATABASE_URL is empty; running in API-only mode")
	}

	var authService *auth.Service
	var botService *bots.Service
	var botPoller *telegramruntime.Poller
	var telegramIdentityManager *telegramauthbot.Manager
	var platformBotSettings *platformsettings.Service
	if db != nil {
		authService = auth.NewService(db.Pool)
		if err := authService.EnsureDemoAdmin(startupCtx); err != nil { logger.Error("bootstrap demo administrator", "error", err); os.Exit(1) }
		logger.Info("demo administrator bootstrap checked (admin / 123456; existing password is never reset)")
		telegramIdentityManager = telegramauthbot.NewManager(authService,logger)
		if cfg.BotTokenEncryptionKey != "" {
			platformBotSettings,err=platformsettings.New(db.Pool,cfg.BotTokenEncryptionKey)
			if err!=nil {logger.Error("configure platform Bot settings encryption","error",err);os.Exit(1)}
			storedStatus,statusErr:=platformBotSettings.Status(startupCtx)
			if statusErr!=nil {logger.Error("load platform Bot settings","error",statusErr);os.Exit(1)}
			if storedStatus.Managed {
				if storedStatus.Configured {
					token,username,loadErr:=platformBotSettings.LoadToken(startupCtx)
					if loadErr!=nil {logger.Error("decrypt platform Bot settings","error",loadErr);os.Exit(1)}
					cfg.TelegramAuthBotToken,cfg.TelegramAuthBotUsername=token,username
				} else { cfg.TelegramAuthBotToken,cfg.TelegramAuthBotUsername="","" }
			}
		}
		if cfg.BotTokenEncryptionKey != "" {
			botService, err = bots.NewService(db.Pool, cfg.BotTokenEncryptionKey)
			if err != nil { logger.Error("configure Telegram bot token encryption", "error", err); os.Exit(1) }
			botPoller = telegramruntime.NewPoller(db.Pool, botService, logger)
		} else {
			logger.Warn("BOT_TOKEN_ENCRYPTION_KEY is empty; Telegram Bot management is disabled")
		}
		telegramIdentityManager.SetBotService(botService)
		if cfg.TelegramAuthBotToken!="" {telegramIdentityManager.Configure(cfg.TelegramAuthBotToken,cfg.TelegramAuthBotUsername)}
	}
	captchaVerifier, err := httpapi.NewCaptchaVerifier(cfg.CAPBaseURL, cfg.CAPSiteKey, cfg.CAPSecret, cfg.CAPWidgetURL)
	if err != nil { logger.Error("configure CAP.js verification", "error", err); os.Exit(1) }
	if captchaVerifier != nil { logger.Info("CAP.js server-side verification enabled") }
	handler := httpapi.NewHandlerWithPlatformBot(logger, authService, botService, captchaVerifier, cfg.TelegramAuthBotUsername, platformBotSettings, telegramIdentityManager)
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: handler,
		ReadHeaderTimeout: 5*time.Second, IdleTimeout: 60*time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", "address", cfg.HTTPAddr, "environment", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var pollerDone chan struct{}
	defer func(){if telegramIdentityManager!=nil {telegramIdentityManager.Stop()}}()
	if botPoller != nil {
		pollerDone = make(chan struct{})
		go func() {
			defer close(pollerDone)
			botPoller.Run(ctx)
		}()
		logger.Info("Telegram Bot long-polling manager started")
	}
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) { logger.Error("HTTP server stopped unexpectedly", "error", err); os.Exit(1) }
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil { logger.Error("graceful shutdown failed", "error", err); os.Exit(1) }
		if telegramIdentityManager!=nil {telegramIdentityManager.Stop()}
		if pollerDone != nil {
			select {
			case <-pollerDone:
			case <-shutdownCtx.Done():
				logger.Warn("Telegram Bot workers did not stop before shutdown deadline")
			}
		}
		logger.Info("HTTP server stopped")
	}
}
