package athena

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

func TestClientCurrentOccupancyConsumesAthenaReadSurface(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/v1/presence/count" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/presence/count")
		}
		if r.URL.Query().Get("facility") != "ashtonbee" {
			t.Fatalf("facility query = %q, want %q", r.URL.Query().Get("facility"), "ashtonbee")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","current_count":9,"observed_at":"2026-04-03T11:05:00Z"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CurrentOccupancy(context.Background(), testTool(), map[string]string{"facility_id": "ashtonbee"})
	if err != nil {
		t.Fatalf("CurrentOccupancy() error = %v", err)
	}

	if result.Source != "athena" {
		t.Fatalf("CurrentOccupancy() source = %q, want %q", result.Source, "athena")
	}
	if result.CurrentCount != 9 {
		t.Fatalf("CurrentOccupancy() current_count = %d, want 9", result.CurrentCount)
	}
}

func TestClientCurrentOccupancyPreservesValidZeroCountResults(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"missing","current_count":0,"observed_at":"2026-04-03T11:05:00Z"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	result, err := client.CurrentOccupancy(context.Background(), testTool(), map[string]string{"facility_id": "missing"})
	if err != nil {
		t.Fatalf("CurrentOccupancy() error = %v", err)
	}

	if result.CurrentCount != 0 {
		t.Fatalf("CurrentOccupancy() current_count = %d, want 0", result.CurrentCount)
	}
	if result.FacilityID != "missing" {
		t.Fatalf("CurrentOccupancy() facility_id = %q, want %q", result.FacilityID, "missing")
	}
}

func TestClientCurrentOccupancyMapsUpstreamFailuresClearly(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"adapter offline"}`, http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.CurrentOccupancy(context.Background(), testTool(), map[string]string{"facility_id": "ashtonbee"})
	if err == nil {
		t.Fatal("CurrentOccupancy() error = nil, want upstream failure")
	}

	upstreamErr, ok := err.(*UpstreamError)
	if !ok {
		t.Fatalf("CurrentOccupancy() error type = %T, want *UpstreamError", err)
	}
	if upstreamErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("CurrentOccupancy() status code = %d, want %d", upstreamErr.StatusCode, http.StatusBadGateway)
	}
}

func TestClientCurrentOccupancyMapsMalformedPayloadClearly(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"ashtonbee"`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.CurrentOccupancy(context.Background(), testTool(), map[string]string{"facility_id": "ashtonbee"})
	if err == nil {
		t.Fatal("CurrentOccupancy() error = nil, want malformed payload failure")
	}

	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("CurrentOccupancy() error = %q, want malformed response text", err)
	}
	if !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("CurrentOccupancy() error = %q, want decode failure", err)
	}
}

func TestClientCurrentOccupancyRejectsMissingObservedAt(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","current_count":9}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	_, err := client.CurrentOccupancy(context.Background(), testTool(), map[string]string{"facility_id": "ashtonbee"})
	if err == nil {
		t.Fatal("CurrentOccupancy() error = nil, want missing observed_at failure")
	}
	if !strings.Contains(err.Error(), "observed_at is required") {
		t.Fatalf("CurrentOccupancy() error = %q, want missing observed_at failure", err)
	}
}

func TestClientCurrentOccupancyMapsTimeoutClearly(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","current_count":9,"observed_at":"2026-04-03T11:05:00Z"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, &http.Client{Timeout: 10 * time.Millisecond})
	_, err := client.CurrentOccupancy(context.Background(), testTool(), map[string]string{"facility_id": "ashtonbee"})
	if err == nil {
		t.Fatal("CurrentOccupancy() error = nil, want timeout failure")
	}
	if !strings.Contains(err.Error(), "athena occupancy request failed") {
		t.Fatalf("CurrentOccupancy() error = %q, want timeout wrapper", err)
	}
}

func testTool() manifest.Tool {
	var tool manifest.Tool
	tool.Name = "athena.get_current_occupancy"
	tool.ReadOnly = true
	tool.Upstream.Service = "athena"
	tool.Upstream.Method = "GET"
	tool.Upstream.Path = "/api/v1/presence/count"
	tool.Upstream.Query = map[string]string{"facility": "facility_id"}
	return tool
}

func TestClientCurrentOccupancyMapsZoneQueryWhenConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("zone") != "gym-floor" {
			t.Fatalf("zone query = %q, want %q", r.URL.Query().Get("zone"), "gym-floor")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","zone_id":"gym-floor","current_count":4,"observed_at":"2026-04-03T11:05:00Z"}`))
	}))
	defer server.Close()

	tool := testTool()
	tool.Name = "athena.get_current_zone_occupancy"
	tool.Upstream.Query["zone"] = "zone_id"

	client := NewClient(server.URL, server.Client())
	result, err := client.CurrentOccupancy(context.Background(), tool, map[string]string{
		"facility_id": "ashtonbee",
		"zone_id":     "gym-floor",
	})
	if err != nil {
		t.Fatalf("CurrentOccupancy() error = %v", err)
	}
	if result.ZoneID != "gym-floor" {
		t.Fatalf("CurrentOccupancy() zone_id = %q, want %q", result.ZoneID, "gym-floor")
	}
}
