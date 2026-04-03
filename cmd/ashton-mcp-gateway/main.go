package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/config"
	"github.com/ixxet/ashton-mcp-gateway/internal/gateway"
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

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	athenaClient := athena.NewClient(cfg.AthenaBaseURL, httpClient)
	service := gateway.NewService(registry, athenaClient, slog.Default())

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: server.NewHandler(registry, service),
	}

	slog.Info(
		"starting gateway server",
		"addr", cfg.HTTPAddr,
		"manifest_dir", cfg.ManifestDir,
		"athena_base_url", cfg.AthenaBaseURL,
		"tools_loaded", len(registry.Tools),
	)

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("gateway server stopped", "error", err)
		os.Exit(1)
	}
}
