package table

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger is a wrapper around zerolog, which we use for tables
// using the go-kit logger
type Logger struct {
	zerolog.Logger
}

// Log logs a message, implementing log.Logger interface
func (l *Logger) Log(keyValuePairs ...interface{}) error {
	log.Logger.Info().Msg(fmt.Sprint(keyValuePairs...))
	return nil
}

// NewOsqueryLogger returns the Logger struct.
func NewOsqueryLogger() *Logger {
	// Return a Logger struct with our global logger, and use the existing global config for the log level.
	return &Logger{log.Logger}
}
