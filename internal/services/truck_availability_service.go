package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
	"github.com/aleksjovanovic/cargo-agent/internal/validation"
)

type TruckAvailabilityService struct {
	db *sql.DB
	q  *store.Queries
}

func NewTruckAvailabilityService(db *sql.DB, q *store.Queries) *TruckAvailabilityService {
	return &TruckAvailabilityService{db: db, q: q}
}

func (s *TruckAvailabilityService) Create(ctx context.Context, userID int32, req request.CreateTruckAvailabilityRequest) (map[string]any, *response.AppError) {
	if err := validation.ValidateCreateTruckAvailability(&req); err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: 400}
	}

	from, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableFrom))
	if err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: "invalid available_from", Status: 400}
	}
	to, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableTo))
	if err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: "invalid available_to", Status: 400}
	}

	var expAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expAt = &t
		}
	}

	// poster_name snapshot (pokuša name, fallback na username)
	u, _ := s.q.GetUser(ctx, userID)
	poster := strings.TrimSpace(u.Name)
	if poster == "" {
		poster = u.Username
	}

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

		PublishedAt: utils.SqlNullTimePtr(nil), // default now() u DB
		ExpiresAt:   utils.SqlNullTimePtr(expAt),

		PosterName: poster,
		PricePerKm: utils.ToNullNumericString(req.PricePerKm),
		Currency:   utils.ToLowerNullString(req.Currency),
		Notes:      utils.ToNullString(req.Notes),

		Status: "published",
	})
	if err != nil {
		return nil, &response.AppError{Code: "create_failed", Message: "Failed to create truck availability", Status: 500}
	}

	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *TruckAvailabilityService) Get(ctx context.Context, id int32) (map[string]any, *response.AppError) {
	row, err := s.q.GetTruckAvailability(ctx, id)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "Truck availability not found", Status: 404}
	}
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *TruckAvailabilityService) List(ctx context.Context, qy request.ListTruckAvailabilityQuery) ([]map[string]any, *response.AppError) {
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

	parseTime := func(p *string) (*time.Time, *response.AppError) {
		if p == nil || strings.TrimSpace(*p) == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*p))
		if err != nil {
			return nil, &response.AppError{Code: "bad_request", Message: "invalid time format (RFC3339)", Status: 400}
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

	rows, err := s.q.ListTruckAvailability(ctx, store.ListTruckAvailabilityParams{
		StartCountryID: utils.ToNullInt32(qy.StartCountryID),
		StartCityID:    utils.ToNullInt32(qy.StartCityID),
		EndCountryID:   utils.ToNullInt32(qy.EndCountryID),
		EndCityID:      utils.ToNullInt32(qy.EndCityID),

		TruckType: utils.ToLowerNullString(qy.TruckType), // enum string filter
		Status:    utils.ToLowerNullString(qy.Status),    // enum string filter

		AvailableFrom: utils.SqlNullTimePtr(afrom),
		AvailableTo:   utils.SqlNullTimePtr(ato),

		FullLoad:    utils.ToNullBool(qy.FullLoad),
		PartialLoad: utils.ToNullBool(qy.PartialLoad),

		Limit:  limit,
		Offset: offset,
	})

	if err != nil {
		return nil, &response.AppError{Code: "list_failed", Message: "Failed to list truck availability", Status: 500}
	}

	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *TruckAvailabilityService) UpdateStatus(ctx context.Context, id int32, userID int32, status string) (map[string]any, *response.AppError) {
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "draft", "published", "cancelled", "expired", "closed":
	default:
		return nil, &response.AppError{
			Code:    "bad_request",
			Message: "status must be one of: draft, published, cancelled, expired, closed",
			Status:  400,
		}
	}

	// 1) Provera vlasništva (da ne možemo menjati tuđ oglas)
	row, err := s.q.GetTruckAvailability(ctx, id)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "Truck availability not found", Status: 404}
	}
	if row.CreatedBy != userID {
		return nil, &response.AppError{Code: "forbidden", Message: "You cannot modify this resource", Status: 403}
	}

	// 2) Update status-a
	updated, err := s.q.UpdateTruckAvailabilityStatus(ctx, store.UpdateTruckAvailabilityStatusParams{
		ID:     id,
		Status: st,
	})
	if err != nil {
		return nil, &response.AppError{Code: "update_failed", Message: "Failed to update status", Status: 500}
	}

	// 3) Povratak kao map[string]any
	raw, _ := json.Marshal(updated)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}
