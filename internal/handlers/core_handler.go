package handlers

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/store"
)

type Handler struct {
	DB      *sql.DB
	Queries *store.Queries
}

// NewHandlers returns a new Handlers struct with queries
func NewHandlers(db *sql.DB, queries *store.Queries) *Handler {
	return &Handler{
		DB:      db,
		Queries: queries,
	}
}
