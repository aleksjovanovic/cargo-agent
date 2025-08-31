package routes

import (
	"net/http"
	"os"

	"github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
)

func SetupUserRoutes(mux *http.ServeMux, handler *handlers.Handler) {
	userMux := http.NewServeMux()

	// “Instanciraj” auth middleware
	auth := middlewares.Authz([]byte(os.Getenv("JWT_SECRET_KEY")))

	// Public
	userMux.HandleFunc("POST /signup", handler.CreateUserHandler())
	userMux.HandleFunc("POST /login", handler.LoginUserHandler())
	userMux.HandleFunc("GET /verify-email", handler.VerifyEmailHandler()) // ← NOVO (bez auth)

	// Protected
	userMux.Handle("GET /me", auth(http.HandlerFunc(handler.UserProfileHandler())))
	userMux.Handle("PATCH /me", auth(http.HandlerFunc(handler.UpdateUserProfileHandler())))
	userMux.Handle("PUT /me/password", auth(http.HandlerFunc(handler.ChangePasswordHandler())))
	userMux.Handle("DELETE /{id}", auth(http.HandlerFunc(handler.DeleteUserHandler())))

	// Mount pod /cargo-agent/v1/users/
	mux.Handle("/cargo-agent/v1/users/", http.StripPrefix("/cargo-agent/v1/users", userMux))
}
