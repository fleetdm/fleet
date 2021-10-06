package service

import (
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

// ApplyAppConfig sends the application config to be applied to the Fleet instance.
func (c *Client) ApplyAppConfig(payload interface{}) error {
	response, err := c.AuthenticatedDo("PATCH", "/api/v1/fleet/config", "", payload)
	if err != nil {
		return errors.Wrap(err, "PATCH /api/v1/fleet/config")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf(
			"apply config received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody appConfigResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode apply config response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("apply config: %s", responseBody.Err)
	}
	return nil
}

// GetAppConfig fetches the application config from the server API
func (c *Client) GetAppConfig() (*fleet.EnrichedAppConfig, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/config", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/fleet/config")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get config received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody *fleet.EnrichedAppConfig
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get config response")
	}

	return responseBody, nil
}

// GetEnrollSecretSpec fetches the enroll secrets stored on the server
func (c *Client) GetEnrollSecretSpec() (*fleet.EnrollSecretSpec, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/spec/enroll_secret", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/fleet/spec/enroll_secret")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get enroll_secrets received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getEnrollSecretSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get enroll secret spec response")
	}

	if responseBody.Err != nil {
		return nil, errors.Errorf("get enroll secret spec: %s", responseBody.Err)
	}

	return responseBody.Spec, nil
}

// ApplyEnrollSecretSpec applies the enroll secrets.
func (c *Client) ApplyEnrollSecretSpec(spec *fleet.EnrollSecretSpec) error {
	req := applyEnrollSecretSpecRequest{Spec: spec}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/spec/enroll_secret", "", req)
	if err != nil {
		return errors.Wrap(err, "POST /api/v1/fleet/spec/enroll_secret")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf(
			"apply enroll secret received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody applyEnrollSecretSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode apply enroll secret response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("apply enroll secret: %s", responseBody.Err)
	}
	return nil
}
