// Package logger builds the process-wide structured logger. log/slog is
// used directly (stdlib, allocation-light) instead of a third-party
// logging library.
package logger

import (
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	level := slog.LevelInfo
	if env == "development" {
		level = slog.LevelDebug
	}

	var handler slog.Handler
	if env == "development" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
