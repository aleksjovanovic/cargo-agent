package routes

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
)

func SetupUserRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	userMux := http.NewServeMux()

	// Define user routes with method-based routing
	userMux.HandleFunc("POST /signup", handler.CreateUserHandler())
	userMux.HandleFunc("POST /login", handler.LoginUserHandler())
	userMux.Handle("GET /profile", middlewares.AuthzMiddleware(http.HandlerFunc(handler.UserProfileHandler())))
	userMux.Handle("PATCH /profile", middlewares.AuthzMiddleware(http.HandlerFunc(handler.UpdateUserProfileHandler())))
	userMux.Handle("PUT /me/password", middlewares.AuthzMiddleware(http.HandlerFunc(handler.ChangePasswordHandler())))
	userMux.Handle("DELETE /{id}", middlewares.AuthzMiddleware(http.HandlerFunc(handler.DeleteUserHandler())))
	mux.Handle("/cargo-agent/v1/users/", http.StripPrefix("/cargo-agent/v1/users", userMux))
}
