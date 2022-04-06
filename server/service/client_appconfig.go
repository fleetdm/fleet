package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/kolide/kit/version"
)

// ApplyAppConfig sends the application config to be applied to the Fleet instance.
func (c *Client) ApplyAppConfig(payload interface{}) error {
	verb, path := "PATCH", "/api/latest/fleet/config"
	var responseBody appConfigResponse
	return c.authenticatedRequest(payload, verb, path, &responseBody)
}

// GetAppConfig fetches the application config from the server API
func (c *Client) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	verb, path := "GET", "/api/latest/fleet/config"
	var responseBody fleet.EnrichedAppConfig
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return &responseBody, err
}

// GetEnrollSecretSpec fetches the enroll secrets stored on the server
func (c *Client) GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error) {
	verb, path := "GET", "/api/latest/fleet/spec/enroll_secret"
	var responseBody getEnrollSecretSpecResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Spec, err
}

// ApplyEnrollSecretSpec applies the enroll secrets.
func (c *Client) ApplyEnrollSecretSpec(spec *fleet.EnrollSecretSpec) error {
	req := applyEnrollSecretSpecRequest{Spec: spec}
	verb, path := "POST", "/api/latest/fleet/spec/enroll_secret"
	var responseBody applyEnrollSecretSpecResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

func (c *Client) Version() (*version.Info, error) {
	verb, path := "GET", "/api/latest/fleet/version"
	var responseBody versionResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Info, err
}
