package config

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/redis/go-redis/v9"
)

func CloseAll(db *sql.DB, rdb *redis.Client) {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Warn("db close error", "error", err)
		} else {
			logger.Info("db closed")
		}
	}
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			logger.Warn("redis close error", "error", err)
		} else {
			logger.Info("redis closed")
		}
	}
}
