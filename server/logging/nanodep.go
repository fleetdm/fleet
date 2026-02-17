package logging

import (
	"fmt"
	"log/slog"

	nanodep_log "github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
)

// NanoDEPLogger is a logger adapter for nanodep.
type NanoDEPLogger struct {
	logger *slog.Logger
}

func NewNanoDEPLogger(logger *slog.Logger) *NanoDEPLogger {
	return &NanoDEPLogger{
		logger: logger,
	}
}

func (l *NanoDEPLogger) Info(keyvals ...interface{}) {
	msg, attrs := extractMsg(keyvals)
	l.logger.Info(msg, attrs...)
}

func (l *NanoDEPLogger) Debug(keyvals ...interface{}) {
	msg, attrs := extractMsg(keyvals)
	l.logger.Debug(msg, attrs...)
}

func (l *NanoDEPLogger) With(keyvals ...interface{}) nanodep_log.Logger {
	return &NanoDEPLogger{
		logger: l.logger.With(keyvals...),
	}
}

// extractMsg extracts the "msg" key-value pair from kitlog-style keyvals,
// returning the message string and the remaining keyvals.
func extractMsg(keyvals []interface{}) (string, []interface{}) {
	for i := 0; i < len(keyvals)-1; i += 2 {
		if key, ok := keyvals[i].(string); ok && key == "msg" {
			msg := fmt.Sprint(keyvals[i+1])
			remaining := make([]interface{}, 0, len(keyvals)-2)
			remaining = append(remaining, keyvals[:i]...)
			remaining = append(remaining, keyvals[i+2:]...)
			return msg, remaining
		}
	}
	return "", keyvals
}
