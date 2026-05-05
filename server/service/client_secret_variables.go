package service

import "github.com/fleetdm/fleet/v4/server/fleet"

func (c *Client) SaveSecretVariables(secretVariables []fleet.SecretVariable, dryRun bool) error {
	verb, path := "PUT", "/api/latest/fleet/spec/secret_variables"
	params := createSecretVariablesRequest{
		SecretVariables: secretVariables,
		DryRun:          dryRun,
	}
	var responseBody createSecretVariablesResponse
	return c.authenticatedRequest(params, verb, path, &responseBody)
}
