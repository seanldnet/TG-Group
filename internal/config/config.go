package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr    string
	Environment string
	DatabaseURL string
	BotTokenEncryptionKey string
	CAPBaseURL string
	CAPSiteKey string
	CAPSecret string
	CAPWidgetURL string
	TelegramAuthBotToken string
	TelegramAuthBotUsername string
}

func Load() (Config, error) {
	port := os.Getenv("TG_GROUP_HTTP_PORT")
	if port == "" { port = "8080" }
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("TG_GROUP_HTTP_PORT must be a valid TCP port: %q", port)
	}
	environment := os.Getenv("TG_GROUP_ENV")
	if environment == "" { environment = "development" }
	capBaseURL, capSiteKey, capSecret := os.Getenv("CAP_BASE_URL"), os.Getenv("CAP_SITE_KEY"), os.Getenv("CAP_SECRET")
	if (capBaseURL != "") != (capSiteKey != "") || (capSiteKey != "") != (capSecret != "") {
		return Config{}, fmt.Errorf("CAP_BASE_URL, CAP_SITE_KEY and CAP_SECRET must be configured together")
	}
	telegramAuthBotToken := os.Getenv("TELEGRAM_AUTH_BOT_TOKEN")
	telegramAuthBotUsername := strings.TrimPrefix(strings.TrimSpace(os.Getenv("TELEGRAM_AUTH_BOT_USERNAME")), "@")
	if (telegramAuthBotToken == "") != (telegramAuthBotUsername == "") { return Config{}, fmt.Errorf("TELEGRAM_AUTH_BOT_TOKEN and TELEGRAM_AUTH_BOT_USERNAME must be configured together") }
	return Config{
		TelegramAuthBotToken: telegramAuthBotToken, TelegramAuthBotUsername: telegramAuthBotUsername,
		CAPBaseURL: capBaseURL, CAPSiteKey: capSiteKey, CAPSecret: capSecret, CAPWidgetURL: os.Getenv("CAP_WIDGET_URL"),
		HTTPAddr: fmt.Sprintf(":%d", portNumber),
		Environment: environment,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		BotTokenEncryptionKey: os.Getenv("BOT_TOKEN_ENCRYPTION_KEY"),
	}, nil
}
