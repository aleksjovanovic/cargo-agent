package utils

import (
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

// Password utilities backed by bcrypt.
// Notes:
// - bcrypt includes a built-in salt and is resistant to rainbow tables.
// - DefaultCost is a sensible baseline; raise it if you can afford more CPU.
// - Do NOT trim or otherwise mutate user-provided passwords here; normalization
//   may surprise users (e.g., trailing spaces). Validate in the API layer instead.

// HashPassword hashes a plain-text password using bcrypt.DefaultCost.
// Returns the bcrypt hash string on success (includes algorithm, cost, and salt).
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// Log at error level but return the original error upward.
		logger.Error("Failed to hash password", "error", err)
		return "", err
	}
	return string(hashedPassword), nil
}

// HashPasswordWithCost is a helper if you want to tune the cost factor per environment.
// For example, use a higher cost in production if latency allows.
func HashPasswordWithCost(password string, cost int) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		logger.Error("Failed to hash password with custom cost", "error", err, "cost", cost)
		return "", err
	}
	return string(hashedPassword), nil
}

// ComparePassword compares a plain-text password with a previously hashed password.
// It uses bcrypt's constant-time comparison to avoid timing side-channels.
func ComparePassword(storedPassword, providedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(providedPassword))
	return err == nil
}
