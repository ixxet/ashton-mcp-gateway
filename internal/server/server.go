package server

import (
	"encoding/json"
	"net/http"

	"github.com/ixxet/ashton-mcp-gateway/internal/manifest"
)

type healthResponse struct {
	Service       string `json:"service"`
	Status        string `json:"status"`
	ManifestsLoad int    `json:"manifests_loaded"`
}

type toolListResponse struct {
	Tools []toolSummary `json:"tools"`
}

type toolSummary struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ReadOnly      bool     `json:"read_only"`
	RequiredInput []string `json:"required_input"`
	SourceService string   `json:"source_service"`
}

func NewHandler(registry manifest.Registry) http.Handler {
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

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
