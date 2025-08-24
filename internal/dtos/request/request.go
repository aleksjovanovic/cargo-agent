package request

type CreateUserRequest struct {
	Username     string `json:"username" validate:"required,min=3,max=150"`
	Email        string `json:"email" validate:"required,email"`
	Password     string `json:"password" validate:"required,min=8"`
	Name         string `json:"name" validate:"required,min=3,max=150"`
	Country      string `json:"country" validate:"required,min=2,max=2"`
	City         string `json:"city" validate:"required,min=3,max=100"`
	LegalAddress string `json:"legal_address" validate:"required,min=3,max=100"`
	VatNumber    string `json:"vat_number" validate:"required,min=3,max=30"`
	Status       string `json:"status" validate:"required,min=5,max=10"`
	Language     string `json:"language" validate:"required,min=2,max=2"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=150"`
	Password string `json:"password" validate:"required,min=8"`
}
