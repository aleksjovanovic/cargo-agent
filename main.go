package main

import (
	"fmt"
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/dbconfig"
	"github.com/aleksjovanovic/cargo-agent/internal/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/routes"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
)

func main() {

	// Load configuration
	config, err := dbconfig.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}

	// Connect to the database
	db := dbconfig.ConnectDB(config.DatabaseURL)
	defer db.Close()

	// Initialize sqlc queries
	queries := store.New(db)

	// Create a new handler with queries
	handler := handlers.NewHandlers(db, queries)

	// Set up HTTP server and routes
	mux := http.NewServeMux()

	// Setup routes without the prefix
	routes.SetupRoutes(mux, handler)

	serverAddr := fmt.Sprintf(":%s", config.ServerPort)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	// Start server
	logger.Info("Starting server", "addr", serverAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Server failed to start", "error", err)
	}
}
