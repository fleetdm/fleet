package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsEnterprise(t *testing.T) {
	testingSuite := new(integrationEnterpriseTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationEnterpriseTestSuite struct {
	withServer
	suite.Suite
}

func (s *integrationEnterpriseTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationEnterpriseTestSuite")

	users, server := RunServerForTestsWithDS(
		s.T(), s.ds, TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecs() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}

	s.Do("POST", "/api/v1/fleet/teams", team, http.StatusOK)

	// updates a team
	agentOpts := json.RawMessage(`{"config": {"foo": "bar"}, "overrides": {"platforms": {"darwin": {"foo": "override"}}}}`)
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: &agentOpts}}}
	s.Do("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)

	assert.Len(t, team.Secrets, 0)
	require.JSONEq(t, string(agentOpts), string(*team.AgentOptions))

	// creates a team with default agent options
	user, err := s.ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)

	teams, err := s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	require.True(t, len(teams) >= 1)

	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2"}}}
	s.Do("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusOK)

	teams, err = s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.True(t, len(teams) >= 2)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)

	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/v1/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 0)
	require.NotNil(t, team.AgentOptions)
	require.JSONEq(t, defaultOpts, string(*team.AgentOptions))

	// updates secrets
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2", Secrets: []fleet.EnrollSecret{{Secret: "ABC"}}}}}
	s.Do("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)

	require.Len(t, team.Secrets, 1)
	assert.Equal(t, "ABC", team.Secrets[0].Secret)
}

func (s *integrationEnterpriseTestSuite) TestTeamSchedule() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	ts := getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)

	qr, err := s.ds.NewQuery(
		context.Background(),
		&fleet.Query{Name: "TestQueryTeamPolicy", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true},
	)
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{QueryID: &qr.ID, Interval: ptr.Uint(42)}}
	r := teamScheduleQueryResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule", team1.ID), gsParams, http.StatusOK, &r)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Equal(t, "TestQueryTeamPolicy", ts.Scheduled[0].Name)
	assert.Equal(t, qr.ID, ts.Scheduled[0].QueryID)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule/%d", team1.ID, id), modifyParams, http.StatusOK, &modifyResp)

	// just to satisfy my paranoia, wanted to make sure the contents of the json would work
	s.DoRaw("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule/%d", team1.ID, id), []byte(`{"interval": 77}`), http.StatusOK)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(77), ts.Scheduled[0].Interval)

	deleteResp := deleteTeamScheduleResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule/%d", team1.ID, id), nil, http.StatusOK, &deleteResp)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)
}

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

	password := "garbage"
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

	ts := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{Name: "TestQuery2", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true})
	require.NoError(t, err)

	tpParams := teamPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some team resolution",
	}
	r := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &r)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)

	deletePolicyParams := deleteTeamPoliciesRequest{IDs: []uint{ts.Policies[0].ID}}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
}

func (s *integrationEnterpriseTestSuite) TestModifyTeamEnrollSecrets() {
	t := s.T()

	// Create new team and set initial secret
	teamName := t.Name() + "secretTeam"
	team := &fleet.Team{
		Name:        teamName,
		Description: "secretTeam description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "initialSecret"}},
	}

	s.Do("POST", "/api/v1/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, team.Secrets[0].Secret, "initialSecret")

	// Test replace existing secrets
	req := json.RawMessage(`{"secrets": [{"secret": "testSecret1"},{"secret": "testSecret2"}]}`)
	var resp teamEnrollSecretsResponse

	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/secrets", team.ID), req, http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 2)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, "testSecret1", team.Secrets[0].Secret)
	assert.Equal(t, "testSecret2", team.Secrets[1].Secret)

	// Test delete all enroll secrets
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": []}`), http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 0)

	// Test bad requests
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"foo": [{"secret": "testSecret3"}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{}`), http.StatusUnprocessableEntity, &resp)
}

func (s *integrationEnterpriseTestSuite) TestAvailableTeams() {
	t := s.T()

	// create a new team
	team := &fleet.Team{
		Name:        "Available Team",
		Description: "Available Team description",
	}

	s.Do("POST", "/api/v1/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), "Available Team")
	require.NoError(t, err)

	// create a new user
	user := &fleet.User{
		Name:       "Available Teams User",
		Email:      "available@example.com",
		GlobalRole: ptr.String("observer"),
	}
	err = user.SetPassword("foobar123#", 10, 10)
	require.Nil(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.Nil(t, err)

	// test available teams for user assigned to global role
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Equal(t, ptr.String("observer"), getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 0)     // teams is empty if user has a global role
	assert.Len(t, getResp.AvailableTeams, 1) // available teams includes all teams if user has a global role
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

	// assign user to a team
	user.GlobalRole = nil
	user.Teams = []fleet.UserTeam{{Team: *team, Role: "maintainer"}}
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)

	// test available teams for user assigned to team role
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

	// test available teams returned by `/me` endpoint
	key := make([]byte, 64)
	sessionKey := base64.StdEncoding.EncodeToString(key)
	session := &fleet.Session{
		UserID:     user.ID,
		Key:        sessionKey,
		AccessedAt: time.Now().UTC(),
	}

	_, err = s.ds.NewSession(context.Background(), session)
	require.NoError(t, err)
	resp := s.DoRawWithHeaders("GET", "/api/v1/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", sessionKey),
	})
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	require.NoError(t, err)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")
}
