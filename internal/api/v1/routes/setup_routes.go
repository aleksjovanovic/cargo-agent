// internal/api/v1/routes/routes.go
package routes

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middleware"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// authFn describes a decorator that wraps an http.Handler with auth middleware.
// Using a type alias makes route registration signatures clearer.
type authFn func(http.Handler) http.Handler

// buildAuth chooses the appropriate auth middleware flavor.
// If Redis is available we enable JWT blacklist checks; otherwise plain auth.
func buildAuth(rdb *redis.Client) authFn {
	// Note: main.go keeps JWT_SECRET_KEY in sync with cfg.JWTSecret for legacy usage.
	secret := []byte(os.Getenv("JWT_SECRET_KEY"))
	if rdb != nil {
		return middleware.AuthzWithBlacklist(secret, rdb)
	}
	return middleware.Authz(secret)
}

// SetupRoutes is the single entry-point for wiring HTTP routes to the provided mux.
// We keep each resource family in a dedicated helper to avoid a monolithic function.
func SetupRoutes(mux *http.ServeMux, handler *handlers.Handler, db *sql.DB, rdb *redis.Client) {
	auth := buildAuth(rdb)

	// Health & docs: mounted first since they are shared infra endpoints.
	SetupHealthCheckRoute(mux, handler, db, rdb)
	SetupYamlRoute(mux, handler)

	// Business resources grouped by domain.
	SetupUserRoutes(mux, handler, auth)
	SetupCountryRoutes(mux, handler, auth)
	SetupCargoOfferRoutes(mux, handler, auth)
	SetupTruckAvailabilityRoutes(mux, handler, auth)

	// Optional root: keep a conservative default that rejects non-GET early.
	// You could render a tiny HTML/JSON banner here if desired.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})
}

// SetupYamlRoute exposes the OpenAPI UI and raw YAML under the versioned prefix.
// We use an inner mux + StripPrefix to keep handler paths clean.
func SetupYamlRoute(mux *http.ServeMux, handler *handlers.Handler) {
	docsMux := http.NewServeMux()
	docsMux.HandleFunc("GET /", handler.OpenAPIUIHandler())               // UI index (e.g., ReDoc/Swagger UI)
	docsMux.HandleFunc("GET /cargo-agent.yaml", handler.OpenAPIHandler()) // Raw YAML
	mux.Handle("/cargo-agent/v1/docs/", http.StripPrefix("/cargo-agent/v1/docs", docsMux))
}

// SetupHealthCheckRoute mounts liveness/readiness/metrics endpoints.
// Note: /readyz and /metrics are on the *root* server (no versioned prefix)
// so that infra (k8s, Prometheus) can scrape/probe without knowing API versions.
func SetupHealthCheckRoute(mux *http.ServeMux, handler *handlers.Handler, db *sql.DB, rdb *redis.Client) {
	// Lightweight liveness (always 200 if process is healthy).
	mux.HandleFunc("GET /cargo-agent/v1/health-check", handler.HealthCheckHandler())

	// Readiness depends on DB and Redis; returns 200 when both are reachable, otherwise 503.
	// This is a package-level function, not a method on handlers.Handler, by design.
	mux.Handle("GET /readyz", handlers.ReadyzHandler(db, rdb))

	// Prometheus metrics in OpenMetrics text format. Must be mounted with Handle (not HandleFunc).
	mux.Handle("GET /metrics", promhttp.Handler())
}

// SetupCountryRoutes registers country and nested city routes under /cargo-agent/v1/countries/*.
func SetupCountryRoutes(mux *http.ServeMux, handler *handlers.Handler, _ authFn) {
	cMux := http.NewServeMux()

	// Country lookups (public)
	cMux.Handle("GET /name/{name}", http.HandlerFunc(handler.GetCountryByNameHandler()))
	cMux.Handle("GET /name/{name}/", http.HandlerFunc(handler.GetCountryByNameHandler())) // trailing slash variant
	cMux.Handle("GET /{id}", http.HandlerFunc(handler.GetCountryByIDHandler()))
	cMux.Handle("GET /{id}/", http.HandlerFunc(handler.GetCountryByIDHandler())) // trailing slash variant
	cMux.Handle("GET /", http.HandlerFunc(handler.ListCountriesHandler()))

	// Cities under country (public)
	cMux.Handle("GET /id/{id}/cities", http.HandlerFunc(handler.ListCitiesByCountryIDHandler()))
	cMux.Handle("GET /id/{id}/cities/", http.HandlerFunc(handler.ListCitiesByCountryIDHandler())) // trailing slash variant
	cMux.Handle("GET /name/{name}/cities", http.HandlerFunc(handler.ListCitiesByCountryNameHandler()))
	cMux.Handle("GET /name/{name}/cities/", http.HandlerFunc(handler.ListCitiesByCountryNameHandler())) // trailing slash variant

	// Guard routes to provide helpful 400s when path params are missing.
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

	// Mount sub-mux under the versioned prefix.
	mux.Handle("/cargo-agent/v1/countries/", http.StripPrefix("/cargo-agent/v1/countries", cMux))
}

