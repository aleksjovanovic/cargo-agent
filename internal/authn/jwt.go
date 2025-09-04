// internal/authn/jwt.go
package authn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims represents the custom + standard JWT payload we issue/expect.
// We embed jwt.RegisteredClaims to leverage library validations (exp/iat/nbf, aud, iss, sub).
// NOTE: Keep field json tags stable to avoid breaking existing tokens.
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"user_name"`
	jwt.RegisteredClaims
}

// Options controls the shape and lifetime of issued tokens.
// - Issuer/Audience are baked into the token on issue (helpful for downstream validation).
// - TTL controls Exp (ExpiresAt) relative to "now".
// - NotBeforeSkew allows slight "clock in the future" grace for nbf claim.
type Options struct {
	Issuer        string
	Audience      []string
	TTL           time.Duration
	NotBeforeSkew time.Duration
}

// GenerateJWT builds a HS256-signed JWT with our custom Claims.
// We set Exp, Iat and Nbf relative to the current UTC time to keep values deterministic.
// NOTE: secretKey must be the same when parsing/validating on the other end.
func GenerateJWT(userID int64, username string, secretKey []byte, opt Options) (string, error) {
	now := time.Now().UTC() // use UTC for consistency across services

	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    opt.Issuer,
			Subject:   username,
			Audience:  jwt.ClaimStrings(opt.Audience),
			ExpiresAt: jwt.NewNumericDate(now.Add(opt.TTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(opt.NotBeforeSkew)),
		},
	}

	// HS256 signing; keep in sync with ParseJWT expectations.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ParseJWT validates and parses a HS256 token into our Claims.
// It enforces the signing method and verifies standard claims via the library.
// IMPORTANT:
//   - This function DOES NOT validate Issuer/Audience beyond what jwt.ParseWithClaims
//     does by default. If you need strict iss/aud enforcement, extend this function
//     to accept Options and call claims.VerifyIssuer/VerifyAudience accordingly.
func ParseJWT(tokenString string, secretKey []byte) (*Claims, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		// Defensive: accept only HMAC and exactly HS256.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secretKey, nil
	}

	// jwt.WithValidMethods ensures parser rejects tokens with other algs.
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		keyFunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// ParseExpiryUnsafe extracts the "exp" (expiry) from the JWT payload WITHOUT verifying
// the signature or the header. Use only for non-security logic (e.g., UX hints).
// Returns (time, true) on success; (zero, false) on failure.
// Caveats:
//   - Since we don't verify the signature, never use this for authorization decisions.
//   - json.Unmarshal of generic map decodes numbers as float64 by default; the json.Number
//     branch is included for robustness in case a different decoder is used.
func ParseExpiryUnsafe(raw string) (time.Time, bool) {
	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return time.Time{}, false
	}
	payloadB64 := parts[1]

	// JWT uses base64url without padding (RawURLEncoding).
	b, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return time.Time{}, false
	}

	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return time.Time{}, false
	}

	switch v := payload["exp"].(type) {
	case float64:
		// Typical path when using standard json.Unmarshal into map[string]any.
		return time.Unix(int64(v), 0).UTC(), true
	case json.Number:
		// Only hits if a decoder was configured with UseNumber.
		if i, err := v.Int64(); err == nil {
			return time.Unix(i, 0).UTC(), true
		}
	}

	return time.Time{}, false
}
