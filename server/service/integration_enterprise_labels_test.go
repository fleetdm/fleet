package service

// Label tests for the enterprise (premium) suite.
//
// Belongs here: fleet labels (CRUD, specs, host counts, team filtering, deletion) and fleet label
// membership through osquery distributed reads and writes.
//
// Does not belong here: label scoping of policies (integration_enterprise_policies_test.go),
// reports (integration_enterprise_reports_test.go), or software
// (integration_enterprise_software_test.go, integration_enterprise_software_installers_test.go).

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestTeamLabels() {
	t := s.T()

	// Create teams for testing team labels
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)

	// Create hosts on different teams
	teamHosts := s.createHosts(t, "darwin", "darwin")
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{teamHosts[0].ID})))
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team2.ID, []uint{teamHosts[1].ID})))

	// Create team labels using datastore directly (LabelPayload doesn't support TeamID)
	team1Label, err := s.ds.NewLabel(context.Background(), &fleet.Label{
		Name:                t.Name() + "_team1_label",
		Query:               "SELECT 1",
		TeamID:              &team1.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	team1LabelID := team1Label.ID

	// Create a team2 label using datastore
	team2Label, err := s.ds.NewLabel(context.Background(), &fleet.Label{
		Name:                "__team2_label",
		Query:               "SELECT 2",
		TeamID:              &team2.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	team2LabelID := team2Label.ID

	// Get the team label via API
	var getResp fleet.GetLabelResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", team1LabelID), nil, http.StatusOK, &getResp)
	require.Equal(t, team1LabelID, getResp.Label.ID)
	require.NotNil(t, getResp.Label.TeamID)
	require.Equal(t, team1.ID, *getResp.Label.TeamID)

	// List labels should include team labels for admin
	var listResp fleet.ListLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	foundTeamLabel := false
	foundTeam2Label := false
	for _, lbl := range listResp.Labels {
		if lbl.ID == team1LabelID {
			foundTeamLabel = true
			require.NotNil(t, lbl.TeamID)
			require.Equal(t, team1.ID, *lbl.TeamID)
		}
		if lbl.ID == team2LabelID {
			foundTeam2Label = true
			require.NotNil(t, lbl.TeamID)
			require.Equal(t, team2.ID, *lbl.TeamID)
		}
	}
	require.True(t, foundTeamLabel, "team label should be in list")
	require.True(t, foundTeam2Label, "team 2 label should be in list")

	// Filter to specific team
	listResp = fleet.ListLabelsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "team_id", fmt.Sprint(team1.ID))
	foundTeamLabel = false
	for _, lbl := range listResp.Labels {
		if lbl.ID == team1LabelID {
			foundTeamLabel = true
		}
		// Should not find team2 labels when filtering to team1
		if lbl.TeamID != nil && *lbl.TeamID == team2.ID {
			t.Fatal("should not find team2 labels when filtering to team1")
		}
	}
	require.True(t, foundTeamLabel, "team1 label should be in filtered list")

	// Filter to global only (team_id=0) should not include team labels
	listResp = fleet.ListLabelsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "team_id", "0")
	for _, lbl := range listResp.Labels {
		require.Nil(t, lbl.TeamID, "global-only filter should not return team labels")
	}

	// Modify team label
	var modResp fleet.ModifyLabelResponse
	newName := t.Name() + "_team1_label_modified"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", team1LabelID), &fleet.ModifyLabelPayload{
		Name: &newName,
	}, http.StatusOK, &modResp)
	require.Equal(t, newName, modResp.Label.Name)

	// Delete team labels
	var delResp fleet.DeleteLabelResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", team1LabelID), nil, http.StatusOK, &delResp)

	// Verify it's deleted
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", team1LabelID), nil, http.StatusNotFound, &getResp)

	// Delete team2 label by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", team2Label.Name), nil, http.StatusOK, &delResp)

	// Verify it's deleted
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", team2LabelID), nil, http.StatusNotFound, &getResp)

	/////// LABEL SPECS

	// Apply a team label spec
	teamLabelName := strings.ReplaceAll(t.Name(), "/", "_") + "_team1_label"
	var applyResp fleet.ApplyLabelSpecsResponse
	// make sure we fail loudly when we spec the request wrong
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                teamLabelName,
				Query:               "select 1",
				Platform:            "darwin",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				TeamID:              &team1.ID,
			},
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/spec/labels?team_id=%d", team1.ID), fleet.ApplyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                teamLabelName,
				Query:               "select 1",
				Platform:            "darwin",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
			},
		},
	}, http.StatusOK, &applyResp)

	// List label specs should include team label for admin
	var specsResp fleet.GetLabelSpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &specsResp)
	foundTeamLabel = false

	for _, spec := range specsResp.Specs {
		if spec.Name == teamLabelName {
			foundTeamLabel = true
			require.NotNil(t, spec.TeamID)
			require.Equal(t, team1.ID, *spec.TeamID)
		}
	}
	require.True(t, foundTeamLabel, "team label spec should be in list")

	// Get the specific team label spec
	var getSpecResp fleet.GetLabelSpecResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(teamLabelName)), nil, http.StatusOK, &getSpecResp)
	require.Equal(t, teamLabelName, getSpecResp.Spec.Name)
	require.NotNil(t, getSpecResp.Spec.TeamID)
	require.Equal(t, team1.ID, *getSpecResp.Spec.TeamID)

	// Filter specs to specific team
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &specsResp, "team_id", fmt.Sprint(team1.ID))
	foundTeamLabel = false
	for _, spec := range specsResp.Specs {
		if spec.Name == teamLabelName {
			foundTeamLabel = true
		}
		// Team2 labels should not be present
		if spec.TeamID != nil && *spec.TeamID == team2.ID {
			t.Fatal("should not find team2 labels when filtering to team1")
		}
	}
	require.True(t, foundTeamLabel, "team1 label spec should be in filtered list")

	// Filter specs to global only (team_id=0)
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &specsResp, "team_id", "0")
	for _, spec := range specsResp.Specs {
		require.Nil(t, spec.TeamID, "global-only filter should not return team labels")
	}

	/////// TEAM MANUAL LABELS RESTRICTIONS

	ctx := context.Background()

	// Create a global host for testing restrictions
	globalHost, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name() + "_global_host"),
		NodeKey:         new(t.Name() + "_global_host"),
		UUID:            t.Name() + "_global_host",
		Hostname:        t.Name() + "_global_host.local",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// Create a team admin user for team1
	teamAdminEmail := t.Name() + "_team_admin@example.com"
	teamAdmin := &fleet.User{
		Name:  teamAdminEmail,
		Email: teamAdminEmail,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, teamAdmin.SetPassword(test.GoodPassword, 10, 10))
	teamAdmin, err = s.ds.NewUser(ctx, teamAdmin)
	require.NoError(t, err)

	// Create a team label (manual) on team1
	teamManualLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                t.Name() + "_teamManualLabel1",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
		TeamID:              &team1.ID,
	})
	require.NoError(t, err)

	// Helper function to get host label names
	getHostLabels := func(host *fleet.Host) []string {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		labels := make([]string, 0, len(hostResp.Host.Labels))
		for _, lbl := range hostResp.Host.Labels {
			labels = append(labels, lbl.Name)
		}
		return labels
	}

	// Admin can add team label to team host (teamHosts[0] is on team1)
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHosts[0].ID), addLabelsToHostRequest{
		Labels: []string{teamManualLabel.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	team1HostLabels := getHostLabels(teamHosts[0])
	require.Contains(t, team1HostLabels, teamManualLabel.Name)

	// Cannot add team1 label to team2 host (teamHosts[1] is on team2)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHosts[1].ID), addLabelsToHostRequest{
		Labels: []string{teamManualLabel.Name},
	}, http.StatusBadRequest, &addLabelsToHostResp)

	// Cannot add team1 label to global host
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), addLabelsToHostRequest{
		Labels: []string{teamManualLabel.Name},
	}, http.StatusBadRequest, &addLabelsToHostResp)

	// Team admin can add their team's label to their team's host
	// First remove the label added by admin
	var removeLabelsFromHostResp fleet.RemoveLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHosts[0].ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{teamManualLabel.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHosts[0].ID), addLabelsToHostRequest{
		Labels: []string{teamManualLabel.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	team1HostLabels = getHostLabels(teamHosts[0])
	require.Contains(t, team1HostLabels, teamManualLabel.Name)

	// Team admin can remove their team's label from their team's host
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHosts[0].ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{teamManualLabel.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	team1HostLabels = getHostLabels(teamHosts[0])
	require.NotContains(t, team1HostLabels, teamManualLabel.Name)

	// Reset token to admin
	s.token = s.getTestAdminToken()

	// Clean up teams
	require.NoError(t, s.ds.DeleteTeam(context.Background(), team1.ID))
	require.NoError(t, s.ds.DeleteTeam(context.Background(), team2.ID))
}

func (s *integrationEnterpriseTestSuite) TestLabelsHostsCounts() {
	// ensure that on exit, the admin token is used
	defer func() { s.token = s.getTestAdminToken() }()

	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t, "debian", "linux", "fedora", "darwin", "darwin")
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// move a couple hosts to tm1, one to tm2
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm1.ID, []uint{hosts[0].ID, hosts[1].ID}))
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm2.ID, []uint{hosts[2].ID}))
	require.NoError(t, err)

	// create new users for tm1, tm2 and one with both tm1 and tm2
	users := []fleet.UserPayload{
		{
			Name:                     new("team1 user"),
			Email:                    new("tm1user@example.com"),
			Password:                 new(test.GoodPassword),
			AdminForcedPasswordReset: new(false),
			Teams: &[]fleet.UserTeam{
				{Team: fleet.Team{ID: tm1.ID}, Role: fleet.RoleMaintainer},
			},
		},
		{
			Name:                     new("team2 user"),
			Email:                    new("tm2user@example.com"),
			Password:                 new(test.GoodPassword),
			AdminForcedPasswordReset: new(false),
			Teams: &[]fleet.UserTeam{
				{Team: fleet.Team{ID: tm2.ID}, Role: fleet.RoleAdmin},
			},
		},
		{
			Name:                     new("team1and2 user"),
			Email:                    new("tm1and2user@example.com"),
			Password:                 new(test.GoodPassword),
			AdminForcedPasswordReset: new(false),
			Teams: &[]fleet.UserTeam{
				{Team: fleet.Team{ID: tm1.ID}, Role: fleet.RoleObserver},
				{Team: fleet.Team{ID: tm2.ID}, Role: fleet.RoleObserverPlus},
			},
		},
	}
	for _, u := range users {
		var createResp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users/admin", u, http.StatusOK, &createResp)
	}

	// create a manual label with hosts across no team, team1 and team2
	var createLbl fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "manual1",
			Hosts: []string{hosts[0].UUID, hosts[1].UUID, hosts[2].UUID, hosts[3].UUID},
		},
	}, http.StatusOK, &createLbl)
	// user is admin, count contains all hosts
	require.Equal(t, 4, createLbl.Label.Count)
	lblM1 := createLbl.Label.ID
	require.NotZero(t, lblM1)

	// create a dynamic label always returns a count of 0 (no members yet)
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "dynamic1",
			Query: "select 1",
		},
	}, http.StatusOK, &createLbl)
	require.Equal(t, 0, createLbl.Label.Count)
	lblD1 := createLbl.Label.ID
	require.NotZero(t, lblD1)

	// record membership for hosts across no team, team1 and team2
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[4], map[uint]*bool{lblD1: new(true)}, time.Now(), false)
	require.NoError(t, err)
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[2], map[uint]*bool{lblD1: new(true)}, time.Now(), false)
	require.NoError(t, err)
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[1], map[uint]*bool{lblD1: new(true)}, time.Now(), false)
	require.NoError(t, err)
	err = s.ds.RecordLabelQueryExecutions(ctx, hosts[0], map[uint]*bool{lblD1: new(true)}, time.Now(), false)
	require.NoError(t, err)

	// create another dynamic label which will stay empty
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "dynamic2",
			Query: "select 2",
		},
	}, http.StatusOK, &createLbl)
	require.Equal(t, 0, createLbl.Label.Count)
	lblD2 := createLbl.Label.ID
	require.NotZero(t, lblD2)

	// test access with each team user
	adminUserPayload := fleet.UserPayload{
		Name:     new("admin1"),
		Email:    new(testUsers["admin1"].Email),
		Password: new(testUsers["admin1"].PlaintextPassword),
	}
	cases := []struct {
		desc  string
		u     fleet.UserPayload
		lblID uint
		want  int
	}{
		{"team1 user, manual1", users[0], lblM1, 2},
		{"team1 user, dynamic1", users[0], lblD1, 2},
		{"team1 user, dynamic2", users[0], lblD2, 0},
		{"team2 user, manual1", users[1], lblM1, 1},
		{"team2 user, dynamic1", users[1], lblD1, 1},
		{"team2 user, dynamic2", users[1], lblD2, 0},
		{"team1 and 2 user, manual1", users[2], lblM1, 3},
		{"team1 and 2 user, dynamic1", users[2], lblD1, 3},
		{"team1 and 2 user, dynamic2", users[2], lblD2, 0},
		{"admin user, manual1", adminUserPayload, lblM1, 4},
		{"admin user, dynamic1", adminUserPayload, lblD1, 4},
		{"admin user, dynamic2", adminUserPayload, lblD2, 0},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			s.setTokenForTest(t, *c.u.Email, *c.u.Password)

			var getLbl fleet.GetLabelResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", c.lblID), nil, http.StatusOK, &getLbl)
			require.Equal(t, c.want, getLbl.Label.Count)

			var listLbls fleet.ListLabelsResponse
			s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listLbls)
			var found bool
			for _, lbl := range listLbls.Labels {
				if lbl.ID == c.lblID {
					found = true
					require.Equal(t, c.want, lbl.Count)
					break
				}
			}
			require.True(t, found)

			// create and update label and just not possible for non-global users
			if c.u != adminUserPayload && (*c.u.Teams)[0].Role != fleet.RoleMaintainer && (*c.u.Teams)[0].Role != fleet.RoleAdmin {
				s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
					LabelPayload: fleet.LabelPayload{
						Name:  "will fail",
						Query: "select 3",
					},
				}, http.StatusForbidden, &createLbl)

				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", c.lblID), fleet.ModifyLabelRequest{
					ModifyLabelPayload: fleet.ModifyLabelPayload{
						Name: new("will fail"),
					},
				}, http.StatusForbidden, &fleet.ModifyLabelResponse{})
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestLabelSpecHostsAreTeamFiltered() {
	defer func() { s.token = s.getTestAdminToken() }()

	t := s.T()
	ctx := t.Context()

	hosts := s.createHosts(t, "debian", "darwin", "windows")
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "label spec team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "label spec team2"})
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm1.ID, []uint{hosts[0].ID})))
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm2.ID, []uint{hosts[1].ID})))

	observer := fleet.UserPayload{
		Name:                     new("label spec team1 observer"),
		Email:                    new("labelspec-tm1observer@example.com"),
		Password:                 new(test.GoodPassword),
		AdminForcedPasswordReset: new(false),
		Teams:                    &[]fleet.UserTeam{{Team: fleet.Team{ID: tm1.ID}, Role: fleet.RoleObserver}},
	}
	var createUser createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users/admin", observer, http.StatusOK, &createUser)

	labelName := "labelspec-global-manual"
	var createLbl fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  labelName,
			Hosts: []string{hosts[0].UUID, hosts[1].UUID, hosts[2].UUID},
		},
	}, http.StatusOK, &createLbl)

	var getSpec fleet.GetLabelSpecResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(labelName)), nil, http.StatusOK, &getSpec)
	require.ElementsMatch(t,
		[]string{fmt.Sprint(hosts[0].ID), fmt.Sprint(hosts[1].ID), fmt.Sprint(hosts[2].ID)},
		[]string(getSpec.Spec.Hosts),
	)

	// the team observer must not learn about hosts outside their team
	s.setTokenForTest(t, *observer.Email, *observer.Password)

	getSpec = fleet.GetLabelSpecResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(labelName)), nil, http.StatusOK, &getSpec)
	require.Equal(t, []string{fmt.Sprint(hosts[0].ID)}, []string(getSpec.Spec.Hosts))

	var listSpecs fleet.GetLabelSpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listSpecs)
	var found bool
	for _, spec := range listSpecs.Specs {
		if spec.Name == labelName {
			found = true
			require.Equal(t, []string{fmt.Sprint(hosts[0].ID)}, []string(spec.Hosts))
		}
	}
	require.True(t, found, "global manual label spec should be listed for the team observer")

	// the members left out of the spec are the ones the observer can't reach directly
	for _, h := range []*fleet.Host{hosts[1], hosts[2]} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusForbidden, &getHostResponse{})
	}
}

