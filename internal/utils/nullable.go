package utils

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

func ToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *p, Valid: true}
}

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

func SqlNullTimePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func ToNumericString(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func ToNullNumericString(p *float64) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: strconv.FormatFloat(*p, 'f', 2, 64), Valid: true}
}

func ToNullInt32(p *int32) sql.NullInt32 {
	if p == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *p, Valid: true}
}

func ToNullBool(p *bool) sql.NullBool {
	if p == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *p, Valid: true}
}

func ToLowerPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*p))
	return &s
}
