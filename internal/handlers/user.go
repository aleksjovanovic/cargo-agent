package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/errorhandler"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/successresponse"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
	"github.com/aleksjovanovic/cargo-agent/internal/validation"
)

// user profile
func (h *Handler) UserProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			errorhandler.RespondWithError(w, http.StatusBadRequest, "please log in to continue")
			return
		}

		userID := claims.UserID

		user, err := h.Queries.GetUser(r.Context(), int32(userID))
		if err != nil {
			errorhandler.RespondWithError(w, http.StatusNotFound, "user not found")
		}
		successresponse.RespondWithSuccess(w, http.StatusOK, "success", user)
	}
}

// Login a user
func (h *Handler) LoginUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req request.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorhandler.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		// Validate the request
		if err := validation.Validate(&req); err != nil {
			errorhandler.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Fetch the user from database using the store queries
		user, err := h.Queries.GetUserByUsernameOrEmail(ctx, req.Username)
		if err != nil {
			errorhandler.RespondWithError(w, http.StatusUnauthorized, "invalid credential")
			return
		}
		if !utils.ComparePassword(user.Password, req.Password) {
			errorhandler.RespondWithError(w, http.StatusUnauthorized, "invalid credential")
			return
		}
		jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
		token, err := authn.GenerateJWT(int64(user.ID), user.Username, jwtKey)
		if err != nil {
			errorhandler.RespondWithError(w, http.StatusInternalServerError, "error generating a token")
			return
		}
		successresponse.RespondWithSuccess(w, http.StatusOK, "login successful", map[string]string{
			"token": token,
		})
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
			errorhandler.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
			return
		}

		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			errorhandler.RespondWithError(w, http.StatusInternalServerError, "error while hashing password")
			return
		}
		// Validate the request
		if err := validation.Validate(&req); err != nil {
			errorhandler.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Start a transaction
		tx, err := h.DB.BeginTx(ctx, nil)
		if err != nil {
			errorhandler.RespondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
			return
		}
		defer tx.Rollback()

		// Create a new Queries instance bound to the transaction
		_, err = h.Queries.CreateUser(ctx, store.CreateUserParams{
			Username:     req.Username,
			Email:        req.Email,
			Password:     hashedPassword,
			Name:         req.Name,
			Country:      req.Country,
			City:         req.City,
			LegalAddress: req.LegalAddress,
			VatNumber:    req.VatNumber,
			Status:       req.Status,
			Language:     req.Language,
			Created:      sql.NullTime{Time: now, Valid: true},
			Updated:      sql.NullTime{Time: now, Valid: true},
		})
		if err != nil {
			errorhandler.RespondWithError(w, http.StatusInternalServerError, "error while creating user")
			return
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			errorhandler.RespondWithError(w, http.StatusInternalServerError, "failed to commit transaction")
			return
		}

		successresponse.RespondWithSuccess(w, http.StatusCreated, "user created", req.Name)

	}
}
