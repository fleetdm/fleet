package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListTeams retrieves the list of teams.
func (c *Client) ListTeams() ([]fleet.Team, error) {
	verb, path := "GET", "/api/v1/fleet/teams"
	var responseBody listTeamsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Teams, nil
}

// ApplyTeams sends the list of Teams to be applied to the
// Fleet instance.
func (c *Client) ApplyTeams(specs []*fleet.TeamSpec) error {
	req := applyTeamSpecsRequest{Specs: specs}
	verb, path := "POST", "/api/v1/fleet/spec/teams"
	var responseBody applyTeamSpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// ApplyPolicies sends the list of Policies to be applied to the
// Fleet instance.
func (c *Client) ApplyPolicies(specs []*fleet.PolicySpec) error {
	req := applyPolicySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/v1/fleet/spec/policies"
	var responseBody applyPolicySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
