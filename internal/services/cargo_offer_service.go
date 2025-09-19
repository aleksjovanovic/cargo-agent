package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
	"github.com/aleksjovanovic/cargo-agent/internal/validation"
)

// CargoOfferService groups the data access for cargo offers.
// It stays thin: validation + shaping params, while SQL lives in sqlc-generated queries.
type CargoOfferService struct {
	db *sql.DB
	q  *store.Queries
}

// NewCargoOfferService wires the service with a DB handle and sqlc queries.
func NewCargoOfferService(db *sql.DB, q *store.Queries) *CargoOfferService {
	return &CargoOfferService{db: db, q: q}
}

// Create validates payload, normalizes inputs, persists a new cargo offer and returns a generic map.
// Returning map[string]any avoids exposing internal sqlc structs over the API layer.
func (s *CargoOfferService) Create(ctx context.Context, userID int32, req request.CreateCargoOfferRequest) (map[string]any, *response.AppError) {
	logger.Info("cargooffersvc.create.start",
		"req_id", ctxmeta.RequestID(ctx), "user_id", userID,
		"origin_country_id", req.OriginCountryID, "origin_city_id", req.OriginCityID,
		"dest_country_id", req.DestinationCountryID, "dest_city_id", req.DestinationCityID,
		"load_type", req.LoadType, "truck_type", req.TruckType,
	)

	// 1) Validate shape and business rules (enum values, date ordering, positive numbers, etc.).
	if err := validation.ValidateCreateCargoOffer(&req); err != nil {
		logger.Warn("cargooffersvc.create.validation_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: http.StatusBadRequest}
	}

	// 2) Parse required timestamps; validation already checked format/order, but parsing is still needed.
	rtl, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ReadyToLoadBy))
	if err != nil {
		logger.Warn("cargooffersvc.create.invalid_ready_to_load_by", "req_id", ctxmeta.RequestID(ctx), "value", req.ReadyToLoadBy)
		return nil, &response.AppError{Code: "bad_request", Message: "invalid ready_to_load_by", Status: http.StatusBadRequest}
	}
	ddl, err := time.Parse(time.RFC3339, strings.TrimSpace(req.DeliveryDeadline))
	if err != nil {
		logger.Warn("cargooffersvc.create.invalid_delivery_deadline", "req_id", ctxmeta.RequestID(ctx), "value", req.DeliveryDeadline)
		return nil, &response.AppError{Code: "bad_request", Message: "invalid delivery_deadline", Status: http.StatusBadRequest}
	}

	// 3) Optional timestamps; accept empty values and normalize RFC3339.
	var pubAt *time.Time
	if req.PublishedAt != nil && strings.TrimSpace(*req.PublishedAt) != "" {
		if t, err := time.Parse(time.RFC3339, *req.PublishedAt); err == nil {
			pubAt = &t
		} else {
			logger.Warn("cargooffersvc.create.invalid_published_at", "req_id", ctxmeta.RequestID(ctx), "value", *req.PublishedAt)
			return nil, &response.AppError{Code: "bad_request", Message: "invalid published_at", Status: http.StatusBadRequest}
		}
	}
	var expAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expAt = &t
		} else {
			logger.Warn("cargooffersvc.create.invalid_expires_at", "req_id", ctxmeta.RequestID(ctx), "value", *req.ExpiresAt)
			return nil, &response.AppError{Code: "bad_request", Message: "invalid expires_at", Status: http.StatusBadRequest}
		}
	}

	// 4) Build a human-friendly poster snapshot from user row (non-fatal if missing).
	u, uErr := s.q.GetUser(ctx, userID)
	poster := ""
	if uErr == nil {
		poster = strings.TrimSpace(u.Name)
		if poster == "" {
			poster = u.Username
		}
	}
	if strings.TrimSpace(poster) == "" {
		poster = fmt.Sprintf("user-%d", userID)
	}

	// 5) Prepare insert parameters, normalizing case for enums and converting optionals to sql-friendly types.
	params := store.CreateCargoOfferParams{
		CreatedBy:            userID,
		OriginCountryID:      req.OriginCountryID,
		OriginCityID:         req.OriginCityID,
		DestinationCountryID: req.DestinationCountryID,
		DestinationCityID:    req.DestinationCityID,

		LoadingPlaces:   req.LoadingPlaces,
		UnloadingPlaces: req.UnloadingPlaces,

		ReadyToLoadBy:    rtl,
		DeliveryDeadline: ddl,

		LoadType:  strings.ToLower(req.LoadType),
		TruckType: strings.ToLower(req.TruckType),

		// Store decimals as strings when needed to avoid float precision quirks at the DB boundary.
		WeightT:         utils.ToNumericString(req.WeightT),
		VolumeM3:        utils.ToNullNumericString(req.VolumeM3),
		Pallets:         utils.ToNullInt32(req.Pallets),
		Palletized:      sql.NullBool{Bool: req.Palletized, Valid: true},
		TemperatureMinC: utils.ToNullNumericString(req.TemperatureMinC),
		TemperatureMaxC: utils.ToNullNumericString(req.TemperatureMaxC),

		PublishedAt: utils.SqlNullTimePtr(pubAt),
		ExpiresAt:   utils.SqlNullTimePtr(expAt),
		PosterName:  poster,

		Price:    utils.ToNullNumericString(req.Price),
		Currency: utils.ToNullString(req.Currency),
		Notes:    utils.ToNullString(req.Notes),

		// Default status for newly created offers (business decision).
		Status: "published",
	}

	// 6) Persist to DB.
	row, err := s.q.CreateCargoOffer(ctx, params)
	if err != nil {
		logger.Error("cargooffersvc.create.db_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{
			Code: "create_failed", Message: "failed to create cargo offer", Status: http.StatusInternalServerError, Details: err.Error(),
		}
	}

	// 7) Convert the sqlc row struct to a generic map to keep handler responses decoupled from store layer structs.
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("cargooffersvc.create.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "id", out["id"])
	return out, nil
}

