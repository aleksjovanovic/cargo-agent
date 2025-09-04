package request

// CreateCargoOfferRequest represents the payload used to create a new cargo offer.
// Notes:
//   - All timestamps are expected in RFC3339 format (e.g. "2025-09-02T08:00:00Z").
//   - Weight is in metric tons; volume is in cubic meters.
//   - Optional fields use pointers so that "omitted" can be distinguished from zero values.
type CreateCargoOfferRequest struct {
	// Location: origin and destination (country/city IDs reference your DB)
	OriginCountryID      int32 `json:"origin_country_id"`
	OriginCityID         int32 `json:"origin_city_id"`
	DestinationCountryID int32 `json:"destination_country_id"`
	DestinationCityID    int32 `json:"destination_city_id"`

	// Number of loading/unloading places (>= 1)
	LoadingPlaces   int32 `json:"loading_places"`
	UnloadingPlaces int32 `json:"unloading_places"`

	// Scheduling windows (RFC3339 timestamps)
	ReadyToLoadBy    string  `json:"ready_to_load_by"`  // earliest pickup time
	DeliveryDeadline string  `json:"delivery_deadline"` // latest delivery time
	PublishedAt      *string `json:"published_at,omitempty"`
	ExpiresAt        *string `json:"expires_at,omitempty"`

	// Load and vehicle metadata (free-text enums validated at service layer)
	LoadType  string `json:"load_type"`  // e.g. "ftl" or "ltl"
	TruckType string `json:"truck_type"` // e.g. "refrigerator", "curtain", ...

	// Capacity & packaging
	WeightT    float64  `json:"weight_t"`            // metric tons
	VolumeM3   *float64 `json:"volume_m3,omitempty"` // cubic meters
	Pallets    *int32   `json:"pallets,omitempty"`   // number of pallets if palletized
	Palletized bool     `json:"palletized"`          // whether the cargo is palletized

	// Temperature control (in °C, both optional; service layer may enforce min<=max)
	TemperatureMinC *float64 `json:"temperature_min_c,omitempty"`
	TemperatureMaxC *float64 `json:"temperature_max_c,omitempty"`

	// Commercial terms
	Price    *float64 `json:"price,omitempty"`
	Currency *string  `json:"currency,omitempty"` // e.g. "EUR"

	// Free-form notes
	Notes *string `json:"notes,omitempty"`
}

// ListCargoOffersQuery holds optional filters for listing offers.
// Omitted fields (nil) are ignored by the filtering logic.
type ListCargoOffersQuery struct {
	// Location filters
	OriginCountryID      *int32 `json:"origin_country_id"`
	OriginCityID         *int32 `json:"origin_city_id"`
	DestinationCountryID *int32 `json:"destination_country_id"`
	DestinationCityID    *int32 `json:"destination_city_id"`

	// Type/status filters (free-text enums validated upstream)
	LoadType  *string `json:"load_type"`  // e.g. "ftl", "ltl"
	TruckType *string `json:"truck_type"` // e.g. "refrigerator", "curtain", ...
	Status    *string `json:"status"`     // e.g. "published", "draft", "closed", ...

	// Time-range filters (RFC3339)
	// Use *From/*To pairs to constrain "ready_to_load_by" and "delivery_deadline".
	ReadyFrom    *string `json:"ready_from"`
	ReadyTo      *string `json:"ready_to"`
	DeliveryFrom *string `json:"delivery_from"`
	DeliveryTo   *string `json:"delivery_to"`

	// Pagination (Page starts at 1; Limit usually capped at 100 in service/DAO)
	Page  *int32 `json:"page"`
	Limit *int32 `json:"limit"`
}

// UpdateCargoOfferStatusRequest updates a cargo offer's lifecycle status.
// The actual allowed values are validated in the service layer.
type UpdateCargoOfferStatusRequest struct {
	Status string `json:"status"`
}

// --- Optional helpers (string constants) --------------------------------------
// These constants are not referenced by the structs directly; they are provided
// as documentation aids and can be used by callers to avoid typos. Keeping them
// as plain strings prevents breaking changes to existing code.

const (
	// Load types
	LoadTypeFTL = "ftl"
	LoadTypeLTL = "ltl"

	// Common truck types (extend as needed to match DB/domain)
	TruckTypeRefrigerator = "refrigerator"
	TruckTypeCurtain      = "curtain"
	TruckTypeBox          = "box"
	TruckTypeFlatbed      = "flatbed"
	TruckTypeTanker       = "tanker"
	TruckTypeContainer    = "container"
	TruckTypeOther        = "other"

	// Typical statuses
	StatusPublished = "published"
	StatusDraft     = "draft"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
	StatusClosed    = "closed"
)
