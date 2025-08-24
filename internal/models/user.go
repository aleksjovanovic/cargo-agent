package models

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	City         string `json:"city"`
	LegalAddress string `json:"legal_address"`
	VatNumber    int    `json:"vat_number"`
	Status       string `json:"status"` // active, suspended, deleted, draft
	Language     string `json:"language"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
}
