package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// Envelope is a lightweight map wrapper used to build response payloads.
// It keeps responses flexible without forcing a rigid schema per endpoint.
type Envelope map[string]any

// Option is a small functional option used to mutate HTTP headers
// when writing the response (e.g., setting caching or custom headers).
type Option func(http.Header)

// WithHeader sets a single HTTP header on the response.
func WithHeader(key, value string) Option {
	return func(h http.Header) { h.Set(key, value) }
}

// WithHeaders sets multiple HTTP headers on the response.
func WithHeaders(m map[string]string) Option {
	return func(h http.Header) {
		for k, v := range m {
			h.Set(k, v)
		}
	}
}

// AppError is an application-level error that carries a machine-friendly code,
// a human message, optional details (for structured context), and the HTTP status
// that should be used. Status is not serialized into the response body directly.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	Status  int    `json:"-"` // HTTP status (excluded from JSON body)
}

// Error implements the error interface for AppError.
func (e *AppError) Error() string { return e.Message }

// WriteAppError renders an AppError to the client using RespondWithError.
// If e is nil it does nothing.
func WriteAppError(w http.ResponseWriter, e *AppError, opts ...Option) {
	if e == nil {
		return
	}
	RespondWithError(w, e.Status, e.Code, e.Message, e.Details, opts...)
}

// JSON writes a JSON response with a consistent envelope:
//
//	{
//	  "timestamp": "<RFC3339>",
//	  "status":    <int>,
//	  ...payload (merged or under "data")
//	}
//
// If `data` is an Envelope, it is merged into the top-level (useful for {"message": "...", "data": ...}).
// Otherwise, non-Envelope data is put under "data".
func JSON(w http.ResponseWriter, status int, data any, opts ...Option) {
	for _, opt := range opts {
		opt(w.Header())
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	envelope := Envelope{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"status":    status,
	}

	if data != nil {
		switch v := data.(type) {
		case Envelope:
			for k, val := range v {
				envelope[k] = val
			}
		default:
			envelope["data"] = v
		}
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(envelope) // best-effort; if this fails the connection is likely gone
}

// RespondWithSuccess is a convenience wrapper for JSON that semantically
// indicates a successful response.
func RespondWithSuccess(w http.ResponseWriter, status int, payload any, opts ...Option) {
	JSON(w, status, payload, opts...)
}

// RespondWithStatus is a back-compat alias for RespondWithSuccess.
// Useful if some handlers still call the old name.
func RespondWithStatus(w http.ResponseWriter, status int, payload any, opts ...Option) {
	JSON(w, status, payload, opts...)
}

// RespondWithError writes a normalized error response:
//
//	{
//	  "timestamp": "...",
//	  "status": <int>,
//	  "error": "<code>",
//	  "message": "<human readable>",
//	  "details": <optional>
//	}
func RespondWithError(w http.ResponseWriter, status int, code, message string, details any, opts ...Option) {
	errBody := Envelope{
		"error":   code,
		"message": message,
	}
	if details != nil {
		errBody["details"] = details
	}
	JSON(w, status, errBody, opts...)
}

// RespondWithNotFound is a small helper for 404 responses with an optional resource name.
func RespondWithNotFound(w http.ResponseWriter, resource string, opts ...Option) {
	msg := "Resource not found"
	if resource != "" {
		msg = resource + " not found"
	}
	RespondWithError(w, http.StatusNotFound, "not_found", msg, nil, opts...)
}
