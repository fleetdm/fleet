package service

// Policy tests for the enterprise (premium) suite.
//
// Belongs here: global, fleet, and "No team" policy CRUD and listing, policy specs, autofill
// authorization, policy label scoping (include/exclude) and how out-of-scope results are handled,
// and resetting a policy's results.
//
// Does not belong here: anything a policy triggers when it fails — webhooks, scripts, software
// installs, profile resends, calendar events (integration_enterprise_policy_automations_test.go,
// integration_enterprise_calendar_test.go) — and Fleet-maintained app patch policies
// (integration_enterprise_software_fma_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestTeamPolicies() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	password := test.GoodPassword
	email := "testteam@user.com"

	u := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)

	ts := fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", fmt.Sprint(team1.ID))
	require.Empty(t, ts.Policies)
	require.Empty(t, s.listFleetInheritedPolicies(t, team1.ID))

	// create a global policy
	gpol, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: "TestGlobalPolicy", Query: "SELECT 1"})
	require.NoError(t, err)
	defer func() {
		_, err := s.ds.DeleteGlobalPolicies(context.Background(), []uint{gpol.ID})
		require.NoError(t, err)
	}()

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery2",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	tpParams := fleet.CreatePolicyRequest{
		TeamID:      &team1.ID,
		Name:        qr.Name,
		Query:       qr.Query,
		Description: qr.Description,
		Resolution:  "some team resolution",
	}
	r := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusOK, &r)

	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", fmt.Sprint(team1.ID))
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	inherited := s.listFleetInheritedPolicies(t, team1.ID)
	require.Len(t, inherited, 1)
	assert.Equal(t, gpol.Name, inherited[0].Name)
	assert.Equal(t, gpol.ID, inherited[0].ID)

	tc := fleet.CountPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &tc, "fleet_id", fmt.Sprint(team1.ID))
	require.NoError(t, tc.Err)
	require.Equal(t, 1, tc.Count)

	gc := fleet.CountPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &gc)
	require.NoError(t, gc.Err)
	require.Equal(t, 1, gc.Count)

	// Test merge inherited
	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", fmt.Sprint(team1.ID), "merge_inherited", "true", "order_key", "fleet_id", "order_direction", "desc")
	require.Len(t, ts.Policies, 2)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	assert.Equal(t, gpol.Name, ts.Policies[1].Name)
	assert.Equal(t, gpol.ID, ts.Policies[1].ID)

	countResp := fleet.CountPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &countResp, "fleet_id", fmt.Sprint(team1.ID), "merge_inherited", "true")
	require.NoError(t, countResp.Err)
	require.Equal(t, 2, countResp.Count)
	// The inherited count is the merged count less the fleet's own.
	require.Equal(t, 1, countResp.Count-tc.Count)

	// Test delete
	deletePolicyParams := fleet.DeletePoliciesRequest{IDs: []uint{ts.Policies[0].ID}}
	deletePolicyResp := fleet.DeletePoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", fmt.Sprint(team1.ID))
	require.Empty(t, ts.Policies)
}

// listFleetInheritedPolicies returns the "All fleets" policies listed with a fleet's policies.
func (s *integrationEnterpriseTestSuite) listFleetInheritedPolicies(t *testing.T, teamID uint) []*fleet.Policy {
	var resp fleet.ListPoliciesResponse
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &resp, "fleet_id", fmt.Sprint(teamID), "merge_inherited", "true")
	var inherited []*fleet.Policy
	for _, p := range resp.Policies {
		if p.TeamID == nil {
			inherited = append(inherited, p)
		}
	}
	return inherited
}

