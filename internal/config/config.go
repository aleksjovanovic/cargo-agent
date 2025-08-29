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

	_ "github.com/lib/pq"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	Environment string
	LogLevel    string
}

func LoadConfig() (*Config, error) {
	loadEnvWithLogging()

	return &Config{
		ServerPort:  getEnv("SERVER_PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://cargo_agent_user:goldsink561@localhost:5432/cargo_agent"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}, nil
}

func loadEnvWithLogging() {
	loaded := 0

	try := func(label, path string, overload bool) {
		var err error
		if overload {
			err = godotenv.Overload(path)
		} else {
			err = godotenv.Load(path)
		}
		if err != nil {
			logger.Debug("env: not found or failed", "where", label, "path", path, "error", err)
			return
		}
		loaded++
		logger.Info("env: loaded", "where", label, "path", path)
	}

	try("CWD", ".env", false)

	try("repo path", "cmd/cargo-agent/.env", true)
	try("repo path", "../.env", true)

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

func getEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// ————— REDIS —————

func ConnectRedis() *redis.Client {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	password := getEnv("REDIS_PASSWORD", "")
	db := getEnvInt("REDIS_DB", 0)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		logger.Fatal("Failed to establish redis connection", "addr", addr, "db", db, "error", err)
	}

	logger.Info("Connected to Redis", "addr", addr, "db", db)
	return rdb
}

// ————— POSTGRES —————

func ConnectDB(databaseURL string) *sql.DB {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to initialize database connection", "error", err)
	}

	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 10)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 5)
	lifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(lifetime)

	// kratak timeout na ping
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
	)

	return db
}

func dsnHostUser(dsn string) (host, user string) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "(unknown)", "(unknown)"
	}
	host = u.Host
	if u.User != nil {
		user = u.User.Username()
	}
	return
}
