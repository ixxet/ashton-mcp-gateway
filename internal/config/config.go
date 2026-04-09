package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ixxet/ashton-mcp-gateway/internal/identity"
)

type Config struct {
	HTTPAddr           string
	ManifestDir        string
	AthenaBaseURL      string
	AuditDatabaseURL   string
	TrustedCallerToken string
	APIKeys            []identity.APIKeyCaller
	HTTPTimeout        time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           getEnv("GATEWAY_HTTP_ADDR", ":8080"),
		ManifestDir:        os.Getenv("GATEWAY_MANIFEST_DIR"),
		AthenaBaseURL:      os.Getenv("ATHENA_BASE_URL"),
		AuditDatabaseURL:   os.Getenv("GATEWAY_AUDIT_DATABASE_URL"),
		TrustedCallerToken: strings.TrimSpace(os.Getenv("GATEWAY_TRUSTED_CALLER_TOKEN")),
		HTTPTimeout:        5 * time.Second,
	}

	if cfg.ManifestDir == "" {
		return Config{}, fmt.Errorf("GATEWAY_MANIFEST_DIR is required")
	}
	if cfg.AthenaBaseURL == "" {
		return Config{}, fmt.Errorf("ATHENA_BASE_URL is required")
	}
	if cfg.AuditDatabaseURL == "" {
		return Config{}, fmt.Errorf("GATEWAY_AUDIT_DATABASE_URL is required")
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

	if value := strings.TrimSpace(os.Getenv("GATEWAY_API_KEYS_JSON")); value != "" {
		if err := json.Unmarshal([]byte(value), &cfg.APIKeys); err != nil {
			return Config{}, fmt.Errorf("GATEWAY_API_KEYS_JSON is invalid: %w", err)
		}
		for _, apiKey := range cfg.APIKeys {
			if strings.TrimSpace(apiKey.Key) == "" {
				return Config{}, fmt.Errorf("GATEWAY_API_KEYS_JSON entries must include key")
			}
			if strings.TrimSpace(apiKey.ID) == "" {
				return Config{}, fmt.Errorf("GATEWAY_API_KEYS_JSON entries must include id")
			}
		}
	}

	if cfg.TrustedCallerToken == "" && len(cfg.APIKeys) == 0 {
		return Config{}, fmt.Errorf("at least one caller identity method must be configured")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
