package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/google/uuid"
)

// Canonical header names we read/write for correlation.
// We prefer W3C Trace Context (Traceparent) if provided,
// but we still support common de-facto headers for compatibility.
const (
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
	HeaderTraceparent   = "Traceparent"

	// Back-compat alias (kept exported in case other packages import it)
	RequestIDHeader = HeaderRequestID
)

// RequestID is a top-level middleware that ensures every request carries a
// stable correlation ID:
// 1) Try to extract trace-id from W3C Trace Context (Traceparent header).
// 2) Else try X-Request-ID or X-Correlation-ID.
// 3) Else generate a UUIDv4.
// The chosen ID is stored into the request context (via ctxmeta.WithRequestID)
// and echoed back in both X-Request-ID and X-Correlation-ID response headers.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Extract or generate an ID
		id := pickIncomingRequestID(r)
		if id == "" {
			id = uuid.NewString()
		}

		// Echo correlation headers so downstream services/clients can use them.
		// We set both, regardless of which one was supplied.
		w.Header().Set(HeaderRequestID, id)
		w.Header().Set(HeaderCorrelationID, id)

		// Store the ID in context for the rest of the pipeline (logging, services, etc.)
		ctx := ctxmeta.WithRequestID(r.Context(), id)

		// Lightweight debug log helps when tracing issues locally.
		logger.Debug("incoming request", "method", r.Method, "path", r.URL.Path, "req_id", id)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// pickIncomingRequestID returns a reasonable request identifier from incoming headers.
// Priority:
//  1. Traceparent (extract the 32-hex trace-id)
//  2. X-Request-ID
//  3. X-Correlation-ID
func pickIncomingRequestID(r *http.Request) string {
	// 1) W3C traceparent:  "00-<trace-id>-<parent-id>-<flags>"
	if tp := strings.TrimSpace(r.Header.Get(HeaderTraceparent)); tp != "" {
		if tid := parseTraceparentTraceID(tp); tid != "" {
			return tid
		}
	}
	// 2) X-Request-ID
	if v := strings.TrimSpace(r.Header.Get(HeaderRequestID)); v != "" {
		return v
	}
	// 3) X-Correlation-ID
	if v := strings.TrimSpace(r.Header.Get(HeaderCorrelationID)); v != "" {
		return v
	}
	return ""
}

// parseTraceparentTraceID tries to extract <trace-id> from a valid Traceparent header.
// Expected format: "00-<32hex>-<16hex>-<2hex>". We only do minimal sanity checks here.
func parseTraceparentTraceID(tp string) string {
	parts := strings.Split(tp, "-")
	if len(parts) < 4 {
		return ""
	}
	traceID := strings.TrimSpace(parts[1])
	if len(traceID) == 32 { // minimal sanity check
		return traceID
	}
	return ""
}

// RequestIDFromContext is a helper that fetches the request ID from the context.
// Returns (id, true) if present, ("", false) otherwise.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id := ctxmeta.RequestID(ctx)
	if strings.TrimSpace(id) == "" {
		return "", false
	}
	return id, true
}

// ReqID returns the best-effort request ID for the given request.
// It prefers the value stored in context (set by this middleware), and
// falls back to common headers if the middleware was bypassed.
func ReqID(r *http.Request) string {
	if id, ok := RequestIDFromContext(r.Context()); ok && id != "" {
		return id
	}
	if v := strings.TrimSpace(r.Header.Get(HeaderRequestID)); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get(HeaderCorrelationID)); v != "" {
		return v
	}
	return ""
}
