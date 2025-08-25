package dbconfig

import (
	"database/sql"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	_ "github.com/lib/pq"
)

func ConnectDB(databaseURL string) *sql.DB {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", "error", err)
	}

	// Ping to secure connection is established
	if err = db.Ping(); err != nil {
		logger.Fatal("Failed to establish database connection", "error", err)
	}

	logger.Info("Connected to database successfully", "driver", "postgres", "host", databaseURL)

	return db
}
