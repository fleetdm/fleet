package service

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) SaveSecretVariables(secretVariables []fleet.SecretVariable, dryRun bool) error {
	verb, path := "PUT", "/api/latest/fleet/spec/secret_variables"
	params := createSecretVariablesRequest{
		SecretVariables: secretVariables,
	}
	var responseBody createSecretVariablesResponse
	return c.authenticatedRequestWithQuery(params, verb, path, &responseBody, fmt.Sprintf("dry_run=%t", dryRun))
}
