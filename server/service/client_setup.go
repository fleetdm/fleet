package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
			OrgName: org,
		},
		ServerURL: &c.addr,
	}

	response, err := c.Do("POST", "/api/v1/setup", "", params)
	if err != nil {
		return "", fmt.Errorf("POST /api/v1/setup: %w", err)
	}
	defer response.Body.Close()

	// If setup has already been completed, Fleet will not serve the setup
	// route, which will cause the request to 404
	if response.StatusCode == http.StatusNotFound {
		return "", setupAlreadyErr{}
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"setup received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("setup got HTTP %d, expected 200", response.StatusCode)
	}

	var responseBody setupResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", fmt.Errorf("decode setup response: %w", err)
	}

	if responseBody.Err != nil {
		return "", fmt.Errorf("setup: %s", responseBody.Err)
	}

	return *responseBody.Token, nil
}
