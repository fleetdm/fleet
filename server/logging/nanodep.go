package logging

import (
	nanodep_log "github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// NanoDEPLogger is a logger adapter for nanodep.
type NanoDEPLogger struct {
	logger kitlog.Logger
}

func NewNanoDEPLogger(logger kitlog.Logger) *NanoDEPLogger {
	return &NanoDEPLogger{
		logger: logger,
	}
}

func (l *NanoDEPLogger) Info(keyvals ...interface{}) {
	level.Info(l.logger).Log(keyvals...)
}

func (l *NanoDEPLogger) Debug(keyvals ...interface{}) {
	level.Debug(l.logger).Log(keyvals...)
}

func (l *NanoDEPLogger) With(keyvals ...interface{}) nanodep_log.Logger {
	newLogger := kitlog.With(l.logger, keyvals...)
	return &NanoDEPLogger{
		logger: newLogger,
	}
}
