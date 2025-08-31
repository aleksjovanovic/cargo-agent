package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
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

type UserService struct {
	db        *sql.DB
	q         *store.Queries
	rdb       *redis.Client
	jwtSecret []byte
	jwtOpt    authn.Options
	mailer    mailer.Mailer
}

func NewUserService(db *sql.DB, q *store.Queries, rdb *redis.Client, jwtSecret []byte, jwtOpt authn.Options, m mailer.Mailer) *UserService {
	return &UserService{db: db, q: q, rdb: rdb, jwtSecret: jwtSecret, jwtOpt: jwtOpt, mailer: m}
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

// SendVerificationEmail generiše verifikacioni token, snimi ga u bazu i pošalje e-mail sa linkom.
func (s *UserService) SendVerificationEmail(ctx context.Context, user map[string]any) error {
	// 1) Izvuci userID/email/username iz map-e (robusno)
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
			return fmt.Errorf("invalid user id string: %w", err)
		}
		userID = int32(n)
	default:
		return errors.New("invalid user id format")
	}

	email, _ := user["email"].(string)
	username, _ := user["username"].(string)
	if strings.TrimSpace(email) == "" {
		return errors.New("user email missing")
	}

	// 2) Generiši kratak, kriptografski jak token (base64url bez paddinga)
	token, err := generateVerificationToken(32) // 32B -> ~43-44 base64url karaktera
	if err != nil {
		return fmt.Errorf("token generate failed: %w", err)
	}

	// 3) TTL iz ENV (default 24h)
	ttl := 24 * time.Hour
	if v := strings.TrimSpace(os.Getenv("EMAIL_VERIFICATION_TTL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}
	validUntil := time.Now().Add(ttl)

	// 4) Upis u tabelu email_verification_tokens
	if _, err := s.q.CreateEmailVerificationToken(ctx, store.CreateEmailVerificationTokenParams{
		UserID:     userID,
		Token:      token,
		ValidUntil: validUntil,
	}); err != nil {
		return fmt.Errorf("create email verification token failed: %w", err)
	}

	// 5) Napravi verifikacioni link (prefer front bazu iz ENV-a; fallback je API ruta)
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("EMAIL_VERIFY_BASE")), "/")
	if base == "" {
		base = "http://localhost:8081/cargo-agent/v1/users/verify-email"
	}
	link := fmt.Sprintf("%s?token=%s", base, token)

	// 6) Renderuj HTML/TXT templejte i pošalji e-mail
	//    (Pretpostavka: templates.MustInit() je pozvan u main() pri startu.)
	subject, htmlBody, textBody, err := templates.RenderVerificationEmail(username, link, validUntil)
	if err != nil {
		return fmt.Errorf("render verification email failed: %w", err)
	}

	m, err := mailer.NewFromEnv()
	if err != nil {
		return fmt.Errorf("mailer init failed: %w", err)
	}
	if err := m.Send(email, subject, htmlBody, textBody); err != nil {
		return fmt.Errorf("send verification email failed: %w", err)
	}

	return nil
}

// VerifyEmail potvrđuje token i aktivira korisnika.
func (s *UserService) VerifyEmail(ctx context.Context, token string) *response.AppError {
	token = strings.TrimSpace(token)
	if token == "" {
		return &response.AppError{Code: "invalid_token", Message: "Token is required", Status: 400}
	}

	// 1) Učitaj token iz DB
	row, err := s.q.GetEmailVerificationToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &response.AppError{Code: "invalid_token", Message: "Token not found", Status: 400}
		}
		return &response.AppError{Code: "db_error", Message: "Unable to load token", Status: 500}
	}

	// 2) Proveri istekao?
	if time.Now().After(row.ValidUntil) {
		// očisti token
		_ = s.q.DeleteEmailVerificationToken(ctx, token)
		return &response.AppError{Code: "token_expired", Message: "Token has expired", Status: 410}
	}

	// 3) Aktiviraj korisnika (iz draft → active; može i bez obzira na prethodni status)
	now := time.Now()
	err = s.q.UpdateUserStatus(ctx, store.UpdateUserStatusParams{
		ID:     int32(row.UserID),
		Status: models.UserStatusActive,
		// updated_at se setuje u SQL-u na now(), ali ako ti treba dodatno:
		// ovde ništa – query već radi SET updated_at = now()
	})
	if err != nil {
		return &response.AppError{Code: "user_update_failed", Message: "Failed to activate user", Status: 500}
	}

	// 4) Očisti token (više nije potreban)
	if err := s.q.DeleteEmailVerificationToken(ctx, token); err != nil {
		// ne blokiramo uspeh – samo loguj
		logger.Warn("failed to delete verification token", "error", err)
	}

	// 5) Po želji: očisti user cache
	_ = s.rdb.Del(ctx, cacheKeyUser(int32(row.UserID))).Err()

	logger.Info("user verified by email", "user_id", row.UserID, "at", now.Format(time.RFC3339))
	return nil
}

// === helpers ===

func generateVerificationToken(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand failed: %w", err)
	}
	// URL-safe bez '=' paddinga
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// --- helpers ---

func cacheKeyUser(id int32) string { return fmt.Sprintf("user:%d", id) }
