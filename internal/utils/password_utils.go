package utils

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain-text password
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return "", err
	}

	return string(hashedPassword), nil
}

// ComparePAssword compares a plain-text password with a hashed password
func ComparePassword(storedPassword, providedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(providedPassword))
	return err == nil
}
