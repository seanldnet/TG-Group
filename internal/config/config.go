package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr    string
	Environment string
}

func Load() (Config, error) {
	port := os.Getenv("TG_GROUP_HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("TG_GROUP_HTTP_PORT must be a valid TCP port: %q", port)
	}

	environment := os.Getenv("TG_GROUP_ENV")
	if environment == "" {
		environment = "development"
	}

	return Config{
		HTTPAddr:    fmt.Sprintf(":%d", portNumber),
		Environment: environment,
	}, nil
}
