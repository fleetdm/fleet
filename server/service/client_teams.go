package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListTeams retrieves the list of teams.
func (c *Client) ListTeams(query string) ([]fleet.Team, error) {
	verb, path := "GET", "/api/latest/fleet/teams"
	var responseBody listTeamsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.Teams, nil
}

// CreateTeam creates a new team.
func (c *Client) CreateTeam(teamPayload fleet.TeamPayload) (*fleet.Team, error) {
	req := createTeamRequest{
		TeamPayload: teamPayload,
	}
	verb, path := "POST", "/api/latest/fleet/teams"
	var responseBody teamResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Team, nil
}

func (c *Client) GetTeam(teamID uint) (*fleet.Team, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/teams/%d", teamID)
	var responseBody getTeamResponse
	if err := c.authenticatedRequest(getTeamRequest{}, verb, path, &responseBody); err != nil {
		return nil, err
	}
	return responseBody.Team, nil
}

// DeleteTeam deletes a team.
func (c *Client) DeleteTeam(teamID uint) error {
	verb, path := "DELETE", "/api/latest/fleet/teams/"+strconv.FormatUint(uint64(teamID), 10)
	var responseBody deleteTeamResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

// ApplyTeams sends the list of Teams to be applied to the
// Fleet instance.
func (c *Client) ApplyTeams(specs []json.RawMessage, opts fleet.ApplySpecOptions) (map[string]uint, error) {
	verb, path := "POST", "/api/latest/fleet/spec/teams"
	var responseBody applyTeamSpecsResponse
	err := c.authenticatedRequestWithQuery(map[string]interface{}{"specs": specs}, verb, path, &responseBody, opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return responseBody.TeamIDsByName, nil
}

// ApplyTeamProfiles sends the list of profiles to be applied for the specified
// team.
func (c *Client) ApplyTeamProfiles(tmName string, profiles [][]byte, opts fleet.ApplySpecOptions) error {
	verb, path := "POST", "/api/latest/fleet/mdm/apple/profiles/batch"
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return err
	}
	query.Add("team_name", tmName)
	return c.authenticatedRequestWithQuery(map[string]interface{}{"profiles": profiles}, verb, path, nil, query.Encode())
}

// ApplyPolicies sends the list of Policies to be applied to the
// Fleet instance.
func (c *Client) ApplyPolicies(specs []*fleet.PolicySpec) error {
	req := applyPolicySpecsRequest{Specs: specs}
	verb, path := "POST", "/api/latest/fleet/spec/policies"
	var responseBody applyPolicySpecsResponse
	return c.authenticatedRequest(req, verb, path, &responseBody)
}
