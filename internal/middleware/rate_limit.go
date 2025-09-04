package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aleksjovanovic/cargo-agent/internal/authn"
	"github.com/aleksjovanovic/cargo-agent/internal/response"
	"github.com/redis/go-redis/v9"
)

// RateLimit returns a middleware that enforces a simple fixed-window rate limit
// in Redis. Identity is derived from the authenticated user ID (if present in
// context via Authz middleware) or falls back to client IP.
// It also sets standard rate-limit headers:
//
//   - X-RateLimit-Limit: total requests allowed in the current window
//   - X-RateLimit-Remaining: remaining requests in the current window
//   - Retry-After: seconds until next window (only on 429)
//
// Behavior notes:
//   - If rdb is nil, this middleware becomes a no-op (always allows).
//   - On Redis failures, it gracefully degrades to pass-through (never blocks).
func RateLimit(rdb *redis.Client, max int, window time.Duration) func(http.Handler) http.Handler {
	// No Redis? No limiter.
	if rdb == nil || max <= 0 || window <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now().UTC()
			// Start of the current fixed window (e.g., minute boundary)
			slotStart := now.Truncate(window)
			nextSlot := slotStart.Add(window)
			secondsToReset := int(nextSlot.Sub(now).Seconds())
			if secondsToReset < 1 {
				secondsToReset = 1
			}

			ctx := r.Context()

			// Compose a stable Redis key for the current identity and window slot.
			// Example: rl:v1:u:12345:2025-09-02T10:35:00Z  or  rl:v1:ip:203.0.113.7:...
			identity := extractIdentity(ctx, r)
			key := "rl:v1:" + identity + ":" + slotStart.Format(time.RFC3339)

			// Atomically increment the counter and ensure it expires after the window.
			pipe := rdb.TxPipeline()
			cnt := pipe.Incr(ctx, key)
			// Expire slightly past the window boundary to absorb clock/network jitter.
			pipe.Expire(ctx, key, window+5*time.Second)
			_, execErr := pipe.Exec(ctx)

			// On Redis error: don't block traffic, just pass through without headers.
			if execErr != nil {
				next.ServeHTTP(w, r)
				return
			}

			current := int(cnt.Val())
			remaining := max - current
			if remaining < 0 {
				remaining = 0
			}

			// Always set limit headers so clients can introspect.
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(max))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if current > max {
				// Too many requests in this window -> 429 + Retry-After
				w.Header().Set("Retry-After", strconv.Itoa(secondsToReset))
				// Use a unified JSON error shape.
				response.RespondWithError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Please try again later.", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractIdentity returns a stable identity string used for rate limiting:
//   - "u:<userID>" when auth claims are present
//   - otherwise "ip:<clientIP>"
func extractIdentity(ctx context.Context, r *http.Request) string {
	if v := ctx.Value(UserClaimsKey); v != nil {
		if claims, ok := v.(*authn.Claims); ok && claims != nil && claims.UserID > 0 {
			return "u:" + strconv.FormatInt(claims.UserID, 10)
		}
	}
	return "ip:" + clientIP(r)
}

// clientIP determines the best-effort client IP:
//
//	X-Forwarded-For (first) > X-Real-Ip > RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xr != "" {
		return xr
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