func (s *integrationEnterpriseTestSuite) TestListTeamPoliciesAutomationTypeSoftware() {
	t := s.T()
	ctx := context.Background()

	// Create a team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team_automation_type_" + t.Name(),
	})
	require.NoError(t, err)

	// Upload a software installer for the software install policy
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "ruby.deb",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	rubyTitleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

	// Upload a second software installer for the patch policy (supported for FMAs)
	payload2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, payload2, http.StatusOK, "")
	dummyTitleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	// Create a Fleet Maintained App and associate it with the dummy installer
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
			VALUES ('DummyApp', 'dummy-autofilter/darwin', 'darwin', 'com.example.dummy-autofilter')`)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, `UPDATE software_installers SET fleet_maintained_app_id = ? WHERE global_or_team_id = ? AND title_id = ?`, id, team.ID, dummyTitleID)
		require.NoError(t, err)
		return nil
	})

	// Create a regular policy (no automation)
	regularPolicy := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		TeamID: &team.ID,
		Name:   "regular policy",
		Query:  "SELECT 1;",
	}, http.StatusOK, &regularPolicy)

	// Create a software install policy
	installPolicy := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		TeamID:          &team.ID,
		Name:            "install software policy",
		Query:           "SELECT 1 FROM some_table;",
		SoftwareTitleID: &rubyTitleID,
		AutomationType:  new(fleet.PolicyAutomationSoftware),
	}, http.StatusOK, &installPolicy)
	require.NotNil(t, installPolicy.Policy.InstallSoftware)

	// Create a patch policy
	patchPolicy := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		TeamID:          &team.ID,
		Type:            new("patch"),
		SoftwareTitleID: &dummyTitleID,
		Hidden:          true,
	}, http.StatusOK, &patchPolicy)
	require.NotNil(t, patchPolicy.Policy.PatchSoftware)
	require.Equal(t, fleet.PolicyTypePatch, patchPolicy.Policy.Type)
	// Patch policies can be hidden from end users like any other policy.
	require.True(t, patchPolicy.Policy.Hidden)
	unhiddenPatch := fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", patchPolicy.Policy.ID),
		json.RawMessage(`{"hidden": false}`), http.StatusOK, &unhiddenPatch)
	require.False(t, unhiddenPatch.Policy.Hidden)

	// List all policies (no filter) - should return all 3
	listResp := fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(team.ID))
	require.Len(t, listResp.Policies, 3)

	// List with automation_type=software - should return only the install policy, not the patch policy
	listResp = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(team.ID), "automation_type", "software")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, installPolicy.Policy.ID, listResp.Policies[0].ID)

	// List with merge_inherited and automation_type=software
	listResp = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(team.ID), "merge_inherited", "true", "automation_type", "software")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, installPolicy.Policy.ID, listResp.Policies[0].ID)

	// Patch policies are listed with type=patch (automation_type=patch is rejected on /policies).
	listResp = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(team.ID), "type", "patch")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, patchPolicy.Policy.ID, listResp.Policies[0].ID)

	// List with merge_inherited and type=patch
	listResp = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &listResp, "fleet_id", fmt.Sprint(team.ID), "merge_inherited", "true", "type", "patch")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, patchPolicy.Policy.ID, listResp.Policies[0].ID)
}

func (s *integrationEnterpriseTestSuite) TestNoTeamPolicies() {
	t := s.T()
	ctx := context.Background()

	//
	// Test a global admin can read and write "No team" policies.
	//

	// List "No team" policies.
	ts := fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", "0")
	require.Empty(t, ts.Policies)
	require.Empty(t, s.listFleetInheritedPolicies(t, 0))
	// Create a placeholder global policy.
	_, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:  "globalPolicy1",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	// Create a "No team" policy.
	noTeamID := uint(0)
	tpParams := fleet.CreatePolicyRequest{
		TeamID: &noTeamID,
		Name:   "noTeamPolicy1",
		Query:  "SELECT 1;",
	}
	r := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusOK, &r)
	require.NotNil(t, r.Policy.TeamID)
	require.Zero(t, *r.Policy.TeamID)
	// Test that we can't create a policy with the same name under "No team" domain.
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusConflict, &r)
	// Create a second "No team" policy.
	tpParams = fleet.CreatePolicyRequest{
		TeamID: &noTeamID,
		Name:   "noTeamPolicy2",
		Query:  "SELECT 2;",
	}
	r = fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusOK, &r)
	require.NotNil(t, r.Policy.TeamID)
	require.Zero(t, *r.Policy.TeamID)
	// List "No team" policies.
	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", "0")
	require.Len(t, ts.Policies, 2)
	assert.Equal(t, "noTeamPolicy1", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 1;", ts.Policies[0].Query)
	require.NotNil(t, ts.Policies[0].TeamID)
	require.Zero(t, *ts.Policies[0].TeamID)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[1].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[1].Query)
	require.NotNil(t, ts.Policies[1].TeamID)
	require.Zero(t, *ts.Policies[1].TeamID)
	inherited := s.listFleetInheritedPolicies(t, 0)
	require.Len(t, inherited, 1)
	assert.Equal(t, "globalPolicy1", inherited[0].Name)
	assert.Equal(t, "SELECT 0;", inherited[0].Query)
	assert.Nil(t, inherited[0].TeamID)
	// Test policy count for "No team" policies.
	tc := fleet.CountPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &tc, "fleet_id", "0")
	require.Equal(t, 2, tc.Count)
	// Test merge inherited for "No team" policies.
	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", "0", "merge_inherited", "true", "order_key", "fleet_id", "order_direction", "desc")
	require.Len(t, ts.Policies, 3)
	assert.Equal(t, "noTeamPolicy1", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 1;", ts.Policies[0].Query)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[1].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[1].Query)
	assert.Equal(t, "globalPolicy1", ts.Policies[2].Name)
	assert.Equal(t, "SELECT 0;", ts.Policies[2].Query)
	// Test merge inherited count for "No team" policies.
	countResp := fleet.CountPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &countResp, "fleet_id", "0", "merge_inherited", "true")
	require.NoError(t, countResp.Err)
	require.Equal(t, 3, countResp.Count)
	// The inherited count is the merged count less the fleet's own.
	require.Equal(t, 1, countResp.Count-tc.Count)
	// Test deleting "No team" policies.
	deletePolicyParams := fleet.DeletePoliciesRequest{
		IDs: []uint{ts.Policies[0].ID},
	}
	deletePolicyResp := fleet.DeletePoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)
	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &ts, "fleet_id", "0")
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[0].Query)
	noTeamPolicy2 := ts.Policies[0]

	//
	// Test that a team admin is not allowed to access "No team" policies.
	//

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})
	password := test.GoodPassword
	email := "testteam@user.com"
	team1Admin := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, team1Admin.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), team1Admin)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)

	ts = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusForbidden, &ts, "fleet_id", "0")
	tpParams = fleet.CreatePolicyRequest{
		TeamID: &noTeamID,
		Name:   "noTeamPolicy1",
		Query:  "SELECT 1;",
	}
	r = fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusForbidden, &r)
	tc = fleet.CountPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusForbidden, &tc, "fleet_id", "0")
	deletePolicyParams = fleet.DeletePoliciesRequest{
		IDs: []uint{noTeamPolicy2.ID},
	}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusForbidden, &fleet.DeletePoliciesResponse{})
}

func (s *integrationEnterpriseTestSuite) TestGlobalPolicyCreateReadPatch() {
	t := s.T()
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	createPol1 := &fleet.CreatePolicyResponse{}
	createPol1Req := &fleet.CreatePolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol1Req, http.StatusOK, &createPol1)
	allEqual(t, createPol1Req, createPol1.Policy, fields...)

	createPol2 := &fleet.CreatePolicyResponse{}
	createPol2Req := &fleet.CreatePolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol2Req, http.StatusOK, &createPol2)
	allEqual(t, createPol2Req, createPol2.Policy, fields...)

	listPol := &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(t, listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(t, createPol1.Policy, listPol.Policies[0])
	require.Equal(t, createPol2.Policy, listPol.Policies[1])

	// match policy by name with leading/trailing whitespace
	listPolByName := &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPolByName, "query", " name1 ")
	require.Len(t, listPolByName.Policies, 1)
	require.Equal(t, "name1", listPolByName.Policies[0].Name)

	patchPol1Req := &fleet.ModifyPolicyRequest{
		Name:        new("newName1"),
		Query:       new("newQuery"),
		Description: new("newDescription"),
		Resolution:  new("newResolution"),
		Platform:    new("windows"),
		Critical:    new(false),
	}
	patchPol1 := &fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(t, patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &fleet.ModifyPolicyRequest{
		Name:        new("newName2"),
		Query:       new("newQuery"),
		Description: new("newDescription"),
		Resolution:  new("newResolution"),
		Platform:    new("windows"),
		Critical:    new(true),
	}
	patchPol2 := &fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(t, patchPol2Req, patchPol2.Policy, fields...)

	listPol = &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(t, listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(t, patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(t, patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(t, listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicyCreateReadPatch() {
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical", "Hidden"}

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	s.Require().NoError(err)

	createPol1 := &fleet.CreatePolicyResponse{}
	createPol1Req := &fleet.CreatePolicyRequest{
		TeamID:         &team1.ID,
		Query:          "query",
		Name:           "name1",
		Description:    "description",
		Resolution:     "resolution",
		Platform:       "linux",
		Critical:       true,
		AutomationType: new(fleet.PolicyAutomationCalendar),
		Hidden:         true,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, append(fields, "AutomationType")...)

	createPol2 := &fleet.CreatePolicyResponse{}
	createPol2Req := &fleet.CreatePolicyRequest{
		TeamID:      &team1.ID,
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, append(fields, "AutomationType")...)

	listPol := &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol, "fleet_id", fmt.Sprint(team1.ID))
	s.Require().Len(listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	s.Require().Equal(createPol1.Policy, listPol.Policies[0])
	s.Require().Equal(createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &fleet.ModifyPolicyRequest{
		Name:           new("newName1"),
		Query:          new("newQuery"),
		Description:    new("newDescription"),
		Resolution:     new("newResolution"),
		Platform:       new("windows"),
		Critical:       new(false),
		AutomationType: optjson.Any[fleet.PolicyAutomation]{Set: true},
		Hidden:         new(false),
	}
	patchPol1 := &fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)
	s.Require().Nil(patchPol1.Policy.AutomationType)

	patchPol2Req := &fleet.ModifyPolicyRequest{
		Name:           new("newName2"),
		Query:          new("newQuery"),
		Description:    new("newDescription"),
		Resolution:     new("newResolution"),
		Platform:       new("windows"),
		Critical:       new(true),
		AutomationType: optjson.Any[fleet.PolicyAutomation]{Set: true, Valid: true, Value: fleet.PolicyAutomationCalendar},
		Hidden:         new(true),
	}
	patchPol2 := &fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)
	s.Require().NotNil(patchPol2.Policy.AutomationType)
	s.Require().Equal(fleet.PolicyAutomationCalendar, *patchPol2.Policy.AutomationType)

	listPol = &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol, "fleet_id", fmt.Sprint(team1.ID))
	s.Require().Len(listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], append(fields, "AutomationType")...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], append(fields, "AutomationType")...)

	getPol2 := &fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	s.Require().Equal(listPol.Policies[1], getPol2.Policy)

	// A patch that omits hidden leaves it untouched.
	patchPol2 = &fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID),
		json.RawMessage(`{"name": "newName2b"}`), http.StatusOK, patchPol2)
	s.Require().True(patchPol2.Policy.Hidden)

	// hidden and conditional access are mutually exclusive, on create and on either side of a patch.
	res := s.Do("POST", "/api/latest/fleet/policies", &fleet.CreatePolicyRequest{
		TeamID: &team1.ID, Query: "query", Name: "hidden-ca", Platform: "darwin", Hidden: true,
		AutomationType: new(fleet.PolicyAutomationConditionalAccess),
	}, http.StatusBadRequest)
	s.Require().Contains(extractServerErrorText(res.Body), `"hidden" and "conditional_access_enabled" cannot both be set`)

	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID),
		json.RawMessage(`{"automation_type": "conditional_access"}`), http.StatusBadRequest)
	s.Require().Contains(extractServerErrorText(res.Body), `"hidden" and "conditional_access_enabled" cannot both be set`)

	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID),
		json.RawMessage(`{"automation_type": "conditional_access"}`), http.StatusOK)
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID),
		json.RawMessage(`{"hidden": true}`), http.StatusBadRequest)
	s.Require().Contains(extractServerErrorText(res.Body), `"hidden" and "conditional_access_enabled" cannot both be set`)

	// "All fleets" policies can be hidden too.
	gpResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name: "hidden global", Query: "SELECT 1;", Hidden: true,
	}, http.StatusOK, &gpResp)
	s.Require().True(gpResp.Policy.Hidden)
	modGP := &fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID),
		json.RawMessage(`{"hidden": false}`), http.StatusOK, modGP)
	s.Require().False(modGP.Policy.Hidden)
	getGP := &fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), nil, http.StatusOK, getGP)
	s.Require().False(getGP.Policy.Hidden)
}

// allEqual compares all fields of a struct.
// If a field is a pointer on one side but not on the other, then it follows that pointer. This is useful for optional
// arguments.
func allEqual(t *testing.T, expect, actual any, fields ...string) {
	require.NotEmpty(t, fields)
	t.Helper()
	expV := reflect.Indirect(reflect.ValueOf(expect))
	actV := reflect.Indirect(reflect.ValueOf(actual))
	for _, f := range fields {
		e, a := expV.FieldByName(f), actV.FieldByName(f)
		switch {
		case e.Kind() == reflect.Ptr && a.Kind() != reflect.Ptr && !e.IsZero():
			e = e.Elem()
		case a.Kind() == reflect.Ptr && e.Kind() != reflect.Ptr && !a.IsZero():
			a = a.Elem()
		}
		require.Equal(t, e.Interface(), a.Interface(), "%s", f)
	}
}

func (s *integrationEnterpriseTestSuite) TestAutofillPoliciesAuthTeamUser() {
	t := s.T()
	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					switch r.URL.Path {
					case "/ok":
						var body map[string]any
						err := json.NewDecoder(r.Body).Decode(&body)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)
	originalUrl := getHumanInterpretationFromOsquerySqlUrl
	originalTimeout := getHumanInterpretationFromOsquerySqlTimeout
	t.Cleanup(
		func() {
			getHumanInterpretationFromOsquerySqlUrl = originalUrl
			getHumanInterpretationFromOsquerySqlTimeout = originalTimeout
		},
	)

	// Create teams
	team1, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			ID:          42,
			Name:        "team1" + t.Name(),
			Description: "desc team1",
		},
	)
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			ID:          43,
			Name:        "team2" + t.Name(),
			Description: "desc team2",
		},
	)
	require.NoError(t, err)

	oldToken := s.token
	t.Cleanup(
		func() {
			s.token = oldToken
		},
	)

	switchUser := func(t *testing.T, role string) {
		password := test.GoodPassword
		email := role + "-testteam@user.com"
		u := &fleet.User{
			Name:       "test team user",
			Email:      email,
			GlobalRole: nil,
			Teams: []fleet.UserTeam{
				{
					Team: *team2,
					Role: fleet.RoleObserver,
				},
				{
					Team: *team1,
					Role: role,
				},
			},
		}
		require.NoError(t, u.SetPassword(password, 10, 10))
		_, err = s.ds.NewUser(context.Background(), u)
		require.NoError(t, err)

		s.token = s.getTestToken(email, password)
	}

	req := fleet.AutofillPoliciesRequest{
		SQL: "select 1",
	}
	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/ok"

	tests := []struct {
		role string
		pass bool
	}{
		{role: fleet.RoleAdmin, pass: true},
		{role: fleet.RoleMaintainer, pass: true},
		{role: fleet.RoleGitOps, pass: true},
		{role: fleet.RoleObserver, pass: false},
		{role: fleet.RoleObserverPlus, pass: false},
	}

	for _, tt := range tests {
		t.Run(
			tt.role, func(t *testing.T) {
				switchUser(t, tt.role)
				if tt.pass {
					var res fleet.AutofillPoliciesResponse
					s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
					assert.Equal(t, "description", res.Description)
					assert.Equal(t, "resolution", res.Resolution)
				} else {
					_ = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusForbidden)
				}
			},
		)
	}
}

// TestPolicyLabelsIncludeAll exercises the full create/modify/spec stack for the
// include_all label scope on policies: combining one include scope with one
// exclude scope, rejecting two include or two exclude scopes at every API entry
// point, and end-to-end host targeting.
func (s *integrationEnterpriseTestSuite) TestPolicyLabelsIncludeAll() {
	t := s.T()
	ctx := context.Background()

	// Two labels.
	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lblA := lblResp.Label
	lblResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 2"}, http.StatusOK, &lblResp)
	lblB := lblResp.Label

	// Three hosts: none, A only, A+B.
	mkHost := func(suffix string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(t.Name() + suffix),
			NodeKey:         new(t.Name() + suffix),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s-%s.local", t.Name(), suffix),
			Platform:        "linux",
		})
		require.NoError(t, err)
		return h
	}
	hostNone := mkHost("none")
	hostA := mkHost("a")
	hostBoth := mkHost("both")
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostA.ID, []uint{lblA.ID}))
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostBoth.ID, []uint{lblA.ID, lblB.ID}))

	// 1. Create global policy with include_all=[A,B] via POST.
	var createResp fleet.CreatePolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name:             "include-all-" + t.Name(),
		Query:            "SELECT 1",
		LabelsIncludeAll: []string{lblA.Name, lblB.Name},
	}, http.StatusOK, &createResp)
	require.NotNil(t, createResp.Policy)
	require.Empty(t, createResp.Policy.LabelsIncludeAny)
	require.Empty(t, createResp.Policy.LabelsExcludeAny)
	require.Len(t, createResp.Policy.LabelsIncludeAll, 2)

	// 2. Mutex rejection on POST: include_all + include_any. Assert on the
	// specific validation error so a generic 400 (e.g., name required) cannot
	// silently satisfy the test.
	rej1Resp := s.Do("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name:             "rej1-" + t.Name(),
		Query:            "SELECT 1",
		LabelsIncludeAll: []string{lblA.Name},
		LabelsIncludeAny: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rej1Resp.Body), fleet.ErrPolicyConflictingIncludeLabels.Error())

	// include_all + exclude_any is now a valid combination (one include scope
	// plus one exclude scope), so it succeeds and persists both.
	var combinedResp fleet.CreatePolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name:             "combined-" + t.Name(),
		Query:            "SELECT 1",
		LabelsIncludeAll: []string{lblA.Name},
		LabelsExcludeAny: []string{lblB.Name},
	}, http.StatusOK, &combinedResp)
	require.Len(t, combinedResp.Policy.LabelsIncludeAll, 1)
	require.Len(t, combinedResp.Policy.LabelsExcludeAny, 1)

	// Two exclude scopes (exclude_any + exclude_all) are still rejected.
	rejExclResp := s.Do("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name:             "rej-excl-" + t.Name(),
		Query:            "SELECT 1",
		LabelsExcludeAny: []string{lblA.Name},
		LabelsExcludeAll: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejExclResp.Body), fleet.ErrPolicyConflictingExcludeLabels.Error())

	// 3. PATCH: switch existing include_all policy to include_any. Other slices should clear.
	switchAny := []string{lblA.Name}
	var patchResp fleet.ModifyPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createResp.Policy.ID), fleet.ModifyPolicyRequest{
		LabelsIncludeAny: switchAny,
	}, http.StatusOK, &patchResp)
	require.Empty(t, patchResp.Policy.LabelsIncludeAll)
	require.Empty(t, patchResp.Policy.LabelsExcludeAny)
	require.Len(t, patchResp.Policy.LabelsIncludeAny, 1)

	// Switch back to include_all so end-to-end targeting test below uses it.
	switchAll := []string{lblA.Name, lblB.Name}
	patchResp = fleet.ModifyPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createResp.Policy.ID), fleet.ModifyPolicyRequest{
		LabelsIncludeAll: switchAll,
	}, http.StatusOK, &patchResp)
	require.Len(t, patchResp.Policy.LabelsIncludeAll, 2)
	require.Empty(t, patchResp.Policy.LabelsIncludeAny)

	// PATCH mutex rejection. Assert on the specific validation error so a
	// generic 400 cannot silently satisfy the test.
	rejPatchResp := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createResp.Policy.ID), fleet.ModifyPolicyRequest{
		LabelsIncludeAll: []string{lblA.Name},
		LabelsIncludeAny: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejPatchResp.Body), fleet.ErrPolicyConflictingIncludeLabels.Error())

	// 4. End-to-end host targeting: only hostBoth should match the include_all policy.
	policy, err := s.ds.Policy(ctx, createResp.Policy.ID)
	require.NoError(t, err)

	for _, tc := range []struct {
		host  *fleet.Host
		match bool
	}{
		{hostNone, false},
		{hostA, false},
		{hostBoth, true},
	} {
		queries, err := s.ds.PolicyQueriesForHost(ctx, tc.host)
		require.NoError(t, err)
		_, ok := queries[fmt.Sprint(policy.ID)]
		require.Equal(t, tc.match, ok, "host %s match expectation", tc.host.Hostname)
	}
}

// A host authenticates with its node key and fully controls the fleet_policy_query_<id> keys it
// sends to distributed/write, so results for policies outside its scope must never be persisted.
func (s *integrationEnterpriseTestSuite) TestPolicyResultsForOutOfScopePoliciesAreDiscarded() {
	t := s.T()
	ctx := t.Context()

	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lbl := lblResp.Label

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// The victim host: darwin, no team, not a member of lbl.
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + ".local",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	inScope, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "in-scope-" + t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)

	// Out of scope three different ways:
	// 1. another fleet
	otherFleet, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{Name: "other-fleet-" + t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)
	// 2. another platform
	otherPlatform, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name: "other-platform-" + t.Name(), Query: "SELECT 1", Platform: "linux",
	})
	require.NoError(t, err)
	// 3. label the host lacks
	otherLabel, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name: "other-label-" + t.Name(), Query: "SELECT 1", LabelsIncludeAny: []string{lbl.Name},
	})
	require.NoError(t, err)

	// Only the in-scope policy is distributed to the host.
	distributedQueries, err := s.ds.PolicyQueriesForHost(ctx, host)
	require.NoError(t, err)
	require.Contains(t, distributedQueries, fmt.Sprint(inScope.ID))
	for _, p := range []*fleet.Policy{otherFleet, otherPlatform, otherLabel} {
		require.NotContains(t, distributedQueries, fmt.Sprint(p.ID))
	}

	// Forge the write: the payload keys are entirely host-controlled, so a node key plus a hand-rolled body is
	// enough to claim a failing result for all four policies -- including the three never distributed above.
	// The write is accepted (200) and the out-of-scope results are dropped during ingestion rather than
	// rejecting the whole check-in, which would also discard the host's legitimate results.
	var distributedResp submitDistributedQueryResultsResponse
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(host, map[uint]*bool{
		inScope.ID:       new(false),
		otherFleet.ID:    new(false),
		otherPlatform.ID: new(false),
		otherLabel.ID:    new(false),
	}), http.StatusOK, &distributedResp)

	// Nothing was persisted for the out-of-scope policies. This asserts on policy_membership rather than
	// ListPoliciesForHost because the latter re-applies the scope filter on read and would hide the rows.
	var storedPolicyIDs []uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &storedPolicyIDs, `SELECT policy_id FROM policy_membership WHERE host_id = ?`, host.ID)
	})
	require.Equal(t, []uint{inScope.ID}, storedPolicyIDs)

	// The forged results do not surface the host in policy-filtered host listings either.
	countFailingHosts := func(policyID uint) int {
		var listResp listHostsResponse
		s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp,
			"policy_id", fmt.Sprint(policyID), "policy_response", "failing")
		return len(listResp.Hosts)
	}
	require.Equal(t, 1, countFailingHosts(inScope.ID))
	for _, p := range []*fleet.Policy{otherFleet, otherPlatform, otherLabel} {
		require.Zero(t, countFailingHosts(p.ID), "policy %s", p.Name)
	}
}

// Enrollment sets label_updated_at to the "never" sentinel and makes labels and policies come due on the same check-in, so an
// exclude-label-scoped policy must not be handed to (or accepted from) a host that has not yet computed the label's membership --
// otherwise its automations fire on a host the exclusion was meant to protect.
func (s *integrationEnterpriseTestSuite) TestPolicyExcludeLabelHeldUntilLabelsReported() {
	t := s.T()
	ctx := t.Context()

	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lbl := lblResp.Label

	// labels.created_at is second-granular and this test runs well inside one second, so the label report below would tie with
	// the label's creation and stay (correctly) unknown. Backdate it to the realistic case: the label predates the check-in.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE labels SET created_at = ? WHERE id = ?`, time.Now().Add(-time.Minute), lbl.ID)
		return err
	})

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  common_mysql.GetDefaultNonZeroTime(),
		PolicyUpdatedAt: common_mysql.GetDefaultNonZeroTime(),
		SeenTime:        time.Now(),
		LastEnrolledAt:  time.Now(),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + ".local",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	unscoped, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "unscoped-" + t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)
	excluded, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name: "excluded-" + t.Name(), Query: "SELECT 1", LabelsExcludeAny: []string{lbl.Name},
	})
	require.NoError(t, err)

	storedPolicyIDs := func() []uint {
		var ids []uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &ids, `SELECT policy_id FROM policy_membership WHERE host_id = ?`, host.ID)
		})
		return ids
	}

	distributedQueries, err := s.ds.PolicyQueriesForHost(ctx, host)
	require.NoError(t, err)
	require.Contains(t, distributedQueries, fmt.Sprint(unscoped.ID))
	require.NotContains(t, distributedQueries, fmt.Sprint(excluded.ID),
		"exclude-scoped policy sent before the host could evaluate the label")

	// Even if the host reports a failing result for it anyway, ingestion must drop it: everything downstream of this
	// (script, software, calendar and conditional-access automations) reads the surviving results.
	var distributedResp submitDistributedQueryResultsResponse
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(host, map[uint]*bool{
		unscoped.ID: new(false),
		excluded.ID: new(false),
	}), http.StatusOK, &distributedResp)
	require.Equal(t, []uint{unscoped.ID}, storedPolicyIDs())

	// Once the host has reported its labels and is confirmed not to be a member, the policy applies as normal.
	s.DoJSON("POST", "/api/osquery/distributed/write",
		genDistributedReqWithLabelResults(host, map[uint]*bool{lbl.ID: new(false)}), http.StatusOK, &distributedResp)

	host, err = s.ds.Host(ctx, host.ID)
	require.NoError(t, err)
	distributedQueries, err = s.ds.PolicyQueriesForHost(ctx, host)
	require.NoError(t, err)
	require.Contains(t, distributedQueries, fmt.Sprint(excluded.ID))

	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(host, map[uint]*bool{
		unscoped.ID: new(false),
		excluded.ID: new(false),
	}), http.StatusOK, &distributedResp)
	require.ElementsMatch(t, []uint{unscoped.ID, excluded.ID}, storedPolicyIDs())
}

