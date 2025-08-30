package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
)

func SetupHealthCheckRoute(mux *http.ServeMux, handler *handlers.Handler) {
	userMux := http.NewServeMux()

	// Define healt-check route with method-based routing
	userMux.HandleFunc("GET /health-check", http.HandlerFunc(handler.HealthCheckHandler()))
	mux.Handle("/cargo-agent/v1/", http.StripPrefix("/cargo-agent/v1", userMux))
}