func (s *integrationEnterpriseTestSuite) TestDeleteLabels() {
	t := s.T()

	// create a couple labels
	var newLabelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:     "TestDeleteLabels1",
		Platform: "darwin",
		Query:    "SELECT 1",
	}, http.StatusOK, &newLabelResp)
	lbl1 := newLabelResp.Label.ID
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:     "TestDeleteLabels2",
		Platform: "darwin",
		Query:    "SELECT 2",
	}, http.StatusOK, &newLabelResp)
	lbl2 := newLabelResp.Label.ID

	// create a software installer associated with the label
	installer := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "install",
		Filename:         "ruby.deb",
		SelfService:      false,
		TeamID:           nil,
		LabelsIncludeAny: []string{"TestDeleteLabels1"},
	}
	s.uploadSoftwareInstaller(t, installer, http.StatusOK, "")

	// try to delete the label associated with the installer
	res := s.Do("DELETE", "/api/v1/fleet/labels/id/"+fmt.Sprint(lbl1), nil, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "foreign key constraint on labels: TestDeleteLabels1")

	// try to delete a label that does not exist by id and name
	var delLabelResp fleet.DeleteLabelByIDResponse
	s.DoJSON("DELETE", "/api/v1/fleet/labels/id/"+fmt.Sprint(lbl1+1000), nil, http.StatusNotFound, &delLabelResp)
	s.DoJSON("DELETE", "/api/v1/fleet/labels/no-such-label", nil, http.StatusNotFound, &delLabelResp)

	// delete the unused label2
	s.DoJSON("DELETE", "/api/v1/fleet/labels/id/"+fmt.Sprint(lbl2), nil, http.StatusOK, &delLabelResp)
}

