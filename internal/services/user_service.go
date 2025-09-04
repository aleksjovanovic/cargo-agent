package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/dtos/request"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/mailer"
	"github.com/aleksjovanovic/cargo-agent/internal/models"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/templates"
	"github.com/aleksjovanovic/cargo-agent/internal/utils"
	"github.com/aleksjovanovic/cargo-agent/internal/validation"
	"github.com/redis/go-redis/v9"
)

// UserService encapsulates user-related business logic.
type UserService struct {
	db        *sql.DB
	q         *store.Queries
	rdb       *redis.Client
	jwtSecret []byte
	jwtOpt    authn.Options
	mailer    mailer.Mailer

	mailerOnce sync.Once
	mailerErr  error
}

func NewUserService(db *sql.DB, q *store.Queries, rdb *redis.Client, jwtSecret []byte, jwtOpt authn.Options, m mailer.Mailer) *UserService {
	return &UserService{db: db, q: q, rdb: rdb, jwtSecret: jwtSecret, jwtOpt: jwtOpt, mailer: m}
}

func tokenFP(raw string) string {
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8]) // 16 hex chars fingerprint
}

// Expose internals (temporary accessors).
func (s *UserService) Q() *store.Queries  { return s.q }
func (s *UserService) RDB() *redis.Client { return s.rdb }

