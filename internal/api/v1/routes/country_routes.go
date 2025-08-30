package routes

import (
	"net/http"
	"os"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

func SetupCountryRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	cMux := http.NewServeMux()

	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// Country
	cMux.Handle("GET /name/{name}", auth(http.HandlerFunc(handler.GetCountryByNameHandler())))
	cMux.Handle("GET /{id}", auth(http.HandlerFunc(handler.GetCountryByIDHandler())))
	cMux.Handle("GET /", auth(http.HandlerFunc(handler.ListCountriesHandler())))

	// Cities (nested)
	cMux.Handle("GET /id/{id}/cities", auth(http.HandlerFunc(handler.ListCitiesByCountryIDHandler())))

	// Guards
	cMux.Handle("GET /id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "Country ID is required", nil)
	}))
	cMux.Handle("GET /id/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "Country ID is required", nil)
	}))
	cMux.Handle("GET /name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name is required", nil)
	}))
	cMux.Handle("GET /name/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name is required", nil)
	}))

	// Mount pod /cargo-agent/v1/countries/
	mux.Handle("/cargo-agent/v1/countries/", http.StripPrefix("/cargo-agent/v1/countries", cMux))
}
