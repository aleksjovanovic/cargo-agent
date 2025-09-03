package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/middleware"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

// GET / (ako je mount na /countries/, onda očekujemo "/" posle StripPrefix-a)
func (h *Handler) ListCountriesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// čuvamo postojeći guard jer handler očekuje "/" unutar submux-a
		if r.URL.Path != "/" {
			response.RespondWithError(w, http.StatusNotFound, "not_found", "Unknown path", nil)
			return
		}

		if _, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		countries, appErr := h.Countries.List(r.Context())
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    countries,
		})
	}
}

// GET /countries/{id}
func (h *Handler) GetCountryByIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		id := r.PathValue("id")
		countryID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || countryID <= 0 {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		country, appErr := h.Countries.GetByID(r.Context(), int32(countryID))
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    country,
		})
	}
}

// GET /countries/name/{name}
func (h *Handler) GetCountryByNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		countryName := strings.TrimSpace(r.PathValue("name"))
		if countryName == "" {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name cannot be empty", nil)
			return
		}

		country, appErr := h.Countries.GetByName(r.Context(), countryName)
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    country,
		})
	}
}

// GET /countries/id/{id}/cities
func (h *Handler) ListCitiesByCountryIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		id := r.PathValue("id")
		countryID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || countryID <= 0 {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		cities, appErr := h.Countries.ListCities(r.Context(), int32(countryID))
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    cities,
		})
	}
}

// GET /countries/name/{name}/cities
func (h *Handler) ListCitiesByCountryNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		name := strings.TrimSpace(r.PathValue("name"))
		if name == "" {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name cannot be empty", nil)
			return
		}

		country, appErr := h.Countries.GetByName(r.Context(), name)
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		// Izvuci ID iz map-e (kao float64 itd.)
		idVal, _ := country["id"]
		var cid int32
		switch v := idVal.(type) {
		case float64:
			cid = int32(v)
		case int:
			cid = int32(v)
		case int32:
			cid = v
		case int64:
			cid = int32(v)
		default:
			response.RespondWithError(w, http.StatusInternalServerError, "type_error", "Unexpected country id type", nil)
			return
		}

		cities, appErr := h.Countries.ListCities(r.Context(), cid)
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    cities,
		})
	}
}
