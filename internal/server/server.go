package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ixxet/ashton-mcp-gateway/internal/gateway"
	"github.com/ixxet/ashton-mcp-gateway/internal/identity"
	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

const maxToolCallRequestBytes int64 = 64 << 10

type healthResponse struct {
	Service       string `json:"service"`
	Status        string `json:"status"`
	ManifestsLoad int    `json:"manifests_loaded"`
}

type toolListResponse struct {
	Tools []toolSummary `json:"tools"`
}

type toolCallRequest struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
}

type toolCallResponse struct {
	Result gateway.ToolCallResult `json:"result"`
}

type toolSummary struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ReadOnly      bool     `json:"read_only"`
	RequiredInput []string `json:"required_input"`
	SourceService string   `json:"source_service"`
}

func NewHandler(registry manifest.Registry, service *gateway.Service, resolver *identity.Resolver) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Service:       "ashton-mcp-gateway",
			Status:        "ok",
			ManifestsLoad: len(registry.Tools),
		})
	})
	mux.HandleFunc("/mcp/v1/tools/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		tools := make([]toolSummary, 0, len(registry.Tools))
		for _, tool := range registry.Tools {
			tools = append(tools, toolSummary{
				Name:          tool.Name,
				Description:   tool.Description,
				ReadOnly:      tool.ReadOnly,
				RequiredInput: append([]string(nil), tool.Input.Required...),
				SourceService: tool.Upstream.Service,
			})
		}

		writeJSON(w, http.StatusOK, toolListResponse{Tools: tools})
	})
	mux.HandleFunc("/mcp/v1/tools/call", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method not allowed",
			})
			return
		}

		var request toolCallRequest
		if err := decodeToolCallRequest(w, r, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}

		caller, err := resolver.Resolve(r)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, identity.ErrAmbiguousIdentity) || (errors.Is(err, identity.ErrInvalidIdentity) && !errors.Is(err, identity.ErrUnknownAPIKey)) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, identity.ErrMissingIdentity) {
				status = http.StatusUnauthorized
			}
			writeJSON(w, status, map[string]string{
				"error": err.Error(),
			})
			return
		}

		result, err := service.CallTool(r.Context(), caller, request.ToolName, request.Arguments)
		if err != nil {
			callErr, ok := err.(*gateway.ToolCallError)
			if !ok {
				writeJSON(w, http.StatusBadGateway, map[string]string{
					"error": err.Error(),
				})
				return
			}

			writeJSON(w, callErr.StatusCode, map[string]string{
				"error": callErr.Message,
			})
			return
		}

		writeJSON(w, http.StatusOK, toolCallResponse{Result: result})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeToolCallRequest(w http.ResponseWriter, r *http.Request, request *toolCallRequest) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolCallRequestBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		return decodeBodyError(err)
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain exactly one JSON object")
	}

	request.ToolName = strings.TrimSpace(request.ToolName)
	if request.ToolName == "" {
		return errors.New("tool_name is required")
	}
	if request.Arguments == nil {
		request.Arguments = map[string]any{}
	}

	return nil
}

func decodeBodyError(err error) error {
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxBytesErr):
		return errors.New("request body is too large")
	case errors.Is(err, io.EOF):
		return errors.New("request body must be valid JSON")
	default:
		return errors.New("request body must be valid JSON")
	}
}
