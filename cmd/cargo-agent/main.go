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
	"github.com/aleksjovanovic/cargo-agent/internal/middlewares"
	"github.com/aleksjovanovic/cargo-agent/internal/services"
	"github.com/aleksjovanovic/cargo-agent/internal/store"
	"github.com/aleksjovanovic/cargo-agent/internal/templates"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 0) Učitaj template-ove (HTML e-mail itd.)
	templates.MustInit()

	// 1) Konfiguracija
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration", "error", err)
	}
	if cfg.JWTSecret == "" {
		logger.Fatal("JWT secret missing", "hint", "set JWT_SECRET_KEY in .env or environment")
	}
	// (legacy) — ako negde middleware čita direktno iz env-a
	_ = os.Setenv("JWT_SECRET_KEY", cfg.JWTSecret)

	// 2) Infrastruktura: DB & Redis
	db := config.ConnectDB(cfg.DatabaseURL)
	defer db.Close()

	rdb := config.ConnectRedis()
	defer func(rdb *redis.Client) { _ = rdb.Close() }(rdb)

	// Mailer (iz .env)
	m, err := mailer.NewFromEnv()
	if err != nil {
		logger.Fatal("Mailer init failed", "error", err)
	}

	// 3) sqlc queries
	queries := store.New(db)

	// 4) JWT opcije
	jwtOpt := authn.Options{
		Issuer:        "AJ",
		Audience:      []string{"cargo-agent"},
		TTL:           time.Hour,
		NotBeforeSkew: 0,
	}

	// 5) Servisi
	usersSvc := services.NewUserService(db, queries, rdb, []byte(cfg.JWTSecret), jwtOpt, m)
	countriesSvc := services.NewCountryService(queries, rdb)
	cargoSvc := services.NewCargoOfferService(db, queries)
	truckAvailabilitySvc := services.NewTruckAvailabilityService(db, queries)

	// 6) HTTP handleri (tanki) — jedan “glavni” handler koji sadrži sve servise
	handler := v1handlers.NewHandlers(usersSvc, countriesSvc, cargoSvc, truckAvailabilitySvc)

	// 7) Rute (v1)
	mux := http.NewServeMux()
	v1routes.SetupRoutes(mux, handler)

	// 8) Global middleware chain
	root := middlewares.RequestID(middlewares.Recoverer(mux))

	// 9) HTTP server sa timeout-ima
	serverAddr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      root,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 10) Start + graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logger.Info("Starting server", "addr", serverAddr, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Warn("Shutting down server", "signal", sig.String())
	case err := <-errCh:
		logger.Fatal("Server failed", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed, forcing close", "error", err)
		_ = srv.Close()
	} else {
		logger.Info("Server stopped gracefully")
	}
}
