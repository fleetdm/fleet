package service

import (
	"encoding/json"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// ApplyAppConfig sends the application config to be applied to the Fleet instance.
func (c *Client) ApplyAppConfig(payload *kolide.AppConfigPayload) error {
	response, err := c.AuthenticatedDo("PATCH", "/api/v1/kolide/config", payload)
	if err != nil {
		return errors.Wrap(err, "PATCH /api/v1/kolide/config")
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
func (c *Client) GetAppConfig() (*kolide.AppConfigPayload, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/kolide/config", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/kolide/config")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get config received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody *kolide.AppConfigPayload
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get config response")
	}

	return responseBody, nil
}

// GetServerSettings fetches the server settings from the server API
func (c *Client) GetServerSettings() (*kolide.ServerSettings, error) {
	appConfig, err := c.GetAppConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get server settings")
	}
	return appConfig.ServerSettings, nil
}
