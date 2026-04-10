package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/audit"
	"github.com/ixxet/ashton-mcp-gateway/internal/gateway"
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
}

func (s *stubAuditRecorder) Record(ctx context.Context, entry audit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "\"service\":\"ashton-mcp-gateway\"") {
		t.Fatalf("body = %q, want gateway service field", body)
	}
	if !strings.Contains(body, "\"status\":\"ok\"") {
		t.Fatalf("body = %q, want ok status", body)
	}
	if !strings.Contains(body, "\"manifests_loaded\":2") {
		t.Fatalf("body = %q, want manifests_loaded 2", body)
	}
}

func TestToolsListEndpointReturnsRegisteredToolsWithoutIdentity(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/list", bytes.NewBufferString(`{}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "\"name\":\"athena.get_current_occupancy\"") {
		t.Fatalf("body = %q, want first tool name", body)
	}
	if !strings.Contains(body, "\"name\":\"athena.get_current_zone_occupancy\"") {
		t.Fatalf("body = %q, want second tool name", body)
	}
}

func TestToolsCallEndpointRequiresCallerIdentity(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {"facility_id": "ashtonbee"}
	}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestToolsCallEndpointRejectsMalformedTrustedHeaders(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {"facility_id": "ashtonbee"}
	}`))
	request.Header.Set(identity.HeaderTrustedCallerToken, "trusted-token")
	request.Header.Set(identity.HeaderCallerType, "interactive")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestToolsCallEndpointRejectsUnknownAPIKey(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {"facility_id": "ashtonbee"}
	}`))
	request.Header.Set(identity.HeaderAPIKey, "missing")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestToolsCallEndpointRoutesOccupancyToolWithTrustedCaller(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {"facility_id": "ashtonbee"}
	}`))
	request.Header.Set(identity.HeaderTrustedCallerToken, "trusted-token")
	request.Header.Set(identity.HeaderCallerType, "interactive")
	request.Header.Set(identity.HeaderCallerID, "operator-001")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "\"facility_id\":\"ashtonbee\"") {
		t.Fatalf("body = %q, want facility_id ashtonbee", body)
	}
	if !strings.Contains(body, "\"source_service\":\"athena\"") {
		t.Fatalf("body = %q, want source_service athena", body)
	}
}

func TestToolsCallEndpointRoutesZoneOccupancyToolWithAPIKey(t *testing.T) {
	handler := NewHandler(testRegistry(), testServiceWithZoneResult(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_zone_occupancy",
		"arguments": {"facility_id": "ashtonbee", "zone_id": "gym-floor"}
	}`))
	request.Header.Set(identity.HeaderAPIKey, "automation-secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "\"zone_id\":\"gym-floor\"") {
		t.Fatalf("body = %q, want zone_id gym-floor", recorder.Body.String())
	}
}

func TestToolsCallEndpointRejectsUnknownTool(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "missing.tool",
		"arguments": {"facility_id": "ashtonbee"}
	}`))
	request.Header.Set(identity.HeaderAPIKey, "automation-secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestToolsCallEndpointRejectsMalformedJSON(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{"tool_name":`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestToolsCallEndpointRejectsUnknownTopLevelFields(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {"facility_id": "ashtonbee"},
		"unexpected": true
	}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestToolsCallEndpointRejectsOversizedJSONBody(t *testing.T) {
	handler := NewHandler(testRegistry(), testService(), testResolver())

	oversizedArgument := strings.Repeat("a", int(maxToolCallRequestBytes))
	payload := fmt.Sprintf(`{"tool_name":"athena.get_current_occupancy","arguments":{"facility_id":"%s"}}`, oversizedArgument)
	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(payload))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "request body is too large") {
		t.Fatalf("body = %q, want size failure", recorder.Body.String())
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

func testService() *gateway.Service {
	return gateway.NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			CurrentCount: 9,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		&stubAuditRecorder{},
		nil,
	)
}

func testServiceWithZoneResult() *gateway.Service {
	return gateway.NewService(
		testRegistry(),
		stubAthenaClient{result: athena.Occupancy{
			FacilityID:   "ashtonbee",
			ZoneID:       "gym-floor",
			CurrentCount: 4,
			ObservedAt:   "2026-04-03T11:05:00Z",
			Source:       "athena",
		}},
		&stubAuditRecorder{},
		nil,
	)
}

func testResolver() *identity.Resolver {
	return identity.NewResolver("trusted-token", []identity.APIKeyCaller{{
		Key:     "automation-secret",
		ID:      "ci-bot",
		Display: "CI Bot",
	}})
}
