// internal/config/config.go
package config

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	_ "github.com/lib/pq" // Postgres driver for database/sql
)

// Config holds runtime configuration loaded from env/.env.
// Keep only values that are broadly useful across the app.
// Service-specific toggles can live closer to the service.
type Config struct {
	ServerPort  string
	DatabaseURL string
	Environment string
	LogLevel    string
	JWTSecret   string
}

// LoadConfig loads env variables (with .env fallbacks) and returns the Config.
// It does not validate semantics beyond presence of values with defaults;
// higher-level validation happens in main() (e.g., JWT secret must be set).
func LoadConfig() (*Config, error) {
	loadEnvWithLogging()

	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://cargo_agent_user:goldsink561@localhost:5432/cargo_agent"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		JWTSecret:   getEnv("JWT_SECRET_KEY", ""),
	}, nil
}

// loadEnvWithLogging tries loading .env files from a few sensible locations,
// in order, and emits logs about successes/failures. This is helpful both
// in local dev and in packaged binaries where CWD may not contain .env.
func loadEnvWithLogging() {
	loaded := 0

	try := func(label, path string, overload bool) {
		var err error
		if overload {
			err = godotenv.Overload(path) // later files override earlier/real env
		} else {
			err = godotenv.Load(path) // load only if present
		}
		if err != nil {
			logger.Debug("env: not found or failed", "where", label, "path", path, "error", err)
			return
		}
		loaded++
		logger.Info("env: loaded", "where", label, "path", path)
	}

	// 1) Common local dev case: project root
	try("CWD", ".env", false)

	// 2) Typical repo layouts: nested cmd dir or parent
	try("repo path", "cmd/cargo-agent/.env", true)
	try("repo path", "../.env", true)

	// 3) Alongside the built executable (useful in deployments)
	if exe, err := os.Executable(); err != nil {
		logger.Warn("env: cannot resolve executable path", "error", err)
	} else {
		base := filepath.Dir(exe)
		try("exe dir", filepath.Join(base, ".env"), true)
		try("exe dir parent", filepath.Join(base, "../.env"), true)
	}

	if loaded == 0 {
		logger.Debug("env: no .env files loaded (running with real env?)")
	} else {
		logger.Info("env: total files loaded", "count", loaded)
	}
}

// getEnv returns env var value or a default if missing.
func getEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

// getEnvInt parses an int env var, returning def on parse failure or absence.
func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// getEnvDuration parses a time.Duration env var (e.g., "30s", "5m"),
// returning def on parse failure or absence.
func getEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// ConnectRedis builds a Redis client from env and verifies connectivity
// with a short ping. The process is terminated with a fatal log if the
// connection cannot be established (startup should fail fast).
//
// Env vars:
//
//	REDIS_ADDR           (default "localhost:6379")
//	REDIS_PASSWORD       (default "")
//	REDIS_DB             (default 0)
func ConnectRedis() *redis.Client {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	password := getEnv("REDIS_PASSWORD", "")
	db := getEnvInt("REDIS_DB", 0)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Fast readiness check to fail early on boot if misconfigured.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		logger.Fatal("Failed to establish redis connection", "addr", addr, "db", db, "error", err)
	}

	logger.Info("Connected to Redis", "addr", addr, "db", db)
	return rdb
}

// ConnectDB opens a Postgres connection pool, sets pool parameters,
// and verifies connectivity with a short ping. Fatal on failure.
//
// Env vars (optional):
//
//	DB_MAX_OPEN_CONNS        (default 10)
//	DB_MAX_IDLE_CONNS        (default 5)
//	DB_CONN_MAX_LIFETIME     (default 30m)
//	DB_CONN_MAX_IDLE_TIME    (default 0; 0 means unlimited / not set)
func ConnectDB(databaseURL string) *sql.DB {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", "error", err)
	}

	// Pool configuration (tuneable via env in different environments).
	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 10)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	lifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	idleTime := getEnvDuration("DB_CONN_MAX_IDLE_TIME", 0)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)
	// Only apply idle timeout if explicitly configured (keeps prior behavior).
	if idleTime > 0 {
		db.SetConnMaxIdleTime(idleTime)
	}

	// Fast readiness check to fail early on boot if misconfigured.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		logger.Fatal("Failed to establish database connection", "error", err)
	}

	host, user := dsnHostUser(databaseURL)
	logger.Info("Connected to Postgres",
		"host", host,
		"user", user,
		"maxOpen", maxOpen,
		"maxIdle", maxIdle,
		"lifetime", lifetime.String(),
		"idleTime", idleTime.String(),
	)

	return db
}

// dsnHostUser extracts host and username from a Postgres DSN for logging.
// Returns "(unknown)" when parsing fails or values are absent.
func dsnHostUser(dsn string) (host, user string) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "(unknown)", "(unknown)"
	}
	host = u.Host
	if u.User != nil {
		user = u.User.Username()
	} else {
		user = "(unknown)"
	}
	return
}
