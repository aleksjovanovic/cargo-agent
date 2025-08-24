package errorhandler

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Message: message})
}

func RespondWithNotFound(w http.ResponseWriter) {
	RespondWithError(w, http.StatusNotFound, "Resource not found")
}
