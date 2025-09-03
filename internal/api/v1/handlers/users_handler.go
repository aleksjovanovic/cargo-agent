package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/middleware"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
)

// GET /users/profile
func (h *Handler) UserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		data, appErr := h.Users.Profile(r.Context(), int32(claims.UserID))
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    data,
		})
	}
}

// PUT /users/me/password
func (h *Handler) ChangePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var req request.ChangePasswordRequest
		if !h.decodeOr400(w, r, &req) {
			return
		}

		if appErr := h.Users.ChangePassword(r.Context(), int32(claims.UserID), req.OldPassword, req.NewPassword); appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Password updated successfully",
			"data":    nil,
		})
	}
}

// POST /users/login
func (h *Handler) LoginUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req request.LoginRequest
		if !h.decodeOr400(w, r, &req) {
			return
		}

		token, appErr := h.Users.Login(r.Context(), req)
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Login successful",
			"data":    map[string]string{"token": token},
		})
	}
}

// POST /users/logout
func (h *Handler) LogoutUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth guard (middleware već validira JWT i puni claims)
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		// Izvuci Bearer token iz header-a (konzistentna 400 poruka)
		authz := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_token", "Authorization: Bearer <token> is required", nil)
			return
		}
		rawToken := strings.TrimSpace(parts[1])

		// Servis: blacklist-uj token do isteka
		if appErr := h.Users.Logout(r.Context(), int32(claims.UserID), rawToken); appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Logout successful",
			"data":    nil,
		})
	}
}

// POST /users/signup
func (h *Handler) CreateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req request.CreateUserRequest
		if !h.decodeOr400(w, r, &req) {
			return
		}

		out, appErr := h.Users.Signup(r.Context(), req)
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		// fire-and-forget (ne blokiramo 201 ako email padne)
		if err := h.Users.SendVerificationEmail(r.Context(), out); err != nil {
			logger.Warn("failed to send verification email", "error", err)
		}

		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "User created successfully",
			"data":    out,
		})
	}
}

// GET /users/verify-email?token=...
func (h *Handler) VerifyEmailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_token", "Token is required", nil)
			return
		}

		if appErr := h.Users.VerifyEmail(r.Context(), token); appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Email verified successfully",
			"data":    nil,
		})
	}
}

// PATCH /users/profile
func (h *Handler) UpdateUserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var req request.UpdateUserProfileRequest
		if !h.decodeOr400(w, r, &req) {
			return
		}

		out, appErr := h.Users.UpdateProfile(r.Context(), int32(claims.UserID), req)
		if appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User profile updated successfully",
			"data":    out,
		})
	}
}

// DELETE /users/{id}
func (h *Handler) DeleteUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(middleware.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		id := r.PathValue("id")
		userID, err := strconv.ParseInt(id, 10, 32)
		if err != nil || userID <= 0 {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		if appErr := h.Users.Delete(r.Context(), int32(userID)); appErr != nil {
			response.WriteAppError(w, appErr)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User deleted successfully",
			"data":    nil,
		})
	}
}
