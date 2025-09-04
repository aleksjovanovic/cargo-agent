package validation

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
)

// Small helper to turn a slice of validation messages into an error (or nil).
func errsToError(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	// Use a consistent separator across the module for easier client parsing.
	return fmt.Errorf(strings.Join(errs, "; "))
}

//
// ========== CREATE USER ==========
//

// ValidateCreateUserRequest performs basic shape and policy checks for a signup payload.
// It does not check uniqueness (username/email) or any DB-related constraints.
func ValidateCreateUserRequest(req *request.CreateUserRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	var errs []string

	trim := func(s string) string { return strings.TrimSpace(s) }
	minMax := func(field, v string, min, max int) {
		if l := len(v); l < min || l > max {
			errs = append(errs, fmt.Sprintf("%s must be between %d and %d characters long", field, min, max))
		}
	}
	exact := func(field, v string, n int) {
		if len(v) != n {
			errs = append(errs, fmt.Sprintf("%s must be exactly %d characters long", field, n))
		}
	}

	// Username: required, 3-150
	if v := trim(req.Username); v == "" {
		errs = append(errs, "username is required")
	} else {
		minMax("username", v, 3, 150)
	}

	// Email: required + basic format
	if v := trim(req.Email); v == "" {
		errs = append(errs, "email is required")
	} else if !isEmailValidBasic(v) {
		errs = append(errs, "email must be in a valid format (e.g. name@example.com)")
	}

	// Password: required, policy (length + at least 1 upper, lower, digit, special)
	if v := trim(req.Password); v == "" {
		errs = append(errs, "password is required")
	} else {
		if len(v) < 8 {
			errs = append(errs, "password must be at least 8 characters long")
		}
		var up, lo, di, sp bool
		for _, c := range v {
			switch {
			case unicode.IsUpper(c):
				up = true
			case unicode.IsLower(c):
				lo = true
			case unicode.IsDigit(c):
				di = true
			case unicode.IsPunct(c) || unicode.IsSymbol(c):
				sp = true
			}
		}
		if !up {
			errs = append(errs, "password must contain at least one uppercase letter")
		}
		if !lo {
			errs = append(errs, "password must contain at least one lowercase letter")
		}
		if !di {
			errs = append(errs, "password must contain at least one digit")
		}
		if !sp {
			errs = append(errs, "password must contain at least one special character")
		}
	}

	// Name: required, 3-150
	if v := trim(req.Name); v == "" {
		errs = append(errs, "name is required")
	} else {
		minMax("name", v, 3, 150)
	}

	// Country: required, exactly 2 chars (ISO-3166 alpha-2 expected by API)
	if v := trim(req.Country); v == "" {
		errs = append(errs, "country is required")
	} else {
		exact("country", v, 2)
	}

	// City: required, 3-100
	if v := trim(req.City); v == "" {
		errs = append(errs, "city is required")
	} else {
		minMax("city", v, 3, 100)
	}

	// LegalAddress: required, 3-100
	if v := trim(req.LegalAddress); v == "" {
		errs = append(errs, "legal_address is required")
	} else {
		minMax("legal_address", v, 3, 100)
	}

	// VatNumber: required, 3-30
	if v := trim(req.VatNumber); v == "" {
		errs = append(errs, "vat_number is required")
	} else {
		minMax("vat_number", v, 3, 30)
	}

	// Language: required, exactly 2 (ISO-639-1)
	if v := trim(req.Language); v == "" {
		errs = append(errs, "language is required")
	} else {
		exact("language", v, 2)
	}

	return errsToError(errs)
}

//
// ========== UPDATE USER (partial) ==========
//

