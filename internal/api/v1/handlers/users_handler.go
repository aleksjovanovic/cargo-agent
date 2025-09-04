// internal/api/v1/handlers/user_handler.go
package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/authz"
	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/middleware"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

// maskToken avoids logging sensitive tokens in full. Keep the prefix/suffix for debugging.
func maskToken(t string) string {
	if len(t) <= 12 {
		return "***"
	}
	return t[:6] + "..." + t[len(t)-6:]
}

// UserProfileHandler returns the authenticated user's profile.
// Requires auth middleware to inject Claims into context.
func (h *Handler) UserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "users.profile"
		rid := ctxmeta.RequestID(r.Context())

		// Read auth claims set by middleware. If missing → 401.
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized access", "op", op, "method", r.Method, "path", r.URL.Path, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		logger.Info("Loading profile", "op", op, "user_id", claims.UserID, "rid", rid)

		data, appErr := h.Users.Profile(r.Context(), int32(claims.UserID))
		if appErr != nil {
			logger.Error("Profile load failed", "op", op, "user_id", claims.UserID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Profile loaded", "op", op, "user_id", claims.UserID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    data,
		})
	}
}

// ChangePasswordHandler updates the password for the authenticated user.
// Validates payload and delegates to service. Does not log plaintext secrets.
func (h *Handler) ChangePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "users.change_password"
		rid := ctxmeta.RequestID(r.Context())

		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized change password", "op", op, "method", r.Method, "path", r.URL.Path, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var reqBody request.ChangePasswordRequest
		// decodeOr400: shared helper that writes 400 if JSON is invalid.
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid change password payload", "op", op, "user_id", claims.UserID, "rid", rid)
			return
		}

		logger.Info("Changing password", "op", op, "user_id", claims.UserID, "rid", rid)

		if appErr := h.Users.ChangePassword(r.Context(), int32(claims.UserID), reqBody.OldPassword, reqBody.NewPassword); appErr != nil {
			logger.Error("Change password failed", "op", op, "user_id", claims.UserID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Password changed", "op", op, "user_id", claims.UserID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Password updated successfully",
			"data":    nil,
		})
	}
}

// PasswordResetRequestHandler starts the password-reset flow using username or email.
// Always returns 200 to prevent user enumeration (do not reveal whether account exists).
func (h *Handler) PasswordResetRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "auth.password_reset.request"
		rid := ctxmeta.RequestID(r.Context())

		var reqBody request.PasswordResetRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("pwdreset.request.bad_payload", "op", op, "rid", rid)
			return
		}

		identifier := strings.TrimSpace(reqBody.UsernameOrEmail)

		// Service encapsulates all privacy/security decisions (e.g., timing, token TTL).
		if err := h.Users.RequestPasswordReset(r.Context(), identifier); err != nil {
			// Do NOT leak whether the identifier exists (security).
			logger.Warn("pwdreset.request.failed", "op", op, "identifier", identifier, "error", err, "rid", rid)
		} else {
			logger.Info("pwdreset.request.ok", "op", op, "identifier", identifier, "rid", rid)
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "If the account exists, a reset email has been sent.",
			"data":    nil,
		})
	}
}

// PasswordResetConfirmHandler completes the reset using a token + new password.
// On error, returns the exact AppError from the service (expired/invalid/etc.).
func (h *Handler) PasswordResetConfirmHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "auth.password_reset.confirm"
		rid := ctxmeta.RequestID(r.Context())

		var reqBody request.PasswordResetConfirmRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("pwdreset.confirm.bad_payload", "op", op, "rid", rid)
			return
		}

		if appErr := h.Users.ResetPassword(r.Context(), strings.TrimSpace(reqBody.Token), reqBody.NewPassword); appErr != nil {
			logger.Warn("pwdreset.confirm.failed", "op", op, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("pwdreset.confirm.ok", "op", op, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Password has been reset successfully.",
			"data":    nil,
		})
	}
}

// LoginUserHandler authenticates a user and returns a JWT on success.
// Never logs plaintext passwords; logs username and request id for traceability.
func (h *Handler) LoginUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "auth.login"
		rid := ctxmeta.RequestID(r.Context())

		var reqBody request.LoginRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid login payload", "op", op, "rid", rid)
			return
		}

		logger.Info("Login attempt", "op", op, "username", reqBody.Username, "rid", rid)

		token, appErr := h.Users.Login(r.Context(), reqBody)
		if appErr != nil {
			logger.Warn("Login failed", "op", op, "username", reqBody.Username, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Login success", "op", op, "username", reqBody.Username, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Login successful",
			"data":    map[string]string{"token": token},
		})
	}
}

