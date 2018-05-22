package service

import (
	"encoding/json"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// GetServerSettings fetches the server settings from the server API
func (c *Client) GetServerSettings() (*kolide.ServerSettings, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/kolide/config", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/kolide/config")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get server settings received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody appConfigResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get app config response")
	}

	if responseBody.Err != nil {
		return nil, errors.Errorf("get server settings: %s", responseBody.Err)
	}

	return responseBody.ServerSettings, nil
}
