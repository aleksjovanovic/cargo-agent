package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/redis/go-redis/v9"
)

// CountryService encapsulates read-only country/city lookups.
// It optionally uses Redis for simple read-through caching to reduce DB load.
type CountryService struct {
	q   *store.Queries // sqlc-generated queries
	rdb *redis.Client  // optional Redis client (can be nil)
}

const (
	// Default TTL for cached country/city payloads.
	cacheTTL = 24 * time.Hour

	// Cache key formats (v1 allows future schema changes without breaking old keys).
	keyCountriesListFmt = "countries:list:v1"
	keyCountryByIDFmt   = "countries:id:%d:v1"
	keyCountryCitiesFmt = "countries:%d:cities:v1"
)

// NewCountryService wires the sqlc queries and (optionally) a Redis client.
// If rdb is nil, the service will work without caching.
func NewCountryService(q *store.Queries, rdb *redis.Client) *CountryService {
	return &CountryService{q: q, rdb: rdb}
}

// List returns all countries. If Redis is configured, it tries a cache hit first.
// On cache miss, it loads from DB, caches the result, then returns it.
func (s *CountryService) List(ctx context.Context) ([]map[string]any, *response.AppError) {
	key := keyCountriesListFmt
	logger.Debug("countrysvc.list.start", "req_id", ctxmeta.RequestID(ctx))

	// Try cache (when available).
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
			var out []map[string]any
			if err := json.Unmarshal([]byte(cached), &out); err == nil {
				logger.Debug("countrysvc.list.cache_hit", "req_id", ctxmeta.RequestID(ctx), "count", len(out))
				return out, nil
			}
			logger.Warn("countrysvc.list.cache_corrupt", "req_id", ctxmeta.RequestID(ctx), "error", err)
		}
	}

	// Fallback to DB.
	rows, err := s.q.ListCountries(ctx)
	if err != nil {
		logger.Warn("countrysvc.list.not_found", "req_id", ctxmeta.RequestID(ctx), "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "Countries not found", Status: 404}
	}

	// Convert to []map[string]any to decouple API layer from sqlc structs.
	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)

	// Best-effort cache set.
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, key, raw, cacheTTL).Err(); err != nil {
			logger.Warn("countrysvc.list.cache_set_failed", "req_id", ctxmeta.RequestID(ctx), "error", err)
		}
	}
	logger.Info("countrysvc.list.ok", "req_id", ctxmeta.RequestID(ctx), "count", len(out))
	return out, nil
}

// GetByID returns a single country by its ID. It is cached by ID if Redis is available.
func (s *CountryService) GetByID(ctx context.Context, id int32) (map[string]any, *response.AppError) {
	key := fmt.Sprintf(keyCountryByIDFmt, id)
	logger.Debug("countrysvc.get_by_id.start", "req_id", ctxmeta.RequestID(ctx), "id", id)

	// Try cache (when available).
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
			var out map[string]any
			if err := json.Unmarshal([]byte(cached), &out); err == nil {
				logger.Debug("countrysvc.get_by_id.cache_hit", "req_id", ctxmeta.RequestID(ctx), "id", id)
				return out, nil
			}
			logger.Warn("countrysvc.get_by_id.cache_corrupt", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		}
	}

	// Fallback to DB.
	row, err := s.q.GetCountryByID(ctx, id)
	if err != nil {
		logger.Warn("countrysvc.get_by_id.not_found", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "Country not found", Status: 404}
	}

	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	// Best-effort cache set.
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, key, raw, cacheTTL).Err(); err != nil {
			logger.Warn("countrysvc.get_by_id.cache_set_failed", "req_id", ctxmeta.RequestID(ctx), "id", id, "error", err)
		}
	}
	logger.Info("countrysvc.get_by_id.ok", "req_id", ctxmeta.RequestID(ctx), "id", id)
	return out, nil
}

// GetByName returns a single country by its exact name (case/DB collation dependent).
// This call is not cached because name lookups are typically less frequent and may be used
// as a precursor to city listing (which is cached by ID).
func (s *CountryService) GetByName(ctx context.Context, name string) (map[string]any, *response.AppError) {
	logger.Debug("countrysvc.get_by_name.start", "req_id", ctxmeta.RequestID(ctx), "name", name)

	row, err := s.q.GetCountryByName(ctx, name)
	if err != nil {
		logger.Warn("countrysvc.get_by_name.not_found", "req_id", ctxmeta.RequestID(ctx), "name", name, "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "Country not found", Status: 404}
	}

	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("countrysvc.get_by_name.ok", "req_id", ctxmeta.RequestID(ctx), "name", name)
	return out, nil
}

// ListCities returns all cities for a given country ID. Results are cached per-country.
func (s *CountryService) ListCities(ctx context.Context, countryID int32) ([]map[string]any, *response.AppError) {
	key := fmt.Sprintf(keyCountryCitiesFmt, countryID)
	logger.Debug("countrysvc.list_cities.start", "req_id", ctxmeta.RequestID(ctx), "country_id", countryID)

	// Try cache (when available).
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
			var out []map[string]any
			if err := json.Unmarshal([]byte(cached), &out); err == nil {
				logger.Debug("countrysvc.list_cities.cache_hit", "req_id", ctxmeta.RequestID(ctx), "country_id", countryID, "count", len(out))
				return out, nil
			}
			logger.Warn("countrysvc.list_cities.cache_corrupt", "req_id", ctxmeta.RequestID(ctx), "country_id", countryID, "error", err)
		}
	}

	// Fallback to DB.
	rows, err := s.q.ListCitiesByCountryID(ctx, countryID)
	if err != nil {
		logger.Warn("countrysvc.list_cities.not_found", "req_id", ctxmeta.RequestID(ctx), "country_id", countryID, "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "Cities not found", Status: 404}
	}

	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)

	// Best-effort cache set.
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, key, raw, cacheTTL).Err(); err != nil {
			logger.Warn("countrysvc.list_cities.cache_set_failed", "req_id", ctxmeta.RequestID(ctx), "country_id", countryID, "error", err)
		}
	}
	logger.Info("countrysvc.list_cities.ok", "req_id", ctxmeta.RequestID(ctx), "country_id", countryID, "count", len(out))
	return out, nil
}
