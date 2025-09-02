package validation

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
)

// ========== CREATE USER ==========

func ValidateCreateUserRequest(req *request.CreateUserRequest) error {
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

	// Password: required, policy
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

	// Country: required, exactly 2 chars
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

	// // Status: required, enum
	// if v := trim(req.Status); v == "" {
	// 	errs = append(errs, "status is required")
	// } else {
	// 	switch v {
	// 	case string(models.UserStatusActive),
	// 		string(models.UserStatusInactive),
	// 		string(models.UserStatusSuspended),
	// 		string(models.UserStatusDeleted),
	// 		string(models.UserStatusDraft):
	// 		// ok
	// 	default:
	// 		errs = append(errs, "status must be one of: active, inactive, suspended, deleted, draft")
	// 	}
	// }

	// Language: required, exactly 2
	if v := trim(req.Language); v == "" {
		errs = append(errs, "language is required")
	} else {
		exact("language", v, 2)
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

// ========== UPDATE USER (partial) ==========

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

	// Email: optional, format
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

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

// ========== LOGIN ==========

func ValidateLoginRequest(req request.LoginRequest) error {
	var errs []string

	if strings.TrimSpace(req.Username) == "" {
		errs = append(errs, "username/email is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		errs = append(errs, "password is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

// ========== Helpers ==========

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

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

func ValidateCreateCargoOffer(req *request.CreateCargoOfferRequest) error {
	var errs []string
	add := func(s string) { errs = append(errs, s) }

	// ids
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

	// times
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

	// enums
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

	// weight/volume
	if req.WeightT <= 0 {
		add("weight_t must be > 0")
	}
	if req.VolumeM3 != nil && *req.VolumeM3 < 0 {
		add("volume_m3 must be >= 0")
	}
	if req.Pallets != nil && *req.Pallets < 0 {
		add("pallets must be >= 0")
	}

	// temperature (ako se koristi)
	if req.TemperatureMinC != nil && req.TemperatureMaxC != nil {
		if *req.TemperatureMaxC < *req.TemperatureMinC {
			add("temperature_max_c must be >= temperature_min_c")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}
	return nil
}

func ValidateCreateTruckAvailability(req *request.CreateTruckAvailabilityRequest) error {
	var errs []string

	if req.StartCountryID <= 0 || req.StartCityID <= 0 {
		errs = append(errs, "start country/city must be provided")
	}

	rt := strings.ToLower(strings.TrimSpace(req.TruckType))
	switch rt {
	case "refrigerator", "curtain", "box", "flatbed", "tanker", "container", "other":
	default:
		errs = append(errs, "invalid truck_type")
	}

	if req.MaxWeightT <= 0 {
		errs = append(errs, "max_weight_t must be > 0")
	}

	// times
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableFrom)); err != nil {
		errs = append(errs, "available_from must be RFC3339")
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(req.AvailableTo)); err != nil {
		errs = append(errs, "available_to must be RFC3339")
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}
