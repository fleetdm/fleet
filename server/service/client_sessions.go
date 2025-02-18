package service

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Login attempts to login to the current Fleet instance. If login is successful,
// an auth token is returned.
func (c *Client) Login(email, password string) (string, error) {
	params := loginRequest{
		Email:    email,
		Password: password,
	}

	response, err := c.Do("POST", "/api/latest/fleet/login", "", params)
	if err != nil {
		return "", fmt.Errorf("POST /api/latest/fleet/login: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return "", notSetupErr{}
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"login received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody loginResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}

	if responseBody.Err != nil {
		return "", fmt.Errorf("login: %s", responseBody.Err)
	}

	return responseBody.Token, nil
}

// Logout attempts to logout to the current Fleet instance.
func (c *Client) Logout() error {
	verb, path := "POST", "/api/latest/fleet/logout"
	var responseBody logoutResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
