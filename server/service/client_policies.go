package service

import (
	"net/url"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) CreateGlobalPolicy(name, query, description, resolution, platform string) error {
	req := fleet.CreatePolicyRequest{
		Name:        name,
		Query:       query,
		Description: description,
		Resolution:  resolution,
		Platform:    platform,
	}
	var responseBody fleet.CreatePolicyResponse
	return c.authenticatedRequest(req, "POST", "/api/latest/fleet/policies", &responseBody)
}

// ApplyPolicies sends the list of Policies to be applied to the
// Fleet instance.
func (c *Client) ApplyPolicies(specs []*fleet.PolicySpec) error {
	req := fleet.ApplyPolicySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/policies"
	var responseBody fleet.ApplyPolicySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}

// GetPolicies retrieves the policies of a scope: "All fleets" when teamID is nil, "Unassigned" when it's 0, and a fleet
// otherwise. Inherited policies are excluded.
func (c *Client) GetPolicies(teamID *uint) ([]*fleet.Policy, error) {
	var query string
	if teamID != nil {
		query = url.Values{"fleet_id": {strconv.FormatUint(uint64(*teamID), 10)}}.Encode()
	}
	var responseBody fleet.ListPoliciesResponse
	if err := c.authenticatedRequestWithQuery(nil, "GET", "/api/latest/fleet/policies", &responseBody, query); err != nil {
		return nil, err
	}
	return responseBody.Policies, nil
}

// DeletePolicies deletes several policies, which can be in any scope.
func (c *Client) DeletePolicies(ids []uint) error {
	var responseBody fleet.DeletePoliciesResponse
	return c.authenticatedRequest(fleet.DeletePoliciesRequest{IDs: ids}, "POST", "/api/latest/fleet/policies/delete", &responseBody)
}
