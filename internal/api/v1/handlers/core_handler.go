package handlers

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	DB      *sql.DB
	Queries *store.Queries
	Redis   *redis.Client
}

// NewHandlers returns a new Handlers struct with queries
func NewHandlers(db *sql.DB, queries *store.Queries, redisClient *redis.Client) *Handler {
	return &Handler{
		DB:      db,
		Queries: queries,
		Redis:   redisClient,
	}
}
