package handlers

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{
			"message": "PONG",
		}
		json.NewEncoder(w).Encode(response)
	}
}
