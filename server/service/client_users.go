package service

import (
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

// CreateUser creates a new user, skipping the invitation process.
func (c *Client) CreateUser(p kolide.UserPayload) error {
	verb, path := "POST", "/api/v1/fleet/users/admin"
	response, err := c.AuthenticatedDo(verb, path, "", p)
	if err != nil {
		return errors.Wrapf(err, "%s %s", verb, path)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf(
			"create user received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody createUserResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode create user response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("create user: %s", responseBody.Err)
	}

	return nil
}