func (s *integrationEnterpriseTestSuite) TestApplyPolicySpecsBatchMixedScopes() {
	t := s.T()
	ctx := context.Background()

	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lblA := lblResp.Label
	lblResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 2"}, http.StatusOK, &lblResp)
	lblB := lblResp.Label

	validAnyName := "batch-any-" + t.Name()
	validAllName := "batch-all-" + t.Name()
	invalidName := "batch-invalid-" + t.Name()

	assertNonePersisted := func(label string) {
		policies, err := s.ds.ListGlobalPolicies(ctx, fleet.ListOptions{}, "")
		require.NoError(t, err)
		for _, p := range policies {
			require.NotEqual(t, validAnyName, p.Name, "%s: no spec from rejected batch should persist", label)
			require.NotEqual(t, validAllName, p.Name, "%s: no spec from rejected batch should persist", label)
			require.NotEqual(t, invalidName, p.Name, "%s: no spec from rejected batch should persist", label)
		}
	}

	// Batch with an invalid (two-include) spec at the END — proves we don't persist
	// preceding valid specs once a later one fails validation.
	rejResp := s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{Name: validAnyName, Query: "SELECT 1", LabelsIncludeAny: []string{lblA.Name}},
			{Name: validAllName, Query: "SELECT 1", LabelsIncludeAll: []string{lblA.Name, lblB.Name}},
			{Name: invalidName, Query: "SELECT 1", LabelsIncludeAll: []string{lblA.Name}, LabelsIncludeAny: []string{lblB.Name}},
		},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejResp.Body), fleet.ErrPolicyConflictingIncludeLabels.Error())
	assertNonePersisted("violator-last")

	// Batch with an invalid (two-include) spec at the FRONT — proves we don't persist
	// trailing valid specs after a per-spec validation pass that the loop never reaches.
	rejResp = s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{Name: invalidName, Query: "SELECT 1", LabelsIncludeAll: []string{lblA.Name}, LabelsIncludeAny: []string{lblB.Name}},
			{Name: validAnyName, Query: "SELECT 1", LabelsIncludeAny: []string{lblA.Name}},
			{Name: validAllName, Query: "SELECT 1", LabelsIncludeAll: []string{lblA.Name, lblB.Name}},
		},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejResp.Body), fleet.ErrPolicyConflictingIncludeLabels.Error())
	assertNonePersisted("violator-first")

	// Fully-valid 3-spec batch (one per scope) succeeds.
	validExclName := "batch-excl-" + t.Name()
	s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{Name: validAnyName, Query: "SELECT 1", LabelsIncludeAny: []string{lblA.Name}},
			{Name: validAllName, Query: "SELECT 1", LabelsIncludeAll: []string{lblA.Name, lblB.Name}},
			{Name: validExclName, Query: "SELECT 1", LabelsExcludeAny: []string{lblB.Name}},
		},
	}, http.StatusOK)

	policies, err := s.ds.ListGlobalPolicies(ctx, fleet.ListOptions{}, "")
	require.NoError(t, err)
	byName := make(map[string]*fleet.Policy, len(policies))
	for _, p := range policies {
		byName[p.Name] = p
	}
	validAny, ok := byName[validAnyName]
	require.True(t, ok)
	validAll, ok := byName[validAllName]
	require.True(t, ok)
	validExcl, ok := byName[validExclName]
	require.True(t, ok)
	require.Len(t, validAny.LabelsIncludeAny, 1)
	require.Len(t, validAll.LabelsIncludeAll, 2)
	require.Len(t, validExcl.LabelsExcludeAny, 1)
}

