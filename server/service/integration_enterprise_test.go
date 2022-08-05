package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
		s.T(), s.ds, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})
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

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	// updates a team, no secret is provided so it will keep the one generated
	// automatically when the team was created.
	agentOpts := json.RawMessage(`{"config": {"foo": "bar"}, "overrides": {"platforms": {"darwin": {"foo": "override"}}}}`)
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: &agentOpts}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)

	assert.Len(t, team.Secrets, 1)
	require.JSONEq(t, string(agentOpts), string(*team.Config.AgentOptions))

	// creates a team with default agent options
	user, err := s.ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)

	teams, err := s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	require.True(t, len(teams) >= 1)

	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2"}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	teams, err = s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.True(t, len(teams) >= 2)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)

	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 0) // no secret gets created automatically when creating a team via apply spec
	require.NotNil(t, team.Config.AgentOptions)
	require.JSONEq(t, defaultOpts, string(*team.Config.AgentOptions))

	// updates secrets
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2", Secrets: []fleet.EnrollSecret{{Secret: "ABC"}}}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)

	qr, err := s.ds.NewQuery(
		context.Background(),
		&fleet.Query{Name: "TestQueryTeamPolicy", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true},
	)
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{QueryID: &qr.ID, Interval: ptr.Uint(42)}}
	r := teamScheduleQueryResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), gsParams, http.StatusOK, &r)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Equal(t, "TestQueryTeamPolicy", ts.Scheduled[0].Name)
	assert.Equal(t, qr.ID, ts.Scheduled[0].QueryID)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), modifyParams, http.StatusOK, &modifyResp)

	// just to satisfy my paranoia, wanted to make sure the contents of the json would work
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), []byte(`{"interval": 77}`), http.StatusOK)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(77), ts.Scheduled[0].Interval)

	deleteResp := deleteTeamScheduleResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), nil, http.StatusOK, &deleteResp)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
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

	ts := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{Name: "TestQuery2", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true})
	require.NoError(t, err)

	tpParams := teamPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some team resolution",
	}
	r := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &r)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)

	deletePolicyParams := deleteTeamPoliciesRequest{IDs: []uint{ts.Policies[0].ID}}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
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

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, team.Secrets[0].Secret, "initialSecret")

	// Test replace existing secrets
	req := json.RawMessage(`{"secrets": [{"secret": "testSecret1"},{"secret": "testSecret2"}]}`)
	var resp teamEnrollSecretsResponse

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), req, http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 2)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, "testSecret1", team.Secrets[0].Secret)
	assert.Equal(t, "testSecret2", team.Secrets[1].Secret)

	// Test delete all enroll secrets
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": []}`), http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 0)

	// Test bad requests
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"foo": [{"secret": "testSecret3"}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{}`), http.StatusUnprocessableEntity, &resp)
}

func (s *integrationEnterpriseTestSuite) TestAvailableTeams() {
	t := s.T()

	// create a new team
	team := &fleet.Team{
		Name:        "Available Team",
		Description: "Available Team description",
	}

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), "Available Team")
	require.NoError(t, err)

	// create a new user
	user := &fleet.User{
		Name:       "Available Teams User",
		Email:      "available@example.com",
		GlobalRole: ptr.String("observer"),
	}
	err = user.SetPassword(test.GoodPassword, 10, 10)
	require.Nil(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.Nil(t, err)

	// test available teams for user assigned to global role
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

	// test available teams returned by `/me` endpoint
	key := make([]byte, 64)
	sessionKey := base64.StdEncoding.EncodeToString(key)
	_, err = s.ds.NewSession(context.Background(), user.ID, sessionKey)
	require.NoError(t, err)
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
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

func (s *integrationEnterpriseTestSuite) TestTeamEndpoints() {
	t := s.T()

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// create a new team
	team := &fleet.Team{
		Name:        name,
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "DEF"}},
	}

	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	assert.Equal(t, team.Name, tmResp.Team.Name)
	require.Len(t, tmResp.Team.Secrets, 1)
	assert.Equal(t, "DEF", tmResp.Team.Secrets[0].Secret)

	// create a duplicate team (same name)
	team2 := &fleet.Team{
		Name:        name,
		Description: "Team2 description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "GHI"}},
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team2, http.StatusConflict, &tmResp)

	// list teams
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp, "query", name, "per_page", "2")
	require.Len(t, listResp.Teams, 1)
	assert.Equal(t, team.Name, listResp.Teams[0].Name)
	assert.NotNil(t, listResp.Teams[0].Config.AgentOptions)
	tm1ID := listResp.Teams[0].ID

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, team.Name, getResp.Team.Name)
	assert.NotNil(t, getResp.Team.Config.AgentOptions)

	// modify team
	team.Description = "Alt " + team.Description
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Contains(t, tmResp.Team.Description, "Alt ")

	// modify non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID+1), team, http.StatusNotFound, &tmResp)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp)
	assert.Len(t, usersResp.Users, 0)

	// list team users - non-existing team
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), nil, http.StatusNotFound, &usersResp)

	// create a new user
	user := &fleet.User{
		Name:       "Team User",
		Email:      "user@example.com",
		GlobalRole: ptr.String("observer"),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	user, err := s.ds.NewUser(context.Background(), user)
	require.NoError(t, err)

	// add a team user
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 1)
	assert.Equal(t, user.ID, tmResp.Team.Users[0].ID)

	// add a team user - non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusNotFound, &tmResp)

	// add a team user - invalid user role
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: "foobar"}}}, http.StatusUnprocessableEntity, &tmResp)

	// search for that user
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp, "query", "user")
	require.Len(t, usersResp.Users, 1)
	assert.Equal(t, user.ID, usersResp.Users[0].ID)

	// search for unknown user
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp, "query", "notauser")
	require.Len(t, usersResp.Users, 0)

	// delete team user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID}}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 0)

	// delete team user - unknown user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID + 1}}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 0)

	// delete team user - unknown team
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID}}}}, http.StatusNotFound, &tmResp)

	// modify team agent options (options for orbit/osquery)
	tmResp.Team = nil
	opts := map[string]string{"x": "y"}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), opts, http.StatusOK, &tmResp)
	var m map[string]string
	require.NoError(t, json.Unmarshal(*tmResp.Team.Config.AgentOptions, &m))
	assert.Equal(t, opts, m)

	// modify team agent options - unknown team
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID+1), opts, http.StatusNotFound, &tmResp)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1ID), nil, http.StatusOK, &secResp)
	require.Len(t, secResp.Secrets, 1)
	assert.Equal(t, team.Secrets[0].Secret, secResp.Secrets[0].Secret)

	// get team enroll secrets- unknown team: does not return 404 because reads directly
	// the secrets table, does not load the team first (which would be unnecessary except
	// for checking that it exists)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1ID+1), nil, http.StatusOK, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &delResp)

	// delete team again, now an unknown team
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusNotFound, &delResp)
}

func (s *integrationEnterpriseTestSuite) TestExternalIntegrationsTeamConfig() {
	t := s.T()

	// create a test http server to act as the Jira and Zendesk server
	srvURL := startExternalServiceWebServer(t)

	// create a new team
	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	require.Len(t, tmResp.Team.Secrets, 1)
	require.Equal(t, "XYZ", tmResp.Team.Secrets[0].Secret)
	team.ID = tmResp.Team.ID

	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusOK, &tmResp)
	require.True(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://example.com", tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	// add an unknown automation - does not exist at the global level
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Jira: []*fleet.TeamJiraIntegration{
			{
				URL:                   srvURL,
				ProjectKey:            "qux",
				EnableFailingPolicies: false,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a couple Jira integrations at the global level (qux and qux2)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// enable an automation - should fail as the webhook is enabled too
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Jira: []*fleet.TeamJiraIntegration{
			{
				URL:                   srvURL,
				ProjectKey:            "qux",
				EnableFailingPolicies: true,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// get the team, no integration was saved
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 0)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 0)

	// disable the webhook and enable the automation
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a second, disabled Jira integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux2",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 2)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.Equal(t, "qux2", tmResp.Team.Config.Integrations.Jira[1].ProjectKey)

	// enabling the second without disabling the first fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux2",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// updating to use the same project key fails (must be unique)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// remove second integration, disable first so that nothing is enabled now
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// enable the webhook now works
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusOK, &tmResp)

	// set environmental varible to use Zendesk test client
	t.Setenv("TEST_ZENDESK_CLIENT", "true")

	// add an unknown automation - does not exist at the global level
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Zendesk: []*fleet.TeamZendeskIntegration{
			{
				URL:                   srvURL,
				GroupID:               122,
				EnableFailingPolicies: false,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a couple Zendesk integrations at the global level (122 and 123), keep the jira ones too
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "a@b.c",
					"api_token": "ok",
					"group_id": 122
				},
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// enable a Zendesk automation - should fail as the webhook is enabled too
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Zendesk: []*fleet.TeamZendeskIntegration{
			{
				URL:                   srvURL,
				GroupID:               122,
				EnableFailingPolicies: true,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// disable the webhook and enable the automation
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a second, disabled Zendesk integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 2)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.Equal(t, int64(123), tmResp.Team.Config.Integrations.Zendesk[1].GroupID)

	// enabling the second without disabling the first fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// updating to use the same group ID fails (must be unique per group ID)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// remove second Zendesk integration, add disabled Jira integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enabling a Jira integration when a Zendesk one is enabled fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// set additional integrations on the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// removing Zendesk 122 from the global config removes it from the team too
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// get the team, only one Zendesk integration remains, none are enabled
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.False(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), getResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.False(t, getResp.Team.Config.Integrations.Zendesk[0].EnableFailingPolicies)

	// removing Jira qux2 from the global config does not impact the team as it is unused.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// get the team, integrations are unchanged
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.False(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), getResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.False(t, getResp.Team.Config.Integrations.Zendesk[0].EnableFailingPolicies)

	// enable Jira qux for the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// removing Zendesk 123 from the global config removes it from the team but
	// leaves the Jira integration enabled.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.True(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 0)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{},
			Jira:    []*fleet.TeamJiraIntegration{},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 0)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 0)
	require.False(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Empty(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {}
	}`), http.StatusOK)
}

