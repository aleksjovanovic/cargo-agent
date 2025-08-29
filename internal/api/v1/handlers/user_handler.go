package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/models"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
	"github.com/aleksjovanovic/cargo-agent/internal/validation"
)

// ===========================
// Helpers
// ===========================

func decodeOr400(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := utils.DecodeJSONBody(w, r, dst, 1<<20); err != nil {
		if je, ok := err.(*utils.JSONError); ok {
			response.RespondWithError(w, je.Status, "invalid_payload", je.Msg, nil)
		} else {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_payload", "Invalid request payload", nil)
		}
		return false
	}
	return true
}

// ===========================
// User Profile (GET /profile)
// ===========================

func (h *Handler) UserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		userID := claims.UserID

		// Redis cache first
		cacheKey := fmt.Sprintf("user:%d", userID)
		if cached, err := h.Redis.Get(r.Context(), cacheKey).Result(); err == nil {
			var user store.User
			if err := json.Unmarshal([]byte(cached), &user); err == nil {
				response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
					"message": "success (from cache/redis)",
					"data":    user,
				})
				return
			}
		}

		// DB fallback
		user, err := h.Queries.GetUser(r.Context(), int32(userID))
		if err != nil {
			response.RespondWithError(w, http.StatusNotFound, "not_found", "User not found", nil)
			return
		}

		// Set cache
		if b, err := json.Marshal(user); err == nil {
			_ = h.Redis.Set(r.Context(), cacheKey, b, 1*time.Hour).Err()
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "success",
			"data":    user,
		})
	}
}

// ===========================
// Change Password (PUT /me/password)
// ===========================

func (h *Handler) ChangePasswordHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now()

		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var req request.ChangePasswordRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		// Fetch existing hash
		passwordHash, err := h.Queries.GetUserPassword(ctx, int32(claims.UserID))
		if err != nil {
			response.RespondWithError(w, http.StatusNotFound, "not_found", "User not found", nil)
			return
		}

		// Old must match
		if !utils.ComparePassword(passwordHash, req.OldPassword) {
			response.RespondWithError(w, http.StatusUnauthorized, "invalid_old_password", "Old password is incorrect", nil)
			return
		}
		// New must differ
		if utils.ComparePassword(passwordHash, req.NewPassword) {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_new_password", "New password must be different from the old password", nil)
			return
		}

		// Validate new password policy
		if err := validation.ValidateChangePasswordRequest(req.NewPassword); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}

		// Hash new
		hashed, err := utils.HashPassword(req.NewPassword)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "password_hash_failed", "Failed to hash new password", nil)
			return
		}

		// Persist
		if err := h.Queries.ChangePassword(ctx, store.ChangePasswordParams{
			ID:       int32(claims.UserID),
			Password: hashed,
			Updated:  sql.NullTime{Time: now, Valid: true},
		}); err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "pasword_update_failed", "Failed to change password", nil)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Password updated successfully",
			"data":    nil,
		})
	}
}

// ===========================
// Login (POST /login)
// ===========================

func (h *Handler) LoginUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req request.LoginRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		if err := validation.ValidateLoginRequest(req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}

		user, err := h.Queries.GetUserByUsernameOrEmail(ctx, req.Username)
		if err != nil || !utils.ComparePassword(user.Password, req.Password) {
			response.RespondWithError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid credentials", nil)
			return
		}

		jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))

		opt := authn.Options{
			Issuer:        "cargo-agent",
			Audience:      []string{"cargo-agent-gui"},
			TTL:           time.Hour * 1,
			NotBeforeSkew: 0,
		}

		token, err := authn.GenerateJWT(int64(user.ID), user.Username, jwtKey, opt)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "token_generation_error", "Error generating a token", nil)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Login successful",
			"data": map[string]string{
				"token": token,
			},
		})
	}
}

// ===========================
// Signup (POST /signup)
// ===========================

