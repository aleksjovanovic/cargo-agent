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
