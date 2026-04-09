package config

import "testing"

func TestLoadDefaultsHTTPAddr(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", "")
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "trusted-token")

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
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "trusted-token")

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
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "trusted-token")

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
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "trusted-token")

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
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "trusted-token")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want timeout validation failure")
	}
	if err.Error() != "GATEWAY_HTTP_TIMEOUT must be greater than zero" {
		t.Fatalf("Load() error = %q, want timeout validation error", err)
	}
}

func TestLoadRequiresAuditDatabaseURL(t *testing.T) {
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "trusted-token")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want audit database failure")
	}
	if err.Error() != "GATEWAY_AUDIT_DATABASE_URL is required" {
		t.Fatalf("Load() error = %q, want audit database error", err)
	}
}

func TestLoadRequiresIdentityConfiguration(t *testing.T) {
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "")
	t.Setenv("GATEWAY_API_KEYS_JSON", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want identity configuration failure")
	}
	if err.Error() != "at least one caller identity method must be configured" {
		t.Fatalf("Load() error = %q, want identity configuration error", err)
	}
}

func TestLoadParsesAPIKeysJSON(t *testing.T) {
	t.Setenv("GATEWAY_MANIFEST_DIR", "/tmp/manifests")
	t.Setenv("ATHENA_BASE_URL", "http://127.0.0.1:18090")
	t.Setenv("GATEWAY_AUDIT_DATABASE_URL", "postgres://gateway:gateway@127.0.0.1:15432/gateway?sslmode=disable")
	t.Setenv("GATEWAY_TRUSTED_CALLER_TOKEN", "")
	t.Setenv("GATEWAY_API_KEYS_JSON", `[{"id":"ci-bot","display":"CI Bot","key":"secret"}]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.APIKeys) != 1 {
		t.Fatalf("Load() api key count = %d, want 1", len(cfg.APIKeys))
	}
	if cfg.APIKeys[0].ID != "ci-bot" {
		t.Fatalf("Load() api key id = %q, want %q", cfg.APIKeys[0].ID, "ci-bot")
	}
}