func (h *Handler) CreateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now()

		var req request.CreateUserRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		if err := validation.ValidateCreateUserRequest(&req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}

		tx, err := h.DB.BeginTx(ctx, nil)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "transaction_start_failed", "Failed to start transaction", nil)
			return
		}
		defer tx.Rollback()
		qtx := store.New(tx)

		// Uniqueness checks
		if _, err := qtx.GetUserByUsernameOrEmail(ctx, req.Username); err == nil {
			response.RespondWithError(w, http.StatusConflict, "username_exists", "Username already exists", nil)
			return
		}
		if _, err := qtx.GetUserByUsernameOrEmail(ctx, req.Email); err == nil {
			response.RespondWithError(w, http.StatusConflict, "email_exists", "Email already exists", nil)
			return
		}

		hashed, err := utils.HashPassword(req.Password)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "password_hash_failed", "Failed to hash password", nil)
			return
		}

		newUser, err := qtx.CreateUser(ctx, store.CreateUserParams{
			Username:     req.Username,
			Email:        req.Email,
			Password:     hashed,
			Name:         req.Name,
			Country:      req.Country,
			City:         req.City,
			LegalAddress: req.LegalAddress,
			VatNumber:    req.VatNumber,
			Status:       models.UserStatus(req.Status),
			Language:     req.Language,
			Created:      sql.NullTime{Time: now, Valid: true},
			Updated:      sql.NullTime{Time: now, Valid: true},
		})
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "user_creation_failed", "Failed to create user", nil)
			return
		}

		if err := tx.Commit(); err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "transaction_commit_failed", "Failed to commit transaction", nil)
			return
		}

		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "User created successfully",
			"data":    newUser,
		})
	}
}

// ===========================
// Update Profile (PATCH /profile)
// ===========================

func (h *Handler) UpdateUserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now()

		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}
		userID := int32(claims.UserID)

		var req request.UpdateUserProfileRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		// Must have at least one change
		if req.Username == nil && req.Email == nil && req.Name == nil && req.Country == nil &&
			req.City == nil && req.LegalAddress == nil && req.VatNumber == nil && req.Language == nil {
			response.RespondWithError(w, http.StatusBadRequest, "no_changes", "No fields to update.", nil)
			return
		}

		// Validate partial payload
		if err := validation.ValidateUpdateUserProfileRequest(&req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "bad_request", err.Error(), nil)
			return
		}

		tx, err := h.DB.BeginTx(ctx, nil)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "transaction_start_failed", "Failed to start transaction", nil)
			return
		}
		defer tx.Rollback()
		qtx := store.New(tx)

		// Uniqueness checks (ignore same user)
		if req.Username != nil && *req.Username != "" {
			if u, err := qtx.GetUserByUsernameOrEmail(ctx, *req.Username); err == nil && u.ID != userID {
				response.RespondWithError(w, http.StatusConflict, "username_exists", "Username already exists", nil)
				return
			}
		}
		if req.Email != nil && *req.Email != "" {
			if u, err := qtx.GetUserByUsernameOrEmail(ctx, *req.Email); err == nil && u.ID != userID {
				response.RespondWithError(w, http.StatusConflict, "email_exists", "Email already exists", nil)
				return
			}
		}

		updatedUser, err := qtx.UpdateUserProfile(ctx, store.UpdateUserProfileParams{
			ID:           userID,
			Username:     utils.ToNullString(req.Username),
			Email:        utils.ToNullString(req.Email),
			Name:         utils.ToNullString(req.Name),
			Country:      utils.ToNullString(req.Country),
			City:         utils.ToNullString(req.City),
			LegalAddress: utils.ToNullString(req.LegalAddress),
			VatNumber:    utils.ToNullString(req.VatNumber),
			Language:     utils.ToNullString(req.Language),
			Updated:      sql.NullTime{Time: now, Valid: true},
		})
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "user_update_failed", "Failed to update user profile", nil)
			return
		}

		if err := tx.Commit(); err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "transaction_commit_failed", "Failed to commit transaction", nil)
			return
		}

		// Bust cache
		_ = h.Redis.Del(ctx, fmt.Sprintf("user:%d", userID)).Err()

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User profile updated successfully",
			"data":    updatedUser,
		})
	}
}

// ===========================
// Delete (DELETE /{id})
// ===========================

func (h *Handler) DeleteUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now()

		if _, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		id := r.PathValue("id")
		userID, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "invalid_id", "ID must be a valid integer", nil)
			return
		}

		if err := h.Queries.DeleteUser(ctx, store.DeleteUserParams{
			ID:      int32(userID),
			Updated: sql.NullTime{Time: now, Valid: true},
		}); err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, "delete_user_failed", "Failed to delete user", nil)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User deleted successfully",
			"data":    nil,
		})
	}
}
