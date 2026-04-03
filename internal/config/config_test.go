package config

import "testing"

func TestLoadDefaultsHTTPAddr(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", "")
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("Load() http addr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
}

func TestLoadUsesExplicitHTTPAddr(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", "127.0.0.1:18095")
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != "127.0.0.1:18095" {
		t.Fatalf("Load() http addr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:18095")
	}
}

func TestLoadRequiresManifestDir(t *testing.T) {
	t.Setenv("GATEWAY_MANIFEST_DIR", "")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want manifest dir failure")
	}
	if err.Error() != "GATEWAY_MANIFEST_DIR is required" {
		t.Fatalf("Load() error = %q, want manifest dir error", err)
	}
}

func TestLoadRequiresAthenaBaseURL(t *testing.T) {
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want ATHENA base URL failure")
	}
	if err.Error() != "ATHENA_BASE_URL is required" {
		t.Fatalf("Load() error = %q, want ATHENA base URL error", err)
	}
}

func TestLoadRejectsNonPositiveHTTPTimeout(t *testing.T) {
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")
	t.Setenv("GATEWAY_HTTP_TIMEOUT", "-1s")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want timeout validation failure")
	}
	if err.Error() != "GATEWAY_HTTP_TIMEOUT must be greater than zero" {
		t.Fatalf("Load() error = %q, want timeout validation error", err)
	}
}
