package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// JSONError models a structured error with an HTTP status code.
// Callers can inspect Status to choose the response code.
type JSONError struct {
	Status int
	Msg    string
}

func (e *JSONError) Error() string { return e.Msg }

// DecodeJSONBody reads and decodes a JSON request body into dst with strict checks.
// - Validates Content-Type when provided (must be application/json, charset allowed)
// - Limits body size (default 1MB if maxBytes <= 0)
// - Disallows unknown fields to catch typos early
// - Ensures the body contains exactly one JSON value (no trailing junk)
// Returns *JSONError for user-facing problems (400/415/413), or nil on success.
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	// Validate Content-Type if present. We accept "application/json" with optional parameters.
	if ct := r.Header.Get("Content-Type"); ct != "" {
		base := strings.TrimSpace(strings.Split(ct, ";")[0])
		if !strings.EqualFold(base, "application/json") {
			return &JSONError{
				Status: http.StatusUnsupportedMediaType,
				Msg:    "Content-Type must be application/json",
			}
		}
	}

	// Apply a size cap to protect against overly large payloads.
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1 MiB
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	defer r.Body.Close()

	// Strict JSON decoder: unknown fields are rejected.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode into destination.
	if err := dec.Decode(dst); err != nil {
		var se *json.SyntaxError
		var ute *json.UnmarshalTypeError

		switch {
		case errors.As(err, &se):
			return &JSONError{Status: http.StatusBadRequest, Msg: fmt.Sprintf("Malformed JSON at position %d", se.Offset)}
		case errors.Is(err, io.ErrUnexpectedEOF):
			return &JSONError{Status: http.StatusBadRequest, Msg: "Malformed JSON"}
		case errors.As(err, &ute):
			// When Field is empty, the decoder couldn't attribute the error to a named field.
			field := ute.Field
			if field == "" {
				field = "(at position)"
			}
			return &JSONError{Status: http.StatusBadRequest, Msg: fmt.Sprintf("Invalid value for field %q", field)}
		case errors.Is(err, io.EOF):
			return &JSONError{Status: http.StatusBadRequest, Msg: "Request body must not be empty"}
		case strings.HasPrefix(err.Error(), "http: request body too large"):
			return &JSONError{Status: http.StatusRequestEntityTooLarge, Msg: "Request body too large"}
		default:
			return &JSONError{Status: http.StatusBadRequest, Msg: "Invalid request payload"}
		}
	}

	// Ensure there is no trailing data (must be exactly one JSON object).
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return &JSONError{Status: http.StatusBadRequest, Msg: "Request body must contain a single JSON object"}
	}

	return nil
}

// DecodeJSONBodyOr400 is a small helper that decodes into dst and, on failure,
// calls onError with an appropriate status and message. It returns true on success
// (i.e., dst has been populated) or false if an error occurred and onError was invoked.
//
// Typical usage in handlers:
//
//	var body SomeDTO
//	if !utils.DecodeJSONBodyOr400(w, r, &body, 0, func(st int, msg string) {
//	    response.RespondWithError(w, st, "invalid_payload", msg, nil)
//	}) {
//	    return
//	}
func DecodeJSONBodyOr400(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	maxBytes int64,
	onError func(status int, msg string),
) bool {
	if err := DecodeJSONBody(w, r, dst, maxBytes); err != nil {
		if je, ok := err.(*JSONError); ok {
			onError(je.Status, je.Msg)
		} else {
			onError(http.StatusBadRequest, "Invalid request payload")
		}
		return false
	}
	return true
}
