package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Env             string
	HttpAddr        string
	DatabaseURL     string
	ShutdownTimeout time.Duration
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func Load() (Config, error) {
	var errs []error

	var cfg = Config{
		Env:         getEnv("APP_ENV", "local"),
		HttpAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}

	timeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err))
	}
	cfg.ShutdownTimeout = timeout

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
