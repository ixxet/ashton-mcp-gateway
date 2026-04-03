package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/ixxet/ashton-mcp-gateway/internal/config"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
	"github.com/ixxet/ashton-mcp-gateway/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("gateway configuration invalid", "error", err)
		os.Exit(1)
	}

	registry, err := manifest.LoadDir(cfg.ManifestDir)
	if err != nil {
		slog.Error("gateway manifest loading failed", "dir", cfg.ManifestDir, "error", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: server.NewHandler(registry),
	}

	slog.Info("starting gateway server", "addr", cfg.HTTPAddr, "manifest_dir", cfg.ManifestDir, "tools_loaded", len(registry.Tools))

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("gateway server stopped", "error", err)
		os.Exit(1)
	}
}
