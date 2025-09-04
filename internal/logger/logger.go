package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Global logger instance used by helper functions below.
// It is safe for concurrent use (slog is goroutine-safe).
var log *slog.Logger

// levelVar allows changing the log level at runtime without
// re-creating handlers. Defaults to Info.
var levelVar slog.LevelVar

func init() {
	// Default level: INFO (can be overridden by LOG_LEVEL).
	levelVar.Set(slog.LevelInfo)

	// Try to read LOG_LEVEL from environment (e.g., "debug", "info", "warn", "error").
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		_ = SetLevelString(s) // best-effort; ignore error to keep default level
	}

	// JSON handler is good for production and machine parsing.
	// AddSource provides file:line for better diagnostics.
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     &levelVar,
		AddSource: true,
	})

	log = slog.New(h)
}

// SetLogger replaces the global logger instance (e.g., in tests).
// NOTE: Prefer to pass a handler with Level bound to &levelVar if you
// still want SetLevelString to affect the new logger.
func SetLogger(l *slog.Logger) {
	if l == nil {
		return
	}
	log = l
}

// L returns the current global logger.
// Useful when you need chained calls or custom With attrs.
func L() *slog.Logger {
	return log
}

// With returns a child logger enriched with the provided key-value attributes.
// Example: logger.With("rid", rid, "route", pattern).Info("handled")
func With(args ...any) *slog.Logger {
	return log.With(args...)
}

// SetLevel sets the current log level (Debug, Info, Warn, Error).
func SetLevel(lvl slog.Level) {
	levelVar.Set(lvl)
}

// SetLevelString sets the current log level from a string.
// Accepted values (case-insensitive): "debug", "info", "warn", "warning", "error".
// Returns an error if the value is unknown (level remains unchanged).
func SetLevelString(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		levelVar.Set(slog.LevelDebug)
	case "info":
		levelVar.Set(slog.LevelInfo)
	case "warn", "warning":
		levelVar.Set(slog.LevelWarn)
	case "error":
		levelVar.Set(slog.LevelError)
	default:
		return fmt.Errorf("unknown log level: %s", s)
	}
	return nil
}

// Info logs at info level with structured key-value pairs.
func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

// Warn logs at warn level with structured key-value pairs.
func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

// Error logs at error level with structured key-value pairs.
func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

// Debug logs at debug level with structured key-value pairs.
func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

// Fatal logs as error and then terminates the process with status code 1.
// slog does not provide Fatal; this keeps termination behavior obvious.
func Fatal(msg string, args ...any) {
	log.Error(msg, args...)
	os.Exit(1)
}
