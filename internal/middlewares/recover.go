package middlewares

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec)
				response.RespondWithError(
					w,
					http.StatusInternalServerError,
					"internal_error",
					"An unexpected error occurred",
					nil,
				)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
