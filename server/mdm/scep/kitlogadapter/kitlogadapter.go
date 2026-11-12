// Package kitlogadapter provides a thin adapter that wraps *slog.Logger to
// implement the go-kit kitlog.Logger interface. This is used at the few
// remaining boundaries where third-party libraries (go-kit/kit, smallstep/scep, etc.)
// require a kitlog.Logger.
package kitlogadapter

import (
	"context"
	"log/slog"
)

// Logger wraps a *slog.Logger to implement the kitlog.Logger interface.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new kitlog adapter using the provided slog.Logger.
func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{logger: logger}
}

// Log implements kitlog.Logger. It converts key-value pairs to slog attributes
// and logs at the appropriate level based on the "level" key if present.
func (a *Logger) Log(keyvals ...any) error {
	if len(keyvals) == 0 {
		return nil
	}

	level := slog.LevelInfo
	msg := ""
	attrs := make([]slog.Attr, 0, len(keyvals)/2)

	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		val := keyvals[i+1]

		switch key {
		case "level":
			level = levelToSlog(val)
		case "msg":
			if s, ok := val.(string); ok {
				msg = s
			}
		case "ts":
			// Skip timestamp — slog handles this automatically
			continue
		default:
			attrs = append(attrs, slog.Any(key, val))
		}
	}

	a.logger.LogAttrs(context.Background(), level, msg, attrs...)
	return nil
}

// With returns a new Logger with the given key-value pairs added to every log entry.
func (a *Logger) With(keyvals ...any) *Logger {
	return &Logger{logger: a.logger.With(keyvals...)}
}

// levelToSlog converts a kitlog level value to slog.Level.
func levelToSlog(val any) slog.Level {
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
