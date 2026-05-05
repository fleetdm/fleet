package logging

import (
	"context"
	"log/slog"

	nanodep_log "github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
)

// NanoDEPLogger is a logger adapter for nanodep.
type NanoDEPLogger struct {
	ctx    context.Context
	logger *slog.Logger
}

func NewNanoDEPLogger(ctx context.Context, logger *slog.Logger) *NanoDEPLogger {
	return &NanoDEPLogger{
		ctx:    ctx,
		logger: logger,
	}
}

func (l *NanoDEPLogger) Info(keyvals ...any) {
	l.logger.InfoContext(l.ctx, "", keyvals...)
}

func (l *NanoDEPLogger) Debug(keyvals ...any) {
	l.logger.DebugContext(l.ctx, "", keyvals...)
}

func (l *NanoDEPLogger) With(keyvals ...any) nanodep_log.Logger {
	return &NanoDEPLogger{
		ctx:    l.ctx,
		logger: l.logger.With(keyvals...),
	}
}
