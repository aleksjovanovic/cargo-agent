// internal/api/v1/handlers/cargo_offers_handler.go
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

// Create creates a cargo offer for the authenticated user.
// Flow: auth check -> decode/validate JSON -> delegate to service -> return 201 on success.
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "cargo_offers.create"
		rid := ctxmeta.RequestID(r.Context())

		// Claims are injected by Authz middleware; absence => unauthenticated.
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized create cargo offer", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var reqBody request.CreateCargoOfferRequest
		// decodeOr400 writes 400 if body is invalid; we only log and return.
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid create cargo offer payload", "op", op, "user_id", claims.UserID, "rid", rid)
			return
		}

		logger.Info("Creating cargo offer", "op", op, "user_id", claims.UserID, "rid", rid)

		out, appErr := h.CargoOffer.Create(r.Context(), int32(claims.UserID), reqBody)
		if appErr != nil {
			logger.Error("Create cargo offer failed", "op", op, "user_id", claims.UserID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		// Try to log the created resource id if present in the service response.
		var createdID any
		if id, ok := out["id"]; ok {
			createdID = id
		}

		logger.Info("Cargo offer created", "op", op, "user_id", claims.UserID, "offer_id", createdID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "cargo offer created",
			"data":    out,
		})
	}
}

// GetByID fetches a cargo offer by its numeric {id} path parameter.
// Public endpoint (no auth required). Returns 400 on malformed id, 404/other via service errors.
func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "cargo_offers.get"
		rid := ctxmeta.RequestID(r.Context())

		idStr := r.PathValue("id")
		id64, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil || id64 <= 0 {
			logger.Warn("Get cargo offer invalid id", "op", op, "id", idStr, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		logger.Info("Getting cargo offer", "op", op, "id", id64, "rid", rid)

		out, appErr := h.CargoOffer.Get(r.Context(), int32(id64))
		if appErr != nil {
			logger.Warn("Get cargo offer failed", "op", op, "id", id64, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Cargo offer loaded", "op", op, "id", id64, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    out,
		})
	}
}

// List returns a filtered, paginated list of cargo offers.
// Query parsing here is permissive; the service layer validates ranges/formats further.
func (h *Handler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "cargo_offers.list"
		rid := ctxmeta.RequestID(r.Context())
		qv := r.URL.Query()

		// i32 parses a query param string into *int32 or returns nil when empty/malformed.
		i32 := func(s string) *int32 {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			if n, err := strconv.ParseInt(s, 10, 32); err == nil {
				v := int32(n)
				return &v
			}
			return nil
		}

		// str returns a lowercased, trimmed *string or nil when empty.
		// Lowercasing helps unify enum-like filters (status, truck_type, etc.).
		str := func(s string) *string {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			v := strings.ToLower(strings.TrimSpace(s))
			return &v
		}

		// Build DTO consumed by service layer.
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

		logger.Info("Listing cargo offers", "op", op, "query", r.URL.RawQuery, "rid", rid)

		data, appErr := h.CargoOffer.List(r.Context(), req)
		if appErr != nil {
			logger.Error("List cargo offers failed", "op", op, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Cargo offers listed", "op", op, "count", len(data), "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    data,
		})
	}
}

// CargoOfferUpdateStatus updates the status of a cargo offer owned by the authenticated user.
// Service enforces ownership and validates allowed status values.
func (h *Handler) CargoOfferUpdateStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "cargo_offers.update_status"
		rid := ctxmeta.RequestID(r.Context())

		// Must be authenticated; service will verify ownership.
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized update cargo offer status", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		// Parse {id} from path and ensure it's a positive int32.
		idStr := r.PathValue("id")
		id64, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil || id64 <= 0 {
			logger.Warn("Update cargo offer invalid id", "op", op, "id", idStr, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}
		id := int32(id64)

		var reqBody request.UpdateCargoOfferStatusRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid update status payload", "op", op, "user_id", claims.UserID, "id", id, "rid", rid)
			return
		}

		logger.Info("Updating cargo offer status", "op", op, "user_id", claims.UserID, "id", id, "status", reqBody.Status, "rid", rid)

		out, appErr := h.CargoOffer.UpdateStatus(r.Context(), id, int32(claims.UserID), reqBody.Status)
		if appErr != nil {
			logger.Error("Update cargo offer status failed", "op", op, "user_id", claims.UserID, "id", id, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Cargo offer status updated", "op", op, "user_id", claims.UserID, "id", id, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "status updated",
			"data":    out,
		})
	}
}
