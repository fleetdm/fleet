package logging

import (
	"os"

	"github.com/rs/zerolog/log"
)

// ConditionalLog logs if the environment variable is set to "1".
func LogErrIfEnvNotSet(envVarName string, err error, message string) {
	actualValue := os.Getenv(envVarName)
	if actualValue == "1" {
		log.Info().Err(err).Msg(message)
	}
}
