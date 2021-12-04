package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ApplyAppConfig sends the application config to be applied to the Fleet instance.
func (c *Client) ApplyAppConfig(payload interface{}) error {
	response, err := c.AuthenticatedDo("PATCH", "/api/v1/fleet/config", "", payload)
	if err != nil {
		return fmt.Errorf("PATCH /api/v1/fleet/config: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"apply config received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody appConfigResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode apply config response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("apply config: %s", responseBody.Err)
	}
	return nil
}

// GetAppConfig fetches the application config from the server API
func (c *Client) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/config", "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/config: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get config received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody *fleet.EnrichedAppConfig
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get config response: %w", err)
	}

	return responseBody, nil
}

// GetEnrollSecretSpec fetches the enroll secrets stored on the server
func (c *Client) GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/spec/enroll_secret", "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/spec/enroll_secret: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get enroll_secrets received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getEnrollSecretSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode get enroll secret spec response: %w", err)
	}

	if responseBody.Err != nil {
		return nil, fmt.Errorf("get enroll secret spec: %s", responseBody.Err)
	}

	return responseBody.Spec, nil
}

// ApplyEnrollSecretSpec applies the enroll secrets.
func (c *Client) ApplyEnrollSecretSpec(spec *fleet.EnrollSecretSpec) error {
	req := applyEnrollSecretSpecRequest{Spec: spec}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/spec/enroll_secret", "", req)
	if err != nil {
		return fmt.Errorf("POST /api/v1/fleet/spec/enroll_secret: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"apply enroll secret received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody applyEnrollSecretSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode apply enroll secret response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("apply enroll secret: %s", responseBody.Err)
	}
	return nil
}
