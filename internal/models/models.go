package models

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
	UserStatusDraft     UserStatus = "draft"
)

type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Password     string     `json:"password"`
	Name         string     `json:"name"`
	Country      string     `json:"country"`
	City         string     `json:"city"`
	LegalAddress string     `json:"legal_address"`
	VatNumber    string     `json:"vat_number"`
	Status       UserStatus `json:"status"`
	Language     string     `json:"language"`
	Created      string     `json:"created_at"`
	Updated      string     `json:"updated_at"`
	Deleted      string     `json:"deleted_at"`
}

type City struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	CountryID int32  `json:"country_id"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type Country struct {
	ID         int32   `json:"id"`
	Name       string  `json:"name"`
	Code       string  `json:"code"`
	Alpha3Code string  `json:"alpha3_code"`
	EuMember   *bool   `json:"eu_member,omitempty"`
	Continent  *string `json:"continent,omitempty"`
}
