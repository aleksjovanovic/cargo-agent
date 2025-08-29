package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
)

func SetupRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	SetupHealthCheckRoute(mux, handler)
	SetupUserRoutes(mux, handler)
	SetupCountryRoutes(mux, handler)

	SetupYamlRoute(mux, handler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

	})
}
