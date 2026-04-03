package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirLoadsSingleManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeManifestFile(t, dir, "athena.json", `{
  "name": "athena.get_current_occupancy",
  "description": "Read occupancy",
  "read_only": true,
  "input": {
    "required": ["facility_id"],
    "properties": {
      "facility_id": {"type": "string", "description": "Facility"}
    }
  },
  "upstream": {
    "service": "athena",
    "method": "GET",
    "path": "/api/v1/presence/count",
    "query": {"facility": "facility_id"}
  }
}`)

	registry, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error = %v", err)
	}

	if len(registry.Tools) != 1 {
		t.Fatalf("LoadDir() tool count = %d, want 1", len(registry.Tools))
	}
	if registry.Tools[0].Name != "athena.get_current_occupancy" {
		t.Fatalf("LoadDir() tool name = %q, want %q", registry.Tools[0].Name, "athena.get_current_occupancy")
	}
}

func TestLoadDirRejectsMalformedManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeManifestFile(t, dir, "broken.json", `{"name":`)

	_, err := LoadDir(dir)
	if err == nil {
		t.Fatal("LoadDir() error = nil, want malformed manifest failure")
	}
}

func TestLoadDirRejectsDuplicateNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeManifestFile(t, dir, "one.json", validManifestJSON("athena.get_current_occupancy"))
	writeManifestFile(t, dir, "two.json", validManifestJSON("athena.get_current_occupancy"))

	_, err := LoadDir(dir)
	if err == nil {
		t.Fatal("LoadDir() error = nil, want duplicate manifest failure")
	}
}

func validManifestJSON(name string) string {
	return `{
  "name": "` + name + `",
  "description": "Read occupancy",
  "read_only": true,
  "input": {
    "required": ["facility_id"],
    "properties": {
      "facility_id": {"type": "string", "description": "Facility"}
    }
  },
  "upstream": {
    "service": "athena",
    "method": "GET",
    "path": "/api/v1/presence/count",
    "query": {"facility": "facility_id"}
  }
}`
}

func writeManifestFile(t *testing.T, dir, name, payload string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
