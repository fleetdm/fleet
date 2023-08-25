package logging

import (
	"os"

	"github.com/rs/zerolog/log"
)

// LogErrIfEnvNotSet logs if the environment variable is not set to "1".
func LogErrIfEnvNotSet(envVarName string, err error, message string) {
	actualValue := os.Getenv(envVarName)
	if actualValue != "1" {
		log.Info().Err(err).Msg(message)
	}
}
