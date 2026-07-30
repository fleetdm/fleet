package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Login attempts to login to the current Fleet instance. If login is successful,
// an auth token is returned.
func (c *Client) Login(email, password string) (string, error) {
	params := fleet.LoginRequest{
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

	var responseBody fleet.LoginResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", fmt.Errorf("decode login response: %w", err)
	}

	if responseBody.Err != nil {
		return "", fmt.Errorf("login: %s", responseBody.Err)
	}

	return responseBody.Token, nil
}

// SSOSettings returns the SSO settings for the current Fleet instance.
// This endpoint is unauthenticated and can be called before login.
func (c *Client) SSOSettings() (*fleet.SessionSSOSettings, error) {
	response, err := c.Do("GET", "/api/v1/fleet/sso", "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/sso: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get SSO settings received status %d %s", response.StatusCode, extractServerErrorText(response.Body))
	}

	var responseBody struct {
		Settings *fleet.SessionSSOSettings `json:"settings"`
	}
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("decode SSO settings response: %w", err)
	}

	return responseBody.Settings, nil
}

// Logout attempts to logout to the current Fleet instance.
func (c *Client) Logout() error {
	verb, path := "POST", "/api/latest/fleet/logout"
	var responseBody fleet.LogoutResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