// ValidateUpdateUserProfileRequest validates a partial profile update.
// Only provided fields are validated; missing fields are ignored.
func ValidateUpdateUserProfileRequest(req *request.UpdateUserProfileRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	var errs []string

	trim := func(p *string) (string, bool) {
		if p == nil {
			return "", false
		}
		return strings.TrimSpace(*p), true
	}
	minMax := func(field, v string, min, max int) {
		if l := len(v); l < min || l > max {
			errs = append(errs, fmt.Sprintf("%s must be between %d and %d characters long", field, min, max))
		}
	}
	exact := func(field, v string, n int) {
		if len(v) != n {
			errs = append(errs, fmt.Sprintf("%s must be exactly %d characters long", field, n))
		}
	}

	// Username: optional, non-empty, 3-150
	if v, ok := trim(req.Username); ok {
		if v == "" {
			errs = append(errs, "username cannot be empty")
		} else {
			minMax("username", v, 3, 150)
		}
	}

	// Email: optional, must be valid if present
	if v, ok := trim(req.Email); ok {
		if v == "" {
			errs = append(errs, "email cannot be empty")
		} else if !isEmailValidBasic(v) {
			errs = append(errs, "email must be in a valid format (e.g. name@example.com)")
		}
	}

	// Name: optional, 3-150
	if v, ok := trim(req.Name); ok {
		if v == "" {
			errs = append(errs, "name cannot be empty")
		} else {
			minMax("name", v, 3, 150)
		}
	}

	// Country: optional, exactly 2
	if v, ok := trim(req.Country); ok {
		if v == "" {
			errs = append(errs, "country cannot be empty")
		} else {
			exact("country", v, 2)
		}
	}

	// City: optional, 3-100
	if v, ok := trim(req.City); ok {
		if v == "" {
			errs = append(errs, "city cannot be empty")
		} else {
			minMax("city", v, 3, 100)
		}
	}

	// LegalAddress: optional, 3-100
	if v, ok := trim(req.LegalAddress); ok {
		if v == "" {
			errs = append(errs, "legal_address cannot be empty")
		} else {
			minMax("legal_address", v, 3, 100)
		}
	}

	// VatNumber: optional, 3-30
	if v, ok := trim(req.VatNumber); ok {
		if v == "" {
			errs = append(errs, "vat_number cannot be empty")
		} else {
			minMax("vat_number", v, 3, 30)
		}
	}

	// Language: optional, exactly 2
	if v, ok := trim(req.Language); ok {
		if v == "" {
			errs = append(errs, "language cannot be empty")
		} else {
			exact("language", v, 2)
		}
	}

	return errsToError(errs)
}

//
// ========== LOGIN ==========
//

