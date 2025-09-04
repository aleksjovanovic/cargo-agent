package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
)

// contextKey is a private type to avoid collisions when storing values in context.
type contextKey string

// UserClaimsKey is the context key under which verified JWT claims are stored.
const UserClaimsKey contextKey = "claims"

// bearerToken extracts and validates a Bearer token from the Authorization header.
// Expected format: "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errors.New("no_token")
	}
	parts := strings.Fields(h)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("invalid_auth_header")
	}
	return parts[1], nil
}

// Authz returns an auth middleware that requires a valid HS256 JWT.
// This variant does NOT check a blacklist (suitable when Redis is not available).
func Authz(secret []byte) func(http.Handler) http.Handler {
	return authz(secret, nil)
}

// AuthzWithBlacklist returns an auth middleware that also checks a Redis-backed blacklist.
// A token is considered revoked if the key "jwt:blacklist:<sha256(token)>" exists in Redis.
func AuthzWithBlacklist(secret []byte, rdb *redis.Client) func(http.Handler) http.Handler {
	return authz(secret, rdb)
}

// authz is the shared implementation used by both Authz(...) and AuthzWithBlacklist(...).
func authz(secret []byte, rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1) Pull the bearer token from the request
			tokenStr, err := bearerToken(r)
			if err != nil {
				// RFC6750-compliant-ish responses for missing/malformed headers
				if err.Error() == "no_token" {
					w.Header().Set("WWW-Authenticate", `Bearer realm="cargo-agent", error="invalid_token", error_description="No token provided"`)
					response.RespondWithError(w, http.StatusUnauthorized, "no_token", "No token provided", nil)
					return
				}
				w.Header().Set("WWW-Authenticate", `Bearer realm="cargo-agent", error="invalid_request", error_description="Malformed Authorization header"`)
				response.RespondWithError(w, http.StatusUnauthorized, "invalid_auth_header", "Invalid Authorization header", nil)
				return
			}

			// 2) Validate and parse the JWT (HS256 only)
			keyFunc := func(token *jwt.Token) (any, error) {
				// Accept only HMAC (HS256) tokens
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, jwt.ErrTokenUnverifiable
				}
				return secret, nil
			}

			claims := &authn.Claims{}
			parsed, err := jwt.ParseWithClaims(
				tokenStr,
				claims,
				keyFunc,
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			)
			if err != nil {
				// Fine-grained error mapping for better client feedback
				var ve *jwt.ValidationError
				if errors.As(err, &ve) {
					switch {
					case ve.Errors&jwt.ValidationErrorExpired != 0:
						w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="Token expired"`)
						response.RespondWithError(w, http.StatusUnauthorized, "token_expired", "Token has expired", nil)
						return
					case ve.Errors&jwt.ValidationErrorNotValidYet != 0:
						response.RespondWithError(w, http.StatusUnauthorized, "token_not_valid_yet", "Token not valid yet", nil)
						return
					case ve.Errors&jwt.ValidationErrorSignatureInvalid != 0:
						response.RespondWithError(w, http.StatusUnauthorized, "invalid_token_signature", "Invalid token signature", nil)
						return
					default:
						response.RespondWithError(w, http.StatusUnauthorized, "invalid_token", "Invalid token", nil)
						return
					}
				}
				response.RespondWithError(w, http.StatusUnauthorized, "invalid_token", "Invalid token", nil)
				return
			}

			if !parsed.Valid {
				response.RespondWithError(w, http.StatusUnauthorized, "invalid_token", "Invalid token", nil)
				return
			}

			// 3) Optional: check Redis blacklist (if Redis is configured)
			if rdb != nil {
				sum := sha256.Sum256([]byte(tokenStr))
				key := "jwt:blacklist:" + hex.EncodeToString(sum[:])

				// If the key exists and contains any value, treat the token as revoked.
				if val, err := rdb.Get(r.Context(), key).Result(); err == nil && val != "" {
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="Token revoked"`)
					response.RespondWithError(w, http.StatusUnauthorized, "token_revoked", "Token has been revoked", nil)
					return
				}
			}

			// 4) Attach claims to context and continue
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
