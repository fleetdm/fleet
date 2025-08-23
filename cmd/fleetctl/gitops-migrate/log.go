package main

import (
	"context"
	"log/slog"
)

type LoggerKey struct{}

var loggerKey LoggerKey

// LoggerIntoContext burns a '*slog.Logger' into the provided context, making it
// retrievable via 'LoggerFromContext'.
//
// This is useful for passing contextual ('slog.With(...)', 'slog.Group(...)')
// loggers across function boundaries.
func LoggerIntoContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext attempts to retrieve a '*slog.Logger' from the provided
// context. If the logger key does not exist, or the value returned is not a
// '*slog.Logger', the default '*slog.Logger' is returned.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	// Attempt to fetch the value stored at 'loggerKey'.
	value := ctx.Value(loggerKey)
	if value == nil {
		return slog.Default()
	}

	// Attempt to assert its type as a '*slog.Logger'.
	logger, ok := value.(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return logger
}
