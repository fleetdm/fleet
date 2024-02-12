package logging

import (
	"github.com/rs/zerolog"
	"os"

	"github.com/rs/zerolog/log"
)

// LogErrIfEnvNotSet logs if the environment variable is not set to "1".
func LogErrIfEnvNotSet(envVarName string, err error, message string) {
	LogErrIfEnvNotSetWithEvent(envVarName, err, message, log.Info())
}

// LogErrIfEnvNotSetWithEvent logs if the environment variable is not set to "1".
func LogErrIfEnvNotSetWithEvent(envVarName string, err error, message string, event *zerolog.Event) {
	actualValue := os.Getenv(envVarName)
	if actualValue != "1" {
		event.Err(err).Msg(message)
	}
}
