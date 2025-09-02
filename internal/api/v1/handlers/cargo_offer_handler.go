package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
)

func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var req request.CreateCargoOfferRequest
		if err := utils.DecodeJSONBody(w, r, &req, 1<<20); err != nil {
			if je, ok := err.(*utils.JSONError); ok {
				response.RespondWithError(w, je.Status, "invalid_payload", je.Msg, nil)
			} else {
				response.RespondWithError(w, http.StatusBadRequest, "invalid_payload", "Invalid request payload", nil)
			}
			return
		}

		out, appErr := h.CargoOffer.Create(r.Context(), int32(claims.UserID), req)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "cargo offer created",
			"data":    out,
		})
	}
}

func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id64, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil || id64 <= 0 {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		out, appErr := h.CargoOffer.Get(r.Context(), int32(id64))
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    out,
		})
	}
}

// GET /cargo-offers
func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qv := r.URL.Query()

		i32 := func(s string) *int32 {
			if s == "" {
				return nil
			}
			if n, err := strconv.ParseInt(s, 10, 32); err == nil {
				v := int32(n)
				return &v
			}
			return nil
		}
		str := func(s string) *string {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			v := strings.ToLower(strings.TrimSpace(s))
			return &v
		}

		req := request.ListCargoOffersQuery{
			OriginCountryID:      i32(qv.Get("origin_country_id")),
			OriginCityID:         i32(qv.Get("origin_city_id")),
			DestinationCountryID: i32(qv.Get("destination_country_id")),
			DestinationCityID:    i32(qv.Get("destination_city_id")),
			LoadType:             str(qv.Get("load_type")),
			TruckType:            str(qv.Get("truck_type")),
			Status:               str(qv.Get("status")),
			ReadyFrom:            str(qv.Get("ready_from")),
			ReadyTo:              str(qv.Get("ready_to")),
			DeliveryFrom:         str(qv.Get("delivery_from")),
			DeliveryTo:           str(qv.Get("delivery_to")),
			Limit:                i32(qv.Get("limit")),
			Page:                 i32(qv.Get("page")),
		}

		data, appErr := h.CargoOffer.List(r.Context(), req)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    data,
		})
	}
}