// Profile returns a cached user profile if available, otherwise fetches from DB.
func (s *UserService) Profile(ctx context.Context, userID int32) (map[string]any, *response.AppError) {
	key := cacheKeyUser(userID)
	logger.Debug("usersvc.profile.start", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)

	if cached, err := s.rdb.Get(ctx, key).Result(); err == nil && cached != "" {
		var out map[string]any
		if err := json.Unmarshal([]byte(cached), &out); err == nil {
			logger.Debug("usersvc.profile.cache_hit", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
			return out, nil
		}
		logger.Warn("usersvc.profile.cache_corrupt", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
	}

	row, err := s.q.GetUser(ctx, userID)
	if err != nil {
		logger.Warn("usersvc.profile.not_found", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "not_found", Message: "User not found", Status: httpStatusNotFound}
	}

	raw, _ := json.Marshal(row)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	if err := s.rdb.Set(ctx, key, raw, time.Hour).Err(); err != nil {
		logger.Warn("usersvc.profile.cache_set_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
	}

	logger.Info("usersvc.profile.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
	return out, nil
}

// ChangePassword validates and updates the user's password.
func (s *UserService) ChangePassword(ctx context.Context, userID int32, oldPass, newPass string) *response.AppError {
	logger.Info("usersvc.change_password.start", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)

	hash, err := s.q.GetUserPassword(ctx, userID)
	if err != nil {
		logger.Warn("usersvc.change_password.user_not_found", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return &response.AppError{Code: "not_found", Message: "User not found", Status: httpStatusNotFound}
	}

	if !utils.ComparePassword(hash, oldPass) {
		logger.Warn("usersvc.change_password.old_mismatch", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
		return &response.AppError{Code: "invalid_old_password", Message: "Old password is incorrect", Status: httpStatusUnauthorized}
	}
	if utils.ComparePassword(hash, newPass) {
		logger.Warn("usersvc.change_password.same_password", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
		return &response.AppError{Code: "invalid_new_password", Message: "New password must be different from the old password", Status: httpStatusBadRequest}
	}
	if err := validation.ValidateChangePasswordRequest(newPass); err != nil {
		logger.Warn("usersvc.change_password.validation_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return &response.AppError{Code: "bad_request", Message: err.Error(), Status: httpStatusBadRequest}
	}

	hashed, err := utils.HashPassword(newPass)
	if err != nil {
		logger.Error("usersvc.change_password.hash_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return &response.AppError{Code: "password_hash_failed", Message: "Failed to hash new password", Status: httpStatusInternalServerError}
	}

	now := time.Now()
	if err := s.q.ChangePassword(ctx, store.ChangePasswordParams{
		ID:        userID,
		Password:  hashed,
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		logger.Error("usersvc.change_password.persist_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return &response.AppError{Code: "password_update_failed", Message: "Failed to change password", Status: httpStatusInternalServerError}
	}

	if err := s.rdb.Del(ctx, cacheKeyUser(userID)).Err(); err != nil {
		logger.Warn("usersvc.change_password.cache_bust_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
	}
	logger.Info("usersvc.change_password.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
	return nil
}

// RequestPasswordReset creates a reset token (Redis) and sends email.
// Always returns nil for non-existent users to avoid user enumeration.
func (s *UserService) RequestPasswordReset(ctx context.Context, usernameOrEmail string) error {
	usernameOrEmail = strings.TrimSpace(usernameOrEmail)
	if usernameOrEmail == "" {
		return nil
	}

	u, err := s.q.GetUserByUsernameOrEmail(ctx, usernameOrEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("db error: %w", err)
	}

	ttl := time.Hour
	if v := strings.TrimSpace(os.Getenv("PASSWORD_RESET_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}

	token, err := generateVerificationToken(32)
	if err != nil {
		return fmt.Errorf("token generate failed: %w", err)
	}
	key := "pwdreset:" + token
	if err := s.rdb.Set(ctx, key, fmt.Sprintf("%d", u.ID), ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("PASSWORD_RESET_BASE")), "/")
	if base == "" {
		base = "http://localhost:8081/cargo-agent/v1/users/password-reset/confirm"
	}
	link := fmt.Sprintf("%s?token=%s", base, token)

	validUntil := time.Now().Add(ttl)
	subject, htmlBody, textBody, err := templates.RenderPasswordResetEmail(u.Username, link, validUntil)
	if err != nil {
		return fmt.Errorf("render password reset email failed: %w", err)
	}

	s.mailerOnce.Do(func() {
		if s.mailer == nil {
			s.mailer, s.mailerErr = mailer.NewFromEnv()
		}
	})
	if s.mailer == nil {
		return fmt.Errorf("mailer init failed: %w", s.mailerErr)
	}
	if err := s.mailer.Send(u.Email, subject, htmlBody, textBody); err != nil {
		return fmt.Errorf("send password reset email failed: %w", err)
	}
	return nil
}

// ResetPassword validates token, updates password, and cleans up.
func (s *UserService) ResetPassword(ctx context.Context, token, newPass string) *response.AppError {
	token = strings.TrimSpace(token)
	if token == "" {
		return &response.AppError{Code: "invalid_token", Message: "Token is required", Status: httpStatusBadRequest}
	}
	if err := validation.ValidateChangePasswordRequest(newPass); err != nil {
		return &response.AppError{Code: "bad_request", Message: err.Error(), Status: httpStatusBadRequest}
	}

	key := "pwdreset:" + token
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &response.AppError{Code: "invalid_token", Message: "Token not found or expired", Status: httpStatusBadRequest}
		}
		return &response.AppError{Code: "server_error", Message: "Please try again later", Status: httpStatusInternalServerError, Details: err.Error()}
	}

	uid, err := strconv.ParseInt(val, 10, 32)
	if err != nil || uid <= 0 {
		return &response.AppError{Code: "invalid_token", Message: "Invalid token payload", Status: httpStatusBadRequest}
	}

	hashed, err := utils.HashPassword(newPass)
	if err != nil {
		return &response.AppError{Code: "password_hash_failed", Message: "Failed to hash new password", Status: httpStatusInternalServerError}
	}
	now := time.Now()
	if err := s.q.ChangePassword(ctx, store.ChangePasswordParams{
		ID:        int32(uid),
		Password:  hashed,
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return &response.AppError{Code: "password_update_failed", Message: "Failed to change password", Status: httpStatusInternalServerError}
	}

	_ = s.rdb.Del(ctx, key).Err()
	_ = s.rdb.Del(ctx, cacheKeyUser(int32(uid))).Err()
	return nil
}

// Login authenticates the user and returns a JWT.
func (s *UserService) Login(ctx context.Context, req request.LoginRequest) (string, *response.AppError) {
	logger.Info("usersvc.login.start", "req_id", ctxmeta.RequestID(ctx), "username_or_email", req.Username)

	if err := validation.ValidateLoginRequest(req); err != nil {
		logger.Warn("usersvc.login.validation_failed", "req_id", ctxmeta.RequestID(ctx), "username_or_email", req.Username, "error", err)
		return "", &response.AppError{Code: "validation_error", Message: err.Error(), Status: httpStatusBadRequest}
	}

	user, err := s.q.GetUserByUsernameOrEmail(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn("usersvc.login.invalid_credentials", "req_id", ctxmeta.RequestID(ctx), "username_or_email", req.Username)
			return "", &response.AppError{Code: "invalid_credentials", Message: "Invalid credentials", Status: httpStatusUnauthorized}
		}
		logger.Error("usersvc.login.db_error", "req_id", ctxmeta.RequestID(ctx), "username_or_email", req.Username, "error", err)
		return "", &response.AppError{Code: "db_error", Message: "Failed to load user", Status: httpStatusInternalServerError, Details: err.Error()}
	}
	if !utils.ComparePassword(user.Password, req.Password) {
		logger.Warn("usersvc.login.invalid_password", "req_id", ctxmeta.RequestID(ctx), "user_id", user.ID)
		return "", &response.AppError{Code: "invalid_credentials", Message: "Invalid credentials", Status: httpStatusUnauthorized}
	}

	token, err := authn.GenerateJWT(int64(user.ID), user.Username, s.jwtSecret, s.jwtOpt)
	if err != nil {
		logger.Error("usersvc.login.token_generation_error", "req_id", ctxmeta.RequestID(ctx), "user_id", user.ID, "error", err)
		return "", &response.AppError{Code: "token_generation_error", Message: "Error generating a token", Status: httpStatusInternalServerError}
	}
	logger.Info("usersvc.login.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", user.ID)
	return token, nil
}

// Logout blacklists the current JWT until it expires.
func (s *UserService) Logout(ctx context.Context, userID int32, rawToken string) *response.AppError {
	fp := tokenFP(rawToken)
	logger.Info("usersvc.logout.start", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "token_fp", fp)

	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		logger.Warn("usersvc.logout.missing_token", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
		return &response.AppError{Code: "invalid_token", Message: "Bearer token is required", Status: httpStatusBadRequest}
	}

	ttl := 24 * time.Hour
	if exp, ok := authn.ParseExpiryUnsafe(rawToken); ok {
		if d := time.Until(exp); d > 0 {
			ttl = d
		} else {
			logger.Info("usersvc.logout.already_expired", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "token_fp", fp)
			return nil
		}
	} else if v := strings.TrimSpace(os.Getenv("JWT_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}

	sum := sha256.Sum256([]byte(rawToken))
	key := "jwt:blacklist:" + hex.EncodeToString(sum[:])

	if err := s.rdb.Set(ctx, key, fmt.Sprintf("%d", userID), ttl).Err(); err != nil {
		logger.Error("usersvc.logout.redis_set_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "token_fp", fp, "error", err)
		return &response.AppError{Code: "logout_failed", Message: "Failed to revoke token", Status: httpStatusInternalServerError, Details: err.Error()}
	}
	logger.Info("usersvc.logout.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "token_fp", fp, "ttl_sec", int64(ttl.Seconds()))
	return nil
}

// Signup creates a new user with uniqueness checks and returns the created record.
func (s *UserService) Signup(ctx context.Context, req request.CreateUserRequest) (map[string]any, *response.AppError) {
	logger.Info("usersvc.signup.start", "req_id", ctxmeta.RequestID(ctx), "username", req.Username, "email", req.Email)

	if err := validation.ValidateCreateUserRequest(&req); err != nil {
		logger.Warn("usersvc.signup.validation_failed", "req_id", ctxmeta.RequestID(ctx), "error", err)
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: httpStatusBadRequest}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("usersvc.signup.tx_begin_failed", "req_id", ctxmeta.RequestID(ctx), "error", err)
		return nil, &response.AppError{Code: "transaction_start_failed", Message: "Failed to start transaction", Status: httpStatusInternalServerError}
	}
	defer tx.Rollback()

	qtx := store.New(tx)

	if _, err := qtx.GetUserByUsernameOrEmail(ctx, req.Username); err == nil {
		logger.Warn("usersvc.signup.username_exists", "req_id", ctxmeta.RequestID(ctx), "username", req.Username)
		return nil, &response.AppError{Code: "username_exists", Message: "Username already exists", Status: httpStatusConflict}
	}
	if _, err := qtx.GetUserByUsernameOrEmail(ctx, req.Email); err == nil {
		logger.Warn("usersvc.signup.email_exists", "req_id", ctxmeta.RequestID(ctx), "email", req.Email)
		return nil, &response.AppError{Code: "email_exists", Message: "Email already exists", Status: httpStatusConflict}
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		logger.Error("usersvc.signup.hash_failed", "req_id", ctxmeta.RequestID(ctx), "error", err)
		return nil, &response.AppError{Code: "password_hash_failed", Message: "Failed to hash password", Status: httpStatusInternalServerError}
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
		Language:     req.Language,
		CreatedAt:    sql.NullTime{Time: now, Valid: true},
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		logger.Error("usersvc.signup.create_failed", "req_id", ctxmeta.RequestID(ctx), "username", req.Username, "email", req.Email, "error", err)
		return nil, &response.AppError{Code: "user_creation_failed", Message: "Failed to create user", Status: httpStatusInternalServerError}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("usersvc.signup.tx_commit_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", newUser.ID, "error", err)
		return nil, &response.AppError{Code: "transaction_commit_failed", Message: "Failed to commit transaction", Status: httpStatusInternalServerError}
	}

	raw, _ := json.Marshal(newUser)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("usersvc.signup.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", newUser.ID)
	return out, nil
}

// UpdateProfile applies partial updates with uniqueness checks, then busts cache.
func (s *UserService) UpdateProfile(ctx context.Context, userID int32, req request.UpdateUserProfileRequest) (map[string]any, *response.AppError) {
	logger.Info("usersvc.update_profile.start", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)

	if req.Username == nil && req.Email == nil && req.Name == nil && req.Country == nil &&
		req.City == nil && req.LegalAddress == nil && req.VatNumber == nil && req.Language == nil {
		logger.Warn("usersvc.update_profile.no_changes", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
		return nil, &response.AppError{Code: "no_changes", Message: "No fields to update.", Status: httpStatusBadRequest}
	}
	if err := validation.ValidateUpdateUserProfileRequest(&req); err != nil {
		logger.Warn("usersvc.update_profile.validation_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "bad_request", Message: err.Error(), Status: httpStatusBadRequest}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("usersvc.update_profile.tx_begin_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "transaction_start_failed", Message: "Failed to start transaction", Status: httpStatusInternalServerError}
	}
	defer tx.Rollback()

	qtx := store.New(tx)

	if req.Username != nil && *req.Username != "" {
		if u, err := qtx.GetUserByUsernameOrEmail(ctx, *req.Username); err == nil && u.ID != userID {
			logger.Warn("usersvc.update_profile.username_exists", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "taken_by", u.ID)
			return nil, &response.AppError{Code: "username_exists", Message: "Username already exists", Status: httpStatusConflict}
		}
	}
	if req.Email != nil && *req.Email != "" {
		if u, err := qtx.GetUserByUsernameOrEmail(ctx, *req.Email); err == nil && u.ID != userID {
			logger.Warn("usersvc.update_profile.email_exists", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "taken_by", u.ID)
			return nil, &response.AppError{Code: "email_exists", Message: "Email already exists", Status: httpStatusConflict}
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
		logger.Error("usersvc.update_profile.update_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "user_update_failed", Message: "Failed to update user profile", Status: httpStatusInternalServerError}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("usersvc.update_profile.tx_commit_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return nil, &response.AppError{Code: "transaction_commit_failed", Message: "Failed to commit transaction", Status: httpStatusInternalServerError}
	}

	if err := s.rdb.Del(ctx, cacheKeyUser(userID)).Err(); err != nil {
		logger.Warn("usersvc.update_profile.cache_bust_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
	}

	raw, _ := json.Marshal(res)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)

	logger.Info("usersvc.update_profile.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
	return out, nil
}

// Delete performs a soft-delete and clears caches.
func (s *UserService) Delete(ctx context.Context, userID int32) *response.AppError {
	logger.Info("usersvc.delete.start", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)

	now := time.Now()
	if err := s.q.DeleteUser(ctx, store.DeleteUserParams{
		ID:        userID,
		DeletedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		logger.Error("usersvc.delete.failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return &response.AppError{Code: "delete_user_failed", Message: "Failed to delete user", Status: httpStatusInternalServerError}
	}
	if err := s.rdb.Del(ctx, cacheKeyUser(userID)).Err(); err != nil {
		logger.Warn("usersvc.delete.cache_bust_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
	}
	logger.Info("usersvc.delete.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
	return nil
}

// SendVerificationEmail creates a token, stores it, and sends the verification email.
func (s *UserService) SendVerificationEmail(ctx context.Context, user map[string]any) error {
	logger.Info("usersvc.email_verification.start", "req_id", ctxmeta.RequestID(ctx))

	var userID int32
	switch v := user["id"].(type) {
	case float64:
		userID = int32(v)
	case int:
		userID = int32(v)
	case int32:
		userID = v
	case int64:
		userID = int32(v)
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 32)
		if err != nil {
			logger.Error("usersvc.email_verification.id_invalid", "req_id", ctxmeta.RequestID(ctx), "value", v, "error", err)
			return fmt.Errorf("invalid user id string: %w", err)
		}
		userID = int32(n)
	default:
		logger.Error("usersvc.email_verification.id_type_invalid", "req_id", ctxmeta.RequestID(ctx), "type", fmt.Sprintf("%T", v))
		return errors.New("invalid user id format")
	}

	email, _ := user["email"].(string)
	username, _ := user["username"].(string)
	if strings.TrimSpace(email) == "" {
		logger.Error("usersvc.email_verification.email_missing", "req_id", ctxmeta.RequestID(ctx), "user_id", userID)
		return errors.New("user email missing")
	}

	token, err := generateVerificationToken(32)
	if err != nil {
		logger.Error("usersvc.email_verification.token_generate_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return fmt.Errorf("token generate failed: %w", err)
	}

	ttl := 24 * time.Hour
	if v := strings.TrimSpace(os.Getenv("EMAIL_VERIFICATION_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}
	validUntil := time.Now().Add(ttl)

	if _, err := s.q.CreateEmailVerificationToken(ctx, store.CreateEmailVerificationTokenParams{
		UserID:     userID,
		Token:      token,
		ValidUntil: validUntil,
	}); err != nil {
		logger.Error("usersvc.email_verification.create_token_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return fmt.Errorf("create email verification token failed: %w", err)
	}

	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("EMAIL_VERIFY_BASE")), "/")
	if base == "" {
		base = "http://localhost:8081/cargo-agent/v1/users/verify-email"
	}
	link := fmt.Sprintf("%s?token=%s", base, token)

	subject, htmlBody, textBody, err := templates.RenderVerificationEmail(username, link, validUntil)
	if err != nil {
		logger.Error("usersvc.email_verification.render_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return fmt.Errorf("render verification email failed: %w", err)
	}

	s.mailerOnce.Do(func() {
		if s.mailer == nil {
			s.mailer, s.mailerErr = mailer.NewFromEnv()
		}
	})
	if s.mailer == nil {
		logger.Error("usersvc.email_verification.mailer_init_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", s.mailerErr)
		return fmt.Errorf("mailer init failed: %w", s.mailerErr)
	}

	if err := s.mailer.Send(email, subject, htmlBody, textBody); err != nil {
		logger.Error("usersvc.email_verification.send_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "error", err)
		return fmt.Errorf("send verification email failed: %w", err)
	}

	logger.Info("usersvc.email_verification.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", userID, "email", email)
	return nil
}

// VerifyEmail validates the token, activates the user, and removes the token.
func (s *UserService) VerifyEmail(ctx context.Context, token string) *response.AppError {
	fp := tokenFP(token)
	logger.Info("usersvc.verify_email.start", "req_id", ctxmeta.RequestID(ctx), "token_fp", fp)

	token = strings.TrimSpace(token)
	if token == "" {
		logger.Warn("usersvc.verify_email.missing_token", "req_id", ctxmeta.RequestID(ctx))
		return &response.AppError{Code: "invalid_token", Message: "Token is required", Status: httpStatusBadRequest}
	}

	row, err := s.q.GetEmailVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Warn("usersvc.verify_email.token_not_found", "req_id", ctxmeta.RequestID(ctx), "token_fp", fp)
			return &response.AppError{Code: "invalid_token", Message: "Token not found", Status: httpStatusBadRequest}
		}
		logger.Error("usersvc.verify_email.db_error", "req_id", ctxmeta.RequestID(ctx), "token_fp", fp, "error", err)
		return &response.AppError{Code: "db_error", Message: "Unable to load token", Status: httpStatusInternalServerError}
	}

	if time.Now().After(row.ValidUntil) {
		_ = s.q.DeleteEmailVerificationToken(ctx, token)
		logger.Warn("usersvc.verify_email.token_expired", "req_id", ctxmeta.RequestID(ctx), "user_id", row.UserID, "token_fp", fp)
		return &response.AppError{Code: "token_expired", Message: "Token has expired", Status: 410}
	}

	if err := s.q.UpdateUserStatus(ctx, store.UpdateUserStatusParams{
		ID:     int32(row.UserID),
		Status: models.UserStatusActive,
	}); err != nil {
		logger.Error("usersvc.verify_email.activate_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", row.UserID, "error", err)
		return &response.AppError{Code: "user_update_failed", Message: "Failed to activate user", Status: httpStatusInternalServerError}
	}

	if err := s.q.DeleteEmailVerificationToken(ctx, token); err != nil {
		logger.Warn("usersvc.verify_email.delete_token_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", row.UserID, "error", err)
	}
	if err := s.rdb.Del(ctx, cacheKeyUser(int32(row.UserID))).Err(); err != nil {
		logger.Warn("usersvc.verify_email.cache_bust_failed", "req_id", ctxmeta.RequestID(ctx), "user_id", row.UserID, "error", err)
	}

	logger.Info("usersvc.verify_email.ok", "req_id", ctxmeta.RequestID(ctx), "user_id", row.UserID, "at", time.Now().Format(time.RFC3339))
	return nil
}

// Helpers

func generateVerificationToken(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func cacheKeyUser(id int32) string { return fmt.Sprintf("user:%d", id) }

// local HTTP status aliases
const (
	httpStatusBadRequest          = 400
	httpStatusUnauthorized        = 401
	httpStatusForbidden           = 403
	httpStatusConflict            = 409
	httpStatusNotFound            = 404
	httpStatusInternalServerError = 500
)
