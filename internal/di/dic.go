package di

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
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
	return &Container{
		Users: services.NewUserService(db, q, rdb, jwtSecret, jwtOpt),
	}
}
