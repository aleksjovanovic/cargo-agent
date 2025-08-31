package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
	"github.com/aleksjovanovic/cargo-agent/internal/validation"
	"github.com/redis/go-redis/v9"
)

type UserService struct {
	db        *sql.DB
	q         *store.Queries
	rdb       *redis.Client
	jwtSecret []byte
	jwtOpt    authn.Options
}

func NewUserService(db *sql.DB, q *store.Queries, rdb *redis.Client, jwtSecret []byte, jwtOpt authn.Options) *UserService {
	return &UserService{db: db, q: q, rdb: rdb, jwtSecret: jwtSecret, jwtOpt: jwtOpt}
}

// --- Read-only “accessors” za druge delove aplikacije (privremeno rešenje) ---

func (s *UserService) Q() *store.Queries  { return s.q }
func (s *UserService) RDB() *redis.Client { return s.rdb }

// --- Business logika ---

func (s *UserService) Profile(ctx context.Context, userID int32) (map[string]any, *response.AppError) {
	key := cacheKeyUser(userID)

	// cache hit
	if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
		var out map[string]any
		if err := json.Unmarshal([]byte(cached), &out); err == nil {
			return out, nil
		}
	}

	// DB
	row, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return nil, &response.AppError{Code: "not_found", Message: "User not found", Status: 404}
	}

	// canonical JSON (stabilno za klijente)
	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	// cache 1h
	_ = s.rdb.Set(ctx, key, raw, time.Hour).Err()

	return out, nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID int32, oldPass, newPass string) *response.AppError {
	// get hash
	hash, err := s.q.GetUserPassword(ctx, userID)
	if err != nil {
		return &response.AppError{Code: "not_found", Message: "User not found", Status: 404}
	}

	// old must match
	if !utils.ComparePassword(hash, oldPass) {
		return &response.AppError{Code: "invalid_old_password", Message: "Old password is incorrect", Status: 401}
	}

	// new must differ
	if utils.ComparePassword(hash, newPass) {
		return &response.AppError{Code: "invalid_new_password", Message: "New password must be different from the old password", Status: 400}
	}

	// policy
	if err := validation.ValidateChangePasswordRequest(newPass); err != nil {
		return &response.AppError{Code: "bad_request", Message: err.Error(), Status: 400}
	}

	// hash new
	hashed, err := utils.HashPassword(newPass)
	if err != nil {
		return &response.AppError{Code: "password_hash_failed", Message: "Failed to hash new password", Status: 500}
	}

	// persist
	now := time.Now()
	if err := s.q.ChangePassword(ctx, store.ChangePasswordParams{
		ID:        userID,
		Password:  hashed,
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return &response.AppError{Code: "password_update_failed", Message: "Failed to change password", Status: 500}
	}

	// bust cache
	_ = s.rdb.Del(ctx, cacheKeyUser(userID)).Err()
	return nil
}

func (s *UserService) Login(ctx context.Context, req request.LoginRequest) (string, *response.AppError) {
	if err := validation.ValidateLoginRequest(req); err != nil {
		return "", &response.AppError{Code: "validation_error", Message: err.Error(), Status: 400}
	}

	user, err := s.q.GetUserByUsernameOrEmail(ctx, req.Username)
	if err != nil || !utils.ComparePassword(user.Password, req.Password) {
		return "", &response.AppError{Code: "invalid_credentials", Message: "Invalid credentials", Status: 401}
	}

	token, err := authn.GenerateJWT(int64(user.ID), user.Username, s.jwtSecret, s.jwtOpt)
	if err != nil {
		return "", &response.AppError{Code: "token_generation_error", Message: "Error generating a token", Status: 500}
	}
	return token, nil
}

func (s *UserService) Signup(ctx context.Context, req request.CreateUserRequest) (map[string]any, *response.AppError) {
	if err := validation.ValidateCreateUserRequest(&req); err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: 400}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, &response.AppError{Code: "transaction_start_failed", Message: "Failed to start transaction", Status: 500}
	}
	defer tx.Rollback()

	qtx := store.New(tx)

	// uniqueness
	if _, err := qtx.GetUserByUsernameOrEmail(ctx, req.Username); err == nil {
		return nil, &response.AppError{Code: "username_exists", Message: "Username already exists", Status: 409}
	}
	if _, err := qtx.GetUserByUsernameOrEmail(ctx, req.Email); err == nil {
		return nil, &response.AppError{Code: "email_exists", Message: "Email already exists", Status: 409}
	}

	// hash
	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, &response.AppError{Code: "password_hash_failed", Message: "Failed to hash password", Status: 500}
	}

	now := time.Now()

	newUser, err := qtx.CreateUser(ctx, store.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		Password:     hashed,
		Name:         req.Name,
		Country:      req.Country,
		City:         req.City,
		LegalAddress: req.LegalAddress,
		VatNumber:    req.VatNumber,
		// Status:       models.UserStatus(req.Status),
		Language:  req.Language,
		CreatedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return nil, &response.AppError{Code: "user_creation_failed", Message: "Failed to create user", Status: 500}
	}

	if err := tx.Commit(); err != nil {
		return nil, &response.AppError{Code: "transaction_commit_failed", Message: "Failed to commit transaction", Status: 500}
	}

	raw, _ := json.Marshal(newUser)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *UserService) SendVerificationEmail(ctx context.Context, user map[string]any) error {

	// email, ok := user["email"].(string)
	// if !ok {
	// 	return errors.New("invalid email format")
	// }

	userID, ok := user["id"].(float64) // ili string ako je string
	if !ok {
		return errors.New("invalid user ID format")
	}
	username, ok := user["username"].(string) // ili string ako je string
	if !ok {
		return errors.New("invalid username format")
	}

	// Generiši token (može JWT ili UUID)
	token, err := authn.GenerateJWT(int64(userID), username, s.jwtSecret, s.jwtOpt)
	if err != nil {
		return &response.AppError{Code: "token_generation_error", Message: "Error generating a token", Status: 500}
	}
	// Sačuvaj token u bazi ako ne koristiš JWT (npr. tabela "email_verifications")

	// Kreiraj verifikacioni link
	link := fmt.Sprintf("https://tvojfrontend.com/verify?token=%s", token)

	fmt.Println(link)
	// Pošalji email
	// body := fmt.Sprintf("Klikni na sledeći link da verifikuješ nalog: %s", link)
	// return emailer.Send(email, "Verifikuj svoj nalog", body)
	return nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int32, req request.UpdateUserProfileRequest) (map[string]any, *response.AppError) {
	// at least one field
	if req.Username == nil && req.Email == nil && req.Name == nil && req.Country == nil &&
		req.City == nil && req.LegalAddress == nil && req.VatNumber == nil && req.Language == nil {
		return nil, &response.AppError{Code: "no_changes", Message: "No fields to update.", Status: 400}
	}

	if err := validation.ValidateUpdateUserProfileRequest(&req); err != nil {
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: 400}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, &response.AppError{Code: "transaction_start_failed", Message: "Failed to start transaction", Status: 500}
	}
	defer tx.Rollback()
	qtx := store.New(tx)

	// uniqueness ignoring same user
	if req.Username != nil && *req.Username != "" {
		if u, err := qtx.GetUserByUsernameOrEmail(ctx, *req.Username); err == nil && u.ID != userID {
			return nil, &response.AppError{Code: "username_exists", Message: "Username already exists", Status: 409}
		}
	}
	if req.Email != nil && *req.Email != "" {
		if u, err := qtx.GetUserByUsernameOrEmail(ctx, *req.Email); err == nil && u.ID != userID {
			return nil, &response.AppError{Code: "email_exists", Message: "Email already exists", Status: 409}
		}
	}

	now := time.Now()

	res, err := qtx.UpdateUserProfile(ctx, store.UpdateUserProfileParams{
		ID:           userID,
		Username:     utils.ToNullString(req.Username),
		Email:        utils.ToNullString(req.Email),
		Name:         utils.ToNullString(req.Name),
		Country:      utils.ToNullString(req.Country),
		City:         utils.ToNullString(req.City),
		LegalAddress: utils.ToNullString(req.LegalAddress),
		VatNumber:    utils.ToNullString(req.VatNumber),
		Language:     utils.ToNullString(req.Language),
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return nil, &response.AppError{Code: "user_update_failed", Message: "Failed to update user profile", Status: 500}
	}

	if err := tx.Commit(); err != nil {
		return nil, &response.AppError{Code: "transaction_commit_failed", Message: "Failed to commit transaction", Status: 500}
	}

	// bust cache
	_ = s.rdb.Del(ctx, cacheKeyUser(userID)).Err()

	raw, _ := json.Marshal(res)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (s *UserService) Delete(ctx context.Context, userID int32) *response.AppError {
	now := time.Now()
	if err := s.q.DeleteUser(ctx, store.DeleteUserParams{
		ID:        userID,
		DeletedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return &response.AppError{Code: "delete_user_failed", Message: "Failed to delete user", Status: 500}
	}
	_ = s.rdb.Del(ctx, cacheKeyUser(userID)).Err()
	return nil
}

// --- helpers ---

func cacheKeyUser(id int32) string { return fmt.Sprintf("user:%d", id) }
