package routes

import (
	"net/http"
	"os"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
)

func SetupCargoOfferRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	newMux := http.NewServeMux()

	// auth middleware za POST
	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// POST /cargo-offers
	newMux.Handle("POST /cargo-offers", auth(http.HandlerFunc(handler.Create())))

	// GET /cargo-offers/{id}
	newMux.Handle("GET /cargo-offers/{id}", http.HandlerFunc(handler.GetByID()))

	mux.Handle("/cargo-agent/v1/", http.StripPrefix("/cargo-agent/v1", newMux))
}
