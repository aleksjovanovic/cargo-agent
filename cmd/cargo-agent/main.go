package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1handlers "github.com/aleksjovanovic/cargo-agent/internal/api/v1/handlers"
	v1routes "github.com/aleksjovanovic/cargo-agent/internal/api/v1/routes"
	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/config"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/mailer"
	"github.com/aleksjovanovic/cargo-agent/internal/middleware"
	"github.com/aleksjovanovic/cargo-agent/internal/services"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/templates"
	"github.com/redis/go-redis/v9"
)

// main wires together configuration, infrastructure (DB/Redis/Mailer),
// services, HTTP routes and middlewares, then starts the HTTP server with
// graceful shutdown. Comments are placed where intent might not be obvious.
func main() {
	// Preload and validate email templates at startup (fail-fast).
	templates.MustInit()
	logger.Info("Templates initialized")

	// Load strongly-typed configuration from env or .env file.
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}
	logger.Info("Configuration loaded",
		"env", cfg.Environment,
		"log_level", cfg.LogLevel,
		"port", cfg.ServerPort,
	)

	// JWT secret is mandatory; without it we cannot sign tokens.
	if cfg.JWTSecret == "" {
		logger.Fatal("JWT secret missing", "hint", "set JWT_SECRET_KEY in .env or environment")
	}
	// Some legacy code reads JWT secret from env directly; keep it in sync.
	_ = os.Setenv("JWT_SECRET_KEY", cfg.JWTSecret)

	// Establish DB connection (panic on misconfiguration happens inside ConnectDB).
	db := config.ConnectDB(cfg.DatabaseURL)
	defer func() {
		_ = db.Close()
		logger.Info("DB connection closed")
	}()
	logger.Info("DB connected")

	// Establish Redis connection (used for caching, blacklists, rate-limiting, etc.).
	rdb := config.ConnectRedis()
	defer func(rdb *redis.Client) {
		_ = rdb.Close()
		logger.Info("Redis connection closed")
	}(rdb)
	logger.Info("Redis connected")

	// Initialize mailer from environment (SMTP/API credentials).
	m, err := mailer.NewFromEnv()
	if err != nil {
		logger.Fatal("Mailer init failed", "error", err)
	}
	logger.Info("Mailer initialized")

	// Prepare sqlc query layer (thin type-safe wrapper over DB).
	queries := store.New(db)
	logger.Info("SQLC queries ready")

	// JWT issuance options (issuer, audience, TTL, etc.).
	jwtOpt := authn.Options{
		Issuer:        "AJ",
		Audience:      []string{"cargo-agent"},
		TTL:           time.Hour,
		NotBeforeSkew: 0,
	}
	logger.Info("JWT options set", "issuer", jwtOpt.Issuer, "aud", jwtOpt.Audience, "ttl", jwtOpt.TTL.String())

	// Construct services. Keep them stateless where possible; inject shared deps.
	usersSvc := services.NewUserService(db, queries, rdb, []byte(cfg.JWTSecret), jwtOpt, m)
	countriesSvc := services.NewCountryService(queries, rdb)
	cargoSvc := services.NewCargoOfferService(db, queries)
	truckAvailabilitySvc := services.NewTruckAvailabilityService(db, queries)
	logger.Info("Services constructed")

	// Build HTTP handlers (presentation layer is intentionally thin).
	handler := v1handlers.NewHandlers(usersSvc, countriesSvc, cargoSvc, truckAvailabilitySvc)

	// Register versioned routes into a single ServeMux.
	mux := http.NewServeMux()
	// Note: we pass both db and rdb because /readyz probes need them,
	// and /metrics is mounted at the root server (not under /cargo-agent/v1).
	v1routes.SetupRoutes(mux, handler, db, rdb)
	logger.Info("Routes setup complete")

	// Compose global middleware chain:
	// - RateLimit: token-bucket in Redis (60 req/min per user/IP)
	// - Metrics: Prometheus instrumentation for HTTP requests
	// - Recoverer: centralized panic recovery with 500 response
	// - RequestID: inject/propagate X-Request-ID (last so it wraps the outermost handler)
	//
	// The nesting order below ensures the request flows through RateLimit first,
	// then gets instrumented, then protected by Recoverer, and finally annotated
	// with RequestID for consistent correlation across logs.
	root := middleware.RequestID(
		middleware.Recoverer(
			middleware.Metrics(
				middleware.RateLimit(rdb, 60, time.Minute)(
					mux, // the inner-most handler is the router
				),
			),
		),
	)

	// HTTP server with sane timeouts. ReadHeaderTimeout helps mitigate slowloris.
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           root,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second, // safe default; does not break existing behavior
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Run server in a goroutine so we can listen for shutdown signals.
	errCh := make(chan error, 1)
	go func() {
		logger.Info("Starting server", "addr", serverAddr, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// OS signal handling for graceful shutdown (Ctrl+C, docker stop, k8s SIGTERM).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either a fatal server error or a shutdown signal.
	select {
	case sig := <-quit:
		logger.Warn("Shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		logger.Fatal("Server failed", "error", err)
	}

	// Attempt graceful shutdown: stop accepting new connections and
	// allow in-flight requests to finish within the timeout window.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		// If graceful shutdown times out, force-close.
		logger.Error("Graceful shutdown failed, forcing close", "error", err)
		_ = srv.Close()
		logger.Info("Server closed forcefully")
	} else {
		logger.Info("Server stopped gracefully")
	}
}
