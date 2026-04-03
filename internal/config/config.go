package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr      string
	ManifestDir   string
	AthenaBaseURL string
	HTTPTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      getEnv("GATEWAY_HTTP_ADDR", ":8080"),
		ManifestDir:   os.Getenv("GATEWAY_MANIFEST_DIR"),
		AthenaBaseURL: os.Getenv("ATHENA_BASE_URL"),
		HTTPTimeout:   5 * time.Second,
	}

	if cfg.ManifestDir == "" {
		return Config{}, fmt.Errorf("GATEWAY_MANIFEST_DIR is required")
	}
	if cfg.AthenaBaseURL == "" {
		return Config{}, fmt.Errorf("ATHENA_BASE_URL is required")
	}

	if value := os.Getenv("GATEWAY_HTTP_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("GATEWAY_HTTP_TIMEOUT is invalid: %w", err)
		}
		if timeout <= 0 {
			return Config{}, fmt.Errorf("GATEWAY_HTTP_TIMEOUT must be greater than zero")
		}
		cfg.HTTPTimeout = timeout
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