// ValidateLoginRequest ensures username/email and password are present.
// It does not authenticate the user.
func ValidateLoginRequest(req request.LoginRequest) error {
	var errs []string

	if strings.TrimSpace(req.Username) == "" {
		errs = append(errs, "username/email is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		errs = append(errs, "password is required")
	}

	return errsToError(errs)
}

//
// ========== PASSWORD CHANGE ==========
//

// ValidateChangePasswordRequest enforces a basic password policy for new passwords.
func ValidateChangePasswordRequest(newPassword string) error {
	var errs []string

	v := strings.TrimSpace(newPassword)
	if v == "" {
		errs = append(errs, "new password is required")
	} else {
		if len(v) < 8 {
			errs = append(errs, "new password must be at least 8 characters long")
		}
		var up, lo, di, sp bool
		for _, c := range v {
			switch {
			case unicode.IsUpper(c):
				up = true
			case unicode.IsLower(c):
				lo = true
			case unicode.IsDigit(c):
				di = true
			case unicode.IsPunct(c) || unicode.IsSymbol(c):
				sp = true
			}
		}
		if !up {
			errs = append(errs, "new password must contain at least one uppercase letter")
		}
		if !lo {
			errs = append(errs, "new password must contain at least one lowercase letter")
		}
		if !di {
			errs = append(errs, "new password must contain at least one digit")
		}
		if !sp {
			errs = append(errs, "new password must contain at least one special character")
		}
	}

	return errsToError(errs)
}

//
// ========== CARGO OFFER ==========
//

// ValidateCreateCargoOffer validates fields for creating a cargo offer.
// It checks ID presence, time ordering, enums, and basic numeric constraints.
func ValidateCreateCargoOffer(req *request.CreateCargoOfferRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	var errs []string
	add := func(s string) { errs = append(errs, s) }

	// Required IDs
	if req.OriginCountryID <= 0 || req.OriginCityID <= 0 {
		add("origin_country_id and origin_city_id must be > 0")
	}
	if req.DestinationCountryID <= 0 || req.DestinationCityID <= 0 {
		add("destination_country_id and destination_city_id must be > 0")
	}
	if req.LoadingPlaces <= 0 {
		add("loading_places must be >= 1")
	}
	if req.UnloadingPlaces <= 0 {
		add("unloading_places must be >= 1")
	}

	// Time window: RFC3339 and delivery after ready
	rtl, err1 := time.Parse(time.RFC3339, strings.TrimSpace(req.ReadyToLoadBy))
	ddl, err2 := time.Parse(time.RFC3339, strings.TrimSpace(req.DeliveryDeadline))
	if err1 != nil {
		add("ready_to_load_by must be RFC3339")
	}
	if err2 != nil {
		add("delivery_deadline must be RFC3339")
	}
	if err1 == nil && err2 == nil && !ddl.After(rtl) {
		add("delivery_deadline must be after ready_to_load_by")
	}

	// Enums
	switch strings.ToLower(strings.TrimSpace(req.LoadType)) {
	case "ftl", "ltl":
	default:
		add("load_type must be ftl or ltl")
	}
	switch strings.ToLower(strings.TrimSpace(req.TruckType)) {
	case "refrigerator", "curtain", "box", "flatbed", "tanker", "container", "other":
	default:
		add("truck_type invalid")
	}

	// Weight/volume
	if req.WeightT <= 0 {
		add("weight_t must be > 0")
	}
	if req.VolumeM3 != nil && *req.VolumeM3 < 0 {
		add("volume_m3 must be >= 0")
	}
	if req.Pallets != nil && *req.Pallets < 0 {
		add("pallets must be >= 0")
	}

	// Temperature (if both provided, ensure min <= max)
	if req.TemperatureMinC != nil && req.TemperatureMaxC != nil {
		if *req.TemperatureMaxC < *req.TemperatureMinC {
			add("temperature_max_c must be >= temperature_min_c")
		}
	}

	return errsToError(errs)
}

//
// ========== TRUCK AVAILABILITY ==========
//

// ValidateCreateTruckAvailability performs basic checks for creating a truck availability entry.
func ValidateCreateTruckAvailability(req *request.CreateTruckAvailabilityRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	var errs []string

	// Required start location
	if req.StartCountryID <= 0 || req.StartCityID <= 0 {
		errs = append(errs, "start country/city must be provided")
	}

	// Truck type enum
	switch strings.ToLower(strings.TrimSpace(req.TruckType)) {
	case "refrigerator", "curtain", "box", "flatbed", "tanker", "container", "other":
	default:
		errs = append(errs, "invalid truck_type")
	}

	// Capacity
	if req.MaxWeightT <= 0 {
		errs = append(errs, "max_weight_t must be > 0")
	}

	// Time range
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableFrom)); err != nil {
		errs = append(errs, "available_from must be RFC3339")
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableTo)); err != nil {
		errs = append(errs, "available_to must be RFC3339")
	}

	return errsToError(errs)
}

//
// ========== Helpers ==========
//

// isEmailValidBasic performs a pragmatic email validation without full RFC compliance.
// It rejects whitespace, enforces a single "@", non-empty local/domain parts,
// reasonable length limits, and a basic character allowlist.
func isEmailValidBasic(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	if strings.ContainsAny(email, " \t\r\n") {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local, domain := parts[0], parts[1]
	if local == "" || domain == "" {
		return false
	}

	if len(email) > 254 || len(local) > 64 || len(domain) > 253 {
		return false
	}
	dsegs := strings.Split(domain, ".")
	if len(dsegs) < 2 {
		return false
	}
	for _, seg := range dsegs {
		if seg == "" {
			return false
		}
	}

	for _, r := range local {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) ||
			r == '.' || r == '_' || r == '%' || r == '+' || r == '-') {
			return false
		}
	}
	for _, r := range domain {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-') {
			return false
		}
	}
	return true
}
