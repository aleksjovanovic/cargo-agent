package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

func SetupCountryRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	userMux := http.NewServeMux()

	// === COUNTRY ENDPOINTS ===
	// Define country routes with method-based routing
	userMux.Handle("GET /name/{name}", middlewares.AuthzMiddleware(http.HandlerFunc(handler.GetCountryByNameHandler())))
	userMux.Handle("GET /{id}", middlewares.AuthzMiddleware(http.HandlerFunc(handler.GetCountryByIDHandler())))
	userMux.Handle("GET /", middlewares.AuthzMiddleware(http.HandlerFunc(handler.ListCountriesHandler())))

	// === CITY ENDPOINTS (nested under country) ===
	userMux.Handle("GET /id/{id}/cities", middlewares.AuthzMiddleware(http.HandlerFunc(handler.ListCitiesByCountryIDHandler())))

	// GUARD:
	userMux.Handle("GET /id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "Country ID is required", nil)
	}))
	userMux.Handle("GET /id/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "Country ID is required", nil)
	}))
	userMux.Handle("GET /name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name is required", nil)
	}))
	userMux.Handle("GET /name/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name is required", nil)
	}))

	// === REGISTER base prefix ===
	mux.Handle("/cargo-agent/v1/countries/", http.StripPrefix("/cargo-agent/v1/countries", userMux))
}
