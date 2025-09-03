package middleware

import (
	"context"
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/google/uuid"
)

type ctxKey string

const requestIDKey ctxKey = "req_id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)

		logger.Debug("incoming request", "method", r.Method, "path", r.URL.Path, "req_id", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Optional helper
func FromContextRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}
