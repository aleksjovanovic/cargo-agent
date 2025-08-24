package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/handlers"
)

func SetupHealthCheckRoute(mux *http.ServeMux, handler *handlers.Handler) {
	mux.HandleFunc("GET /ping", handler.HealthCheckHandler())
}
