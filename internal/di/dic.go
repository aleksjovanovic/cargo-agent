// internal/di/dic.go
//
// Package di provides a very small dependency-injection container.
// The goal is to centralize wiring of services so the rest of the app
// (handlers, jobs, etc.) can depend on constructed services instead of
// knowing how to assemble them.
package di

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/mailer"
	"github.com/aleksjovanovic/cargo-agent/internal/services"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/redis/go-redis/v9"
)

// Container holds references to constructed services that the application uses.
// Add new fields here as new services are introduced (e.g., Countries, CargoOffers, ...).
type Container struct {
	Users *services.UserService
	// Countries         *services.CountryService
	// CargoOffer        *services.CargoOfferService
	// TruckAvailability *services.TruckAvailabilityService
}

// New builds a Container by wiring concrete dependencies into services.
//
// Arguments:
//   - db:     shared *sql.DB connection pool
//   - q:      sqlc-generated query set bound to db
//   - rdb:    Redis client (can be nil if Redis is not used, services should handle it)
//   - jwtSecret: symmetric key for signing JWTs (HS256)
//   - jwtOpt:    issuer/audience/TTL settings used by the authentication service
//
// Behavior:
//   - Tries to initialize a Mailer from environment; if it fails, logs a warning and
//     passes a nil mailer to the UserService (which should either no-op or handle it
//     gracefully for features that require email).
func New(
	db *sql.DB,
	q *store.Queries,
	rdb *redis.Client,
	jwtSecret []byte,
	jwtOpt authn.Options,
) *Container {
	// Attempt to construct an email sender from environment variables.
	// This keeps bootstrap simple in development while allowing production
	// to configure a real SMTP/Sendgrid provider.
	var m mailer.Mailer
	if mm, err := mailer.NewFromEnv(); err != nil {
		// Non-fatal: password reset / verification email features may be disabled.
		logger.Warn("mailer init failed (service may degrade features that require email)", "error", err)
	} else {
		m = mm
	}

	// Wire services with shared infra. Keep construction here so other layers
	// (handlers, jobs) only receive a ready-to-use service.
	usersSvc := services.NewUserService(db, q, rdb, jwtSecret, jwtOpt, m)

	return &Container{
		Users: usersSvc,
	}
}
