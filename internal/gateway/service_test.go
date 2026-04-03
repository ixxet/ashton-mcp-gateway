package gateway

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

type stubAthenaClient struct {
	result athena.Occupancy
	err    error
}

func (s stubAthenaClient) CurrentOccupancy(ctx context.Context, tool manifest.Tool, facilityID string) (athena.Occupancy, error) {
	if s.err != nil {
		return athena.Occupancy{}, s.err
	}
	return s.result, nil
}

func TestCallToolReturnsSourceBackedOccupancy(t *testing.T) {
	t.Parallel()

	service := NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			CurrentCount: 9,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	result, err := service.CallTool(context.Background(), "athena.get_current_occupancy", map[string]any{
		"facility_id": "ashtonbee",
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if result.SourceService != "athena" {
		t.Fatalf("CallTool() source_service = %q, want %q", result.SourceService, "athena")
	}
	if result.CurrentCount != 9 {
		t.Fatalf("CallTool() current_count = %d, want 9", result.CurrentCount)
	}
}

func TestCallToolLogsSuccessPath(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	service := NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			CurrentCount: 9,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		slog.New(slog.NewTextHandler(&logBuffer, nil)),
	)

	_, err := service.CallTool(context.Background(), "athena.get_current_occupancy", map[string]any{
		"facility_id": "ashtonbee",
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	logLine := logBuffer.String()
	if !strings.Contains(logLine, "tool_name=athena.get_current_occupancy") {
		t.Fatalf("log = %q, want tool name", logLine)
	}
	if !strings.Contains(logLine, "source_service=athena") {
		t.Fatalf("log = %q, want source service", logLine)
	}
	if !strings.Contains(logLine, "outcome=success") {
		t.Fatalf("log = %q, want success outcome", logLine)
	}
}

func TestCallToolRejectsUnknownTool(t *testing.T) {
	t.Parallel()

	service := NewService(testRegistry(), stubAthenaClient{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	_, err := service.CallTool(context.Background(), "missing.tool", map[string]any{
		"facility_id": "ashtonbee",
	})
	if err == nil {
		t.Fatal("CallTool() error = nil, want unknown tool failure")
	}

	callErr, ok := err.(*ToolCallError)
	if !ok {
		t.Fatalf("CallTool() error type = %T, want *ToolCallError", err)
	}
	if callErr.StatusCode != 404 {
		t.Fatalf("CallTool() status code = %d, want 404", callErr.StatusCode)
	}
}

func TestCallToolLogsUpstreamFailures(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	service := NewService(
		testRegistry(),
		stubAthenaClient{err: &athena.UpstreamError{StatusCode: 500, Message: "adapter offline"}},
		slog.New(slog.NewTextHandler(&logBuffer, nil)),
	)

	_, err := service.CallTool(context.Background(), "athena.get_current_occupancy", map[string]any{
		"facility_id": "ashtonbee",
	})
	if err == nil {
		t.Fatal("CallTool() error = nil, want upstream failure")
	}

	logLine := logBuffer.String()
	if !strings.Contains(logLine, "facility_id=ashtonbee") {
		t.Fatalf("log = %q, want facility id", logLine)
	}
	if !strings.Contains(logLine, "outcome=upstream_error") {
		t.Fatalf("log = %q, want upstream_error outcome", logLine)
	}
}

func testRegistry() manifest.Registry {
	var tool manifest.Tool
	tool.Name = "athena.get_current_occupancy"
	tool.Description = "Read occupancy"
	tool.ReadOnly = true
	tool.Input.Required = []string{"facility_id"}
	tool.Input.Properties = map[string]struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	}{
		"facility_id": {Type: "string", Description: "Facility"},
	}
	tool.Upstream.Service = "athena"
	tool.Upstream.Method = "GET"
	tool.Upstream.Path = "/api/v1/presence/count"
	tool.Upstream.Query = map[string]string{"facility": "facility_id"}
	return manifest.Registry{Tools: []manifest.Tool{tool}}
}
