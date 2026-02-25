package logging

import (
	"context"
	"log/slog"
)

// Logger wraps a slog.Logger to implement the kitlog.Logger interface.
// This allows gradual migration from kitlog to slog by providing a drop-in
// replacement that uses slog under the hood.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new adapter that implements kitlog.Logger
// using the provided slog.Logger. It returns *Logger to preserve
// type information, allowing callers to access SlogLogger() directly.
func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// Log implements kitlog.Logger. It converts key-value pairs to slog attributes
// and logs at the appropriate level based on the "level" key if present.
func (a *Logger) Log(keyvals ...any) error {
	if len(keyvals) == 0 {
		return nil
	}

	// Extract level and message from keyvals
	level := slog.LevelInfo
	msg := ""
	attrs := make([]slog.Attr, 0, len(keyvals)/2)

	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			// If key isn't a string, skip this pair
			continue
		}
		val := keyvals[i+1]

		switch key {
		case "level":
			level = kitlogLevelToSlog(val)
		case "msg":
			if s, ok := val.(string); ok {
				msg = s
			}
		case "ts":
			// Skip timestamp - slog handles this automatically
			continue
		default:
			attrs = append(attrs, slog.Any(key, val))
		}
	}

	a.logger.LogAttrs(context.Background(), level, msg, attrs...)
	return nil
}

// With returns a new logger with the given key-value pairs added to every log.
// It returns *Logger (not kitlog.Logger) to preserve type information,
// allowing callers to access SlogLogger() without type assertions.
func (a *Logger) With(keyvals ...any) *Logger {
	return &Logger{
		logger: a.logger.With(keyvals...),
	}
}

// kitlogLevelToSlog converts a kitlog level value to slog.Level.
func kitlogLevelToSlog(val any) slog.Level {
	// kitlog uses level.Value which implements fmt.Stringer
	// Common values are "debug", "info", "warn", "error"
	var levelStr string

	switch v := val.(type) {
	case string:
		levelStr = v
	case interface{ String() string }:
		levelStr = v.String()
	default:
		return slog.LevelInfo
	}

	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SlogLogger returns the underlying slog.Logger.
// This is useful when migrating code from kitlog to slog.
func (a *Logger) SlogLogger() *slog.Logger {
	return a.logger
}

// Wrap slog's ErrorContext method.
func (a *Logger) ErrorContext(ctx context.Context, msg string, keyvals ...any) {
	a.logger.ErrorContext(ctx, msg, keyvals...)
}

// Wrap slog's WarnContext method.
func (a *Logger) WarnContext(ctx context.Context, msg string, keyvals ...any) {
	a.logger.WarnContext(ctx, msg, keyvals...)
}

// Wrap slog's InfoContext method.
func (a *Logger) InfoContext(ctx context.Context, msg string, keyvals ...any) {
	a.logger.InfoContext(ctx, msg, keyvals...)
}

// Wrap slog's DebugContext method.
func (a *Logger) DebugContext(ctx context.Context, msg string, keyvals ...any) {
	a.logger.DebugContext(ctx, msg, keyvals...)
}
