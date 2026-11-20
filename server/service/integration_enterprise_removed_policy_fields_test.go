package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestRemovedPolicyFields() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	removed := func(field string) string {
		return fmt.Sprintf(`The %q parameter is not available anymore`, field)
	}

	// Create rejects the automation booleans and patch_software_title_id.
	for _, field := range []string{"calendar_events_enabled", "conditional_access_enabled", "patch_software_title_id"} {
		value := any(true)
		if field == "patch_software_title_id" {
			value = 1
		}
		res := s.Do("POST", "/api/latest/fleet/policies", map[string]any{
			"fleet_id": team.ID, "name": "removed " + field, "query": "SELECT 1;", field: value,
		}, http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), removed(field))
	}

	// An item field needs its automation_type on create.
	res := s.Do("POST", "/api/latest/fleet/policies", map[string]any{
		"fleet_id": team.ID, "name": "script without automation", "query": "SELECT 1;", "script_id": 1,
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "script_id requires automation_type")

	var createResp fleet.CreatePolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", map[string]any{
		"fleet_id": team.ID, "name": "calendar policy", "query": "SELECT 1;", "automation_type": "calendar",
	}, http.StatusOK, &createResp)
	require.NotNil(t, createResp.Policy)
	policyID := createResp.Policy.ID

	// Update rejects the booleans too.
	for _, field := range []string{"calendar_events_enabled", "conditional_access_enabled"} {
		res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", policyID), map[string]any{field: false},
			http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), removed(field))
	}

	// The policy endpoints report automation_type instead of the booleans.
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", policyID), nil, http.StatusOK)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var got struct {
		Policy map[string]any `json:"policy"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, string(fleet.PolicyAutomationCalendar), got.Policy["automation_type"])
	require.NotContains(t, got.Policy, "calendar_events_enabled")
	require.NotContains(t, got.Policy, "conditional_access_enabled")

	// policy_ids is gone from the fleet's webhook settings.
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), map[string]any{
		"webhook_settings": map[string]any{"failing_policies_webhook": map[string]any{"policy_ids": []uint{policyID}}},
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), removed("policy_ids"))
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", team.ID), nil, http.StatusOK)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NotContains(t, string(body), `"policy_ids"`)
}
