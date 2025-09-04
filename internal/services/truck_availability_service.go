package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// TruckAvailabilityService owns business logic for truck availability posts.
// It coordinates validation, DB access (via sqlc), and shapes responses.
type TruckAvailabilityService struct {
	db *sql.DB        // raw DB handle (kept for future transactions if needed)
	q  *store.Queries // sqlc-generated queries
}

// NewTruckAvailabilityService wires sqlc queries and returns the service.
func NewTruckAvailabilityService(db *sql.DB, q *store.Queries) *TruckAvailabilityService {
	return &TruckAvailabilityService{db: db, q: q}
}

// Create validates input, normalizes values, inserts a truck availability row,
// and returns a generic map payload for the handler layer.
func (s *TruckAvailabilityService) Create(ctx context.Context, userID int32, req request.CreateTruckAvailabilityRequest) (map[string]any, *response.AppError) {
	logger.Info(
		"trucksvc.create.start",
		"req_id", ctxmeta.RequestID(ctx),
		"user_id", userID,
		"start_country_id", req.StartCountryID,
		"start_city_id", req.StartCityID,
	)

	// 1) Validate request at the edge of the service.
	if err := validation.ValidateCreateTruckAvailability(&req); err != nil {
		logger.Warn("trucksvc.create.validation_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: httpStatusBadRequest}
	}

	// 2) Parse required RFC3339 timestamps.
	from, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableFrom))
	if err != nil {
		logger.Warn("trucksvc.create.invalid_available_from", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "value", req.AvailableFrom)
		return nil, &response.AppError{Code: "bad_request", Message: "invalid available_from", Status: httpStatusBadRequest}
	}
	to, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableTo))
	if err != nil {
		logger.Warn("trucksvc.create.invalid_available_to", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "value", req.AvailableTo)
		return nil, &response.AppError{Code: "bad_request", Message: "invalid available_to", Status: httpStatusBadRequest}
	}
	// Small safety improvement: require AvailableTo to be strictly after AvailableFrom.
	if !to.After(from) {
		logger.Warn("trucksvc.create.time_window_invalid", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "from", from, "to", to)
		return nil, &response.AppError{Code: "bad_request", Message: "available_to must be after available_from", Status: httpStatusBadRequest}
	}

	// 3) Parse optional expiry.
	var expAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expAt = &t
		} else {
			// Non-fatal: ignore invalid expires_at but log a warning.
			logger.Warn("trucksvc.create.invalid_expires_at", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "value", *req.ExpiresAt)
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

	// 5) Prepare insert parameters (normalize enums to lowercase, numeric strings, null wrappers).
	row, err := s.q.CreateTruckAvailability(ctx, store.CreateTruckAvailabilityParams{
		CreatedBy:      userID,
		StartCountryID: req.StartCountryID,
		StartCityID:    req.StartCityID,
		EndCountryID:   utils.ToNullInt32(req.EndCountryID),
		EndCityID:      utils.ToNullInt32(req.EndCityID),

		AvailableFrom: from,
		AvailableTo:   to,

		TruckType:   strings.ToLower(req.TruckType),
		MaxWeightT:  utils.ToNumericString(req.MaxWeightT),
		MaxVolumeM3: utils.ToNullNumericString(req.MaxVolumeM3),

		FullLoad:        req.FullLoad,
		PartialLoad:     req.PartialLoad,
		LoadingPlaces:   req.LoadingPlaces,
		UnloadingPlaces: req.UnloadingPlaces,

		PublishedAt: utils.SqlNullTimePtr(nil), // created as "published" now; can be set explicitly later
		ExpiresAt:   utils.SqlNullTimePtr(expAt),

		PosterName: poster,
		PricePerKm: utils.ToNullNumericString(req.PricePerKm),
		Currency:   utils.ToLowerNullString(req.Currency),
		Notes:      utils.ToNullString(req.Notes),

		Status: "published",
	})
	if err != nil {
		logger.Error("trucksvc.create.failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "create_failed", Message: "Failed to create truck availability", Status: httpStatusInternalServerError}
	}

	// 6) Convert sqlc struct to map[string]any to keep handler layer decoupled.
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("trucksvc.create.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "id", out["id"])
	return out, nil
}

// Get loads a single truck availability by ID or returns a typed AppError.
func (s *TruckAvailabilityService) Get(ctx context.Context, id int32) (map[string]any, *response.AppError) {
	logger.Debug("trucksvc.get.start", "req_id", ctxmeta.RequestID(ctx), "id", id)

	row, err := s.q.GetTruckAvailability(ctx, id)
	if err != nil {
		logger.Warn("trucksvc.get.not_found", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "Truck availability not found", Status: httpStatusNotFound}
	}

	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("trucksvc.get.ok", "req_id", ctxmeta.RequestID(ctx), "id", id)
	return out, nil
}

// List returns a filtered and paginated collection of truck availability posts.
// It parses optional RFC3339 windows and passes nullable filters to sqlc.
func (s *TruckAvailabilityService) List(ctx context.Context, qy request.ListTruckAvailabilityQuery) ([]map[string]any, *response.AppError) {
	// Pagination with sensible bounds (1..N pages, limit <= 100).
	limit := int32(20)
	if qy.Limit != nil && *qy.Limit > 0 {
		limit = *qy.Limit
		if limit > 100 {
			limit = 100
		}
	}
	page := int32(1)
	if qy.Page != nil && *qy.Page > 0 {
		page = *qy.Page
	}
	offset := (page - 1) * limit

	logger.Debug(
		"trucksvc.list.start",
		"req_id", ctxmeta.RequestID(ctx),
		"page", page, "limit", limit,
		"start_country_id", qy.StartCountryID, "start_city_id", qy.StartCityID,
		"end_country_id", qy.EndCountryID, "end_city_id", qy.EndCityID,
		"truck_type", qy.TruckType, "status", qy.Status,
		"available_from", qy.AvailableFrom, "available_to", qy.AvailableTo,
		"full_load", qy.FullLoad, "partial_load", qy.PartialLoad,
	)

	// Helper: parse optional RFC3339 strings into *time.Time.
	parseTime := func(p *string) (*time.Time, *response.AppError) {
		if p == nil || strings.TrimSpace(*p) == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*p))
		if err != nil {
			logger.Warn("trucksvc.list.invalid_time", "req_id", ctxmeta.RequestID(ctx), "value", *p)
			return nil, &response.AppError{Code: "bad_request", Message: "invalid time format (RFC3339)", Status: httpStatusBadRequest}
		}
		return &t, nil
	}

	afrom, appErr := parseTime(qy.AvailableFrom)
	if appErr != nil {
		return nil, appErr
	}
	ato, appErr := parseTime(qy.AvailableTo)
	if appErr != nil {
		return nil, appErr
	}

	// Query DB with null-wrapped filters.
	rows, err := s.q.ListTruckAvailability(ctx, store.ListTruckAvailabilityParams{
		StartCountryID: utils.ToNullInt32(qy.StartCountryID),
		StartCityID:    utils.ToNullInt32(qy.StartCityID),
		EndCountryID:   utils.ToNullInt32(qy.EndCountryID),
		EndCityID:      utils.ToNullInt32(qy.EndCityID),

		TruckType: utils.ToLowerNullString(qy.TruckType),
		Status:    utils.ToLowerNullString(qy.Status),

		AvailableFrom: utils.SqlNullTimePtr(afrom),
		AvailableTo:   utils.SqlNullTimePtr(ato),

		FullLoad:    utils.ToNullBool(qy.FullLoad),
		PartialLoad: utils.ToNullBool(qy.PartialLoad),

		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		logger.Error("trucksvc.list.failed", "req_id", ctxmeta.RequestID(ctx), "error", err)
		return nil, &response.AppError{Code: "list_failed", Message: "Failed to list truck availability", Status: httpStatusInternalServerError}
	}

	// Convert to []map for API independence from sqlc structs.
	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("trucksvc.list.ok", "req_id", ctxmeta.RequestID(ctx), "count", len(out))
	return out, nil
}

// UpdateStatus changes the status of a post, with ownership check to prevent
// users from modifying others' posts.
func (s *TruckAvailabilityService) UpdateStatus(ctx context.Context, id int32, userID int32, status string) (map[string]any, *response.AppError) {
	logger.Info("trucksvc.update_status.start", "req_id", ctxmeta.RequestID(ctx), "id", id, "user_id", userID, "status", status)

	// Normalize and validate allowed statuses.
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "draft", "published", "cancelled", "expired", "closed":
	default:
		logger.Warn("trucksvc.update_status.bad_status", "req_id", ctxmeta.RequestID(ctx), "id", id, "status", status)
		return nil, &response.AppError{
			Code:    "bad_request",
			Message: "status must be one of: draft, published, cancelled, expired, closed",
			Status:  httpStatusBadRequest,
		}
	}

	// 1) Ownership guard — only the creator can modify.
	row, err := s.q.GetTruckAvailability(ctx, id)
	if err != nil {
		logger.Warn("trucksvc.update_status.not_found", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "Truck availability not found", Status: httpStatusNotFound}
	}
	if row.CreatedBy != userID {
		logger.Warn("trucksvc.update_status.forbidden", "req_id", ctxmeta.RequestID(ctx), "id", id, "owner_id", row.CreatedBy, "user_id", userID)
		return nil, &response.AppError{Code: "forbidden", Message: "You cannot modify this resource", Status: httpStatusForbidden}
	}

	// 2) Persist the new status.
	updated, err := s.q.UpdateTruckAvailabilityStatus(ctx, store.UpdateTruckAvailabilityStatusParams{
		ID:     id,
		Status: st,
	})
	if err != nil {
		logger.Error("trucksvc.update_status.db_failed", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "update_failed", Message: "Failed to update status", Status: httpStatusInternalServerError}
	}

	raw, _ := json.Marshal(updated)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("trucksvc.update_status.ok", "req_id", ctxmeta.RequestID(ctx), "id", id, "status", st)
	return out, nil
}
