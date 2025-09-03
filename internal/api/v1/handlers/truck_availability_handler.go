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

func (h *Handler) decodeOr400(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := utils.DecodeJSONBody(w, r, dst, 1<<20); err != nil {
		if je, ok := err.(*utils.JSONError); ok {
			response.RespondWithError(w, je.Status, "invalid_payload", je.Msg, nil)
		} else {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_payload", "Invalid request payload", nil)
		}
		return false
	}
	return true
}

func (h *Handler) TruckAvailabilityCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}
		var req request.CreateTruckAvailabilityRequest
		if !h.decodeOr400(w, r, &req) {
			return
		}
		out, appErr := h.TruckAvailability.Create(r.Context(), int32(claims.UserID), req)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}
		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "Truck availability created",
			"data":    out,
		})
	}
}

func (h *Handler) TruckAvailabilityGetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		n, err := strconv.ParseInt(id, 10, 32)
		if err != nil || n <= 0 {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}
		out, appErr := h.TruckAvailability.Get(r.Context(), int32(n))
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

func (h *Handler) TruckAvailabilityList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qp := r.URL.Query()

		parseI32 := func(k string) *int32 {
			if s := qp.Get(k); s != "" {
				if n, err := strconv.ParseInt(s, 10, 32); err == nil {
					v := int32(n)
					return &v
				}
			}
			return nil
		}
		parseBool := func(k string) *bool {
			if s := strings.TrimSpace(qp.Get(k)); s != "" {
				switch strings.ToLower(s) {
				case "1", "true":
					t := true
					return &t
				case "0", "false":
					f := false
					return &f
				}
			}
			return nil
		}

		getStr := func(k string) *string {
			if s := qp.Get(k); s != "" {
				return &s
			}
			return nil
		}

		q := request.ListTruckAvailabilityQuery{
			StartCountryID: parseI32("start_country_id"),
			StartCityID:    parseI32("start_city_id"),
			EndCountryID:   parseI32("end_country_id"),
			EndCityID:      parseI32("end_city_id"),

			TruckType:   getStr("truck_type"),
			Status:      getStr("status"),
			FullLoad:    parseBool("full_load"),
			PartialLoad: parseBool("partial_load"),

			AvailableFrom: getStr("available_from"),
			AvailableTo:   getStr("available_to"),

			Page:  parseI32("page"),
			Limit: parseI32("limit"),
		}

		out, appErr := h.TruckAvailability.List(r.Context(), q)
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

func (h *Handler) TruckAvailabilityUpdateStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		idStr := r.PathValue("id")
		id64, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil || id64 <= 0 {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}
		id := int32(id64)

		var req request.UpdateTruckAvailabilityStatusRequest
		if !h.decodeOr400(w, r, &req) {
			return
		}

		out, appErr := h.TruckAvailability.UpdateStatus(r.Context(), id, int32(claims.UserID), req.Status)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "status updated",
			"data":    out,
		})
	}
}
