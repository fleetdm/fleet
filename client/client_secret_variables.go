package client

import "github.com/fleetdm/fleet/v4/server/fleet"

func (c *Client) SaveSecretVariables(secretVariables []fleet.SecretVariable, dryRun bool) error {
	verb, path := "PUT", "/api/latest/fleet/spec/secret_variables"
	params := fleet.CreateSecretVariablesRequest{
		SecretVariables: secretVariables,
		DryRun:          dryRun,
	}
	var responseBody fleet.CreateSecretVariablesResponse
	return c.authenticatedRequest(params, verb, path, &responseBody)
}
