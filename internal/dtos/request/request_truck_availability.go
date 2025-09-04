package request

// CreateTruckAvailabilityRequest is the payload for creating a new truck availability entry.
// Notes:
//   - All timestamps are RFC3339 strings (e.g. "2025-09-02T08:00:00Z").
//   - Weight is in metric tons; volume is in cubic meters.
//   - Optional fields use pointers so callers can omit them (as opposed to sending zero values).
type CreateTruckAvailabilityRequest struct {
	// Start (required) and optional end location (references your DB IDs).
	StartCountryID int32  `json:"start_country_id"`
	StartCityID    int32  `json:"start_city_id"`
	EndCountryID   *int32 `json:"end_country_id"`
	EndCityID      *int32 `json:"end_city_id"`

	// Availability window (RFC3339). Service layer should validate From <= To.
	AvailableFrom string `json:"available_from"`
	AvailableTo   string `json:"available_to"`

	// Vehicle/type & capacity configuration.
	TruckType   string   `json:"truck_type"`    // e.g. "refrigerator", "curtain", ...
	MaxWeightT  float64  `json:"max_weight_t"`  // metric tons
	MaxVolumeM3 *float64 `json:"max_volume_m3"` // cubic meters (optional)

	// Load modes and places (>= 1 where applicable).
	FullLoad        bool  `json:"full_load"`    // FTL
	PartialLoad     bool  `json:"partial_load"` // LTL
	LoadingPlaces   int32 `json:"loading_places"`
	UnloadingPlaces int32 `json:"unloading_places"`

	// Commercial data and meta.
	ExpiresAt  *string  `json:"expires_at"`   // RFC3339 (optional)
	PricePerKm *float64 `json:"price_per_km"` // optional
	Currency   *string  `json:"currency"`     // default "EUR" (if omitted upstream)
	Notes      *string  `json:"notes"`        // free-form notes
}

// ListTruckAvailabilityQuery contains optional filters for listing availabilities.
// Any nil field is ignored by the filter logic.
type ListTruckAvailabilityQuery struct {
	// Location filters.
	StartCountryID *int32 `json:"start_country_id"`
	StartCityID    *int32 `json:"start_city_id"`
	EndCountryID   *int32 `json:"end_country_id"`
	EndCityID      *int32 `json:"end_city_id"`

	// Type/status filters (validated upstream).
	TruckType   *string `json:"truck_type"`
	Status      *string `json:"status"` // e.g. "published", "draft", "closed", ...
	FullLoad    *bool   `json:"full_load"`
	PartialLoad *bool   `json:"partial_load"`

	// Time range filters (RFC3339).
	AvailableFrom *string `json:"available_from"`
	AvailableTo   *string `json:"available_to"`

	// Pagination (Page starts at 1; Limit typically capped at 100 in service/DAO).
	Page  *int32 `json:"page"`
	Limit *int32 `json:"limit"`
}

// UpdateTruckAvailabilityStatusRequest updates the lifecycle status of a
// truck availability entry. Allowed values are enforced in the service layer.
type UpdateTruckAvailabilityStatusRequest struct {
	Status string `json:"status"`
}
