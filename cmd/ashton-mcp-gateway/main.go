package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/audit"
	"github.com/ixxet/ashton-mcp-gateway/internal/config"
	"github.com/ixxet/ashton-mcp-gateway/internal/gateway"
	"github.com/ixxet/ashton-mcp-gateway/internal/identity"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
	"github.com/ixxet/ashton-mcp-gateway/internal/server"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

var version = "dev"

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

	auditStore, err := audit.Open(context.Background(), cfg.AuditDatabaseURL)
	if err != nil {
		slog.Error("gateway audit store initialization failed", "error", err)
		os.Exit(1)
	}
	defer auditStore.Close()

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	athenaClient := athena.NewClient(cfg.AthenaBaseURL, httpClient)
	resolver := identity.NewResolver(cfg.TrustedCallerToken, cfg.APIKeys)
	service := gateway.NewService(registry, athenaClient, auditStore, slog.Default())

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.NewHandler(registry, service, resolver),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	shutdownContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	slog.Info(
		"starting gateway server",
		"version", version,
		"addr", cfg.HTTPAddr,
		"manifest_dir", cfg.ManifestDir,
		"athena_base_url", cfg.AthenaBaseURL,
		"audit_database_configured", cfg.AuditDatabaseURL != "",
		"tools_loaded", len(registry.Tools),
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("gateway server stopped", "error", err)
			os.Exit(1)
		}
	case <-shutdownContext.Done():
		slog.Info("gateway shutdown requested")
	}

	shutdownDeadline, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownDeadline); err != nil {
		slog.Error("gateway shutdown failed", "error", err)
		os.Exit(1)
	}
}
