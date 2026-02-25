package client

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) CreateGlobalPolicy(name, query, description, resolution, platform string) error {
	req := fleet.GlobalPolicyRequest{
		Name:        name,
		Query:       query,
		Description: description,
		Resolution:  resolution,
		Platform:    platform,
	}
	verb, path := "POST", "/api/latest/fleet/global/policies"
	var responseBody fleet.GlobalPolicyResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// ApplyPolicies sends the list of Policies to be applied to the
// Fleet instance.
func (c *Client) ApplyPolicies(specs []*fleet.PolicySpec) error {
	req := fleet.ApplyPolicySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/policies"
	var responseBody fleet.ApplyPolicySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetPolicies retrieves the list of Policies. Inherited policies are excluded.
func (c *Client) GetPolicies(teamID *uint) ([]*fleet.Policy, error) {
	verb, path := "GET", ""
	if teamID != nil {
		path = fmt.Sprintf("/api/latest/fleet/teams/%d/policies", *teamID)
	} else {
		path = "/api/latest/fleet/policies"
	}
	// The response body also works for fleet.ListTeamPoliciesResponse because they contain some of the same members.
	var responseBody fleet.ListGlobalPoliciesResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Policies, nil
}

// DeletePolicies deletes several policies.
func (c *Client) DeletePolicies(teamID *uint, ids []uint) error {
	verb, path := "POST", ""
	req := fleet.DeleteTeamPoliciesRequest{IDs: ids}
	if teamID != nil {
		path = fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", *teamID)
		req.TeamID = *teamID
	} else {
		path = "/api/latest/fleet/policies/delete"
	}
	// The response body also works for fleet.DeleteTeamPoliciesResponse because they contain some of the same members.
	var responseBody fleet.DeleteGlobalPoliciesResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
