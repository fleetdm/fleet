package service

import "github.com/fleetdm/fleet/v4/server/fleet"

// ApplyMicrosoftGraphCredentials declaratively reconciles the server's Microsoft Graph credentials to creds.
func (c *Client) ApplyMicrosoftGraphCredentials(creds []fleet.MicrosoftGraphCredential, dryRun bool) error {
	req := applyMicrosoftGraphCredentialsRequest{MicrosoftGraphCredentials: creds, DryRun: dryRun}
	var responseBody applyMicrosoftGraphCredentialsResponse
	return c.authenticatedRequest(req, "PUT", "/api/latest/fleet/microsoft_graph_credentials", &responseBody)
}

// GetMicrosoftGraphCredentials returns the stored credentials with their per-tenant sync status.
func (c *Client) GetMicrosoftGraphCredentials() ([]*fleet.MicrosoftGraphCredentialMetadata, error) {
	var responseBody listMicrosoftGraphCredentialsResponse
	err := c.authenticatedRequest(nil, "GET", "/api/latest/fleet/microsoft_graph_credentials", &responseBody)
	return responseBody.MicrosoftGraphCredentials, err
}