func (s *integrationEnterpriseTestSuite) TestTeamLabelsDistributedReadWrite() {
	t := s.T()

	t1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "t1",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "t2",
	})
	require.NoError(t, err)

	newHost := func(name string, platform string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(t.Context(), &fleet.Host{
			OsqueryHostID: new(t.Name() + name),
			NodeKey:       new(t.Name() + name),
			UUID:          uuid.New().String(),
			Hostname:      fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:      platform,
			TeamID:        teamID,

			LabelUpdatedAt: time.Now().Add(-2 * time.Hour),
		})
		require.NoError(t, err)
		return h
	}
	// Create host on team t1.
	macOST1 := newHost("macOST1", "darwin", &t1.ID)
	// Create host on team t2.
	macOST2 := newHost("macOST2", "darwin", &t2.ID)
	// Create host on "No team".
	ubuntuHostNoTeam := newHost("linuxHostNoTeam", "ubuntu", nil)

	// Mock live queries.
	s.lq.On("QueriesForHost", macOST1.ID).Return(map[string]string{}, nil)
	s.lq.On("QueriesForHost", macOST2.ID).Return(map[string]string{}, nil)
	s.lq.On("QueriesForHost", ubuntuHostNoTeam.ID).Return(map[string]string{}, nil)

	// Create label on team t1.
	l1t1, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l1t1",
		Query:               "SELECT t1;",
		TeamID:              &t1.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	// Create two dynamic labels on team t2.
	l2t2, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l2t2",
		Query:               "SELECT t2;",
		TeamID:              &t2.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	l3t2, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l3t2",
		Query:               "SELECT l3t2;",
		TeamID:              &t2.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	// Create a manual label on team t2.
	l4t2, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "l4t2",
		TeamID:              &t2.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	// Create global label.
	globalLabel, err := s.ds.NewLabel(t.Context(), &fleet.Label{
		Name:                "global",
		Query:               "SELECT global;",
		TeamID:              nil,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	user := &fleet.User{
		Name:       "global admin",
		Email:      "global_admin@example.com",
		GlobalRole: new(fleet.RoleAdmin),
	}
	err = user.SetPassword(test.GoodPassword, 10, 10)
	require.NoError(t, err)
	user, err = s.ds.NewUser(t.Context(), user)
	require.NoError(t, err)

	labelQueryName := func(labelID uint) string {
		return hostLabelQueryPrefix + fmt.Sprint(labelID)
	}

	// filterLabelQueries filters out non-label queries and builtin label queries.
	filterLabelQueries := func(queries map[string]string) map[string]string {
		allLabels, err := s.ds.ListLabels(t.Context(), fleet.TeamFilter{
			User: user,
		}, fleet.ListOptions{}, false)
		require.NoError(t, err)
		builtinLabels := make(map[string]struct{})
		for _, label := range allLabels {
			if label.LabelType == fleet.LabelTypeBuiltIn {
				builtinLabels[labelQueryName(label.ID)] = struct{}{}
			}
		}

		labelQueries := make(map[string]string)
		for queryName, querySQL := range queries {
			if !strings.HasPrefix(queryName, hostLabelQueryPrefix) {
				continue
			}
			if _, ok := builtinLabels[queryName]; ok {
				continue
			}
			labelQueries[queryName] = querySQL
		}
		return labelQueries
	}

	// Get distributed queries for macOST1.
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", getDistributedQueriesRequest{NodeKey: *macOST1.NodeKey}, http.StatusOK, &dqResp)
	labelQueries := filterLabelQueries(dqResp.Queries)
	require.Len(t, labelQueries, 2)
	require.Contains(t, labelQueries, labelQueryName(l1t1.ID))
	require.Equal(t, "SELECT t1;", labelQueries[labelQueryName(l1t1.ID)])
	require.Contains(t, labelQueries, labelQueryName(globalLabel.ID))
	require.Equal(t, "SELECT global;", labelQueries[labelQueryName(globalLabel.ID)])

	// Get distributed queries for macOST2.
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", getDistributedQueriesRequest{NodeKey: *macOST2.NodeKey}, http.StatusOK, &dqResp)
	labelQueries = filterLabelQueries(dqResp.Queries)
	require.Len(t, labelQueries, 3)
	require.Contains(t, labelQueries, labelQueryName(l2t2.ID))
	require.Equal(t, "SELECT t2;", labelQueries[labelQueryName(l2t2.ID)])
	require.Contains(t, labelQueries, labelQueryName(l3t2.ID))
	require.Equal(t, "SELECT l3t2;", labelQueries[labelQueryName(l3t2.ID)])
	require.Contains(t, labelQueries, labelQueryName(globalLabel.ID))
	require.Equal(t, "SELECT global;", labelQueries[labelQueryName(globalLabel.ID)])

	// Get distributed queries for ubuntuHostNoTeam.
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", getDistributedQueriesRequest{NodeKey: *ubuntuHostNoTeam.NodeKey}, http.StatusOK, &dqResp)
	labelQueries = filterLabelQueries(dqResp.Queries)
	require.Len(t, labelQueries, 1)
	require.Contains(t, labelQueries, labelQueryName(globalLabel.ID))
	require.Equal(t, "SELECT global;", labelQueries[labelQueryName(globalLabel.ID)])

	// macOST1 returns results for the label queries back.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithLabelResults(macOST1, map[uint]*bool{
		l1t1.ID:        new(true),
		globalLabel.ID: new(true),
		// Send fake result for t2 (*) to verify it gets filtered.
		// This could happen if the host was transferred to another team in between distributed/read and distributed/write.
		l2t2.ID: new(true),
	}), http.StatusOK, &distributedResp)

	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", macOST1.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, getHostResp.Host.Labels, 2)
	sort.Slice(getHostResp.Host.Labels, func(i, j int) bool {
		return getHostResp.Host.Labels[i].ID < getHostResp.Host.Labels[j].ID
	})
	require.Equal(t, getHostResp.Host.Labels[0].ID, l1t1.ID)
	require.Equal(t, getHostResp.Host.Labels[1].ID, globalLabel.ID)

	// macOST2 returns results for the label queries back.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithLabelResults(macOST2, map[uint]*bool{
		l2t2.ID:        new(true),
		l3t2.ID:        new(false), // runs but doesn't belong
		globalLabel.ID: new(true),
	}), http.StatusOK, &distributedResp)

	// Add manual label on macOST2.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", macOST2.ID), addLabelsToHostRequest{
		Labels: []string{l4t2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", macOST2.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, getHostResp.Host.Labels, 3)
	sort.Slice(getHostResp.Host.Labels, func(i, j int) bool {
		return getHostResp.Host.Labels[i].ID < getHostResp.Host.Labels[j].ID
	})
	require.Equal(t, getHostResp.Host.Labels[0].ID, l2t2.ID)
	require.Equal(t, getHostResp.Host.Labels[1].ID, l4t2.ID)
	require.Equal(t, getHostResp.Host.Labels[2].ID, globalLabel.ID)

	// Transferring hosts should clear team labels.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{macOST2.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", macOST2.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, getHostResp.Host.Labels, 1)
	require.Equal(t, getHostResp.Host.Labels[0].ID, globalLabel.ID)

	// Delete team should delete the labels and their membership.
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), nil, http.StatusOK, &delResp)

	// Membership is cleared.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", macOST1.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, getHostResp.Host.Labels, 1)
	require.Equal(t, getHostResp.Host.Labels[0].ID, globalLabel.ID)
	// l1t1 doesn't exist anymore.
	_, _, err = s.ds.Label(t.Context(), l1t1.ID, fleet.TeamFilter{
		User: user,
	})
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	// l2t2 is still around.
	_, _, err = s.ds.Label(t.Context(), l2t2.ID, fleet.TeamFilter{
		User: user,
	})
	require.NoError(t, err)
}
