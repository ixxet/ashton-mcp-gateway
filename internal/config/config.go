package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr    string
	ManifestDir string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getEnv("GATEWAY_HTTP_ADDR", ":8080"),
		ManifestDir: os.Getenv("GATEWAY_MANIFEST_DIR"),
	}

	if cfg.ManifestDir == "" {
		return Config{}, fmt.Errorf("GATEWAY_MANIFEST_DIR is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