// Get loads a single cargo offer by ID; 404 when not found.
func (s *CargoOfferService) Get(ctx context.Context, id int32) (map[string]any, *response.AppError) {
	logger.Debug("cargooffersvc.get.start", "req_id", ctxmeta.RequestID(ctx), "id", id)

	row, err := s.q.GetCargoOffer(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("cargooffersvc.get.not_found", "req_id", ctxmeta.RequestID(ctx), "id", id)
			return nil, &response.AppError{Code: "not_found", Message: "cargo offer not found", Status: http.StatusNotFound}
		}
		logger.Error("cargooffersvc.get.db_error", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "db_error", Message: "failed to load cargo offer", Status: http.StatusInternalServerError, Details: err.Error()}
	}

	// Keep response generic.
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("cargooffersvc.get.ok", "req_id", ctxmeta.RequestID(ctx), "id", id)
	return out, nil
}

// List returns cargo offers with filtering and pagination.
// Query parameters are normalized and validated lightly (e.g., RFC3339 for date filters).
func (s *CargoOfferService) List(ctx context.Context, q request.ListCargoOffersQuery) ([]map[string]any, *response.AppError) {
	const (
		defaultLimit = int32(20)
		maxLimit     = int32(100)
	)

	// Normalize paging.
	limit := defaultLimit
	if q.Limit != nil && *q.Limit > 0 {
		limit = *q.Limit
		if limit > maxLimit {
			limit = maxLimit
		}
	}
	page := int32(1)
	if q.Page != nil && *q.Page > 0 {
		page = *q.Page
	}
	offset := (page - 1) * limit

	logger.Debug("cargooffersvc.list.start", "req_id", ctxmeta.RequestID(ctx),
		"page", page, "limit", limit,
		"origin_country_id", q.OriginCountryID, "origin_city_id", q.OriginCityID,
		"dest_country_id", q.DestinationCountryID, "dest_city_id", q.DestinationCityID,
		"load_type", q.LoadType, "truck_type", q.TruckType, "status", q.Status,
		"ready_from", q.ReadyFrom, "ready_to", q.ReadyTo,
		"delivery_from", q.DeliveryFrom, "delivery_to", q.DeliveryTo,
	)

	// Helper to parse optional RFC3339 timestamps.
	parseTime := func(p *string) (*time.Time, *response.AppError) {
		if p == nil || strings.TrimSpace(*p) == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*p))
		if err != nil {
			logger.Warn("cargooffersvc.list.invalid_time", "req_id", ctxmeta.RequestID(ctx), "value", *p)
			return nil, &response.AppError{Code: "bad_request", Message: "invalid time format (use RFC3339)", Status: http.StatusBadRequest}
		}
		return &t, nil
	}

	// Parse date filters lazily and bail fast on invalid inputs.
	readyFrom, appErr := parseTime(q.ReadyFrom)
	if appErr != nil {
		return nil, appErr
	}
	readyTo, appErr := parseTime(q.ReadyTo)
	if appErr != nil {
		return nil, appErr
	}
	deliveryFrom, appErr := parseTime(q.DeliveryFrom)
	if appErr != nil {
		return nil, appErr
	}
	deliveryTo, appErr := parseTime(q.DeliveryTo)
	if appErr != nil {
		return nil, appErr
	}

	// Delegate to sqlc with properly converted nullable params.
	rows, err := s.q.ListCargoOffers(ctx, store.ListCargoOffersParams{
		OriginCountryID:      utils.ToNullInt32(q.OriginCountryID),
		OriginCityID:         utils.ToNullInt32(q.OriginCityID),
		DestinationCountryID: utils.ToNullInt32(q.DestinationCountryID),
		DestinationCityID:    utils.ToNullInt32(q.DestinationCityID),
		LoadType:             utils.ToLowerNullString(q.LoadType),
		TruckType:            utils.ToLowerNullString(q.TruckType),
		Status:               utils.ToLowerNullString(q.Status),
		ReadyFrom:            utils.SqlNullTimePtr(readyFrom),
		ReadyTo:              utils.SqlNullTimePtr(readyTo),
		DeliveryFrom:         utils.SqlNullTimePtr(deliveryFrom),
		DeliveryTo:           utils.SqlNullTimePtr(deliveryTo),
		Limit:                limit,
		Offset:               offset,
	})
	if err != nil {
		logger.Error("cargooffersvc.list.db_failed", "req_id", ctxmeta.RequestID(ctx), "error", err)
		return nil, &response.AppError{Code: "list_failed", Message: "Failed to list cargo offers", Status: http.StatusInternalServerError}
	}

	// Convert to []map for API layer.
	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("cargooffersvc.list.ok", "req_id", ctxmeta.RequestID(ctx), "count", len(out))
	return out, nil
}

