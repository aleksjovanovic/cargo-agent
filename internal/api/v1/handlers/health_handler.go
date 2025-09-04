// internal/api/v1/handlers/health_handler.go
package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/ctxmeta"
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/redis/go-redis/v9"
)

// HealthCheckHandler returns a very cheap, always-200 liveness signal.
// Use this for container/orchestrator liveness probes.
// Notes:
// - We add Cache-Control: no-store to avoid any proxy caching.
// - Request ID is pulled from context (middleware injects it) for consistent tracing.
func (h *Handler) HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := ctxmeta.RequestID(r.Context())

		// Prevent intermediate proxies from caching health responses.
		w.Header().Set("Cache-Control", "no-store")

		logger.Info("health.liveness", "method", r.Method, "path", r.URL.Path, "rid", rid)

		response.RespondWithSuccess(w, http.StatusOK, response.Envelope{
			"message": "Server is OK",
		})
	}
}

// ReadyzHandler performs dependency checks (DB + Redis) and reports readiness.
// Returns 200 when both dependencies are reachable; otherwise 503.
// Notes:
// - Short timeouts keep the probe fast and prevent resource pileups.
// - We log per-dependency failures with the current request id for correlation.
// - Cache-Control: no-store avoids proxy caching of readiness state.
func ReadyzHandler(db *sql.DB, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := ctxmeta.RequestID(r.Context())

		// Prevent caching of readiness state by intermediaries.
		w.Header().Set("Cache-Control", "no-store")

		// Keep probe snappy — lower than typical app timeouts.
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()

		// Database check
		dbOK := false
		if db != nil {
			if err := db.PingContext(ctx); err == nil {
				dbOK = true
			} else {
				logger.Warn("readyz.db_down", "rid", rid, "error", err)
			}
		} else {
			// If DB is expected in this deployment, leaving dbOK=false will flip readiness to 503.
			logger.Warn("readyz.db_nil", "rid", rid)
		}

		// Redis check
		redisOK := false
		if rdb != nil {
			if _, err := rdb.Ping(ctx).Result(); err == nil {
				redisOK = true
			} else {
				logger.Warn("readyz.redis_down", "rid", rid, "error", err)
			}
		} else {
			// If Redis is expected, treat as not ready; otherwise adapt the logic as needed.
			logger.Warn("readyz.redis_nil", "rid", rid)
		}

		// Overall readiness
		status := http.StatusOK
		if !dbOK || !redisOK {
			status = http.StatusServiceUnavailable
		}

		logger.Info("readyz.status", "rid", rid, "db_ok", dbOK, "redis_ok", redisOK, "status", status)

		// Keep payload schema minimal and stable (as documented in OpenAPI).
		response.RespondWithSuccess(w, status, response.Envelope{
			"message": map[bool]string{true: "ready", false: "not_ready"}[status == http.StatusOK],
			"data": map[string]any{
				"db":    map[bool]string{true: "ok", false: "down"}[dbOK],
				"redis": map[bool]string{true: "ok", false: "down"}[redisOK],
			},
		})
	}
}
