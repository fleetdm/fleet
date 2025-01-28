package logging

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogErrIfEnvNotSet logs an info error if the environment variable is not set to "1".
func LogErrIfEnvNotSet(envVarName string, err error, message string) {
	LogErrIfEnvNotSetWithEvent(envVarName, err, message, log.Info())
}

// LogErrIfEnvNotSetDebug logs a debug error if the environment variable is not set to "1".
func LogErrIfEnvNotSetDebug(envVarName string, err error, message string) {
	LogErrIfEnvNotSetWithEvent(envVarName, err, message, log.Debug())
}

// LogErrIfEnvNotSetWithEvent logs if the environment variable is not set to "1".
func LogErrIfEnvNotSetWithEvent(envVarName string, err error, message string, event *zerolog.Event) {
	actualValue := os.Getenv(envVarName)
	if actualValue != "1" {
		event.Err(err).Msg(message)
	}
}
