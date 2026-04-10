package gateway

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/audit"
	"github.com/ixxet/ashton-mcp-gateway/internal/identity"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

type stubAthenaClient struct {
	result athena.Occupancy
	err    error
}

func (s stubAthenaClient) CurrentOccupancy(ctx context.Context, tool manifest.Tool, arguments map[string]string) (athena.Occupancy, error) {
	if s.err != nil {
		return athena.Occupancy{}, s.err
	}
	return s.result, nil
}

type stubAuditRecorder struct {
	entries []audit.Entry
	err     error
}

func (s *stubAuditRecorder) Record(ctx context.Context, entry audit.Entry) error {
	if s.err != nil {
		return s.err
	}
	s.entries = append(s.entries, entry)
	return nil
}

var testCaller = identity.Caller{
	Type:    "automation",
	ID:      "ci-bot",
	Display: "CI Bot",
	Method:  "api_key",
}

func TestCallToolReturnsSourceBackedOccupancyAndPersistsAudit(t *testing.T) {
	t.Parallel()

	auditRecorder := &stubAuditRecorder{}
	service := NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			CurrentCount: 9,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		auditRecorder,
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	result, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{
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
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
	if auditRecorder.entries[0].Outcome != "success" {
		t.Fatalf("audit outcome = %q, want success", auditRecorder.entries[0].Outcome)
	}
	if auditRecorder.entries[0].CallerID != "ci-bot" {
		t.Fatalf("audit caller_id = %q, want ci-bot", auditRecorder.entries[0].CallerID)
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
		&stubAuditRecorder{},
		slog.New(slog.NewTextHandler(&logBuffer, nil)),
	)

	_, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{
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

func TestCallToolRejectsUnknownToolAndAuditsAttempt(t *testing.T) {
	t.Parallel()

	auditRecorder := &stubAuditRecorder{}
	service := NewService(testRegistry(), stubAthenaClient{}, auditRecorder, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	_, err := service.CallTool(context.Background(), testCaller, "missing.tool", map[string]any{
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
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
	if auditRecorder.entries[0].Outcome != "unknown_tool" {
		t.Fatalf("audit outcome = %q, want unknown_tool", auditRecorder.entries[0].Outcome)
	}
}

func TestCallToolRejectsInvalidArgumentsAndAuditsAttempt(t *testing.T) {
	t.Parallel()

	auditRecorder := &stubAuditRecorder{}
	service := NewService(testRegistry(), stubAthenaClient{}, auditRecorder, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	_, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{})
	if err == nil {
		t.Fatal("CallTool() error = nil, want invalid-argument failure")
	}

	callErr, ok := err.(*ToolCallError)
	if !ok {
		t.Fatalf("CallTool() error type = %T, want *ToolCallError", err)
	}
	if callErr.StatusCode != 400 {
		t.Fatalf("CallTool() status code = %d, want 400", callErr.StatusCode)
	}
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
	if auditRecorder.entries[0].Outcome != "invalid_arguments" {
		t.Fatalf("audit outcome = %q, want invalid_arguments", auditRecorder.entries[0].Outcome)
	}
}

func TestCallToolRejectsUndeclaredArgumentsAndAuditsAttempt(t *testing.T) {
	t.Parallel()

	auditRecorder := &stubAuditRecorder{}
	service := NewService(testRegistry(), stubAthenaClient{}, auditRecorder, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	_, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{
		"facility_id": "ashtonbee",
		"unexpected":  "value",
	})
	if err == nil {
		t.Fatal("CallTool() error = nil, want undeclared-argument failure")
	}

	callErr, ok := err.(*ToolCallError)
	if !ok {
		t.Fatalf("CallTool() error type = %T, want *ToolCallError", err)
	}
	if callErr.StatusCode != 400 {
		t.Fatalf("CallTool() status code = %d, want 400", callErr.StatusCode)
	}
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
	if auditRecorder.entries[0].Outcome != "invalid_arguments" {
		t.Fatalf("audit outcome = %q, want invalid_arguments", auditRecorder.entries[0].Outcome)
	}
}

func TestCallToolRejectsWrongTypeOptionalArgumentsAndAuditsAttempt(t *testing.T) {
	t.Parallel()

	auditRecorder := &stubAuditRecorder{}
	service := NewService(testRegistryWithOptionalArgument(), stubAthenaClient{}, auditRecorder, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))

	_, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{
		"facility_id": "ashtonbee",
		"detail":      5,
	})
	if err == nil {
		t.Fatal("CallTool() error = nil, want wrong-type optional-argument failure")
	}

	callErr, ok := err.(*ToolCallError)
	if !ok {
		t.Fatalf("CallTool() error type = %T, want *ToolCallError", err)
	}
	if callErr.StatusCode != 400 {
		t.Fatalf("CallTool() status code = %d, want 400", callErr.StatusCode)
	}
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
	if auditRecorder.entries[0].Outcome != "invalid_arguments" {
		t.Fatalf("audit outcome = %q, want invalid_arguments", auditRecorder.entries[0].Outcome)
	}
}

func TestCallToolLogsUpstreamFailuresAndAuditsAttempt(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	auditRecorder := &stubAuditRecorder{}
	service := NewService(
		testRegistry(),
		stubAthenaClient{err: &athena.UpstreamError{StatusCode: 500, Message: "adapter offline"}},
		auditRecorder,
		slog.New(slog.NewTextHandler(&logBuffer, nil)),
	)

	_, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{
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
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
	if auditRecorder.entries[0].Outcome != "upstream_error" {
		t.Fatalf("audit outcome = %q, want upstream_error", auditRecorder.entries[0].Outcome)
	}
}

func TestCallToolFailClosesWhenAuditPersistenceFails(t *testing.T) {
	t.Parallel()

	service := NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			CurrentCount: 9,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		&stubAuditRecorder{err: errors.New("database offline")},
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	_, err := service.CallTool(context.Background(), testCaller, "athena.get_current_occupancy", map[string]any{
		"facility_id": "ashtonbee",
	})
	if err == nil {
		t.Fatal("CallTool() error = nil, want audit failure")
	}
	callErr, ok := err.(*ToolCallError)
	if !ok {
		t.Fatalf("CallTool() error type = %T, want *ToolCallError", err)
	}
	if callErr.StatusCode != 500 {
		t.Fatalf("CallTool() status code = %d, want 500", callErr.StatusCode)
	}
	if callErr.Kind != "audit_failure" {
		t.Fatalf("CallTool() kind = %q, want audit_failure", callErr.Kind)
	}
}

func TestCallToolReturnsZoneOccupancyShape(t *testing.T) {
	t.Parallel()

	auditRecorder := &stubAuditRecorder{}
	service := NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			ZoneID:       "gym-floor",
			CurrentCount: 4,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		auditRecorder,
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	)

	result, err := service.CallTool(context.Background(), testCaller, "athena.get_current_zone_occupancy", map[string]any{
		"facility_id": "ashtonbee",
		"zone_id":     "gym-floor",
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.ZoneID != "gym-floor" {
		t.Fatalf("CallTool() zone_id = %q, want gym-floor", result.ZoneID)
	}
	if len(auditRecorder.entries) != 1 {
		t.Fatalf("len(audit entries) = %d, want 1", len(auditRecorder.entries))
	}
}

func testRegistry() manifest.Registry {
	var occupancy manifest.Tool
	occupancy.Name = "athena.get_current_occupancy"
	occupancy.Description = "Read occupancy"
	occupancy.ReadOnly = true
	occupancy.Input.Required = []string{"facility_id"}
	occupancy.Input.Properties = map[string]struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	}{
		"facility_id": {Type: "string", Description: "Facility"},
	}
	occupancy.Upstream.Service = "athena"
	occupancy.Upstream.Method = "GET"
	occupancy.Upstream.Path = "/api/v1/presence/count"
	occupancy.Upstream.Query = map[string]string{"facility": "facility_id"}

	var zone manifest.Tool
	zone.Name = "athena.get_current_zone_occupancy"
	zone.Description = "Read zone occupancy"
	zone.ReadOnly = true
	zone.Input.Required = []string{"facility_id", "zone_id"}
	zone.Input.Properties = map[string]struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	}{
		"facility_id": {Type: "string", Description: "Facility"},
		"zone_id":     {Type: "string", Description: "Zone"},
	}
	zone.Upstream.Service = "athena"
	zone.Upstream.Method = "GET"
	zone.Upstream.Path = "/api/v1/presence/count"
	zone.Upstream.Query = map[string]string{
		"facility": "facility_id",
		"zone":     "zone_id",
	}

	return manifest.Registry{Tools: []manifest.Tool{occupancy, zone}}
}

func testRegistryWithOptionalArgument() manifest.Registry {
	registry := testRegistry()
	registry.Tools[0].Input.Properties["detail"] = struct {
		Type        string `json:"type"`
		Description string `json:"description"`
	}{Type: "string", Description: "Detail level"}
	return registry
}