func (s *integrationEnterpriseTestSuite) TestResetPolicy() {
	t := s.T()
	ctx := context.Background()

	// Create all hosts upfront so each gets a unique osquery_host_id index.
	hosts := s.createHosts(t, "darwin", "darwin", "darwin")
	globalHost, noTeamHost, teamHost := hosts[0], hosts[1], hosts[2]

	// --- global policy ---
	createGlobalResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name:  "reset-test-global",
		Query: "SELECT 1;",
	}, http.StatusOK, &createGlobalResp)
	globalPolicy := createGlobalResp.Policy
	require.NotZero(t, globalPolicy.ID)

	// Record a failing result for the global policy.
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		globalHost,
		map[uint]*bool{globalPolicy.ID: new(false)},
	), http.StatusOK, new(submitDistributedQueryResultsResponse))

	require.NoError(t, s.ds.UpdateHostPolicyCounts(ctx))

	// Confirm the global policy now has a failing host count.
	getGlobalResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicy.ID), nil, http.StatusOK, &getGlobalResp)
	require.Equal(t, uint(1), getGlobalResp.Policy.FailingHostCount)

	// Reset the global policy.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/policies/%d/reset", globalPolicy.ID), nil, http.StatusOK)

	// Counts must be 0 immediately after reset.
	getGlobalResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicy.ID), nil, http.StatusOK, &getGlobalResp)
	require.Equal(t, uint(0), getGlobalResp.Policy.FailingHostCount)
	require.Equal(t, uint(0), getGlobalResp.Policy.PassingHostCount)

	// Counts remain 0 after re-aggregating (policy_membership was wiped).
	require.NoError(t, s.ds.UpdateHostPolicyCounts(ctx))
	getGlobalResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicy.ID), nil, http.StatusOK, &getGlobalResp)
	require.Equal(t, uint(0), getGlobalResp.Policy.FailingHostCount)
	require.Equal(t, uint(0), getGlobalResp.Policy.PassingHostCount)

	// A reset_policy activity was recorded; global policies emit team_id/fleet_id: -1.
	s.lastActivityMatches("reset_policy", fmt.Sprintf(`{"policy_id":%d,"policy_name":"reset-test-global","fleet_id":-1}`, globalPolicy.ID), 0)

	// --- no-team policy ---
	createNoTeamResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		TeamID: new(uint(0)),
		Name:   "reset-test-no-team",
		Query:  "SELECT 1;",
	}, http.StatusOK, &createNoTeamResp)
	noTeamPolicy := createNoTeamResp.Policy
	require.NotZero(t, noTeamPolicy.ID)
	require.NotNil(t, noTeamPolicy.TeamID)
	require.Zero(t, *noTeamPolicy.TeamID)

	// Seed a failing result.
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		noTeamHost,
		map[uint]*bool{noTeamPolicy.ID: new(false)},
	), http.StatusOK, new(submitDistributedQueryResultsResponse))
	require.NoError(t, s.ds.UpdateHostPolicyCounts(ctx))

	getNoTeamResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", noTeamPolicy.ID), nil, http.StatusOK, &getNoTeamResp)
	require.Equal(t, uint(1), getNoTeamResp.Policy.FailingHostCount)

	// Reset the no-team policy — must not error.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/policies/%d/reset", noTeamPolicy.ID), nil, http.StatusOK)

	getNoTeamResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", noTeamPolicy.ID), nil, http.StatusOK, &getNoTeamResp)
	require.Equal(t, uint(0), getNoTeamResp.Policy.FailingHostCount)
	require.Equal(t, uint(0), getNoTeamResp.Policy.PassingHostCount)

	// Activity emitted with team_id/fleet_id: 0 and no team_name.
	s.lastActivityMatches("reset_policy", fmt.Sprintf(`{"policy_id":%d,"policy_name":"reset-test-no-team","fleet_id":0}`, noTeamPolicy.ID), 0)

	// --- team policy ---
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "reset-policy-team"})
	require.NoError(t, err)

	createTeamResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		TeamID: &team.ID,
		Name:   "reset-test-team",
		Query:  "SELECT 1;",
	}, http.StatusOK, &createTeamResp)
	teamPolicy := createTeamResp.Policy
	require.NotZero(t, teamPolicy.ID)

	// Assign host to team and record a passing result.
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID})))

	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		teamHost,
		map[uint]*bool{teamPolicy.ID: new(true)},
	), http.StatusOK, new(submitDistributedQueryResultsResponse))

	require.NoError(t, s.ds.UpdateHostPolicyCounts(ctx))

	getTeamResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", teamPolicy.ID), nil, http.StatusOK, &getTeamResp)
	require.Equal(t, uint(1), getTeamResp.Policy.PassingHostCount)

	// Reset the team policy.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/policies/%d/reset", teamPolicy.ID), nil, http.StatusOK)

	getTeamResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", teamPolicy.ID), nil, http.StatusOK, &getTeamResp)
	require.Equal(t, uint(0), getTeamResp.Policy.PassingHostCount)
	require.Equal(t, uint(0), getTeamResp.Policy.FailingHostCount)

	// A reset_policy activity was recorded with team fields (both team_id/fleet_id and team_name/fleet_name).
	s.lastActivityMatches("reset_policy", fmt.Sprintf(
		`{"policy_id":%d,"policy_name":"reset-test-team","fleet_id":%d,"fleet_name":"reset-policy-team"}`,
		teamPolicy.ID, team.ID,
	), 0)

	// 404 for a nonexistent policy.
	s.Do("POST", "/api/latest/fleet/policies/999999/reset", nil, http.StatusNotFound)

	// Both endpoints are documented under /api/v1 and must be routed there too.
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/policies/%d/reset", globalPolicy.ID), nil, http.StatusOK)
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/policies/%d/automation_activities", globalPolicy.ID), nil, http.StatusOK)

	// --- host-scoped reset (?host_id=) ---
	createHostScopedResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.CreatePolicyRequest{
		Name:  "reset-test-host-scoped",
		Query: "SELECT 1;",
	}, http.StatusOK, &createHostScopedResp)
	hostScopedPolicy := createHostScopedResp.Policy

	for _, h := range []*fleet.Host{globalHost, noTeamHost} {
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			h, map[uint]*bool{hostScopedPolicy.ID: new(false)},
		), http.StatusOK, new(submitDistributedQueryResultsResponse))
	}
	require.NoError(t, s.ds.UpdateHostPolicyCounts(ctx))
	getHostScopedResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", hostScopedPolicy.ID), nil, http.StatusOK, &getHostScopedResp)
	require.Equal(t, uint(2), getHostScopedResp.Policy.FailingHostCount)

	// Reset only globalHost's result; noTeamHost's failing result must survive and the
	// counts must reflect it immediately, without waiting for the counts cron.
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/policies/%d/reset?host_id=%d", hostScopedPolicy.ID, globalHost.ID), nil, http.StatusOK)
	getHostScopedResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", hostScopedPolicy.ID), nil, http.StatusOK, &getHostScopedResp)
	require.Equal(t, uint(1), getHostScopedResp.Policy.FailingHostCount)
	require.Equal(t, uint(0), getHostScopedResp.Policy.PassingHostCount)

	s.lastActivityMatches("reset_policy", fmt.Sprintf(
		`{"policy_id":%d,"policy_name":"reset-test-host-scoped","fleet_id":-1,"host_id":%d,"host_display_name":%q}`,
		hostScopedPolicy.ID, globalHost.ID, globalHost.DisplayName(),
	), 0)

	// A host-scoped reset shows up in that host's activity feed; policy-wide resets don't
	// get linked to any host.
	resetActivityType := fleet.ActivityTypeResetPolicy{}.ActivityName()
	countResetActivities := func(hostID uint) int {
		var hostActivities listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", hostID), nil, http.StatusOK, &hostActivities)
		n := 0
		for _, a := range hostActivities.Activities {
			if a.Type == resetActivityType {
				n++
			}
		}
		return n
	}
	require.Equal(t, 1, countResetActivities(globalHost.ID))
	require.Equal(t, 0, countResetActivities(noTeamHost.ID))

	// 404 for a nonexistent host.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/policies/%d/reset?host_id=999999", hostScopedPolicy.ID), nil, http.StatusNotFound)

	// 404 for a host outside a team policy's team.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/policies/%d/reset?host_id=%d", teamPolicy.ID, noTeamHost.ID), nil, http.StatusNotFound)
}