// UpdateStatus enforces allowed state transitions, checks ownership, updates the record, and returns the updated view.
func (s *CargoOfferService) UpdateStatus(ctx context.Context, id int32, userID int32, status string) (map[string]any, *response.AppError) {
	logger.Info("cargooffersvc.update_status.start", "req_id", ctxmeta.RequestID(ctx), "id", id, "user_id", userID, "status", status)

	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "draft", "published", "cancelled", "expired", "closed":
		// allowed
	default:
		logger.Warn("cargooffersvc.update_status.bad_status", "req_id", ctxmeta.RequestID(ctx), "id", id, "status", status)
		return nil, &response.AppError{
			Code: "bad_request",
			Message: "status must be one of: draft, published, " +
				"cancelled, expired, closed",
			Status: http.StatusBadRequest,
		}
	}

	// 1) Ownership check: only the creator can change the status.
	row, err := s.q.GetCargoOffer(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("cargooffersvc.update_status.not_found", "req_id", ctxmeta.RequestID(ctx), "id", id)
			return nil, &response.AppError{Code: "not_found", Message: "cargo offer not found", Status: http.StatusNotFound}
		}
		logger.Error("cargooffersvc.update_status.db_error", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "db_error", Message: "failed to load cargo offer", Status: http.StatusInternalServerError, Details: err.Error()}
	}
	if row.CreatedBy != userID {
		logger.Warn("cargooffersvc.update_status.forbidden", "req_id", ctxmeta.RequestID(ctx), "id", id, "owner_id", row.CreatedBy, "user_id", userID)
		return nil, &response.AppError{Code: "forbidden", Message: "You cannot modify this resource", Status: http.StatusForbidden}
	}

	// 2) Persist the status change.
	updated, err := s.q.UpdateCargoOfferStatus(ctx, store.UpdateCargoOfferStatusParams{
		ID:     id,
		Status: st,
	})
	if err != nil {
		logger.Error("cargooffersvc.update_status.db_failed", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "update_failed", Message: "Failed to update status", Status: http.StatusInternalServerError}
	}

	// 3) Shape response.
	raw, _ := json.Marshal(updated)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("cargooffersvc.update_status.ok", "req_id", ctxmeta.RequestID(ctx), "id", id, "status", st)
	return out, nil
}
