package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/ixxet/ashton-mcp-gateway/internal/config"
	"github.com/ixxet/ashton-mcp-gateway/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("gateway configuration invalid", "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: server.NewHandler(),
	}

	slog.Info("starting gateway server", "addr", cfg.HTTPAddr)

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("gateway server stopped", "error", err)
		os.Exit(1)
	}
}
