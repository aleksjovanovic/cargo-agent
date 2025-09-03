package handlers

import (
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
)

// Jedinstveni decoder helper za sve handlere.
// I dalje ostaje u handlers sloju jer ovde definišemo kako izgleda HTTP odgovor.
func (h *Handler) decodeOr400(w http.ResponseWriter, r *http.Request, dst any) bool {
	return utils.DecodeJSONBodyOr400(w, r, dst, 1<<20, func(status int, msg string) {
		response.RespondWithError(w, status, "invalid_payload", msg, nil)
	})
}
