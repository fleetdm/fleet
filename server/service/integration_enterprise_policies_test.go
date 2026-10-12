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
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

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

	ts := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
	require.Len(t, ts.InheritedPolicies, 0)

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

	tpParams := fleet.TeamPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some team resolution",
	}
	r := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &r)

	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	require.Len(t, ts.InheritedPolicies, 1)
	assert.Equal(t, gpol.Name, ts.InheritedPolicies[0].Name)
	assert.Equal(t, gpol.ID, ts.InheritedPolicies[0].ID)

	tc := fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tc)
	require.Nil(t, tc.Err)
	require.Equal(t, 1, tc.Count)
	require.Equal(t, 0, tc.InheritedPolicyCount)

	gc := fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &gc)
	require.Nil(t, gc.Err)
	require.Equal(t, 1, gc.Count)

	// Test merge inherited
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts, "merge_inherited", "true", "order_key", "team_id", "order_direction", "desc")
	require.Len(t, ts.Policies, 2)
	require.Nil(t, ts.InheritedPolicies)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	assert.Equal(t, gpol.Name, ts.Policies[1].Name)
	assert.Equal(t, gpol.ID, ts.Policies[1].ID)

	countResp := fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &countResp, "merge_inherited", "true")
	require.Nil(t, countResp.Err)
	require.Equal(t, 2, countResp.Count)
	require.Equal(t, 1, countResp.InheritedPolicyCount)

	// Test delete
	deletePolicyParams := fleet.DeleteTeamPoliciesRequest{IDs: []uint{ts.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
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
	regularPolicy := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Name:  "regular policy",
		Query: "SELECT 1;",
	}, http.StatusOK, &regularPolicy)

	// Create a software install policy
	installPolicy := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Name:            "install software policy",
		Query:           "SELECT 1 FROM some_table;",
		SoftwareTitleID: &rubyTitleID,
	}, http.StatusOK, &installPolicy)
	require.NotNil(t, installPolicy.Policy.InstallSoftware)

	// Create a patch policy
	patchPolicy := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Type:                 new("patch"),
		PatchSoftwareTitleID: &dummyTitleID,
		Hidden:               true,
	}, http.StatusOK, &patchPolicy)
	require.NotNil(t, patchPolicy.Policy.PatchSoftware)
	require.Equal(t, fleet.PolicyTypePatch, patchPolicy.Policy.Type)
	// Patch policies can be hidden from end users like any other policy.
	require.True(t, patchPolicy.Policy.Hidden)
	unhiddenPatch := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", team.ID, patchPolicy.Policy.ID),
		json.RawMessage(`{"hidden": false}`), http.StatusOK, &unhiddenPatch)
	require.False(t, unhiddenPatch.Policy.Hidden)

	// List all policies (no filter) - should return all 3
	listResp := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Policies, 3)

	// List with automation_type=software - should return only the install policy, not the patch policy
	listResp = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), nil, http.StatusOK, &listResp, "automation_type", "software")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, installPolicy.Policy.ID, listResp.Policies[0].ID)

	// List with merge_inherited and automation_type=software
	listResp = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), nil, http.StatusOK, &listResp, "merge_inherited", "true", "automation_type", "software")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, installPolicy.Policy.ID, listResp.Policies[0].ID)

	// List with automation_type=patch - should return only the patch policy
	listResp = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), nil, http.StatusOK, &listResp, "automation_type", "patch")
	require.Len(t, listResp.Policies, 1)
	assert.Equal(t, patchPolicy.Policy.ID, listResp.Policies[0].ID)

	// List with merge_inherited and automation_type=patch
	listResp = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", team.ID), nil, http.StatusOK, &listResp, "merge_inherited", "true", "automation_type", "patch")
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
	ts := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
	require.Len(t, ts.InheritedPolicies, 0)
	// Create a placeholder global policy.
	_, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:  "globalPolicy1",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	// Create a "No team" policy.
	tpParams := fleet.TeamPolicyRequest{
		Name:  "noTeamPolicy1",
		Query: "SELECT 1;",
	}
	r := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusOK, &r)
	require.NotNil(t, r.Policy.TeamID)
	require.Zero(t, *r.Policy.TeamID)
	// Test that we can't create a policy with the same name under "No team" domain.
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusConflict, &r)
	// Create a second "No team" policy.
	tpParams = fleet.TeamPolicyRequest{
		Name:  "noTeamPolicy2",
		Query: "SELECT 2;",
	}
	r = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusOK, &r)
	require.NotNil(t, r.Policy.TeamID)
	require.Zero(t, *r.Policy.TeamID)
	// List "No team" policies.
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 2)
	assert.Equal(t, "noTeamPolicy1", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 1;", ts.Policies[0].Query)
	require.NotNil(t, ts.Policies[0].TeamID)
	require.Zero(t, *ts.Policies[0].TeamID)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[1].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[1].Query)
	require.NotNil(t, ts.Policies[1].TeamID)
	require.Zero(t, *ts.Policies[1].TeamID)
	require.Len(t, ts.InheritedPolicies, 1)
	assert.Equal(t, "globalPolicy1", ts.InheritedPolicies[0].Name)
	assert.Equal(t, "SELECT 0;", ts.InheritedPolicies[0].Query)
	assert.Nil(t, ts.InheritedPolicies[0].TeamID)
	// Test policy count for "No team" policies.
	tc := fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies/count", nil, http.StatusOK, &tc)
	require.Equal(t, 2, tc.Count)
	require.Equal(t, 0, tc.InheritedPolicyCount)
	// Test merge inherited for "No team" policies.
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts, "merge_inherited", "true", "order_key", "team_id", "order_direction", "desc")
	require.Len(t, ts.Policies, 3)
	require.Nil(t, ts.InheritedPolicies)
	assert.Equal(t, "noTeamPolicy1", ts.Policies[0].Name)
	assert.Equal(t, "SELECT 1;", ts.Policies[0].Query)
	assert.Equal(t, "noTeamPolicy2", ts.Policies[1].Name)
	assert.Equal(t, "SELECT 2;", ts.Policies[1].Query)
	assert.Equal(t, "globalPolicy1", ts.Policies[2].Name)
	assert.Equal(t, "SELECT 0;", ts.Policies[2].Query)
	// Test merge inherited count for "No team" policies.
	countResp := fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies/count", nil, http.StatusOK, &countResp, "merge_inherited", "true")
	require.Nil(t, countResp.Err)
	require.Equal(t, 3, countResp.Count)
	require.Equal(t, 1, countResp.InheritedPolicyCount)
	// Test deleting "No team" policies.
	deletePolicyParams := fleet.DeleteTeamPoliciesRequest{
		IDs: []uint{ts.Policies[0].ID},
	}
	deletePolicyResp := fleet.DeleteTeamPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)
	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusOK, &ts)
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

	ts = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies", nil, http.StatusForbidden, &ts)
	tpParams = fleet.TeamPolicyRequest{
		Name:  "noTeamPolicy1",
		Query: "SELECT 1;",
	}
	r = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", tpParams, http.StatusForbidden, &r)
	tc = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0/policies/count", nil, http.StatusForbidden, &tc)
	deletePolicyParams = fleet.DeleteTeamPoliciesRequest{
		IDs: []uint{noTeamPolicy2.ID},
	}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies/delete", deletePolicyParams, http.StatusForbidden, &fleet.DeleteTeamPoliciesResponse{})
}

