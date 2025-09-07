// internal/api/v1/handlers/countries_handler.go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

// NOTE: All country/city endpoints are PUBLIC now (no auth checks here).

// asInt32 tries to coerce a dynamic "id" value (commonly unmarshaled as float64 or int types)
// into int32. Returns (0, false) when the value type is unsupported.
func asInt32(v any) (int32, bool) {
	switch x := v.(type) {
	case int:
		return int32(x), true
	case int32:
		return x, true
	case int64:
		return int32(x), true
	case float64: // JSON numbers often end up as float64
		return int32(x), true
	default:
		return 0, false
	}
}

// ListCountriesHandler returns all countries.
// Notes:
// - Route is mounted under "/cargo-agent/v1/countries/" with a sub mux.
// - We keep a simple path guard to avoid accidental fallthrough.
func (h *Handler) ListCountriesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "countries.list"
		rid := ctxmeta.RequestID(r.Context())

		// Defensive guard: only accept the "root" of the countries submux.
		if r.URL.Path != "/" {
			logger.Warn("Countries unknown path", "op", op, "path", r.URL.Path, "rid", rid)
			response.RespondWithError(w, http.StatusNotFound, "not_found", "Unknown path", nil)
			return
		}

		logger.Info("Listing countries", "op", op, "rid", rid)

		countries, appErr := h.Countries.List(r.Context())
		if appErr != nil {
			logger.Error("List countries failed", "op", op, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Countries listed", "op", op, "count", len(countries), "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    countries,
		})
	}
}

// GetCountryByIDHandler returns a single country by numeric ID extracted from the path.
// Flow: parse {id} -> service call -> 200 or mapped error.
func (h *Handler) GetCountryByIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "countries.get_by_id"
		rid := ctxmeta.RequestID(r.Context())

		id := r.PathValue("id")
		countryID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || countryID <= 0 {
			logger.Warn("Get country invalid id", "op", op, "id", id, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		logger.Info("Getting country", "op", op, "id", countryID, "rid", rid)

		country, appErr := h.Countries.GetByID(r.Context(), int32(countryID))
		if appErr != nil {
			logger.Warn("Get country failed", "op", op, "id", countryID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Country loaded", "op", op, "id", countryID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    country,
		})
	}
}

// GetCountryByNameHandler returns a single country matched by its name provided in the path.
func (h *Handler) GetCountryByNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "countries.get_by_name"
		rid := ctxmeta.RequestID(r.Context())

		countryName := strings.TrimSpace(r.PathValue("name"))
		if countryName == "" {
			logger.Warn("Get country by name empty", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name cannot be empty", nil)
			return
		}

		logger.Info("Getting country by name", "op", op, "name", countryName, "rid", rid)

		country, appErr := h.Countries.GetByName(r.Context(), countryName)
		if appErr != nil {
			logger.Warn("Get country by name failed", "op", op, "name", countryName, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Country by name loaded", "op", op, "name", countryName, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    country,
		})
	}
}

// ListCitiesByCountryIDHandler returns all cities for the given country id.
// Flow: parse {id} -> service call -> 200 or mapped error.
func (h *Handler) ListCitiesByCountryIDHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "countries.cities_by_id"
		rid := ctxmeta.RequestID(r.Context())

		id := r.PathValue("id")
		countryID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || countryID <= 0 {
			logger.Warn("List cities invalid id", "op", op, "id", id, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		logger.Info("Listing cities by country id", "op", op, "country_id", countryID, "rid", rid)

		cities, appErr := h.Countries.ListCities(r.Context(), int32(countryID))
		if appErr != nil {
			logger.Error("List cities by country id failed", "op", op, "country_id", countryID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Cities by country id listed", "op", op, "country_id", countryID, "count", len(cities), "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    cities,
		})
	}
}

// ListCitiesByCountryNameHandler resolves a country by name, then lists its cities.
func (h *Handler) ListCitiesByCountryNameHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "countries.cities_by_name"
		rid := ctxmeta.RequestID(r.Context())

		name := strings.TrimSpace(r.PathValue("name"))
		if name == "" {
			logger.Warn("List cities empty country name", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_name", "Country name cannot be empty", nil)
			return
		}

		logger.Info("Loading country by name (for cities)", "op", op, "name", name, "rid", rid)

		country, appErr := h.Countries.GetByName(r.Context(), name)
		if appErr != nil {
			logger.Warn("Load country by name failed", "op", op, "name", name, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		// Extract ID from the service's dynamic map response.
		idVal := country["id"]
		cid, ok := asInt32(idVal)
		if !ok || cid <= 0 {
			logger.Error("Unexpected country id type", "op", op, "name", name, "type", fmt.Sprintf("%T", idVal), "rid", rid)
			response.RespondWithError(w, http.StatusInternalServerError, "type_error", "Unexpected country id type", nil)
			return
		}

		logger.Info("Listing cities by country name", "op", op, "country_id", cid, "name", name, "rid", rid)

		cities, appErr := h.Countries.ListCities(r.Context(), cid)
		if appErr != nil {
			logger.Error("List cities by country name failed", "op", op, "country_id", cid, "name", name, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Cities by country name listed", "op", op, "country_id", cid, "name", name, "count", len(cities), "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    cities,
		})
	}
}
