package request

type CreateUserRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	City         string `json:"city"`
	LegalAddress string `json:"legal_address"`
	VatNumber    string `json:"vat_number"`
	// Status       string `json:"status"`
	Language string `json:"language"`
}

type UpdateUserProfileRequest struct {
	Username     *string `json:"username,omitempty"`
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	Country      *string `json:"country,omitempty"`
	City         *string `json:"city,omitempty"`
	LegalAddress *string `json:"legal_address,omitempty"`
	VatNumber    *string `json:"vat_number,omitempty"`
	Language     *string `json:"language,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CountryRequest struct {
	ID int64 `json:"id"`
}

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
