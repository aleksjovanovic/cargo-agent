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
	// Status       string `json:"status"`
	Language string `json:"language"`
}

type UpdateUserProfileRequest struct {
	Username     *string `json:"username,omitempty"`
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	Country      *string `json:"country,omitempty"`
	City         *string `json:"city,omitempty"`
	LegalAddress *string `json:"legal_address,omitempty"`
	VatNumber    *string `json:"vat_number,omitempty"`
	Language     *string `json:"language,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