func (s *integrationEnterpriseTestSuite) TestApplyPolicySpecsScriptValidation() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-other"})
	require.NoError(t, err)

	newScript := func(name string, teamID *uint) *fleet.Script {
		script, err := s.ds.NewScript(ctx, &fleet.Script{
			Name:           name,
			ScriptContents: "echo",
			TeamID:         teamID,
		})
		require.NoError(t, err)
		return script
	}
	teamScript := newScript("spec-team.sh", &team.ID)
	otherTeamScript := newScript("spec-other-team.sh", &otherTeam.ID)

	const specURL = "/api/latest/fleet/spec/policies"
	spec := func(name, teamName string, scriptID uint) fleet.ApplyPolicySpecsRequest {
		return fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{{
				Name:     name,
				Query:    "SELECT 1;",
				Platform: "darwin",
				Team:     teamName,
				ScriptID: new(scriptID),
			}},
		}
	}

	// A script on the policy's own team is accepted.
	s.Do("POST", specURL, spec("gitops script", team.Name, teamScript.ID), http.StatusOK)
	list := &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, list, "fleet_id", fmt.Sprint(team.ID))
	require.Len(t, list.Policies, 1)
	require.NotNil(t, list.Policies[0].RunScript)
	require.Equal(t, teamScript.ID, list.Policies[0].RunScript.ID)

	// A spec with no team cannot carry a script.
	res := s.Do("POST", specURL, spec("gitops global script", "", teamScript.ID), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "cannot have script_id set")

	// A script owned by another team is rejected.
	res = s.Do("POST", specURL, spec("gitops cross team script", team.Name, otherTeamScript.ID), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "does not belong to fleet ID")

	// A script that does not exist is rejected with a clear message rather than a database error.
	res = s.Do("POST", specURL, spec("gitops missing script", team.Name, otherTeamScript.ID+999), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "does not exist")

	// None of the rejected specs created a policy.
	list = &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, list, "fleet_id", fmt.Sprint(team.ID))
	require.Len(t, list.Policies, 1)
	globalList := &fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/policies", nil, http.StatusOK, globalList)
	for _, p := range globalList.Policies {
		require.NotEqual(t, "gitops global script", p.Name)
	}
}

