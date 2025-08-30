package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/redis/go-redis/v9"
)

type CountryService struct {
	q   *store.Queries
	rdb *redis.Client
}

func NewCountryService(q *store.Queries, rdb *redis.Client) *CountryService {
	return &CountryService{q: q, rdb: rdb}
}

func (s *CountryService) List(ctx context.Context) ([]map[string]any, *response.AppError) {
	key := "countries:list:v1"

	if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
		var out []map[string]any
		if err := json.Unmarshal([]byte(cached), &out); err == nil {
			return out, nil
		}
	}

	rows, err := s.q.ListCountries(ctx)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "Countries not found", Status: 404}
	}

	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)

	_ = s.rdb.Set(ctx, key, raw, 24*time.Hour).Err()
	return out, nil
}

func (s *CountryService) GetByID(ctx context.Context, id int32) (map[string]any, *response.AppError) {
	key := fmt.Sprintf("countries:id:%d:v1", id)

	if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
		var out map[string]any
		if err := json.Unmarshal([]byte(cached), &out); err == nil {
			return out, nil
		}
	}

	row, err := s.q.GetCountryByID(ctx, id)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "Country not found", Status: 404}
	}

	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	_ = s.rdb.Set(ctx, key, raw, 24*time.Hour).Err()
	return out, nil
}

func (s *CountryService) GetByName(ctx context.Context, name string) (map[string]any, *response.AppError) {
	row, err := s.q.GetCountryByName(ctx, name)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "Country not found", Status: 404}
	}
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *CountryService) ListCities(ctx context.Context, countryID int32) ([]map[string]any, *response.AppError) {
	key := fmt.Sprintf("countries:%d:cities:v1", countryID)

	if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
		var out []map[string]any
		if err := json.Unmarshal([]byte(cached), &out); err == nil {
			return out, nil
		}
	}

	rows, err := s.q.ListCitiesByCountryID(ctx, countryID)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "Cities not found", Status: 404}
	}

	raw, _ := json.Marshal(rows)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)

	_ = s.rdb.Set(ctx, key, raw, 24*time.Hour).Err()
	return out, nil
}
