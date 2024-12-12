package fleet

import (
	"fmt"
	"strings"
)

const FLEET_SECRET_PREFIX = "FLEET_SECRET_"

type MissingSecretsError struct {
	MissingSecrets []string
}

func (e MissingSecretsError) Error() string {
	return fmt.Sprintf("secret variables not present in database: %s", strings.Join(e.MissingSecrets, ", "))
}
