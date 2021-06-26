package service

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

// Login attempts to login to the current Fleet instance. If login is successful,
// an auth token is returned.
func (c *Client) Login(email, password string) (string, error) {
	params := loginRequest{
		Email:    email,
		Password: password,
	}

	response, err := c.Do("POST", "/api/v1/fleet/login", "", params)
	if err != nil {
		return "", errors.Wrap(err, "POST /api/v1/fleet/login")
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return "", notSetupErr{}
	}
	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf(
			"login received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody loginResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", errors.Wrap(err, "decode login response")
	}

	if responseBody.Err != nil {
		return "", errors.Errorf("login: %s", responseBody.Err)
	}

	return responseBody.Token, nil
}

// Logout attempts to logout to the current Fleet instance.
func (c *Client) Logout() error {
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/logout", "", nil)
	if err != nil {
		return errors.Wrap(err, "POST /api/v1/fleet/logout")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf(
			"logout received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody logoutResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode logout response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("logout: %s", responseBody.Err)
	}

	return nil
}
