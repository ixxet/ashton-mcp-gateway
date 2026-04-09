package identity

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	HeaderAPIKey             = "X-Gateway-API-Key"
	HeaderTrustedCallerToken = "X-Gateway-Trusted-Caller-Token"
	HeaderCallerType         = "X-Gateway-Caller-Type"
	HeaderCallerID           = "X-Gateway-Caller-Id"
	HeaderCallerDisplay      = "X-Gateway-Caller-Display"
)

var (
	ErrMissingIdentity   = errors.New("caller identity is required")
	ErrInvalidIdentity   = errors.New("caller identity is invalid")
	ErrAmbiguousIdentity = errors.New("multiple caller identity methods are not allowed")
	ErrUnknownAPIKey     = errors.New("API key is unknown")
)

type Caller struct {
	Type    string
	ID      string
	Display string
	Method  string
}

type APIKeyCaller struct {
	Key     string `json:"key"`
	ID      string `json:"id"`
	Display string `json:"display"`
}

type Resolver struct {
	trustedCallerToken string
	apiKeys            map[string]Caller
}

func NewResolver(trustedCallerToken string, apiKeys []APIKeyCaller) *Resolver {
	resolver := &Resolver{
		trustedCallerToken: strings.TrimSpace(trustedCallerToken),
		apiKeys:            make(map[string]Caller, len(apiKeys)),
	}

	for _, apiKey := range apiKeys {
		resolver.apiKeys[apiKey.Key] = Caller{
			Type:    "automation",
			ID:      strings.TrimSpace(apiKey.ID),
			Display: strings.TrimSpace(apiKey.Display),
			Method:  "api_key",
		}
	}

	return resolver
}

func (r *Resolver) Resolve(request *http.Request) (Caller, error) {
	apiKey := strings.TrimSpace(request.Header.Get(HeaderAPIKey))
	trustedToken := strings.TrimSpace(request.Header.Get(HeaderTrustedCallerToken))
	callerType := strings.TrimSpace(request.Header.Get(HeaderCallerType))
	callerID := strings.TrimSpace(request.Header.Get(HeaderCallerID))
	callerDisplay := strings.TrimSpace(request.Header.Get(HeaderCallerDisplay))

	hasAPIKey := apiKey != ""
	hasTrustedHeaders := trustedToken != "" || callerType != "" || callerID != "" || callerDisplay != ""
	if hasAPIKey && hasTrustedHeaders {
		return Caller{}, ErrAmbiguousIdentity
	}

	if hasAPIKey {
		caller, ok := r.apiKeys[apiKey]
		if !ok {
			return Caller{}, fmt.Errorf("%w: %w", ErrInvalidIdentity, ErrUnknownAPIKey)
		}
		return caller, nil
	}

	if hasTrustedHeaders {
		if r.trustedCallerToken == "" {
			return Caller{}, fmt.Errorf("%w: trusted caller header mode is not configured", ErrInvalidIdentity)
		}
		if trustedToken != r.trustedCallerToken {
			return Caller{}, fmt.Errorf("%w: trusted caller token is not recognized", ErrInvalidIdentity)
		}
		if callerID == "" {
			return Caller{}, fmt.Errorf("%w: caller id header is required", ErrInvalidIdentity)
		}
		switch callerType {
		case "interactive", "internal":
		default:
			return Caller{}, fmt.Errorf("%w: caller type must be interactive or internal", ErrInvalidIdentity)
		}

		return Caller{
			Type:    callerType,
			ID:      callerID,
			Display: callerDisplay,
			Method:  "trusted_headers",
		}, nil
	}

	return Caller{}, ErrMissingIdentity
}
