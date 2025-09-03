package request

type CreateCargoOfferRequest struct {
	OriginCountryID      int32 `json:"origin_country_id"`
	OriginCityID         int32 `json:"origin_city_id"`
	DestinationCountryID int32 `json:"destination_country_id"`
	DestinationCityID    int32 `json:"destination_city_id"`

	LoadingPlaces   int32 `json:"loading_places"`
	UnloadingPlaces int32 `json:"unloading_places"`

	ReadyToLoadBy    string `json:"ready_to_load_by"`  // RFC3339s
	DeliveryDeadline string `json:"delivery_deadline"` // RFC3339

	LoadType  string `json:"load_type"`  // "ftl" | "ltl"
	TruckType string `json:"truck_type"` // "refrigerator" | "curtain" | ...

	WeightT    float64  `json:"weight_t"` // u tonama
	VolumeM3   *float64 `json:"volume_m3,omitempty"`
	Pallets    *int32   `json:"pallets,omitempty"`
	Palletized bool     `json:"palletized"`

	TemperatureMinC *float64 `json:"temperature_min_c,omitempty"`
	TemperatureMaxC *float64 `json:"temperature_max_c,omitempty"`

	PublishedAt *string `json:"published_at,omitempty"` // RFC3339 (opcionalno)
	ExpiresAt   *string `json:"expires_at,omitempty"`

	Price    *float64 `json:"price,omitempty"`
	Currency *string  `json:"currency,omitempty"` // default: EUR

	Notes *string `json:"notes,omitempty"`
}

type ListCargoOffersQuery struct {
	OriginCountryID      *int32 `json:"origin_country_id"`
	OriginCityID         *int32 `json:"origin_city_id"`
	DestinationCountryID *int32 `json:"destination_country_id"`
	DestinationCityID    *int32 `json:"destination_city_id"`

	LoadType  *string `json:"load_type"`  // ftl | ltl
	TruckType *string `json:"truck_type"` // refrigerator | curtain | ...
	Status    *string `json:"status"`     // published | draft | closed...

	// RFC3339 stringovi, npr "2025-09-10T00:00:00Z"
	ReadyFrom    *string `json:"ready_from"`
	ReadyTo      *string `json:"ready_to"`
	DeliveryFrom *string `json:"delivery_from"`
	DeliveryTo   *string `json:"delivery_to"`

	// paginacija
	Page  *int32 `json:"page"`  // 1..N
	Limit *int32 `json:"limit"` // default 20, max 100
}

// Update status (published | draft | closed)
type UpdateCargoOfferStatusRequest struct {
	Status string `json:"status"`
}