func (s *integrationEnterpriseTestSuite) TestGlobalPolicyCreateReadPatch() {
	t := s.T()
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	createPol1 := &fleet.GlobalPolicyResponse{}
	createPol1Req := &fleet.GlobalPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol1Req, http.StatusOK, &createPol1)
	allEqual(t, createPol1Req, createPol1.Policy, fields...)

	createPol2 := &fleet.GlobalPolicyResponse{}
	createPol2Req := &fleet.GlobalPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol2Req, http.StatusOK, &createPol2)
	allEqual(t, createPol2Req, createPol2.Policy, fields...)

	listPol := &fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(t, listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(t, createPol1.Policy, listPol.Policies[0])
	require.Equal(t, createPol2.Policy, listPol.Policies[1])

	// match policy by name with leading/trailing whitespace
	listPolByName := &fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPolByName, "query", " name1 ")
	require.Len(t, listPolByName.Policies, 1)
	require.Equal(t, listPolByName.Policies[0].Name, "name1")

	patchPol1Req := &fleet.ModifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        new("newName1"),
			Query:       new("newQuery"),
			Description: new("newDescription"),
			Resolution:  new("newResolution"),
			Platform:    new("windows"),
			Critical:    new(false),
		},
	}
	patchPol1 := &fleet.ModifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(t, patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &fleet.ModifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        new("newName2"),
			Query:       new("newQuery"),
			Description: new("newDescription"),
			Resolution:  new("newResolution"),
			Platform:    new("windows"),
			Critical:    new(true),
		},
	}
	patchPol2 := &fleet.ModifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(t, patchPol2Req, patchPol2.Policy, fields...)

	listPol = &fleet.ListGlobalPoliciesResponse{}
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
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical", "CalendarEventsEnabled", "Hidden"}

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &fleet.TeamPolicyResponse{}
	createPol1Req := &fleet.TeamPolicyRequest{
		Query:                 "query",
		Name:                  "name1",
		Description:           "description",
		Resolution:            "resolution",
		Platform:              "linux",
		Critical:              true,
		CalendarEventsEnabled: true,
		Hidden:                true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, fields...)

	createPol2 := &fleet.TeamPolicyResponse{}
	createPol2Req := &fleet.TeamPolicyRequest{
		Query:                 "query",
		Name:                  "name2",
		Description:           "description",
		Resolution:            "resolution",
		Platform:              "linux",
		Critical:              false,
		CalendarEventsEnabled: false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, fields...)

	listPol := &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(s.T(), createPol1.Policy, listPol.Policies[0])
	require.Equal(s.T(), createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:                  new("newName1"),
			Query:                 new("newQuery"),
			Description:           new("newDescription"),
			Resolution:            new("newResolution"),
			Platform:              new("windows"),
			Critical:              new(false),
			CalendarEventsEnabled: new(false),
			Hidden:                new(false),
		},
	}
	patchPol1 := &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:                  new("newName2"),
			Query:                 new("newQuery"),
			Description:           new("newDescription"),
			Resolution:            new("newResolution"),
			Platform:              new("windows"),
			Critical:              new(true),
			CalendarEventsEnabled: new(true),
			Hidden:                new(true),
		},
	}
	patchPol2 := &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)

	listPol = &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(s.T(), listPol.Policies[1], getPol2.Policy)

	// A patch that omits hidden leaves it untouched.
	patchPol2 = &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID),
		json.RawMessage(`{"name": "newName2b"}`), http.StatusOK, patchPol2)
	s.Require().True(patchPol2.Policy.Hidden)

	// hidden and conditional_access_enabled are mutually exclusive, on create and on either side of a patch.
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), &fleet.TeamPolicyRequest{
		Query: "query", Name: "hidden-ca", Platform: "darwin", Hidden: true, ConditionalAccessEnabled: true,
	}, http.StatusBadRequest)
	s.Require().Contains(extractServerErrorText(res.Body), `"hidden" and "conditional_access_enabled" cannot both be set`)

	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID),
		json.RawMessage(`{"conditional_access_enabled": true}`), http.StatusBadRequest)
	s.Require().Contains(extractServerErrorText(res.Body), `"hidden" and "conditional_access_enabled" cannot both be set`)

	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol1.Policy.ID),
		json.RawMessage(`{"conditional_access_enabled": true}`), http.StatusOK)
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol1.Policy.ID),
		json.RawMessage(`{"hidden": true}`), http.StatusBadRequest)
	s.Require().Contains(extractServerErrorText(res.Body), `"hidden" and "conditional_access_enabled" cannot both be set`)

	// "All fleets" policies can be hidden too.
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name: "hidden global", Query: "SELECT 1;", Hidden: true,
	}, http.StatusOK, &gpResp)
	s.Require().True(gpResp.Policy.Hidden)
	modGP := &fleet.ModifyGlobalPolicyResponse{}
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
func allEqual(t *testing.T, expect, actual interface{}, fields ...string) {
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
						var body map[string]interface{}
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
	var createResp fleet.GlobalPolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
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
	rej1Resp := s.Do("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:             "rej1-" + t.Name(),
		Query:            "SELECT 1",
		LabelsIncludeAll: []string{lblA.Name},
		LabelsIncludeAny: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rej1Resp.Body), fleet.ErrPolicyConflictingIncludeLabels.Error())

	// include_all + exclude_any is now a valid combination (one include scope
	// plus one exclude scope), so it succeeds and persists both.
	var combinedResp fleet.GlobalPolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:             "combined-" + t.Name(),
		Query:            "SELECT 1",
		LabelsIncludeAll: []string{lblA.Name},
		LabelsExcludeAny: []string{lblB.Name},
	}, http.StatusOK, &combinedResp)
	require.Len(t, combinedResp.Policy.LabelsIncludeAll, 1)
	require.Len(t, combinedResp.Policy.LabelsExcludeAny, 1)

	// Two exclude scopes (exclude_any + exclude_all) are still rejected.
	rejExclResp := s.Do("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:             "rej-excl-" + t.Name(),
		Query:            "SELECT 1",
		LabelsExcludeAny: []string{lblA.Name},
		LabelsExcludeAll: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejExclResp.Body), fleet.ErrPolicyConflictingExcludeLabels.Error())

	// 3. PATCH: switch existing include_all policy to include_any. Other slices should clear.
	switchAny := []string{lblA.Name}
	var patchResp fleet.ModifyGlobalPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createResp.Policy.ID), fleet.ModifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{LabelsIncludeAny: switchAny},
	}, http.StatusOK, &patchResp)
	require.Empty(t, patchResp.Policy.LabelsIncludeAll)
	require.Empty(t, patchResp.Policy.LabelsExcludeAny)
	require.Len(t, patchResp.Policy.LabelsIncludeAny, 1)

	// Switch back to include_all so end-to-end targeting test below uses it.
	switchAll := []string{lblA.Name, lblB.Name}
	patchResp = fleet.ModifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createResp.Policy.ID), fleet.ModifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{LabelsIncludeAll: switchAll},
	}, http.StatusOK, &patchResp)
	require.Len(t, patchResp.Policy.LabelsIncludeAll, 2)
	require.Empty(t, patchResp.Policy.LabelsIncludeAny)

	// PATCH mutex rejection. Assert on the specific validation error so a
	// generic 400 cannot silently satisfy the test.
	rejPatchResp := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createResp.Policy.ID), fleet.ModifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			LabelsIncludeAll: []string{lblA.Name},
			LabelsIncludeAny: []string{lblB.Name},
		},
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
	require.Contains(t, byName, validAnyName)
	require.Contains(t, byName, validAllName)
	require.Contains(t, byName, validExclName)
	require.Len(t, byName[validAnyName].LabelsIncludeAny, 1)
	require.Len(t, byName[validAllName].LabelsIncludeAll, 2)
	require.Len(t, byName[validExclName].LabelsExcludeAny, 1)
}

