package service

import (
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

// Setup attempts to setup the current Fleet instance. If setup is successful,
// an auth token is returned.
func (c *Client) Setup(email, name, password, org string) (string, error) {
	params := setupRequest{
		Admin: &fleet.UserPayload{
			Email:    &email,
			Name:     &name,
			Password: &password,
		},
		OrgInfo: &fleet.OrgInfo{
			OrgName: &org,
		},
		ServerURL: &c.addr,
	}

	response, err := c.Do("POST", "/api/v1/setup", "", params)
	if err != nil {
		return "", errors.Wrap(err, "POST /api/v1/setup")
	}
	defer response.Body.Close()

	// If setup has already been completed, Fleet will not serve the setup
	// route, which will cause the request to 404
	if response.StatusCode == http.StatusNotFound {
		return "", setupAlreadyErr{}
	}
	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf(
			"setup received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("setup got HTTP %d, expected 200", response.StatusCode)
	}

	var responseBody setupResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", errors.Wrap(err, "decode setup response")
	}

	if responseBody.Err != nil {
		return "", errors.Errorf("setup: %s", responseBody.Err)
	}

	return *responseBody.Token, nil
}
