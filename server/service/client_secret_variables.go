package service

import "github.com/fleetdm/fleet/v4/server/fleet"

func (c *Client) SaveSecretVariables(secretVariables []fleet.SecretVariable) error {
	verb, path := "PUT", "/api/latest/fleet/spec/secret_variables"
	params := secretVariablesRequest{
		secretVariables,
	}
	var responseBody secretVariablesResponse
	return c.authenticatedRequest(params, verb, path, &responseBody)
}
