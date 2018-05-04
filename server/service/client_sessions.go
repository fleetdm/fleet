package service

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// Login attempts to login to the current Fleet instance. If login is successful,
// an auth token is returned.
func (c *Client) Login(email, password string) (string, error) {
	params := loginRequest{
		Username: email,
		Password: password,
	}

	response, err := c.Do("POST", "/api/v1/kolide/login", params)
	if err != nil {
		return "", errors.Wrap(err, "error making request")
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return "", notSetup()
	case http.StatusUnauthorized:
		return "", invalidLogin()
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("Received HTTP %d instead of HTTP 200", response.StatusCode)
	}

	var responseBody loginResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return "", errors.Wrap(err, "error decoding HTTP response body")
	}

	if responseBody.Err != nil {
		return "", errors.Wrap(err, "error setting up fleet instance")
	}

	return responseBody.Token, nil
}

// Logout attempts to logout to the current Fleet instance.
func (c *Client) Logout() error {
	response, err := c.AuthenticatedDo("POST", "/api/v1/kolide/logout", nil)
	if err != nil {
		return errors.Wrap(err, "error making request")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("Received HTTP %d instead of HTTP 200", response.StatusCode)
	}

	responeBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrap(err, "error reading response body")
	}

	var responseBody logoutResponse
	err = json.Unmarshal(responeBytes, &responseBody)
	if err != nil {
		return errors.Wrap(err, "error decoding HTTP response body")
	}

	if responseBody.Err != nil {
		return errors.Wrap(err, "error logging out of Fleet")
	}

	return nil
}
