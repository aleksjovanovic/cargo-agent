package middlewares

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/dgrijalva/jwt-go"
)

// Custom type context for context key to avoid collision
type contextKey string

// Constant used in storing user claims
const UserClaimsKey contextKey = "claims"

func AuthzMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Retrieves the authorization header from the request
		authzHeader := r.Header.Get("Authorization")
		if authzHeader == "" {
			response.RespondWithError(
				w,
				http.StatusUnauthorized,
				"no_token",
				"No token provided",
				nil,
			)
			return
		}
		// strips the Berarer from the Bearer token
		tokenString := strings.TrimPrefix(authzHeader, "Bearer ")
		claims := &authn.Claims{}

		// Parse the token and validate it
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			return []byte(os.Getenv("JWT_SECRET_KEY")), nil
		})
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				response.RespondWithError(
					w,
					http.StatusBadRequest,
					"invalid_token_signature",
					"Invalid token signature",
					nil,
				)

				return
			}
			response.RespondWithError(
				w,
				http.StatusBadRequest,
				"invalid_token",
				"Invalid token",
				nil,
			)

			return
		}
		// if token is valid, store th claims in the request context
		if token.Valid {
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			r := r.WithContext(ctx)
			next.ServeHTTP(w, r)
		} else {
			response.RespondWithError(
				w,
				http.StatusUnauthorized,
				"invalid_token",
				"Invalid token",
				nil,
			)

		}
	})
}
