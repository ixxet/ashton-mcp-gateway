package identity

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestResolverAcceptsTrustedCallerHeaders(t *testing.T) {
	resolver := NewResolver("trusted-token", nil)

	request := httptest.NewRequest("POST", "/mcp/v1/tools/call", nil)
	request.Header.Set(HeaderTrustedCallerToken, "trusted-token")
	request.Header.Set(HeaderCallerType, "interactive")
	request.Header.Set(HeaderCallerID, "operator-001")
	request.Header.Set(HeaderCallerDisplay, "Operator One")

	caller, err := resolver.Resolve(request)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if caller.Type != "interactive" {
		t.Fatalf("caller.Type = %q, want interactive", caller.Type)
	}
	if caller.ID != "operator-001" {
		t.Fatalf("caller.ID = %q, want operator-001", caller.ID)
	}
}

func TestResolverAcceptsAPIKeys(t *testing.T) {
	resolver := NewResolver("", []APIKeyCaller{{
		Key:     "automation-secret",
		ID:      "ci-bot",
		Display: "CI Bot",
	}})

	request := httptest.NewRequest("POST", "/mcp/v1/tools/call", nil)
	request.Header.Set(HeaderAPIKey, "automation-secret")

	caller, err := resolver.Resolve(request)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if caller.Type != "automation" {
		t.Fatalf("caller.Type = %q, want automation", caller.Type)
	}
	if caller.ID != "ci-bot" {
		t.Fatalf("caller.ID = %q, want ci-bot", caller.ID)
	}
}

func TestResolverRejectsMissingCallerIDInTrustedHeaderMode(t *testing.T) {
	resolver := NewResolver("trusted-token", nil)

	request := httptest.NewRequest("POST", "/mcp/v1/tools/call", nil)
	request.Header.Set(HeaderTrustedCallerToken, "trusted-token")
	request.Header.Set(HeaderCallerType, "interactive")

	_, err := resolver.Resolve(request)
	if err == nil {
		t.Fatal("Resolve() error = nil, want invalid identity")
	}
}

func TestResolverRejectsUnknownAPIKey(t *testing.T) {
	resolver := NewResolver("", []APIKeyCaller{{
		Key: "automation-secret",
		ID:  "ci-bot",
	}})

	request := httptest.NewRequest("POST", "/mcp/v1/tools/call", nil)
	request.Header.Set(HeaderAPIKey, "missing")

	_, err := resolver.Resolve(request)
	if err == nil {
		t.Fatal("Resolve() error = nil, want unknown API key failure")
	}
	if !errors.Is(err, ErrUnknownAPIKey) {
		t.Fatalf("Resolve() error = %v, want unknown API key", err)
	}
}
