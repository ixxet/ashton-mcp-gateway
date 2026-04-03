package config

import (
	"os"
)

type Config struct {
	HTTPAddr string
}

func Load() (Config, error) {
	return Config{
		HTTPAddr: getEnv("GATEWAY_HTTP_ADDR", ":8080"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