func (s *integrationEnterpriseTestSuite) TestAllFleetsPolicyAutomationsBadRequest() {
	t := s.T()
	ctx := t.Context()

	script, err := s.ds.NewScript(ctx, &fleet.Script{Name: t.Name() + ".sh", ScriptContents: "echo"})
	require.NoError(t, err)
	policy, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)
	policyURL := fmt.Sprintf("/api/latest/fleet/policies/%d", policy.ID)

	// script_id alone can only change the script of a policy that already runs one, and "All fleets" policies can't
	// be given the script automation.
	res := s.Do("PATCH", policyURL, json.RawMessage(fmt.Sprintf(`{"script_id": %d}`, script.ID)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "script_id can only change the item")
	res = s.Do("PATCH", policyURL, json.RawMessage(fmt.Sprintf(`{"automation_type": "script", "script_id": %d}`, script.ID)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `"All fleets" policies can only have automation_type "webhook_or_ticket"`)
	s.Do("PATCH", policyURL, json.RawMessage(`{"script_id": null}`), http.StatusOK)
	s.Do("PATCH", policyURL, json.RawMessage(`{"script_id": 0}`), http.StatusOK)

	res = s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{{Name: t.Name() + " spec", Query: "SELECT 1;", SoftwareTitleID: new(uint(1))}},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "cannot have software_title_id set")
}

