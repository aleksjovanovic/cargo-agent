package utils

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// ToNullString converts a *string to sql.NullString.
// If p is nil, returns an invalid (NULL) value.
// NOTE: This does not trim or normalize the string; use ToLowerNullString if you need trimming/lowercasing.
func ToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *p, Valid: true}
}

// ToLowerNullString converts a *string to a trimmed, lower-cased sql.NullString.
// If p is nil or the trimmed string is empty, returns an invalid (NULL) value.
func ToLowerNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{Valid: false}
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: strings.ToLower(v), Valid: true}
}

// SqlNullTimePtr converts a *time.Time to sql.NullTime.
// If t is nil, returns an invalid (NULL) value.
func SqlNullTimePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// ToNumericString formats a float64 using fixed-point with 2 decimal places (e.g., 12.30).
// Useful for producing a stable textual representation for storage or logs.
func ToNumericString(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// ToNullNumericString converts a *float64 to sql.NullString by formatting with 2 decimals.
// If p is nil, returns an invalid (NULL) value.
func ToNullNumericString(p *float64) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: strconv.FormatFloat(*p, 'f', 2, 64), Valid: true}
}

// ToNullInt32 converts a *int32 to sql.NullInt32.
// If p is nil, returns an invalid (NULL) value.
func ToNullInt32(p *int32) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *p, Valid: true}
}

// ToNullBool converts a *bool to sql.NullBool.
// If p is nil, returns an invalid (NULL) value.
func ToNullBool(p *bool) sql.NullBool {
	if p == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *p, Valid: true}
}

// ToLowerPtr trims and lower-cases the given *string and returns a new pointer.
// If p is nil, returns nil.
func ToLowerPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*p))
	return &s
}
