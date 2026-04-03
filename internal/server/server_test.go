package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler()

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
}
