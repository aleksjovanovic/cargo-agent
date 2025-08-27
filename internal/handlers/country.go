package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
)

// Get all countries
func (h *Handler) ListCountries() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			response.RespondWithError(w, http.StatusNotFound, "not_found", "Unknown path", nil)
			return
		}

		_, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"unauthorized",
				"Please log in to continue",
				nil,
			)
			return
		}

		// Check the Redis first
		ctx := r.Context()
		cacheKey := "countries:list:v1"

		if cached, err := h.Redis.Get(ctx, cacheKey).Result(); err == nil {
			var countries []store.Country
			if err := json.Unmarshal([]byte(cached), &countries); err == nil {
				response.RespondWithSuccess(
					w,
					http.StatusOK,
					response.Envelope{
						"message": "success (from cache/redis)",
						"data":    countries,
					},
				)
				return
			}
		}

		// Fallback to database
		countries, err := h.Queries.ListCountries(r.Context())
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusNotFound,
				"not_found",
				"Countries not found",
				nil,
			)
			return
		}

		// Set to Redis
		if b, err := json.Marshal(countries); err == nil {
			_ = h.Redis.Set(ctx, cacheKey, b, 24*time.Hour).Err() // for this case TTL can be 0
		}

		response.RespondWithSuccess(
			w,
			http.StatusOK,
			response.Envelope{
				"message": "success",
				"data":    countries,
			},
		)
	}
}

// Get country by id
func (h *Handler) GetCountryByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"unauthorized",
				"Please log in to continue",
				nil,
			)
			return
		}

		id := r.PathValue("id")
		countryId, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid_id",
				"ID must be a valid integer",
				nil,
			)
			return
		}

		// Check the Redis first
		ctx := r.Context()
		cacheKey := "countries:id:{id}:v1"

		if cached, err := h.Redis.Get(ctx, cacheKey).Result(); err == nil {
			var country store.Country
			if err := json.Unmarshal([]byte(cached), &country); err == nil {
				response.RespondWithSuccess(
					w,
					http.StatusOK,
					response.Envelope{
						"message": "success (from cache/redis)",
						"data":    country,
					},
				)
				return
			}
		}

		// Fallback to database
		country, err := h.Queries.GetCountryByID(r.Context(), int32(countryId))
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusNotFound,
				"not_found",
				"Country not found",
				nil,
			)
			return
		}

		// Set to Redis
		if b, err := json.Marshal(country); err == nil {
			_ = h.Redis.Set(ctx, cacheKey, b, 24*time.Hour).Err() // for this case TTL can be 0
		}

		response.RespondWithSuccess(
			w,
			http.StatusOK,
			response.Envelope{
				"message": "success",
				"data":    country,
			},
		)
	}
}

// Get country by name
func (h *Handler) GetCountryByName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"unauthorized",
				"Please log in to continue",
				nil,
			)
			return
		}

		countryName := strings.TrimSpace(r.PathValue("name"))
		if countryName == "" {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid_name",
				"Country name cannot be empty",
				nil,
			)
			return
		}

		country, err := h.Queries.GetCountryByName(r.Context(), countryName)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusNotFound,
				"not_found",
				"Country not found",
				nil,
			)
			return
		}

		response.RespondWithSuccess(
			w,
			http.StatusOK,
			response.Envelope{
				"message": "success",
				"data":    country,
			},
		)
	}
}

// Get cities by country id
func (h *Handler) ListCitiesByCountryID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"unauthorized",
				"Please log in to continue",
				nil,
			)
			return
		}

		id := r.PathValue("id")
		countryId, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid_id",
				"ID must be a valid integer",
				nil,
			)
			return
		}

		ctx := r.Context()
		cacheKey := fmt.Sprintf("countries:%d:cities:v1", int32(countryId))

		// Check the Redis first
		if cached, err := h.Redis.Get(ctx, cacheKey).Result(); err == nil {
			var cities []store.City
			if err := json.Unmarshal([]byte(cached), &cities); err == nil {
				response.RespondWithSuccess(
					w,
					http.StatusOK,
					response.Envelope{
						"message": "success (from cache/redis)",
						"data":    cities,
					},
				)
				return
			}
		}

		// Fallback to database
		cities, err := h.Queries.ListCitiesByCountryID(r.Context(), int32(countryId))
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusNotFound,
				"not_found",
				"Cities not found",
				nil,
			)
			return
		}

		// Set to Redis
		if b, err := json.Marshal(cities); err == nil {
			_ = h.Redis.Set(ctx, cacheKey, b, 24*time.Hour).Err() // for this case TTL can be 0
		}

		response.RespondWithSuccess(
			w,
			http.StatusOK,
			response.Envelope{
				"message": "success",
				"data":    cities,
			},
		)
	}
}
