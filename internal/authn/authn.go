package authn

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"user_name"`
	jwt.StandardClaims
}

// GenerateJWT generates a JWT token for the user
func GenerateJWT(userID int64, username string, secretKey []byte) (string, error) {
	// Set claims
	claims := Claims{
		UserID:   userID,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(60 * time.Minute).Unix(), // Expire  in 60 minutes
			Issuer:    "AJ",
		},
	}

	// Create token with claims and sign it
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ParseJWT parses the JWT token and returns the claims
func ParseJWT(tokenString string, secretKey []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	// Extract claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, err
}
