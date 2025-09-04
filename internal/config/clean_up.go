// internal/config/clean_up.go
package config

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/redis/go-redis/v9"
)

// CloseAll attempts to gracefully close DB and Redis clients.
//   - Safe to call with nils (no-ops).
//   - Best-effort: logs errors for each resource independently and continues.
//   - Idempotent in practice: repeated Close calls on already-closed clients
//     will typically return an error; we log and move on.
func CloseAll(db *sql.DB, rdb *redis.Client) {
	// If both resources are nil, emit a low-verbosity info to help diagnose teardown paths.
	if db == nil && rdb == nil {
		logger.Info("nothing to close: db and redis are nil")
		return
	}

	// Close database connection pool if present.
	if db != nil {
		if err := db.Close(); err != nil {
			// Non-fatal: just log and continue with Redis.
			logger.Warn("db close error", "error", err)
		} else {
			logger.Info("db closed")
		}
	}

	// Close Redis client if present.
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			// Non-fatal: just log.
			logger.Warn("redis close error", "error", err)
		} else {
			logger.Info("redis closed")
		}
	}
}
