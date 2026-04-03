package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ixxet/ashton-mcp-gateway/internal/athena"
	"github.com/ixxet/ashton-mcp-gateway/internal/gateway"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler(testRegistry(), testService())

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
	if !strings.Contains(body, "\"manifests_loaded\":1") {
		t.Fatalf("body = %q, want manifests_loaded 1", body)
	}
}

func TestToolsListEndpointReturnsRegisteredTool(t *testing.T) {
	handler := NewHandler(testRegistry(), testService())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/list", bytes.NewBufferString(`{}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "\"name\":\"athena.get_current_occupancy\"") {
		t.Fatalf("body = %q, want tool name", body)
	}
	if !strings.Contains(body, "\"source_service\":\"athena\"") {
		t.Fatalf("body = %q, want source service athena", body)
	}
	if !strings.Contains(body, "\"required_input\":[\"facility_id\"]") {
		t.Fatalf("body = %q, want facility_id required input", body)
	}
}

func TestToolsListEndpointRejectsWrongMethod(t *testing.T) {
	handler := NewHandler(testRegistry(), testService())

	request := httptest.NewRequest(http.MethodGet, "/mcp/v1/tools/list", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestToolsCallEndpointRoutesOccupancyTool(t *testing.T) {
	handler := NewHandler(testRegistry(), testService())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {"facility_id": "ashtonbee"}
	}`))
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

func TestToolsCallEndpointRequiresFacilityID(t *testing.T) {
	handler := NewHandler(testRegistry(), testService())

	request := httptest.NewRequest(http.MethodPost, "/mcp/v1/tools/call", bytes.NewBufferString(`{
		"tool_name": "athena.get_current_occupancy",
		"arguments": {}
	}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func testRegistry() manifest.Registry {
	return manifest.Registry{
		Tools: []manifest.Tool{
			{
				Name:        "athena.get_current_occupancy",
				Description: "Read occupancy",
				ReadOnly:    true,
				Input: struct {
					Required   []string `json:"required"`
					Properties map[string]struct {
						Type        string `json:"type"`
						Description string `json:"description"`
					} `json:"properties"`
				}{
					Required: []string{"facility_id"},
				},
				Upstream: struct {
					Service string            `json:"service"`
					Method  string            `json:"method"`
					Path    string            `json:"path"`
					Query   map[string]string `json:"query"`
				}{
					Service: "athena",
					Method:  "GET",
					Path:    "/api/v1/presence/count",
				},
			},
		},
	}
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
		nil,
	)
}

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
