package handlers

import (
	"net/http"
	"strconv"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
)

// pomoćni decoder (uniform 400)
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

// GET /users/profile
func (h *Handler) UserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		data, err := h.Users.Profile(r.Context(), int32(claims.UserID))
		if err != nil {
			response.RespondWithError(w, err.Status, err.Code, err.Message, err.Details)
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
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var req request.ChangePasswordRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		if err := h.Users.ChangePassword(r.Context(), int32(claims.UserID), req.OldPassword, req.NewPassword); err != nil {
			response.RespondWithError(w, err.Status, err.Code, err.Message, err.Details)
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
		if !decodeOr400(w, r, &req) {
			return
		}

		token, appErr := h.Users.Login(r.Context(), req)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Login successful",
			"data":    map[string]string{"token": token},
		})
	}
}

// POST /users/signup
func (h *Handler) CreateUserHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req request.CreateUserRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		out, appErr := h.Users.Signup(r.Context(), req)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusCreated, response.Envelope{
			"message": "User created successfully",
			"data":    out,
		})
	}
}

// PATCH /users/profile
func (h *Handler) UpdateUserProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(middlewares.UserClaimsKey).(*authn.Claims)
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized", "Please log in to continue", nil)
			return
		}

		var req request.UpdateUserProfileRequest
		if !decodeOr400(w, r, &req) {
			return
		}

		out, appErr := h.Users.UpdateProfile(r.Context(), int32(claims.UserID), req)
		if appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
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

		if appErr := h.Users.Delete(r.Context(), int32(userID)); appErr != nil {
			response.RespondWithError(w, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
			return
		}

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "User deleted successfully",
			"data":    nil,
		})
	}
}
