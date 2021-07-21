package service

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

// CreateUser creates a new user, skipping the invitation process.
func (c *Client) CreateUser(p fleet.UserPayload) error {
	verb, path := "POST", "/api/v1/fleet/users/admin"
	var responseBody createUserResponse

	return c.authenticatedRequest(p, verb, path, &responseBody)
}

// ListUsers retrieves the list of users.
func (c *Client) ListUsers() ([]fleet.User, error) {
	verb, path := "GET", "/api/v1/fleet/users"
	var responseBody listUsersResponse

	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Users, nil
}

// ApplyUsersRoleSecretSpec applies the global and team roles for users.
func (c *Client) ApplyUsersRoleSecretSpec(spec *fleet.UsersRoleSpec) error {
	req := applyUserRoleSpecsRequest{Spec: spec}
	verb, path := "POST", "/api/v1/fleet/users/roles/spec"
	var responseBody applyUserRoleSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

func (c *Client) userIdFromEmail(email string) (uint, error) {
	verb, path := "POST", "/api/v1/fleet/translate"
	var responseBody translatorResponse

	params := translatorRequest{List: []fleet.TranslatePayload{
		{
			Type:    fleet.TranslatorTypeUserEmail,
			Payload: fleet.StringIdentifierToIDPayload{Identifier: email},
		},
	}}

	err := c.authenticatedRequest(&params, verb, path, &responseBody)
	if err != nil {
		return 0, err
	}
	if len(responseBody.List) != 1 {
		return 0, errors.New("Expected 1 item translated, got none")
	}
	return responseBody.List[0].Payload.ID, nil
}

// DeleteUser deletes the user specified by the email
func (c *Client) DeleteUser(email string) error {
	userID, err := c.userIdFromEmail(email)
	if err != nil {
		return err
	}

	verb, path := "DELETE", fmt.Sprintf("/api/v1/fleet/users/%d", userID)
	var responseBody deleteUserResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}
