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

type contextKey string

const UserClaimsKey contextKey = "claims"

func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errors.New("no_token")
	}
	parts := strings.Fields(h)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("invalid_auth_header")
	}
	return parts[1], nil
}

// Authz: kompatibilno sa postojećim kodom (bez Redis blackliste).
func Authz(secret []byte) func(http.Handler) http.Handler {
	return authz(secret, nil)
}

// AuthzWithBlacklist: kao Authz, ali dodatno proverava Redis blacklist
// ("jwt:blacklist:<sha256(token)>"). Ako je token revokovan → 401.
func AuthzWithBlacklist(secret []byte, rdb *redis.Client) func(http.Handler) http.Handler {
	return authz(secret, rdb)
}

func authz(secret []byte, rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := bearerToken(r)
			if err != nil {
				if err.Error() == "no_token" {
					w.Header().Set("WWW-Authenticate", `Bearer realm="cargo-agent", error="invalid_token", error_description="No token provided"`)
					response.RespondWithError(w, http.StatusUnauthorized, "no_token", "No token provided", nil)
					return
				}
				w.Header().Set("WWW-Authenticate", `Bearer realm="cargo-agent", error="invalid_request", error_description="Malformed Authorization header"`)
				response.RespondWithError(w, http.StatusUnauthorized, "invalid_auth_header", "Invalid Authorization header", nil)
				return
			}

			keyFunc := func(token *jwt.Token) (any, error) {
				// Samo HS256
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
			if err != nil { // <= ovde je bila suvišna zagrada u staroj verziji
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

			// (Opcionalno) Redis blacklist check – aktivno samo ako je rdb != nil.
			if rdb != nil {
				sum := sha256.Sum256([]byte(tokenStr))
				key := "jwt:blacklist:" + hex.EncodeToString(sum[:])

				if val, err := rdb.Get(r.Context(), key).Result(); err == nil && val != "" {
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token", error_description="Token revoked"`)
					response.RespondWithError(w, http.StatusUnauthorized, "token_revoked", "Token has been revoked", nil)
					return
				}
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
