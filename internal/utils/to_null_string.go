package utils

import "database/sql"

func ToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *p, Valid: true}
}