// SetupCargoOfferRoutes registers cargo-offers CRUD-ish endpoints.
// Public read endpoints are left unauthenticated; mutations require auth.
func SetupCargoOfferRoutes(mux *http.ServeMux, handler *handlers.Handler, auth authFn) {
	// LIST (public)
	mux.Handle("GET /cargo-agent/v1/cargo-offers", http.HandlerFunc(handler.List()))
	mux.Handle("GET /cargo-agent/v1/cargo-offers/", http.HandlerFunc(handler.List()))

	// CREATE (protected)
	mux.Handle("POST /cargo-agent/v1/cargo-offers", auth(http.HandlerFunc(handler.Create())))
	mux.Handle("POST /cargo-agent/v1/cargo-offers/", auth(http.HandlerFunc(handler.Create())))

	// GET BY ID (public)
	mux.Handle("GET /cargo-agent/v1/cargo-offers/{id}", http.HandlerFunc(handler.GetByID()))
	mux.Handle("GET /cargo-agent/v1/cargo-offers/{id}/", http.HandlerFunc(handler.GetByID())) // trailing slash variant

	// UPDATE STATUS (protected)
	mux.Handle("PATCH /cargo-agent/v1/cargo-offers/{id}/status", auth(http.HandlerFunc(handler.CargoOfferUpdateStatus())))
	mux.Handle("PATCH /cargo-agent/v1/cargo-offers/{id}/status/", auth(http.HandlerFunc(handler.CargoOfferUpdateStatus()))) // trailing slash variant
}

// SetupUserRoutes registers auth and user-profile related endpoints under /cargo-agent/v1/users/*.
func SetupUserRoutes(mux *http.ServeMux, handler *handlers.Handler, auth authFn) {
	userMux := http.NewServeMux()

	// Public auth endpoints
	userMux.HandleFunc("POST /signup", handler.CreateUserHandler())
	userMux.HandleFunc("POST /login", handler.LoginUserHandler())
	userMux.HandleFunc("GET /verify-email", handler.VerifyEmailHandler())

	// Protected auth endpoint (logout)
	userMux.Handle("POST /logout", auth(http.HandlerFunc(handler.LogoutUserHandler())))
	userMux.Handle("POST /logout/", auth(http.HandlerFunc(handler.LogoutUserHandler()))) // trailing slash variant

	// Authenticated "me" endpoints
	userMux.Handle("GET /me", auth(http.HandlerFunc(handler.UserProfileHandler())))
	userMux.Handle("PATCH /me", auth(http.HandlerFunc(handler.UpdateUserProfileHandler())))
	userMux.Handle("PUT /me/password", auth(http.HandlerFunc(handler.ChangePasswordHandler())))

	// Password-reset flow (public entrypoints).
	// The handlers themselves validate inputs and throttle via global rate limit.
	userMux.Handle("POST /password-reset/request", handler.PasswordResetRequestHandler())
	userMux.Handle("POST /password-reset/confirm", handler.PasswordResetConfirmHandler())

	// Admin-ish example: delete by path ID (still wrapped with auth).
	userMux.Handle("DELETE /{id}", auth(http.HandlerFunc(handler.DeleteUserHandler())))

	// Mount under versioned prefix.
	mux.Handle("/cargo-agent/v1/users/", http.StripPrefix("/cargo-agent/v1/users", userMux))
}

// SetupTruckAvailabilityRoutes registers endpoints for truck availability postings.
// Read is public; writes require authentication.
func SetupTruckAvailabilityRoutes(mux *http.ServeMux, handler *handlers.Handler, auth authFn) {
	// LIST (public)
	mux.Handle("GET /cargo-agent/v1/truck-availability", http.HandlerFunc(handler.TruckAvailabilityList()))
	mux.Handle("GET /cargo-agent/v1/truck-availability/", http.HandlerFunc(handler.TruckAvailabilityList()))

	// CREATE (protected)
	mux.Handle("POST /cargo-agent/v1/truck-availability", auth(http.HandlerFunc(handler.TruckAvailabilityCreate())))
	mux.Handle("POST /cargo-agent/v1/truck-availability/", auth(http.HandlerFunc(handler.TruckAvailabilityCreate())))

	// GET BY ID (public)
	mux.Handle("GET /cargo-agent/v1/truck-availability/{id}", http.HandlerFunc(handler.TruckAvailabilityGetByID()))
	mux.Handle("GET /cargo-agent/v1/truck-availability/{id}/", http.HandlerFunc(handler.TruckAvailabilityGetByID())) // trailing slash variant

	// UPDATE STATUS (protected)
	mux.Handle("PATCH /cargo-agent/v1/truck-availability/{id}/status", auth(http.HandlerFunc(handler.TruckAvailabilityUpdateStatus())))
	mux.Handle("PATCH /cargo-agent/v1/truck-availability/{id}/status/", auth(http.HandlerFunc(handler.TruckAvailabilityUpdateStatus()))) // trailing slash variant
}
