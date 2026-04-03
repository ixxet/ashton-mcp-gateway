package config

import "testing"

func TestLoadDefaultsHTTPAddr(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", "")
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")

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

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want manifest dir failure")
	}
	if err.Error() != "GATEWAY_MANIFEST_DIR is required" {
		t.Fatalf("Load() error = %q, want manifest dir error", err)
	}
}
