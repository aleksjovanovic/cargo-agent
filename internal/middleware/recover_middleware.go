package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

// Recoverer is a top-level middleware that prevents the server from crashing
// when a handler panics. It logs the panic (with stack trace) and returns a
// generic 500 JSON response so we don't leak internals to clients.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Defer a recovery block around the whole request lifecycle.
		defer func() {
			if rec := recover(); rec != nil {
				// Try to enrich logs with request metadata (request id, method, path).
				rid := ctxmeta.RequestID(r.Context())

				// Log the panic with a stack trace for debugging/observability.
				logger.Error(
					"panic recovered",
					"rid", rid,
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)

				// Best-effort standardized error response.
				// If headers were already partially written by a handler
				// before the panic, this may not change the outcome,
				// but in most cases the panic happens before the first write.
				response.RespondWithError(
					w,
					http.StatusInternalServerError,
					"internal_error",
					"An unexpected error occurred",
					nil,
				)
			}
		}()

		// Continue down the chain; any panic below will be handled by the defer above.
		next.ServeHTTP(w, r)
	})
}
