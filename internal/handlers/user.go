package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
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

// user profile
func (h *Handler) UserProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"unauthorized",
				"Please log in to continue",
				nil,
			)
			return
		}

		userID := claims.UserID

		user, err := h.Queries.GetUser(r.Context(), int32(userID))
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusNotFound,
				"not_found",
				"User not found",
				nil,
			)
		}
		response.RespondWithSuccess(
			w,
			http.StatusOK,
			response.Envelope{
				"message": "success",
				"data":    user,
			},
		)
	}
}

// Login a user
func (h *Handler) LoginUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req request.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid_payload",
				"Invalid request payload",
				nil,
			)
			return
		}

		if err := validation.ValidateLoginRequest(req); err != nil {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"validation_error",
				err.Error(),
				nil)
			return
		}

		// Fetch the user from database using the store queries
		user, err := h.Queries.GetUserByUsernameOrEmail(ctx, req.Username)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusUnauthorized,
				"invalid_credentials",
				"Invalid credentials",
				nil,
			)
			return
		}
		if !utils.ComparePassword(user.Password, req.Password) {
			response.RespondWithError(
				w,
				http.StatusUnauthorized,
				"invalid_credentials",
				"Invalid credentials",
				nil,
			)
			return
		}
		jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
		token, err := authn.GenerateJWT(int64(user.ID), user.Username, jwtKey)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusInternalServerError,
				"token_generation_error",
				"Error generating a token",
				nil,
			)
			return
		}
		response.RespondWithSuccess(
			w,
			http.StatusOK,
			response.Envelope{
				"message": "Login successful",
				"data": map[string]string{
					"token": token,
				},
			},
		)
	}
}

// CreateUserHandler with Transactions
func (h *Handler) CreateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now()

		// User request (dataTransportObject)
		var req request.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid_payload",
				"Invalid request payload",
				nil,
			)
			return
		}

		// Validate the request
		if err := validation.ValidateCreateUserRequest(&req); err != nil {
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"bad_request",
				err.Error(),
				nil,
			)
			return
		}

		// Start a transaction
		tx, err := h.DB.BeginTx(ctx, nil)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusInternalServerError,
				"transaction_start_failed",
				"Failed to start transaction",
				nil,
			)
			return
		}
		defer tx.Rollback()

		// Create a new Queries instance bound to the transaction
		qtx := store.New(tx)

		// Check if the username already exists
		_, err = qtx.GetUserByUsernameOrEmail(ctx, req.Username)
		if err == nil {
			response.RespondWithError(
				w,
				http.StatusConflict,
				"username_exists",
				"Username already exists",
				nil,
			)
			return
		}

		// Check if the email already exists
		_, err = qtx.GetUserByUsernameOrEmail(ctx, req.Email)
		if err == nil {
			response.RespondWithError(
				w,
				http.StatusConflict,
				"email_exists",
				"Email already exists",
				nil,
			)

			return
		}

		// Hash password
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			response.RespondWithError(
				w,
				http.StatusInternalServerError,
				"password_hash_failed",
				"Failed to hash password",
				nil,
			)
			return
		}

		// Create user within the transaction
		newUser, err := qtx.CreateUser(ctx, store.CreateUserParams{
			Username:     req.Username,
			Email:        req.Email,
			Password:     hashedPassword,
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
			response.RespondWithError(
				w,
				http.StatusInternalServerError,
				"user_creation_failed",
				"Failed to create user",
				nil,
			)
			return
		}

		// Commit the transaction if all operations succeed
		if err := tx.Commit(); err != nil {
			response.RespondWithError(
				w,
				http.StatusInternalServerError,
				"transaction_commit_failed",
				"Failed to commit transaction",
				nil,
			)
			return
		}

		response.RespondWithSuccess(
			w,
			http.StatusCreated,
			response.Envelope{
				"message": "User created successfully",
				"data":    newUser,
			},
		)

	}
}
