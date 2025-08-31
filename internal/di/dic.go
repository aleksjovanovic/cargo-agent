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

type Container struct {
	Users *services.UserService
}

func New(
	db *sql.DB,
	q *store.Queries,
	rdb *redis.Client,
	jwtSecret []byte,
	jwtOpt authn.Options,
) *Container {
	// Mailer iz okruženja (ako fali konfiguracija, samo upozori i ostavi nil).
	var m mailer.Mailer
	if mm, err := mailer.NewFromEnv(); err != nil {
		logger.Warn("mailer init failed (will use nil, service may fallback)", "error", err)
		// m = nil (UserService konstruktor treba da hendluje nil i napravI fallback)
	} else {
		m = mm
	}

	return &Container{
		// Novi potpis: dodaj mailer kao šesti argument
		Users: services.NewUserService(db, q, rdb, jwtSecret, jwtOpt, m),
	}
}
