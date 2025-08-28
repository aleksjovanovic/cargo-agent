package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// JSONError predstavlja grešku dekodiranja koju možeš direktno mapirati u HTTP odgovor.
type JSONError struct {
	Status int
	Msg    string
}

func (e *JSONError) Error() string { return e.Msg }

// DecodeJSONBody striktno dekodira JSON u dst, sa zaštitama i korisnim porukama.
func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	// 1) Content-Type check (dozvoli application/json i varijante sa charset-om)
	if ct := r.Header.Get("Content-Type"); ct != "" {
		base := strings.TrimSpace(strings.Split(ct, ";")[0])
		if !strings.EqualFold(base, "application/json") {
			return &JSONError{
				Status: http.StatusUnsupportedMediaType,
				Msg:    "Content-Type must be application/json",
			}
		}
	}

	// 2) Limitiraj telo (DoS zaštita)
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // podrazumevano 1MB
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	defer r.Body.Close()

	// 3) Striktno dekodiranje
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var se *json.SyntaxError
		var ute *json.UnmarshalTypeError

		switch {
		case errors.As(err, &se):
			return &JSONError{Status: http.StatusBadRequest, Msg: fmt.Sprintf("Malformed JSON at position %d", se.Offset)}
		case errors.Is(err, io.ErrUnexpectedEOF):
			return &JSONError{Status: http.StatusBadRequest, Msg: "Malformed JSON"}
		case errors.As(err, &ute):
			field := ute.Field
			if field == "" {
				// kada polje nije direktno dostupno
				field = "(at position)"
			}
			return &JSONError{Status: http.StatusBadRequest, Msg: fmt.Sprintf("Invalid value for field %q", field)}
		case errors.Is(err, io.EOF):
			return &JSONError{Status: http.StatusBadRequest, Msg: "Request body must not be empty"}
		case strings.HasPrefix(err.Error(), "http: request body too large"):
			return &JSONError{Status: http.StatusRequestEntityTooLarge, Msg: "Request body too large"}
		default:
			// drugi slučajevi (npr. custom unmarshaler error)
			return &JSONError{Status: http.StatusBadRequest, Msg: "Invalid request payload"}
		}
	}

	// 4) Ne dozvoli dodatni trailing JSON (npr. "{}{}")
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return &JSONError{Status: http.StatusBadRequest, Msg: "Request body must contain a single JSON object"}
	}

	return nil
}
