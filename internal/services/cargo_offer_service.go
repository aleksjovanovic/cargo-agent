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

type CargoOfferService struct {
	db *sql.DB
	q  *store.Queries
}

func NewCargoOfferService(db *sql.DB, q *store.Queries) *CargoOfferService {
	return &CargoOfferService{db: db, q: q}
}

func (s *CargoOfferService) Create(
	ctx context.Context,
	userID int32,
	req request.CreateCargoOfferRequest,
) (map[string]any, *response.AppError) {

	// 1) Validacija
	if err := validation.ValidateCreateCargoOffer(&req); err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: 400}
	}

	// 2) Datumi/vremena
	rtl, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ReadyToLoadBy))
	if err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: "invalid ready_to_load_by", Status: 400}
	}
	ddl, err := time.Parse(time.RFC3339, strings.TrimSpace(req.DeliveryDeadline))
	if err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: "invalid delivery_deadline", Status: 400}
	}

	var pubAt *time.Time
	if req.PublishedAt != nil && strings.TrimSpace(*req.PublishedAt) != "" {
		if t, err := time.Parse(time.RFC3339, *req.PublishedAt); err == nil {
			pubAt = &t
		} else {
			return nil, &response.AppError{Code: "bad_request", Message: "invalid published_at", Status: 400}
		}
	}

	var expAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			expAt = &t
		} else {
			return nil, &response.AppError{Code: "bad_request", Message: "invalid expires_at", Status: 400}
		}
	}

	// 3) Parametri za insert (tipovi u skladu sa sqlc generisanim modelom)
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

		LoadType:  strings.ToLower(req.LoadType),  // "ftl"/"ltl"
		TruckType: strings.ToLower(req.TruckType), // "refrigerator"/"curtain"/...

		// NUMERIC → string ili NullString (u skladu s sqlc mapiranjem)
		WeightT:         utils.ToNumericString(req.WeightT),
		VolumeM3:        utils.ToNullNumericString(req.VolumeM3),
		Pallets:         utils.ToNullInt32(req.Pallets),
		Palletized:      sql.NullBool{Bool: req.Palletized, Valid: true},
		TemperatureMinC: utils.ToNullNumericString(req.TemperatureMinC),
		TemperatureMaxC: utils.ToNullNumericString(req.TemperatureMaxC),

		PublishedAt: utils.SqlNullTimePtr(pubAt),
		ExpiresAt:   utils.SqlNullTimePtr(expAt),

		Price:    utils.ToNullNumericString(req.Price),
		Currency: utils.ToNullString(req.Currency), // ✅ koristi postojeći helper
		Notes:    utils.ToNullString(req.Notes),    // ✅ koristi postojeći helper

		Status: "published",
	}

	// 4) Insert
	row, err := s.q.CreateCargoOffer(ctx, params)
	if err != nil {
		return nil, &response.AppError{
			Code: "create_failed", Message: "failed to create cargo offer", Status: 500, Details: err.Error(),
		}
	}

	// 5) Odgovor kao map (stabilan JSON)
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *CargoOfferService) Get(ctx context.Context, id int32) (map[string]any, *response.AppError) {
	row, err := s.q.GetCargoOffer(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &response.AppError{Code: "not_found", Message: "cargo offer not found", Status: 404}
		}
		return nil, &response.AppError{Code: "db_error", Message: "failed to load cargo offer", Status: 500, Details: err.Error()}
	}
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

// List vraća kolekciju ponuda sa filtrima i paginacijom.
func (s *CargoOfferService) List(ctx context.Context, q request.ListCargoOffersQuery) ([]map[string]any, *response.AppError) {
	limit := int32(20)
	if q.Limit != nil && *q.Limit > 0 {
		limit = *q.Limit
		if limit > 100 {
			limit = 100
		}
	}
	page := int32(1)
	if q.Page != nil && *q.Page > 0 {
		page = *q.Page
	}
	offset := (page - 1) * limit

	parseTime := func(p *string) (*time.Time, *response.AppError) {
		if p == nil || strings.TrimSpace(*p) == "" {
			return nil, nil
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*p))
		if err != nil {
			return nil, &response.AppError{Code: "bad_request", Message: "invalid time format (use RFC3339)", Status: 400}
		}
		return &t, nil
	}

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

	rows, err := s.q.ListCargoOffers(ctx, store.ListCargoOffersParams{
		OriginCountryID:      utils.ToNullInt32(q.OriginCountryID),
		OriginCityID:         utils.ToNullInt32(q.OriginCityID),
		DestinationCountryID: utils.ToNullInt32(q.DestinationCountryID),
		DestinationCityID:    utils.ToNullInt32(q.DestinationCityID),
		LoadType:             utils.ToLowerNullString(q.LoadType),  // string → enum u SQL-u (cast)
		TruckType:            utils.ToLowerNullString(q.TruckType), // string → enum u SQL-u (cast)
		Status:               utils.ToLowerNullString(q.Status),    // string → enum u SQL-u (cast)
		ReadyFrom:            utils.SqlNullTimePtr(readyFrom),
		ReadyTo:              utils.SqlNullTimePtr(readyTo),
		DeliveryFrom:         utils.SqlNullTimePtr(deliveryFrom),
		DeliveryTo:           utils.SqlNullTimePtr(deliveryTo),
		Limit:                limit,
		Offset:               offset,
	})
	if err != nil {
		return nil, &response.AppError{Code: "list_failed", Message: "Failed to list cargo offers", Status: 500}
	}

	// marshal → unmarshal da dobijemo []map[string]any (isti izlaz kao create/get)
	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}
