package server

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Service: "ashton-mcp-gateway",
			Status:  "ok",
		})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
