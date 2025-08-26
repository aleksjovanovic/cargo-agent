package validation

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
)

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

	// Username: required, 3-150 characters
	if v := trim(req.Username); v == "" {
		errs = append(errs, "username is required")
	} else {
		minMax("username", v, 3, 150)
	}

	// Email: required + custom struktura (x@y.z)
	if v := trim(req.Email); v == "" {
		errs = append(errs, "email is is required")
	} else if !isEmailValidBasic(v) {
		errs = append(errs, "email must be in a valid format (e.g. name@example.com)")
	}

	// Password: required, min 8 characters, at least 1 uppercase, 1 lowercase, 1 number, 1 special character
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

	// Name: required, 3-150 characters
	if v := trim(req.Name); v == "" {
		errs = append(errs, "name is required")
	} else {
		minMax("name", v, 3, 150)
	}

	// Country: required, exactly 2 characters
	if v := trim(req.Country); v == "" {
		errs = append(errs, "country is required")
	} else {
		exact("country", v, 2)
	}

	// City: required, 3-100 characters
	if v := trim(req.City); v == "" {
		errs = append(errs, "city is required")
	} else {
		minMax("city", v, 3, 100)
	}

	// LegalAddress: required, 3-100 characters
	if v := trim(req.LegalAddress); v == "" {
		errs = append(errs, "legal_address is required")
	} else {
		minMax("legal_address", v, 3, 100)
	}

	// VatNumber: required, 3-30 characters
	if v := trim(req.VatNumber); v == "" {
		errs = append(errs, "vat_number is required")
	} else {
		minMax("vat_number", v, 3, 30)
	}

	// Status: required, 5-10 characters, enum(active, inactive, suspended, deleted, or draft)
	if v := trim(req.Status); v == "" {
		errs = append(errs, "status jis required")
	} else {
		minMax("status", v, 5, 10)
		switch v {
		case "active", "inactive", "suspended", "deleted", "draft":
			// ok
		default:
			errs = append(errs, "status must be one of the following values: active, inactive, suspended, deleted, or draft")
		}
	}

	// Language: required, exactly 2 characters
	if v := trim(req.Language); v == "" {
		errs = append(errs, "language is required")
	} else {
		exact("language", v, 2)
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}
	return nil
}

func ValidateLoginRequest(req request.LoginRequest) error {
	var errs []string

	// Username/Email required
	if strings.TrimSpace(req.Username) == "" {
		errs = append(errs, "username or email is required")
	}

	// Password required
	if strings.TrimSpace(req.Password) == "" {
		errs = append(errs, "password is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}

	return nil
}

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
