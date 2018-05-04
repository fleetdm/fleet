package service

import (
	"encoding/json"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// Setup attempts to setup the current Fleet instance. If setup is successful,
// an auth token is returned.
func (c *Client) Setup(email, password, org string) (string, error) {
	t := true
	params := setupRequest{
		Admin: &kolide.UserPayload{
			Admin:    &t,
			Username: &email,
			Email:    &email,
			Password: &password,
		},
		OrgInfo: &kolide.OrgInfo{
			OrgName: &org,
		},
		KolideServerURL: &c.addr,
	}

	response, err := c.Do("POST", "/api/v1/setup", params)
	if err != nil {
		return "", errors.Wrap(err, "error making request")
	}
	defer response.Body.Close()

	// If setup has already been completed, Kolide Fleet will not serve the
	// setup route, which will cause the request to 404
	if response.StatusCode == http.StatusNotFound {
		return "", setupAlready()
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("Received HTTP %d instead of HTTP 200", response.StatusCode)
	}

	var responseBody setupResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", errors.Wrap(err, "error decoding HTTP response body")
	}

	if responseBody.Err != nil {
		return "", errors.Wrap(err, "error setting up fleet instance")
	}

	return *responseBody.Token, nil
}
