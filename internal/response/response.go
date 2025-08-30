package response

import (
	"encoding/json"
	"net/http"
	"time"
)

type Envelope map[string]any

type Option func(http.Header)

func WithHeader(key, value string) Option {
	return func(h http.Header) { h.Set(key, value) }
}

func WithHeaders(m map[string]string) Option {
	return func(h http.Header) {
		for k, v := range m {
			h.Set(k, v)
		}
	}
}

// -----------------------------
// AppError (koristi ga service sloj)
// -----------------------------
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	Status  int    `json:"-"` // HTTP status, ne ide u JSON telo
}

func (e *AppError) Error() string { return e.Message }

// Ako želiš direktno da ispišeš AppError iz handler-a:
func WriteAppError(w http.ResponseWriter, e *AppError, opts ...Option) {
	if e == nil {
		return
	}
	RespondWithError(w, e.Status, e.Code, e.Message, e.Details, opts...)
}

// -----------------------------
// JSON helperi za odgovore
// -----------------------------
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
	_ = enc.Encode(envelope)
}

func RespondWithSuccess(w http.ResponseWriter, status int, payload any, opts ...Option) {
	JSON(w, status, payload, opts...)
}

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

func RespondWithNotFound(w http.ResponseWriter, resource string, opts ...Option) {
	msg := "Resource not found"
	if resource != "" {
		msg = resource + " not found"
	}
	RespondWithError(w, http.StatusNotFound, "not_found", msg, nil, opts...)
}