func (s *integrationEnterpriseTestSuite) TestCombinedPolicyEndpoints() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + " other"})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID))
		require.NoError(t, s.ds.DeleteTeam(context.Background(), otherTeam.ID))
	})
	script, err := s.ds.NewScript(ctx, &fleet.Script{Name: t.Name() + ".sh", ScriptContents: "echo hi", TeamID: &team.ID})
	require.NoError(t, err)

	type policyResp struct {
		Policy *fleet.Policy `json:"policy"`
	}
	create := func(version, body string, wantStatus int) *fleet.Policy {
		var resp policyResp
		s.DoJSON("POST", "/api/"+version+"/fleet/policies", json.RawMessage(body), wantStatus, &resp)
		return resp.Policy
	}
	names := func(policies []*fleet.Policy) []string {
		out := make([]string, 0, len(policies))
		for _, p := range policies {
			if strings.HasPrefix(p.Name, t.Name()) {
				out = append(out, strings.TrimPrefix(p.Name, t.Name()+" "))
			}
		}
		return out
	}
	list := func(query ...string) []string {
		var resp fleet.ListPoliciesResponse
		s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &resp, query...)
		return names(resp.Policies)
	}
	count := func(query ...string) int {
		var resp fleet.CountPoliciesResponse
		s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &resp, append(query, "query", t.Name())...)
		return resp.Count
	}

	// Create in each scope, on v1 and latest.
	global := create("v1", fmt.Sprintf(`{"name": "%s global", "query": "SELECT 1", "automation_type": "webhook_or_ticket"}`, t.Name()), http.StatusOK)
	require.Nil(t, global.TeamID)
	globalPlain := create("latest", fmt.Sprintf(`{"name": "%s global plain", "query": "SELECT 1"}`, t.Name()), http.StatusOK)
	teamScript := create("v1", fmt.Sprintf(`{"name": "%s fleet script", "query": "SELECT 1", "fleet_id": %d, "automation_type": "script", "script_id": %d}`,
		t.Name(), team.ID, script.ID), http.StatusOK)
	require.Equal(t, team.ID, *teamScript.TeamID)
	teamPlain := create("latest", fmt.Sprintf(`{"name": "%s fleet plain", "query": "SELECT 1", "fleet_id": %d}`, t.Name(), team.ID), http.StatusOK)
	unassigned := create("latest", fmt.Sprintf(`{"name": "%s unassigned", "query": "SELECT 1", "fleet_id": 0}`, t.Name()), http.StatusOK)
	require.Equal(t, uint(0), *unassigned.TeamID)
	other := create("latest", fmt.Sprintf(`{"name": "%s other fleet", "query": "SELECT 1", "fleet_id": %d}`, t.Name(), otherTeam.ID), http.StatusOK)
	t.Cleanup(func() {
		_, err := s.ds.DeleteGlobalPolicies(context.Background(), []uint{global.ID, globalPlain.ID})
		require.NoError(t, err)
		_, err = s.ds.DeleteTeamPolicies(context.Background(), 0, []uint{unassigned.ID})
		require.NoError(t, err)
	})

	// "All fleets" policies reject fleet-only fields and automations.
	for _, body := range []string{
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "automation_type": "calendar"}`, t.Name()),
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "platform": "darwin", "automation_type": "conditional_access"}`, t.Name()),
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "automation_type": "script", "script_id": %d}`, t.Name(), script.ID),
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "automation_type": "software", "software_package_id": 1}`, t.Name()),
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "patch_when_closed": true}`, t.Name()),
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "notify_before_patching": true}`, t.Name()),
		fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "patch_deadline": true}`, t.Name()),
	} {
		create("latest", body, http.StatusBadRequest)
	}

	// List and count by scope.
	require.ElementsMatch(t, []string{"global", "global plain"}, list("query", t.Name()))
	require.Equal(t, 2, count())
	require.ElementsMatch(t, []string{"fleet script", "fleet plain"}, list("fleet_id", fmt.Sprint(team.ID)))
	require.Equal(t, 2, count("fleet_id", fmt.Sprint(team.ID)))
	require.ElementsMatch(t, []string{"fleet script", "fleet plain", "global", "global plain"},
		list("fleet_id", fmt.Sprint(team.ID), "merge_inherited", "true"))
	require.Equal(t, 4, count("fleet_id", fmt.Sprint(team.ID), "merge_inherited", "true"))
	require.ElementsMatch(t, []string{"unassigned"}, list("fleet_id", "0"))
	require.Equal(t, 1, count("fleet_id", "0"))
	var v1List fleet.ListPoliciesResponse
	s.DoJSON("GET", "/api/v1/fleet/policies", nil, http.StatusOK, &v1List, "fleet_id", fmt.Sprint(otherTeam.ID))
	require.Equal(t, []string{"other fleet"}, names(v1List.Policies))
	// Clients such as fleetctl send these GETs without a body.
	res := s.DoRaw("GET", "/api/latest/fleet/policies", nil, http.StatusOK, "fleet_id", fmt.Sprint(otherTeam.ID))
	require.NoError(t, json.NewDecoder(res.Body).Decode(&v1List))
	require.Equal(t, []string{"other fleet"}, names(v1List.Policies))
	s.DoRaw("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, "fleet_id", fmt.Sprint(otherTeam.ID))

	// Filters work with and without fleet_id, and on inherited policies.
	require.Equal(t, []string{"global"}, list("automation_type", "webhook_or_ticket", "query", t.Name()))
	require.Equal(t, []string{"fleet script"}, list("fleet_id", fmt.Sprint(team.ID), "automation_type", "script"))
	require.Equal(t, []string{"global"}, list("fleet_id", fmt.Sprint(team.ID), "merge_inherited", "true", "automation_type", "webhook_or_ticket"))
	require.Equal(t, 1, count("fleet_id", fmt.Sprint(team.ID), "merge_inherited", "true", "automation_type", "webhook_or_ticket"))
	require.ElementsMatch(t, []string{"fleet script", "fleet plain"}, list("fleet_id", fmt.Sprint(team.ID), "type", "dynamic"))
	require.Empty(t, list("fleet_id", fmt.Sprint(team.ID), "type", "patch"))
	for _, q := range [][]string{{"automation_type", "scripts"}, {"automation_type", "other"}, {"automation_type", "patch"}, {"type", "nope"}} {
		res := s.Do("GET", "/api/latest/fleet/policies", nil, http.StatusUnprocessableEntity, q...)
		require.Contains(t, extractServerErrorText(res.Body), q[0], q)
		s.Do("GET", "/api/latest/fleet/policies/count", nil, http.StatusUnprocessableEntity, q...)
	}

	// Get and update by ID in any scope; fleet_id on update doesn't move the policy.
	var getResp fleet.GetPolicyByIDResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/policies/%d", teamScript.ID), nil, http.StatusOK, &getResp)
	require.Equal(t, script.ID, getResp.Policy.RunScript.ID)
	var modResp policyResp
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/policies/%d", teamPlain.ID),
		json.RawMessage(fmt.Sprintf(`{"description": "moved?", "fleet_id": %d}`, otherTeam.ID)), http.StatusOK, &modResp)
	require.Equal(t, team.ID, *modResp.Policy.TeamID)
	require.Equal(t, "moved?", modResp.Policy.Description)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", unassigned.ID), json.RawMessage(`{"automation_type": "webhook_or_ticket"}`),
		http.StatusOK, &modResp)
	require.Equal(t, fleet.PolicyAutomationWebhookOrTicket, *modResp.Policy.AutomationType)

	// Users with only fleet roles: a missing fleet_id is named, other fleets and "All fleets" can't be written, and a
	// delete that includes a policy they can't delete deletes nothing.
	fleetAdmin := &fleet.User{Name: t.Name(), Email: strings.ToLower(t.Name()) + "-admin@example.com",
		Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleAdmin}}}
	require.NoError(t, fleetAdmin.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, fleetAdmin)
	require.NoError(t, err)
	func() {
		old := s.token
		s.token = s.getTestToken(fleetAdmin.Email, test.GoodPassword)
		defer func() { s.token = old }()

		res := s.Do("POST", "/api/latest/fleet/policies", json.RawMessage(fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1"}`, t.Name())),
			http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), "fleet_id")
		create("latest", fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "fleet_id": %d}`, t.Name(), otherTeam.ID), http.StatusForbidden)
		s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", global.ID), json.RawMessage(`{"description": "x"}`), http.StatusForbidden)
		require.ElementsMatch(t, []string{"global", "global plain"}, list("query", t.Name()))

		s.Do("POST", "/api/latest/fleet/policies/delete", fleet.DeletePoliciesRequest{IDs: []uint{teamPlain.ID, other.ID}}, http.StatusForbidden)
		_, err := s.ds.Policy(ctx, teamPlain.ID)
		require.NoError(t, err)
	}()

	// Permission is checked before the "All fleets" fields.
	observer := &fleet.User{Name: t.Name() + " observer", Email: strings.ToLower(t.Name()) + "-observer@example.com",
		GlobalRole: new(fleet.RoleObserver)}
	require.NoError(t, observer.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, observer)
	require.NoError(t, err)
	func() {
		old := s.token
		s.token = s.getTestToken(observer.Email, test.GoodPassword)
		defer func() { s.token = old }()
		create("latest", fmt.Sprintf(`{"name": "%s x", "query": "SELECT 1", "automation_type": "script", "script_id": %d}`, t.Name(), script.ID),
			http.StatusForbidden)
	}()

	// Delete across scopes in one request.
	var delResp fleet.DeletePoliciesResponse
	s.DoJSON("POST", "/api/v1/fleet/policies/delete", fleet.DeletePoliciesRequest{IDs: []uint{teamPlain.ID, other.ID, globalPlain.ID}},
		http.StatusOK, &delResp)
	require.ElementsMatch(t, []uint{teamPlain.ID, other.ID, globalPlain.ID}, delResp.Deleted)
	require.ElementsMatch(t, []string{"global"}, list("query", t.Name()))
	s.Do("POST", "/api/latest/fleet/policies/delete", fleet.DeletePoliciesRequest{IDs: []uint{teamScript.ID, 999999}}, http.StatusNotFound)
	_, err = s.ds.Policy(ctx, teamScript.ID)
	require.NoError(t, err)

	// Patch policies: software_title_id names the app, and automation_type decides whether it patches.
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install", Filename: "dummy_installer.pkg", TeamID: &team.ID,
	}, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier)
			VALUES ('DummyApp', ?, 'darwin', 'com.example.dummy')`, strings.ToLower(t.Name())+"/darwin")
		if err != nil {
			return err
		}
		fmaID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		_, err = q.ExecContext(ctx, `UPDATE software_installers SET fleet_maintained_app_id = ? WHERE title_id = ? AND global_or_team_id = ?`,
			fmaID, titleID, team.ID)
		return err
	})
	installerID := func(policyID uint) *uint {
		p, err := s.ds.Policy(ctx, policyID)
		require.NoError(t, err)
		return p.SoftwareInstallerID
	}

	reportOnly := create("latest", fmt.Sprintf(`{"type": "patch", "fleet_id": %d, "software_title_id": %d}`, team.ID, titleID), http.StatusOK)
	require.Equal(t, titleID, reportOnly.PatchSoftware.SoftwareTitleID)
	require.Nil(t, reportOnly.AutomationType)
	require.Nil(t, installerID(reportOnly.ID))
	var patchList fleet.ListPoliciesResponse
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &patchList, "fleet_id", fmt.Sprint(team.ID), "type", "patch")
	require.Len(t, patchList.Policies, 1)
	require.Equal(t, reportOnly.ID, patchList.Policies[0].ID)

	// A different app is rejected, and automation_type alone turns patching on for the policy's app.
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID),
		json.RawMessage(fmt.Sprintf(`{"automation_type": "software", "software_title_id": %d}`, titleID+1000)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "can't be changed")
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID), json.RawMessage(`{"automation_type": "software"}`),
		http.StatusOK, &modResp)
	require.Equal(t, fleet.PolicyAutomationSoftware, *modResp.Policy.AutomationType)
	require.NotNil(t, installerID(reportOnly.ID))
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID),
		json.RawMessage(`{"patch_when_closed": true, "notify_before_patching": true, "continuous_automations_enabled": true}`), http.StatusBadRequest)

	// With another automation_type, the policy's own app can be sent back and only names the app.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID),
		json.RawMessage(fmt.Sprintf(`{"automation_type": null, "software_title_id": %d}`, titleID)), http.StatusOK, &modResp)
	require.Nil(t, modResp.Policy.AutomationType)
	require.Nil(t, installerID(reportOnly.ID))
	require.Equal(t, titleID, modResp.Policy.PatchSoftware.SoftwareTitleID)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID),
		json.RawMessage(fmt.Sprintf(`{"automation_type": "webhook_or_ticket", "software_title_id": %d}`, titleID)), http.StatusOK, &modResp)
	require.Equal(t, fleet.PolicyAutomationWebhookOrTicket, *modResp.Policy.AutomationType)
	require.Nil(t, installerID(reportOnly.ID))
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID),
		json.RawMessage(fmt.Sprintf(`{"automation_type": null, "software_title_id": %d}`, titleID+1000)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "can't be changed")
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", reportOnly.ID),
		json.RawMessage(fmt.Sprintf(`{"automation_type": "software", "software_title_id": %d}`, titleID)), http.StatusOK, &modResp)
	require.Equal(t, fleet.PolicyAutomationSoftware, *modResp.Policy.AutomationType)
	require.NotNil(t, installerID(reportOnly.ID))

	// Created with automation_type "software", it patches right away.
	_, err = s.ds.DeleteTeamPolicies(ctx, team.ID, []uint{reportOnly.ID})
	require.NoError(t, err)
	patching := create("latest", fmt.Sprintf(`{"type": "patch", "fleet_id": %d, "software_title_id": %d, "automation_type": "software"}`,
		team.ID, titleID), http.StatusOK)
	require.Equal(t, fleet.PolicyAutomationSoftware, *patching.AutomationType)
	require.NotNil(t, installerID(patching.ID))
}

