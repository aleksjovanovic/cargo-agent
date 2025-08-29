package authn

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"user_name"`
	jwt.RegisteredClaims
}

type Options struct {
	Issuer        string
	Audience      []string
	TTL           time.Duration // npr 60 * time.Minute
	NotBeforeSkew time.Duration // npr 0 ili 30s
}

// GenerateJWT generates a JWT token for the user
func GenerateJWT(userID int64, username string, secretKey []byte, opt Options) (string, error) {
	now := time.Now()

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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ParseJWT parses the JWT token and returns the claims
func ParseJWT(tokenString string, secretKey []byte) (*Claims, error) {
	keyFunc := func(token *jwt.Token) (any, error) {
		// zaštita: prihvati isključivo HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secretKey, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
