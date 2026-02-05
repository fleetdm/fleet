package logging

import (
	"context"
	"log/slog"
	"slices"

	kitlog "github.com/go-kit/log"
)

// KitlogAdapter wraps a slog.Logger to implement the kitlog.Logger interface.
// This allows gradual migration from kitlog to slog by providing a drop-in
// replacement that uses slog under the hood.
type KitlogAdapter struct {
	logger *slog.Logger
	// attrs holds any attributes added via With()
	attrs []any
}

// NewKitlogAdapter creates a new adapter that implements kitlog.Logger
// using the provided slog.Logger.
func NewKitlogAdapter(logger *slog.Logger) kitlog.Logger {
	return &KitlogAdapter{
		logger: logger,
	}
}

// Log implements kitlog.Logger. It converts key-value pairs to slog attributes
// and logs at the appropriate level based on the "level" key if present.
func (a *KitlogAdapter) Log(keyvals ...any) error {
	if len(keyvals) == 0 && len(a.attrs) == 0 {
		return nil
	}

	// Combine pre-set attrs with new keyvals
	allKeyvals := slices.Concat(a.attrs, keyvals)

	// Extract level and message from keyvals
	level := slog.LevelInfo
	msg := ""
	attrs := make([]slog.Attr, 0, len(allKeyvals)/2)

	for i := 0; i < len(allKeyvals)-1; i += 2 {
		key, ok := allKeyvals[i].(string)
		if !ok {
			// If key isn't a string, skip this pair
			continue
		}
		val := allKeyvals[i+1]

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
func (a *KitlogAdapter) With(keyvals ...any) kitlog.Logger {
	return &KitlogAdapter{
		logger: a.logger,
		attrs:  slices.Concat(a.attrs, keyvals),
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

// Ensure KitlogAdapter implements kitlog.Logger at compile time.
var _ kitlog.Logger = (*KitlogAdapter)(nil)
