package routes

import (
	"net/http"
	"os"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

func SetupRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	SetupHealthCheckRoute(mux, handler)
	SetupUserRoutes(mux, handler)
	SetupCountryRoutes(mux, handler)
	SetupCargoOfferRoutes(mux, handler)
	SetupYamlRoute(mux, handler)
	SetupTruckAvailabilityRoutes(mux, handler)

	// optional root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})
}

// /cargo-agent/v1/docs
func SetupYamlRoute(mux *http.ServeMux, handler *handlers.Handler) {
	docsMux := http.NewServeMux()
	docsMux.HandleFunc("GET /", handler.OpenAPIUIHandler())               // UI
	docsMux.HandleFunc("GET /cargo-agent.yaml", handler.OpenAPIHandler()) // YAML
	mux.Handle("/cargo-agent/v1/docs/", http.StripPrefix("/cargo-agent/v1/docs", docsMux))
}

// /cargo-agent/v1/health-check (nema submux-a, direktna ruta)
func SetupHealthCheckRoute(mux *http.ServeMux, handler *handlers.Handler) {
	mux.HandleFunc("GET /cargo-agent/v1/health-check", handler.HealthCheckHandler())
}

// /cargo-agent/v1/countries/*
func SetupCountryRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	cMux := http.NewServeMux()
	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// Country
	cMux.Handle("GET /name/{name}", auth(http.HandlerFunc(handler.GetCountryByNameHandler())))
	cMux.Handle("GET /{id}", auth(http.HandlerFunc(handler.GetCountryByIDHandler())))
	cMux.Handle("GET /", auth(http.HandlerFunc(handler.ListCountriesHandler())))

	// Cities (nested)
	cMux.Handle("GET /id/{id}/cities", auth(http.HandlerFunc(handler.ListCitiesByCountryIDHandler())))
	cMux.Handle("GET /name/{name}/cities", auth(http.HandlerFunc(handler.ListCitiesByCountryNameHandler())))

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

	// mount
	mux.Handle("/cargo-agent/v1/countries/", http.StripPrefix("/cargo-agent/v1/countries", cMux))
}

// /cargo-agent/v1/cargo-offers/*
func SetupCargoOfferRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// LIST (public)
	mux.Handle("GET /cargo-agent/v1/cargo-offers", http.HandlerFunc(handler.List()))
	mux.Handle("GET /cargo-agent/v1/cargo-offers/", http.HandlerFunc(handler.List()))

	// CREATE (protected)
	mux.Handle("POST /cargo-agent/v1/cargo-offers", auth(http.HandlerFunc(handler.Create())))
	mux.Handle("POST /cargo-agent/v1/cargo-offers/", auth(http.HandlerFunc(handler.Create())))

	// GET BY ID (public)
	mux.Handle("GET /cargo-agent/v1/cargo-offers/{id}", http.HandlerFunc(handler.GetByID()))
}

// /cargo-agent/v1/users/*
func SetupUserRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	userMux := http.NewServeMux()
	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// Public
	userMux.HandleFunc("POST /signup", handler.CreateUserHandler())
	userMux.HandleFunc("POST /login", handler.LoginUserHandler())
	userMux.HandleFunc("GET /verify-email", handler.VerifyEmailHandler()) // bez auth

	// Protected (me)
	userMux.Handle("GET /me", auth(http.HandlerFunc(handler.UserProfileHandler())))
	userMux.Handle("PATCH /me", auth(http.HandlerFunc(handler.UpdateUserProfileHandler())))
	userMux.Handle("PUT /me/password", auth(http.HandlerFunc(handler.ChangePasswordHandler())))

	// Admin-ish (primer)
	userMux.Handle("DELETE /{id}", auth(http.HandlerFunc(handler.DeleteUserHandler())))

	// mount
	mux.Handle("/cargo-agent/v1/users/", http.StripPrefix("/cargo-agent/v1/users", userMux))
}

func SetupTruckAvailabilityRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// LIST (public)
	mux.Handle("GET /cargo-agent/v1/truck-availability", http.HandlerFunc(handler.TruckAvailabilityList()))
	mux.Handle("GET /cargo-agent/v1/truck-availability/", http.HandlerFunc(handler.TruckAvailabilityList()))

	// CREATE (protected)
	mux.Handle("POST /cargo-agent/v1/truck-availability", auth(http.HandlerFunc(handler.TruckAvailabilityCreate())))
	mux.Handle("POST /cargo-agent/v1/truck-availability/", auth(http.HandlerFunc(handler.TruckAvailabilityCreate())))

	// GET BY ID (public)
	mux.Handle("GET /cargo-agent/v1/truck-availability/{id}", http.HandlerFunc(handler.TruckAvailabilityGetByID()))

	// UPDATE STATUS (protected)
	mux.Handle("PATCH /cargo-agent/v1/truck-availability/{id}/status", auth(http.HandlerFunc(handler.UpdateStatus())))
}
