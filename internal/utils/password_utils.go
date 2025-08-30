package utils

import (
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain-text password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("Failed to hash password", "error", err)
		return "", err
	}

	return string(hashedPassword), nil
}

// ComparePassword compares a plain-text password with a hashed password
func ComparePassword(storedPassword, providedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(providedPassword))
	return err == nil
}