func (s *integrationEnterpriseTestSuite) TestListDevicePolicies() {
	t := s.T()

	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	ac.OrgInfo.OrgLogoURL = "http://example.com/logo"
	err = s.ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          51,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   t.Name(),
		NodeKey:         t.Name(),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	// create an auth token for host
	token := "much_valid"
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, host.ID, token)
		return err
	})

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQueryEnterpriseGlobalPolicy",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	// add a global policy
	gpParams := globalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	// add a policy to team
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	password := test.GoodPassword
	email := "test_enterprise_policies@user.com"

	u := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleMaintainer,
			},
		},
	}

	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)
	tpParams := teamPolicyRequest{
		Name:        "TestQueryEnterpriseTeamPolicy",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), tpParams, http.StatusOK, &tpResp)

	// try with invalid token
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/invalid_token/policies", nil, http.StatusUnauthorized)
	res.Body.Close()

	// GET `/api/_version_/fleet/device/{token}/policies`
	listDevicePoliciesResp := listDevicePoliciesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/policies", nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&listDevicePoliciesResp)
	res.Body.Close()
	require.Len(t, listDevicePoliciesResp.Policies, 2)
	require.NoError(t, listDevicePoliciesResp.Err)

	// GET `/api/_version_/fleet/device/{token}`
	getDeviceHostResp := getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
	res.Body.Close()
	require.NoError(t, getDeviceHostResp.Err)
	require.Equal(t, host.ID, getDeviceHostResp.Host.ID)
	require.False(t, getDeviceHostResp.Host.RefetchRequested)
	require.Equal(t, "http://example.com/logo", getDeviceHostResp.OrgLogoURL)
	require.Len(t, *getDeviceHostResp.Host.Policies, 2)
}

// TestCustomTransparencyURL tests that Fleet Premium licensees can use custom transparency urls.
func (s *integrationEnterpriseTestSuite) TestCustomTransparencyURL() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   t.Name(),
		NodeKey:         t.Name(),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// create device token for host
	token := "token_test_custom_transparency_url"
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, host.ID, token)
		return err
	})

	// confirm intitial default url
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// confirm device endpoint returns initial default url
	deviceResp := &transparencyURLResponse{}
	rawResp := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp)
	rawResp.Body.Close()
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))

	// set custom url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", fleet.AppConfig{FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: "customURL"}}, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, "customURL", acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns custom url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp)
	rawResp.Body.Close()
	require.NoError(t, deviceResp.Err)
	require.Equal(t, "customURL", rawResp.Header.Get("Location"))

	// empty string applies default url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", fleet.AppConfig{FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: ""}}, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns default url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp)
	rawResp.Body.Close()
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))
}
