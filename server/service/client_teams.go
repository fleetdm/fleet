package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// ListTeams retrieves the list of teams.
func (c *Client) ListTeams(query string) ([]fleet.Team, error) {
	verb, path := "GET", "/api/latest/fleet/fleets"
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
	verb, path := "POST", "/api/latest/fleet/fleets"
	var responseBody teamResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Team, nil
}

func (c *Client) GetTeam(teamID uint) (*fleet.Team, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID)
	var responseBody getTeamResponse
	if err := c.authenticatedRequest(nil, verb, path, &responseBody); err != nil {
		return nil, err
	}
	return responseBody.Team, nil
}

// DeleteTeam deletes a team.
func (c *Client) DeleteTeam(teamID uint) error {
	verb, path := "DELETE", "/api/latest/fleet/fleets/"+strconv.FormatUint(uint64(teamID), 10)
	var responseBody deleteTeamResponse
	return c.authenticatedRequest(nil, verb, path, &responseBody)
}

// ApplyTeams sends the list of Teams to be applied to the
// Fleet instance.
func (c *Client) ApplyTeams(specs []json.RawMessage, opts fleet.ApplyTeamSpecOptions) (map[string]uint, error) {
	verb, path := "POST", "/api/latest/fleet/spec/fleets"
	var responseBody applyTeamSpecsResponse
	// Rewrite deprecated key names in each team spec to use the new names.
	rules := endpointer.ExtractAliasRules(fleet.TeamSpec{})
	rewritten := make([]json.RawMessage, len(specs))
	for i, spec := range specs {
		updated, err := endpointer.RewriteOldToNewKeys(spec, rules)
		if err != nil {
			return nil, err
		}
		rewritten[i] = updated
	}
	params := map[string]any{"specs": rewritten}
	if opts.DryRun && opts.DryRunAssumptions != nil {
		params["dry_run_assumptions"] = opts.DryRunAssumptions
	}
	err := c.authenticatedRequestWithQuery(params, verb, path, &responseBody, opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return responseBody.TeamIDsByName, nil
}

// PatchFleet sends a partial update to the specified team.
func (c *Client) PatchFleet(teamID uint, payload fleet.TeamPayload) error {
	verb, path := "PATCH", "/api/latest/fleet/fleets/"+strconv.FormatUint(uint64(teamID), 10)
	var resp teamResponse
	return c.authenticatedRequest(payload, verb, path, &resp)
}

// ApplyTeamProfiles sends the list of profiles to be applied for the specified
// team.
func (c *Client) ApplyTeamProfiles(tmName string, profiles []fleet.MDMProfileBatchPayload, opts fleet.ApplyTeamSpecOptions) error {
	verb, path := "POST", "/api/latest/fleet/mdm/profiles/batch"
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return err
	}
	query.Add("fleet_name", tmName)
	if opts.DryRunAssumptions != nil && opts.DryRunAssumptions.WindowsEnabledAndConfigured.Valid {
		query.Add("assume_enabled", strconv.FormatBool(opts.DryRunAssumptions.WindowsEnabledAndConfigured.Value))
	}
	return c.authenticatedRequestWithQuery(map[string]interface{}{"profiles": profiles}, verb, path, nil, query.Encode())
}

// ApplyTeamScripts sends the list of scripts to be applied for the specified
// team.
func (c *Client) ApplyTeamScripts(tmName string, scripts []fleet.ScriptPayload, opts fleet.ApplySpecOptions) ([]fleet.ScriptResponse, error) {
	verb, path := "POST", "/api/latest/fleet/scripts/batch"
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	query.Add("fleet_name", tmName)

	var resp fleet.BatchSetScriptsResponse
	err = c.authenticatedRequestWithQuery(map[string]interface{}{"scripts": scripts}, verb, path, &resp, query.Encode())
	return resp.Scripts, err
}

func (c *Client) ApplyTeamSoftwareInstallers(tmName string, softwareInstallers []fleet.SoftwareInstallerPayload, opts fleet.ApplySpecOptions) ([]fleet.SoftwarePackageResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	query.Add("fleet_name", tmName)
	return c.applySoftwareInstallers(softwareInstallers, query, opts.DryRun)
}

func (c *Client) ApplyTeamAppStoreAppsAssociation(tmName string, vppBatchPayload []fleet.VPPBatchPayload, opts fleet.ApplySpecOptions) ([]fleet.VPPAppResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	query.Add("fleet_name", tmName)
	return c.applyAppStoreAppsAssociation(vppBatchPayload, query)
}

func (c *Client) ApplyNoTeamAppStoreAppsAssociation(vppBatchPayload []fleet.VPPBatchPayload, opts fleet.ApplySpecOptions) ([]fleet.VPPAppResponse, error) {
	query, err := url.ParseQuery(opts.RawQuery())
	if err != nil {
		return nil, err
	}
	return c.applyAppStoreAppsAssociation(vppBatchPayload, query)
}

func (c *Client) applyAppStoreAppsAssociation(vppBatchPayload []fleet.VPPBatchPayload, query url.Values) ([]fleet.VPPAppResponse, error) {
	verb, path := "POST", "/api/latest/fleet/software/app_store_apps/batch"
	var appsResponse batchAssociateAppStoreAppsResponse
	err := c.authenticatedRequestWithQuery(map[string]interface{}{"app_store_apps": vppBatchPayload}, verb, path, &appsResponse, query.Encode())
	if err != nil {
		return nil, err
	}
	return matchAppStoreAppCustomIcons(vppBatchPayload, appsResponse.Apps), nil
}

// matchAppStoreAppCustomIcons hydrates VPP responses with references to icons in the request payload, so we can track
// which API calls to make to add/update/delete icons
func matchAppStoreAppCustomIcons(request []fleet.VPPBatchPayload, response []fleet.VPPAppResponse) []fleet.VPPAppResponse {
	byAdamID := make(map[string]fleet.VPPBatchPayload)
	for _, clientSide := range request {
		byAdamID[clientSide.AppStoreID] = clientSide
	}

	for i := range response {
		serverSide := &response[i]
		if clientSide, ok := byAdamID[serverSide.AppStoreID]; ok {
			serverSide.LocalIconHash = clientSide.IconHash
			serverSide.LocalIconPath = clientSide.IconPath
		}
	}

	return response
}
