// internal/api/v1/handlers/truck_availability_handler.go
package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/middleware"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

// TruckAvailabilityCreate creates a new truck-availability entry for the authenticated user.
// Validates auth (via middleware), decodes payload, and delegates to the service layer.
func (h *Handler) TruckAvailabilityCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "truck_availability.create"
		rid := ctxmeta.RequestID(r.Context())

		// Claims are injected by the Authz middleware; absence means unauthenticated request.
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized create truck availability", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var reqBody request.CreateTruckAvailabilityRequest
		// decodeOr400 is a shared helper: writes 400 if JSON is invalid and returns false.
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid create truck availability payload", "op", op, "user_id", claims.UserID, "rid", rid)
			return
		}

		logger.Info("Creating truck availability", "op", op, "user_id", claims.UserID, "rid", rid)

		out, appErr := h.TruckAvailability.Create(r.Context(), int32(claims.UserID), reqBody)
		if appErr != nil {
			logger.Error("Create truck availability failed", "op", op, "user_id", claims.UserID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Truck availability created", "op", op, "user_id", claims.UserID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "Truck availability created",
			"data":    out,
		})
	}
}

// TruckAvailabilityGetByID returns a single truck-availability record by id.
// This endpoint is public (no auth required).
func (h *Handler) TruckAvailabilityGetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "truck_availability.get"
		rid := ctxmeta.RequestID(r.Context())

		// Extract {id} from the path and validate it's a positive int32.
		id := r.PathValue("id")
		n, err := strconv.ParseInt(id, 10, 32)
		if err != nil || n <= 0 {
			logger.Warn("Get truck availability invalid id", "op", op, "id", id, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		logger.Info("Getting truck availability", "op", op, "id", n, "rid", rid)

		out, appErr := h.TruckAvailability.Get(r.Context(), int32(n))
		if appErr != nil {
			logger.Warn("Get truck availability failed", "op", op, "id", n, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Truck availability loaded", "op", op, "id", n, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    out,
		})
	}
}

// TruckAvailabilityList lists availabilities with optional filters.
// Query parsing is done defensively; invalid values are ignored (service validates ranges).
func (h *Handler) TruckAvailabilityList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "truck_availability.list"
		rid := ctxmeta.RequestID(r.Context())

		qp := r.URL.Query()

		// parseI32 parses a query param into *int32 or returns nil when missing/malformed.
		parseI32 := func(k string) *int32 {
			if s := qp.Get(k); s != "" {
				if n, err := strconv.ParseInt(s, 10, 32); err == nil {
					v := int32(n)
					return &v
				}
			}
			return nil
		}

		// parseBool accepts 1/0/true/false (case-insensitive) and returns a pointer or nil.
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

		// getStr returns a trimmed non-empty string pointer or nil.
		getStr := func(k string) *string {
			if s := strings.TrimSpace(qp.Get(k)); s != "" {
				v := s
				return &v
			}
			return nil
		}

		// Build the query DTO expected by the service layer.
		req := request.ListTruckAvailabilityQuery{
			StartCountryID: parseI32("start_country_id"),
			StartCityID:    parseI32("start_city_id"),
			EndCountryID:   parseI32("end_country_id"),
			EndCityID:      parseI32("end_city_id"),
			TruckType:      getStr("truck_type"),
			Status:         getStr("status"),
			FullLoad:       parseBool("full_load"),
			PartialLoad:    parseBool("partial_load"),
			AvailableFrom:  getStr("available_from"),
			AvailableTo:    getStr("available_to"),
			Page:           parseI32("page"),
			Limit:          parseI32("limit"),
		}

		logger.Info("Listing truck availability", "op", op, "query", r.URL.RawQuery, "rid", rid)

		out, appErr := h.TruckAvailability.List(r.Context(), req)
		if appErr != nil {
			logger.Error("List truck availability failed", "op", op, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Truck availability list loaded", "op", op, "count", len(out), "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    out,
		})
	}
}

// TruckAvailabilityUpdateStatus updates the status for a specific availability.
// Requires ownership validation in the service; accepts only allowed enum values.
func (h *Handler) TruckAvailabilityUpdateStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "truck_availability.update_status"
		rid := ctxmeta.RequestID(r.Context())

		// Must be authenticated; service enforces ownership.
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized update truck availability status", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		// Parse id from the path and validate it's a positive int32.
		idStr := r.PathValue("id")
		id64, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil || id64 <= 0 {
			logger.Warn("Update truck availability invalid id", "op", op, "id", idStr, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}
		id := int32(id64)

		var reqBody request.UpdateTruckAvailabilityStatusRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid update status payload", "op", op, "user_id", claims.UserID, "id", id, "rid", rid)
			return
		}

		logger.Info("Updating truck availability status", "op", op, "user_id", claims.UserID, "id", id, "status", reqBody.Status, "rid", rid)

		out, appErr := h.TruckAvailability.UpdateStatus(r.Context(), id, int32(claims.UserID), reqBody.Status)
		if appErr != nil {
			logger.Error("Update truck availability status failed", "op", op, "user_id", claims.UserID, "id", id, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Truck availability status updated", "op", op, "user_id", claims.UserID, "id", id, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "status updated",
			"data":    out,
		})
	}
}
