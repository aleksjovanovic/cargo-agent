package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
)

func SetupUserRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	userMux := http.NewServeMux()

	// Define user routes with method-based routing
	userMux.HandleFunc("POST /signup", handler.CreateUserHandler())
	userMux.HandleFunc("POST /login", handler.LoginUserHandler())
	userMux.Handle("GET /profile", middlewares.AuthzMiddleware(http.HandlerFunc(handler.UserProfile())))
	userMux.Handle("PUT /me/password", middlewares.AuthzMiddleware(http.HandlerFunc(handler.ChangePassword())))
	mux.Handle("/cargo-agent/v1/users/", http.StripPrefix("/cargo-agent/v1/users", userMux))
}
