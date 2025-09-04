package request

// CreateUserRequest represents the payload to create a new user.
// Service layer should validate uniqueness (username/email) and password strength.
type CreateUserRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	City         string `json:"city"`
	LegalAddress string `json:"legal_address"`
	VatNumber    string `json:"vat_number"`
	Language     string `json:"language"`
}

// UpdateUserProfileRequest is used for partial updates.
// All fields are pointers so "nil" means "do not update this field".
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

// ChangePasswordRequest is the payload for an authenticated password change.
// The service should verify the old password and the strength of the new one.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// LoginRequest is the standard username/password login payload.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PasswordResetRequest is used to start the password reset flow.
// The identifier can be either a username or an email. For security,
// handlers/services should avoid user enumeration in responses.
type PasswordResetRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
}

// PasswordResetConfirmRequest finalizes the password reset flow.
// It contains the token (from email) and the desired new password.
// The service must validate token integrity/expiry and apply the change.
type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