func (s *integrationEnterpriseTestSuite) TestTeamPoliciesProprietary() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)
	hosts := make([]uint, 2)
	for i := range 2 {
		h, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hosts[i] = h.ID
	}
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, hosts))
	require.NoError(t, err)

	tpName := "TestPolicy3"
	tpParams := fleet.CreatePolicyRequest{
		TeamID:      &team1.ID,
		Name:        tpName,
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)
	assert.Equal(t, tpName, tpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", tpResp.Policy.Query)
	assert.Equal(t, "Some description", tpResp.Policy.Description)
	require.NotNil(t, tpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution", *tpResp.Policy.Resolution)
	assert.NotNil(t, tpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", tpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", tpResp.Policy.AuthorEmail)

	tpNameNew := "TestPolicy4"

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", tpResp.Policy.ID), []byte(fmt.Sprintf(`{
		"name": "%s",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some team resolution updated"
	}`, tpNameNew)), http.StatusOK)
	var mtpResp fleet.ModifyPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mtpResp)
	require.NoError(t, err)

	require.NotNil(t, mtpResp.Policy)
	assert.Equal(t, tpNameNew, mtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mtpResp.Policy.Description)
	require.NotNil(t, mtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *mtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mtpResp.Policy.Platform)

	gtpResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", tpResp.Policy.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK, &gtpResp)
	require.NotNil(t, gtpResp.Policy)
	assert.Equal(t, tpNameNew, gtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", gtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", gtpResp.Policy.Description)
	require.NotNil(t, gtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *gtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", gtpResp.Policy.Platform)

	policiesResponse := fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse, "fleet_id", fmt.Sprint(team1.ID))
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, tpNameNew, policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some team resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	require.Empty(t, s.listFleetInheritedPolicies(t, team1.ID))

	// test fleet policy count; with no inherited policies, the merged count matches.
	for _, q := range []string{"", tpNameNew, " " + tpNameNew + " ", " nomatch"} {
		want := 1
		if q == " nomatch" {
			want = 0
		}
		tpCountResp := fleet.CountPoliciesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &tpCountResp, "fleet_id", fmt.Sprint(team1.ID), "query", q)
		assert.Equal(t, want, tpCountResp.Count, q)
		tpCountResp = fleet.CountPoliciesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &tpCountResp, "fleet_id", fmt.Sprint(team1.ID), "query", q,
			"merge_inherited", "true")
		assert.Equal(t, want, tpCountResp.Count, q)
	}

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 2)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?fleet_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?fleet_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := fleet.DeletePoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeletePoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse, "fleet_id", fmt.Sprint(team1.ID))
	require.Empty(t, policiesResponse.Policies)
}

func (s *integrationEnterpriseTestSuite) TestTeamPoliciesProprietaryInvalid() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies-2",
		Description: "desc team1",
	})
	require.NoError(t, err)

	tpParams := fleet.CreatePolicyRequest{
		TeamID:      &team1.ID,
		Name:        "TestQuery3-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	teamPolicyID := tpResp.Policy.ID

	gpParams := fleet.CreatePolicyRequest{
		Name:        "TestQuery3-Global",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	globalPolicyID := gpResp.Policy.ID

	for _, tc := range []struct {
		tname      string
		testUpdate bool
		name       string
		query      string
		platforms  string
	}{
		{
			tname:      "empty query",
			testUpdate: true,
			name:       "Some name",
			query:      "",
		},
		{
			tname:      "empty name",
			testUpdate: true,
			name:       "",
			query:      "select 1;",
		},
		{
			tname:      "empty with space",
			testUpdate: true,
			name:       " ", // #3704
			query:      "select 1;",
		},
		{
			tname:      "Invalid query",
			testUpdate: true,
			name:       "Invalid query",
			query:      "",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			tpReq := fleet.CreatePolicyRequest{
				TeamID:   &team1.ID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			tpResp := fleet.CreatePolicyResponse{}
			s.DoJSON("POST", "/api/latest/fleet/policies", tpReq, http.StatusBadRequest, &tpResp)
			require.Nil(t, tpResp.Policy)

			if tc.testUpdate {
				tpReq := fleet.ModifyPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  new(tc.name),
						Query: new(tc.query),
					},
				}
				tpResp := fleet.ModifyPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", teamPolicyID), tpReq, http.StatusBadRequest, &tpResp)
				require.Nil(t, tpResp.Policy)
			}

			gpReq := fleet.CreatePolicyRequest{
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			gpResp := fleet.CreatePolicyResponse{}
			s.DoJSON("POST", "/api/latest/fleet/policies", gpReq, http.StatusBadRequest, &gpResp)
			require.Nil(t, tpResp.Policy)

			if tc.testUpdate {
				gpReq := fleet.ModifyPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  new(tc.name),
						Query: new(tc.query),
					},
				}
				gpResp := fleet.ModifyPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), gpReq, http.StatusBadRequest, &gpResp)
				require.Nil(t, tpResp.Policy)
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestHostDetailsPolicies() {
	t := s.T()

	hosts := s.createHosts(t)
	host1 := hosts[0]
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "HostDetailsPolicies-Team",
		Description: "desc team1",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID}))
	require.NoError(t, err)

	gpParams := fleet.CreatePolicyRequest{
		Name:        "HostDetailsPolicies",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)

	tpParams := fleet.CreatePolicyRequest{
		TeamID:      &team1.ID,
		Name:        "HostDetailsPolicies-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := fleet.CreatePolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)

	_, err = s.ds.RecordPolicyQueryExecutions(
		context.Background(),
		host1,
		map[uint]*bool{gpResp.Policy.ID: new(true)},
		time.Now(),
		false,
		nil,
	)
	require.NoError(t, err)

	resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host1.ID), nil, http.StatusOK)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var r struct {
		Host *fleet.HostDetailResponse `json:"host"`
		Err  error                     `json:"error,omitempty"`
	}
	err = json.Unmarshal(b, &r)
	require.NoError(t, err)
	require.NoError(t, r.Err)
	hd := r.Host.HostDetail
	policies := *hd.Policies
	require.Len(t, policies, 2)
	// Policies that did not run are listed before passing policies
	require.True(t, reflect.DeepEqual(tpResp.Policy.PolicyData, policies[0].PolicyData))
	require.Empty(t, policies[0].Response) // policy didn't "run"

	require.True(t, reflect.DeepEqual(gpResp.Policy.PolicyData, policies[1].PolicyData))
	require.Equal(t, "pass", policies[1].Response)

	// Try to create a global policy with an existing name.
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusConflict, &gpResp)
	// Try to create a team policy with an existing name.
	s.DoJSON("POST", "/api/latest/fleet/policies", tpParams, http.StatusConflict, &tpResp)
}

func (s *integrationEnterpriseTestSuite) TestTeamPoliciesTeamNotExists() {
	t := s.T()

	teamPoliciesResponse := fleet.ListPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusNotFound, &teamPoliciesResponse, "fleet_id", fmt.Sprint(9999999))
	require.Empty(t, teamPoliciesResponse.Policies)
}
