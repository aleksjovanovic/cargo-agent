package logger

import (
	"log/slog"
	"os"
)

// globalni logger
var log *slog.Logger

func init() {
	// default: JSON logger (dobar za produkciju)
	log = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// ako želiš "dev friendly" tekstualni logger, promeni:
	// log = slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// SetLogger omogućava da zameniš default logger (npr. test vs prod).
func SetLogger(l *slog.Logger) {
	log = l
}

// Info ...
func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

// Warn ...
func Warn(msg string, args ...any) {
	log.Warn(msg, args...)
}

// Error ...
func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

// Debug ...
func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

// Fatal -> slog nema Fatal, pa ovde završavamo proces.
func Fatal(msg string, args ...any) {
	log.Error(msg, args...)
	os.Exit(1)
}

// EXAMPLES ...
// uspešna poruka
// logger.Info("User created", "id", user.ID, "email", user.Email)

// warning
// logger.Warn("Disk almost full", "free_gb", 1.2)

// error
// logger.Error("Failed to send email", "to", email, "error", err)

// fatal
// if err != nil {
//     logger.Fatal("Database connection failed", "error", err)
// }
