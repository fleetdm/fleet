package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
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