// LogoutUserHandler revokes the current JWT (blacklists it until expiry).
// Requires a valid Bearer token and authenticated user context.
func (h *Handler) LogoutUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "auth.logout"
		rid := ctxmeta.RequestID(r.Context())

		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized logout", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		// Pull raw bearer token from header. We don't rely on middleware here because
		// we need the exact raw token to blacklist it (service will parse/validate).
		authz := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			logger.Warn("Logout missing/invalid bearer", "op", op, "user_id", claims.UserID, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_token", "Authorization: Bearer <token> is required", nil)
			return
		}
		rawToken := strings.TrimSpace(parts[1])

		logger.Info("Logout attempt", "op", op, "user_id", claims.UserID, "token", maskToken(rawToken), "rid", rid)

		if appErr := h.Users.Logout(r.Context(), int32(claims.UserID), rawToken); appErr != nil {
			logger.Error("Logout failed", "op", op, "user_id", claims.UserID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Logout success", "op", op, "user_id", claims.UserID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Logout successful",
			"data":    nil,
		})
	}
}

// CreateUserHandler registers a new user and triggers a verification email.
// Purposefully does not include any sensitive data in logs.
func (h *Handler) CreateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "auth.signup"
		rid := ctxmeta.RequestID(r.Context())

		var reqBody request.CreateUserRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid signup payload", "op", op, "rid", rid)
			return
		}

		logger.Info("Signup attempt", "op", op, "username", reqBody.Username, "email", reqBody.Email, "rid", rid)

		out, appErr := h.Users.Signup(r.Context(), reqBody)
		if appErr != nil {
			logger.Error("Signup failed", "op", op, "username", reqBody.Username, "email", reqBody.Email, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		// Fire-and-forget email; failure is not fatal to signup.
		if err := h.Users.SendVerificationEmail(r.Context(), out); err != nil {
			logger.Warn("Failed to send verification email", "op", op, "username", reqBody.Username, "email", reqBody.Email, "error", err, "rid", rid)
		} else {
			logger.Info("Verification email queued", "op", op, "username", reqBody.Username, "email", reqBody.Email, "rid", rid)
		}

		logger.Info("Signup success", "op", op, "username", reqBody.Username, "email", reqBody.Email, "rid", rid)
		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "User created successfully",
			"data":    out,
		})
	}
}

// VerifyEmailHandler confirms a user's email based on a verification token in the query string.
// Returns generic errors without leaking token internals.
func (h *Handler) VerifyEmailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "auth.verify_email"
		rid := ctxmeta.RequestID(r.Context())

		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			logger.Warn("Verify email missing token", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_token", "Token is required", nil)
			return
		}

		logger.Info("Verify email attempt", "op", op, "token", maskToken(token), "rid", rid)

		if appErr := h.Users.VerifyEmail(r.Context(), token); appErr != nil {
			logger.Warn("Verify email failed", "op", op, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Verify email success", "op", op, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Email verified successfully",
			"data":    nil,
		})
	}
}

// UpdateUserProfileHandler partially updates the authenticated user's profile.
// The service validates allowed fields and conflict errors (email/username duplicates).
func (h *Handler) UpdateUserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "users.update_profile"
		rid := ctxmeta.RequestID(r.Context())

		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized update profile", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var reqBody request.UpdateUserProfileRequest
		if !h.decodeOr400(w, r, &reqBody) {
			logger.Warn("Invalid update profile payload", "op", op, "user_id", claims.UserID, "rid", rid)
			return
		}

		logger.Info("Updating profile", "op", op, "user_id", claims.UserID, "rid", rid)

		out, appErr := h.Users.UpdateProfile(r.Context(), int32(claims.UserID), reqBody)
		if appErr != nil {
			logger.Error("Update profile failed", "op", op, "user_id", claims.UserID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("Profile updated", "op", op, "user_id", claims.UserID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User profile updated successfully",
			"data":    out,
		})
	}
}

// DeleteUserHandler performs a soft-delete by id (admin-like).
// Validates id path param and returns standard errors on failure.
func (h *Handler) DeleteUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		op := "users.delete"
		rid := ctxmeta.RequestID(r.Context())

		// Ensure caller is authenticated; authorization decisions are in the service or middleware.
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			logger.Warn("Unauthorized delete user", "op", op, "rid", rid)
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		// Extract path param {id} using Go 1.22+ pattern matching (ServerMux).
		id := r.PathValue("id")
		userID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || userID <= 0 {
			logger.Warn("Delete user invalid id", "op", op, "id", id, "rid", rid)
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		if !authz.IsAdmin(claims.UserID) {
			logger.Warn("Delete user forbidden", "op", op, "actor_id", claims.UserID, "target_user_id", userID, "rid", rid)
			response.RespondWithError(w, http.StatusForbidden, "forbidden", "You are not allowed to perform this action", nil)
			return
		}

		logger.Info("Deleting user", "op", op, "target_user_id", userID, "rid", rid)

		if appErr := h.Users.Delete(r.Context(), int32(userID)); appErr != nil {
			logger.Error("Delete user failed", "op", op, "target_user_id", userID, "code", appErr.Code, "status", appErr.Status, "err", appErr.Message, "rid", rid)
			response.WriteAppError(w, appErr)
			return
		}

		logger.Info("User deleted", "op", op, "target_user_id", userID, "rid", rid)
		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User deleted successfully",
			"data":    nil,
		})
	}
}
