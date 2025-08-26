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
	Status       string `json:"status"`
	Language     string `json:"language"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
