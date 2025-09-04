// internal/ctxmeta/ctxmeta.go

// Package ctxmeta centralizes context keys and helpers for attaching and
// retrieving per-request metadata (request ID, normalized route pattern, ...).
// Using a private key type avoids collisions with keys from other packages.
package ctxmeta

import "context"

// ctxKey is an unexported key type to prevent collisions with context keys
// used by other packages. Only this package can construct ctxKey values.
type ctxKey string

// Known context keys used by this package.
// Keep keys small and stable; changing them breaks propagation.
const (
	requestIDKey    ctxKey = "req_id"
	routePatternKey ctxKey = "route_pattern"
)

// WithRequestID stores a request ID string in the context.
// Call this from early middleware (e.g., RequestID) so all downstream
// handlers and logs can correlate events with the same ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID reads the request ID from context.
// Returns an empty string when the value is missing or not a string.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// WithRoutePattern stores a normalized route pattern in the context,
// e.g. "GET /cargo-agent/v1/users/{id}". This should be set by routing
// or metrics middleware once the pattern is known, then used for logs
// and Prometheus labels to keep cardinality low.
func WithRoutePattern(ctx context.Context, pattern string) context.Context {
	return context.WithValue(ctx, routePatternKey, pattern)
}

// RoutePattern retrieves the normalized route pattern from context.
// Returns the pattern and true if present and non-empty; otherwise
// returns "", false. Use RoutePatternOr when you want a fallback value.
func RoutePattern(ctx context.Context) (string, bool) {
	if v, ok := ctx.Value(routePatternKey).(string); ok && v != "" {
		return v, true
	}
	return "", false
}

// RoutePatternOr is a convenience helper that returns the route pattern
// if set, otherwise returns the provided fallback (often r.Method+" "+r.URL.Path).
// This is handy in logging code to avoid branching.
func RoutePatternOr(ctx context.Context, fallback string) string {
	if p, ok := RoutePattern(ctx); ok {
		return p
	}
	return fallback
}
