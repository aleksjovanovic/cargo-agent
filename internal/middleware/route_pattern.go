package middleware

import (
	"context"
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
)

// RoutePattern is a no-op middleware kept for compatibility in the chain.
// It does not modify the request; use WithRoutePattern if you want to write
// a normalized route pattern into the context for metrics/logging.
func RoutePattern(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally pass-through. Keeping this in the chain allows
		// you to turn on pattern writing later without touching main.go.
		next.ServeHTTP(w, r)
	})
}

// WithRoutePattern wraps a handler and writes the provided normalized route
// pattern into the request context (and echoes it as a response header).
// Example usage in routes setup:
//
//	mux.Handle("GET /cargo-agent/v1/cargo-offers/{id}",
//	  WithRoutePattern("GET /cargo-agent/v1/cargo-offers/{id}", handler.GetByID()))
func WithRoutePattern(pattern string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ctxmeta.WithRoutePattern(r.Context(), pattern)
		// Optional: expose the normalized pattern for debugging/observability.
		w.Header().Set("X-Route-Pattern", pattern)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContextRoutePattern returns the pattern previously stored in context
// (if any). This is a thin alias around ctxmeta.RoutePattern and is kept for
// backward compatibility with older references in the codebase.
func FromContextRoutePattern(ctx context.Context) (string, bool) {
	return ctxmeta.RoutePattern(ctx)
}
