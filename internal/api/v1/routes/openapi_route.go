package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
)

func SetupYamlRoute(mux *http.ServeMux, handler *handlers.Handler) {
	docsMux := http.NewServeMux()
	docsMux.HandleFunc("GET /", http.HandlerFunc(handler.OpenAPIUIHandler()))               // UI
	docsMux.HandleFunc("GET /cargo-agent.yaml", http.HandlerFunc(handler.OpenAPIHandler())) // YAML
	mux.Handle("/cargo-agent/v1/docs/", http.StripPrefix("/cargo-agent/v1/docs", docsMux))
}
