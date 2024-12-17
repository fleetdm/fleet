package fleet

import (
	"fmt"
	"strings"
)

const ServerSecretPrefix = "FLEET_SECRET_"

type MissingSecretsError struct {
	MissingSecrets []string
}

func (e MissingSecretsError) Error() string {
	secretVars := make([]string, 0, len(e.MissingSecrets))
	for _, secret := range e.MissingSecrets {
		secretVars = append(secretVars, fmt.Sprintf("\"$FLEET_SECRET_%s\"", secret))
	}
	plural := ""
	if len(secretVars) > 1 {
		plural = "s"
	}
	return fmt.Sprintf("Couldn't add. Variable%s %s missing", plural, strings.Join(secretVars, ", "))
}
