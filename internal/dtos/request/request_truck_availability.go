package request

type CreateTruckAvailabilityRequest struct {
	StartCountryID int32  `json:"start_country_id"`
	StartCityID    int32  `json:"start_city_id"`
	EndCountryID   *int32 `json:"end_country_id"`
	EndCityID      *int32 `json:"end_city_id"`

	AvailableFrom string `json:"available_from"` // RFC3339
	AvailableTo   string `json:"available_to"`   // RFC3339

	TruckType   string   `json:"truck_type"`    // refrigerator | curtain | ...
	MaxWeightT  float64  `json:"max_weight_t"`  // t
	MaxVolumeM3 *float64 `json:"max_volume_m3"` // m3 (optional)

	FullLoad        bool  `json:"full_load"`    // FTL
	PartialLoad     bool  `json:"partial_load"` // LTL
	LoadingPlaces   int32 `json:"loading_places"`
	UnloadingPlaces int32 `json:"unloading_places"`

	ExpiresAt  *string  `json:"expires_at"`   // RFC3339
	PricePerKm *float64 `json:"price_per_km"` // optional
	Currency   *string  `json:"currency"`     // default EUR
	Notes      *string  `json:"notes"`
}

type ListTruckAvailabilityQuery struct {
	StartCountryID *int32 `json:"start_country_id"`
	StartCityID    *int32 `json:"start_city_id"`
	EndCountryID   *int32 `json:"end_country_id"`
	EndCityID      *int32 `json:"end_city_id"`

	TruckType   *string `json:"truck_type"`
	Status      *string `json:"status"` // published | draft | closed
	FullLoad    *bool   `json:"full_load"`
	PartialLoad *bool   `json:"partial_load"`

	AvailableFrom *string `json:"available_from"` // RFC3339
	AvailableTo   *string `json:"available_to"`   // RFC3339

	Page  *int32 `json:"page"`  // default 1
	Limit *int32 `json:"limit"` // default 20, max 100
}

// Update status (published | draft | closed)
type UpdateTruckAvailabilityStatusRequest struct {
	Status string `json:"status"`
}
