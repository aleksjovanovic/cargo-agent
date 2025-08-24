package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aleksjovanovic/cargo-agent/internal/dbconfig"
	"github.com/aleksjovanovic/cargo-agent/internal/handlers"
	"github.com/aleksjovanovic/cargo-agent/internal/routes"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
)

func main() {

	// Load configuration
	config, err := dbconfig.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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
	fmt.Printf("Starting server on  %s\n", serverAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed %v", err)
	}
}
