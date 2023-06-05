package table

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	zerolog.Logger
}

// Fatal logs a fatal message, just like Kolide's logutil.Fatal.
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal().Msg(fmt.Sprintln(args...))
}

// Log logs a message, just like Kolide's logutil.Log
func (l *Logger) Log(keyvals ...interface{}) error {
	// Implement this method to satisfy the github.com/go-kit/log.Logger interface.
	// You might want to use zerolog's log.Logger.Info() or similar here.
	return nil
}

// SetLevelKey sets the level key, just like Kolide's logutil.SetLevelKey.
func (l *Logger) SetLevelKey(key interface{}) *Logger {
	// Note: zerolog doesn't support changing the level key dynamically like logutil, so we just log it as a warning.
	l.Logger.Warn().Msgf("Attempted to set level key to: %v", key)

	return l
}

// NewServerLogger returns a new server logger, just like Kolide's logutil.NewServerLogger.
func NewKolideLogger(debug bool) *Logger {
	// Note: we're just setting the global level here, not creating a new logger
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	return &Logger{log.Logger}
}