func (s *integrationEnterpriseTestSuite) TestResetPolicy() {
	t := s.T()
	ctx := context.Background()

	// Create all hosts upfront so each gets a unique osquery_host_id index.
	hosts := s.createHosts(t, "darwin", "darwin", "darwin")
	globalHost, noTeamHost, teamHost := hosts[0], hosts[1], hosts[2]

	// --- global policy ---
	createGlobalResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
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
	s.lastActivityMatches("reset_policy", fmt.Sprintf(`{"policy_id":%d,"policy_name":"reset-test-global","team_id":-1,"fleet_id":-1}`, globalPolicy.ID), 0)

	// --- no-team policy ---
	createNoTeamResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams/0/policies", fleet.TeamPolicyRequest{
		Name:  "reset-test-no-team",
		Query: "SELECT 1;",
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
	s.lastActivityMatches("reset_policy", fmt.Sprintf(`{"policy_id":%d,"policy_name":"reset-test-no-team","team_id":0,"fleet_id":0}`, noTeamPolicy.ID), 0)

	// --- team policy ---
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "reset-policy-team"})
	require.NoError(t, err)

	createTeamResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), fleet.TeamPolicyRequest{
		Name:  "reset-test-team",
		Query: "SELECT 1;",
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
		`{"policy_id":%d,"policy_name":"reset-test-team","team_id":%d,"fleet_id":%d,"team_name":"reset-policy-team","fleet_name":"reset-policy-team"}`,
		teamPolicy.ID, team.ID, team.ID,
	), 0)

	// 404 for a nonexistent policy.
	s.Do("POST", "/api/latest/fleet/policies/999999/reset", nil, http.StatusNotFound)

	// Both endpoints are documented under /api/v1 and must be routed there too.
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/policies/%d/reset", globalPolicy.ID), nil, http.StatusOK)
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/policies/%d/automation_activities", globalPolicy.ID), nil, http.StatusOK)

	// --- host-scoped reset (?host_id=) ---
	createHostScopedResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
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
		`{"policy_id":%d,"policy_name":"reset-test-host-scoped","team_id":-1,"fleet_id":-1,"host_id":%d,"host_display_name":%q}`,
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
	list := &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), nil, http.StatusOK, list)
	require.Len(t, list.Policies, 1)
	require.NotNil(t, list.Policies[0].RunScript)
	require.Equal(t, teamScript.ID, list.Policies[0].RunScript.ID)

	// A spec with no team cannot carry a script.
	res := s.Do("POST", specURL, spec("gitops global script", "", teamScript.ID), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "cannot have script_id set")

	// A script owned by another team is rejected.
	res = s.Do("POST", specURL, spec("gitops cross team script", team.Name, otherTeamScript.ID), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "does not belong to team ID")

	// A script that does not exist is rejected with a clear message rather than a database error.
	res = s.Do("POST", specURL, spec("gitops missing script", team.Name, otherTeamScript.ID+999), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "does not exist")

	// None of the rejected specs created a policy.
	list = &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), nil, http.StatusOK, list)
	require.Len(t, list.Policies, 1)
	globalList := &fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/global/policies", nil, http.StatusOK, globalList)
	for _, p := range globalList.Policies {
		require.NotEqual(t, "gitops global script", p.Name)
	}
}
