package models

// UserStatus represents the lifecycle state of a user account.
// Keep values stable; they are persisted and used in API responses.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
	UserStatusDraft     UserStatus = "draft"
)

// User is the outward-facing user representation used by the API layer.
// NOTE:
//   - Password is deliberately excluded from JSON to prevent accidental exposure.
//   - Timestamps are strings (usually RFC3339) because upstream layers may
//     already format them; switching to time.Time could be a breaking change.
//   - The *_at JSON field names match existing API/DB mapping conventions.
type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Password     string     `json:"-"` // never serialize password/hash
	Name         string     `json:"name"`
	Country      string     `json:"country"`
	City         string     `json:"city"`
	LegalAddress string     `json:"legal_address"`
	VatNumber    string     `json:"vat_number"`
	Status       UserStatus `json:"status"`
	Language     string     `json:"language"`
	Created      string     `json:"created_at"`           // RFC3339 expected
	Updated      string     `json:"updated_at"`           // RFC3339 expected
	Deleted      string     `json:"deleted_at,omitempty"` // RFC3339; often empty
}

// City is a minimal city descriptor used in listing endpoints and lookups.
// Latitude/Longitude are strings to preserve exact formatting from the source.
type City struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	CountryID int32  `json:"country_id"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// Country represents a country row with ISO codes.
// Nullable fields are pointers and omitted when absent.
type Country struct {
	ID         int32   `json:"id"`
	Name       string  `json:"name"`
	Code       string  `json:"code"`        // ISO 3166-1 alpha-2
	Alpha3Code string  `json:"alpha3_code"` // ISO 3166-1 alpha-3
	EuMember   *bool   `json:"eu_member,omitempty"`
	Continent  *string `json:"continent,omitempty"`
}
