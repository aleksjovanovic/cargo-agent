package dbconfig

import (
	"context"

	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func ConnectRedis() *redis.Client {
	addr := getEnv("REDIS_ADDR", "localhost:6379")
	password := getEnv("REDIS_PASSWORD", "")

	db := 0

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	_, err := rdb.Ping(Ctx).Result()
	if err != nil {
		logger.Fatal("Failed to establish redis connection", "error", err)
	}

	logger.Info("Connected to database successfully", "driver", "redis", "host", rdb)

	return rdb
}
