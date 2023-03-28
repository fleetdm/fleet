package service

import (
	"context"
	"encoding/base64"
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

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	redisPool fleet.RedisPool

	lq *live_query_mock.MockLiveQuery
}

func (s *integrationEnterpriseTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationEnterpriseTestSuite")

	s.redisPool = redistest.SetupRedis(s.T(), "integration_enterprise", false, false, false)
	s.lq = live_query_mock.New(s.T())
	config := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Pool: s.redisPool,
		Lq:   s.lq,
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &config)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
}

func (s *integrationEnterpriseTestSuite) TearDownTest() {
	// reset the mock
	s.lq.Mock = mock.Mock{}
	s.withServer.commonTearDownTest(s.T())
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
	agentOpts := json.RawMessage(`{"config": {"views": {"foo": "bar"}}, "overrides": {"platforms": {"darwin": {"views": {"bar": "qux"}}}}}`)
	mdm := fleet.TeamSpecMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: "10.15.0",
			Deadline:       "2021-01-01",
		},
	}
	features := json.RawMessage(`{
    "enable_host_users": false,
    "enable_software_inventory": false,
    "additional_queries": {"foo": "bar"}
  }`)
	teamSpecs := applyTeamSpecsRequest{
		Specs: []*fleet.TeamSpec{
			{
				Name:         teamName,
				AgentOptions: agentOpts,
				Features:     &features,
				MDM:          mdm,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Len(t, team.Secrets, 1)
	require.JSONEq(t, string(agentOpts), string(*team.Config.AgentOptions))
	require.Equal(t, fleet.Features{
		EnableHostUsers:         false,
		EnableSoftwareInventory: false,
		AdditionalQueries:       ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
	}, team.Config.Features)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: "10.15.0",
			Deadline:       "2021-01-01",
		},
	}, team.Config.MDM)

	// an activity was created for team spec applied
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

	// dry-run with invalid agent options
	agentOpts = json.RawMessage(`{"config": {"nope": 1}}`)
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: agentOpts}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest, "dry_run", "true")

	// dry-run with empty body
	res := s.DoRaw("POST", "/api/latest/fleet/spec/teams", nil, http.StatusBadRequest, "force", "true")
	errBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(errBody), `"Expected JSON Body"`)

	// dry-run with invalid top-level key
	s.Do("POST", "/api/latest/fleet/spec/teams", json.RawMessage(`{
		"specs": [
			{"name": "team_name_1", "unknown_key": true}
		]
	}`), http.StatusBadRequest, "dry_run", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged

	// dry-run with valid agent options and custom macos settings
	agentOpts = json.RawMessage(`{"config": {"views": {"foo": "qux"}}}`)
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: agentOpts, MDM: fleet.TeamSpecMDM{MacOSSettings: map[string]interface{}{"custom_settings": []string{"foo", "bar"}}}}}}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

	// dry-run with macos disk encryption set to false, no error
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, MDM: fleet.TeamSpecMDM{MacOSSettings: map[string]interface{}{"enable_disk_encryption": false}}}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")

	// dry-run with macos disk encryption set to true
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, MDM: fleet.TeamSpecMDM{MacOSSettings: map[string]interface{}{"enable_disk_encryption": true}}}}}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

	// dry-run with valid agent options only
	agentOpts = json.RawMessage(`{"config": {"views": {"foo": "qux"}}}`)
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: agentOpts}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged
	require.Empty(t, team.Config.MDM.MacOSSettings.CustomSettings)         // unchanged
	require.False(t, team.Config.MDM.MacOSSettings.EnableDiskEncryption)   // unchanged

	// apply without agent options specified
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// agent options are unchanged, not cleared
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged

	// apply with agent options specified but null
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: json.RawMessage(`null`)}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// agent options are cleared
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Nil(t, team.Config.AgentOptions)

	// force with invalid agent options
	agentOpts = json.RawMessage(`{"config": {"foo": "qux"}}`)
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: agentOpts}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "force", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "qux"`)

	// force create new team with invalid top-level key
	s.Do("POST", "/api/latest/fleet/spec/teams", json.RawMessage(`{
		"specs": [
			{"name": "team_with_invalid_key", "unknown_key": true}
		]
	}`), http.StatusOK, "force", "true")

	_, err = s.ds.TeamByName(context.Background(), "team_with_invalid_key")
	require.NoError(t, err)

	// invalid agent options command-line flag
	agentOpts = json.RawMessage(`{"command_line_flags": {"nope": 1}}`)
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: agentOpts}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)

	// valid agent options command-line flag
	agentOpts = json.RawMessage(`{"command_line_flags": {"enable_tables": "abcd"}}`)
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: agentOpts}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"enable_tables": "abcd"`)

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

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 0) // no secret gets created automatically when creating a team via apply spec
	require.NotNil(t, team.Config.AgentOptions)
	require.JSONEq(t, defaultOpts, string(*team.Config.AgentOptions))
	require.Equal(t, appConfig.Features, team.Config.Features)

	// an activity was created for the newly created team via the applied spec
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

	// updates
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name:     "team2",
		Secrets:  []fleet.EnrollSecret{{Secret: "ABC"}},
		Features: nil,
	}}}
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
	require.Len(t, ts.InheritedPolicies, 0)

	// create a global policy
	gpol, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: "TestGlobalPolicy", Query: "SELECT 1"})
	require.NoError(t, err)
	defer func() {
		_, err := s.ds.DeleteGlobalPolicies(context.Background(), []uint{gpol.ID})
		require.NoError(t, err)
	}()

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
	require.Len(t, ts.InheritedPolicies, 1)
	assert.Equal(t, gpol.Name, ts.InheritedPolicies[0].Name)
	assert.Equal(t, gpol.ID, ts.InheritedPolicies[0].ID)

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

	// too many secrets
	secrets := createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": `+string(jsonMustMarshal(t, secrets))+`}`), http.StatusUnprocessableEntity, &resp)
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

	// create a team with too many secrets
	team3 := &fleet.Team{
		Name:        name + "lots_of_secrets",
		Description: "Team3 description",
		Secrets:     createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team3, http.StatusUnprocessableEntity, &tmResp)

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

	// modify team's disk encryption, impossible without mdm enabled
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSSettings: &fleet.MacOSSettings{EnableDiskEncryption: true},
		},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `Couldn't update macos_settings because MDM features aren't turned on in Fleet.`)

	// modify a team with a NULL config
	defaultFeatures := fleet.Features{}
	defaultFeatures.ApplyDefaultsForNewInstalls()
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = NULL WHERE id = ? `, team.ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features)

	// modify a team with an empty config
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = '{}' WHERE id = ? `, team.ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features)

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

	// modify team agent options with invalid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"x": "y"
	}`), http.StatusBadRequest, &tmResp)

	// modify team agent options with invalid options, but force-apply them
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"x": "y"
		}
	}`), http.StatusOK, &tmResp, "force", "true")
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"x": "y"`)

	// modify team agent options with valid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &tmResp)
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`)

	// modify team agent using invalid options with dry-run
	tmResp.Team = nil
	resp := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": "not-a-bool"
			}
		}
	}`), http.StatusBadRequest, "dry_run", "true")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "invalid value type at 'options.aws_debug': expected bool but got string")

	// modify team agent using valid options with dry-run
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": false
			}
		}
	}`), http.StatusOK, &tmResp, "dry_run", "true")
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`) // left unchanged

	// list activities, it should have created one for edited_agent_options
	s.lastActivityMatches(fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), fmt.Sprintf(`{"global": false, "team_id": %d, "team_name": %q}`, tm1ID, team.Name), 0)

	// modify team agent options - unknown team
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID+1), json.RawMessage(`{}`), http.StatusNotFound, &tmResp)

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

func (s *integrationEnterpriseTestSuite) TestMacOSUpdatesConfig() {
	t := s.T()

	// Create a team
	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	team.ID = tmResp.Team.ID

	// modify the team's config
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				MinimumVersion: "10.15.0",
				Deadline:       "2021-01-01",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline)
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2021-01-01"}`, team.ID, team.Name), 0)

	// only update the deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				MinimumVersion: "10.15.0",
				Deadline:       "2025-10-01",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline)
	lastActivity := s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2025-10-01"}`, team.ID, team.Name), 0)

	// sending a nil MacOSUpdate config doesn't modify anything
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{MDM: nil}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	// sending an empty MacOSUpdate empties both fields
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{MDM: &fleet.TeamPayloadMDM{MacOSUpdates: &fleet.MacOSUpdates{}}}, http.StatusOK, &tmResp)
	require.Empty(t, tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion)
	require.Empty(t, tmResp.Team.Config.MDM.MacOSUpdates.Deadline)
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "", "deadline": ""}`, team.ID, team.Name), 0)

	// error checks:

	// try to set an invalid deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				MinimumVersion: "10.15.0",
				Deadline:       "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an invalid minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				MinimumVersion: "10.15.0 (19A583)",
				Deadline:       "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a deadline but not a minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				Deadline: "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a minimum version but not a deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				MinimumVersion: "10.15.0 (19A583)",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
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

	token := "much_valid"
	host := createHostAndDeviceToken(t, s.ds, token)
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

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

	// add a policy execution
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host,
		map[uint]*bool{gpResp.Policy.ID: ptr.Bool(false)}, time.Now(), false))

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
	json.NewDecoder(res.Body).Decode(&listDevicePoliciesResp) //nolint:errcheck
	res.Body.Close()                                          //nolint:errcheck
	require.Len(t, listDevicePoliciesResp.Policies, 2)
	require.NoError(t, listDevicePoliciesResp.Err)

	// GET `/api/_version_/fleet/device/{token}`
	getDeviceHostResp := getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getDeviceHostResp) //nolint:errcheck
	res.Body.Close()                                     //nolint:errcheck
	require.NoError(t, getDeviceHostResp.Err)
	require.Equal(t, host.ID, getDeviceHostResp.Host.ID)
	require.False(t, getDeviceHostResp.Host.RefetchRequested)
	require.Equal(t, "http://example.com/logo", getDeviceHostResp.OrgLogoURL)
	require.Len(t, *getDeviceHostResp.Host.Policies, 2)

	// GET `/api/_version_/fleet/device/{token}/desktop`
	getDesktopResp := fleetDesktopResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
	require.NoError(t, res.Body.Close())
	require.NoError(t, getDesktopResp.Err)
	require.Equal(t, *getDesktopResp.FailingPolicies, uint(1))
}

// TestCustomTransparencyURL tests that Fleet Premium licensees can use custom transparency urls.
func (s *integrationEnterpriseTestSuite) TestCustomTransparencyURL() {
	t := s.T()

	token := "token_test_custom_transparency_url"
	createHostAndDeviceToken(t, s.ds, token)

	// confirm intitial default url
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// confirm device endpoint returns initial default url
	deviceResp := &transparencyURLResponse{}
	rawResp := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))

	// set custom url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, "customURL", acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns custom url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, "customURL", rawResp.Header.Get("Location"))

	// empty string applies default url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": ""}}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns default url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))
}

func (s *integrationEnterpriseTestSuite) TestDefaultAppleBMTeam() {
	t := s.T()

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(s.T(), err)

	var acResp appConfigResponse

	// try to set an invalid team name
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"apple_bm_default_team": "xyz"
		}
	}`), http.StatusUnprocessableEntity, &acResp)

	// get the appconfig, nothing changed
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.AppleBMDefaultTeam)

	// set to a valid team name
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_bm_default_team": %q
		}
	}`, tm.Name)), http.StatusOK, &acResp)
	require.Equal(t, tm.Name, acResp.MDM.AppleBMDefaultTeam)

	// get the appconfig, set to that team name
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, tm.Name, acResp.MDM.AppleBMDefaultTeam)
}

func (s *integrationEnterpriseTestSuite) TestMDMMacOSUpdates() {
	t := s.T()

	// keep the last activity, to detect newly created ones
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
	var lastActivity uint
	if len(activitiesResp.Activities) > 0 {
		lastActivity = activitiesResp.Activities[0].ID
	}

	checkInvalidConfig := func(config string) {
		// try to set an invalid config
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(config), http.StatusUnprocessableEntity, &acResp)

		// get the appconfig, nothing changed
		acResp = appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.Equal(t, fleet.MacOSUpdates{}, acResp.MDM.MacOSUpdates)

		// no activity got created
		activitiesResp = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
		require.Condition(t, func() bool {
			return (lastActivity == 0 && len(activitiesResp.Activities) == 0) ||
				(len(activitiesResp.Activities) > 0 && activitiesResp.Activities[0].ID == lastActivity)
		})
	}

	// missing minimum_version
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"deadline": "2022-01-01"
		}
	}}`)

	// missing deadline
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1"
		}
	}}`)

	// invalid deadline
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022"
		}
	}}`)

	// deadline includes timestamp
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022-01-01T00:00:00Z"
		}
	}}`)

	// minimum_version includes build info
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1 (ABCD)",
			"deadline": "2022-01-01"
		}
	}}`)

	// valid config
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "12.3.1",
					"deadline": "2022-01-01"
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion)
	require.Equal(t, "2022-01-01", acResp.MDM.MacOSUpdates.Deadline)

	// edited macos min version activity got created
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"2022-01-01", "minimum_version":"12.3.1", "team_id": null, "team_name": null}`, 0)

	// get the appconfig
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion)
	require.Equal(t, "2022-01-01", acResp.MDM.MacOSUpdates.Deadline)

	// update the deadline
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "12.3.1",
					"deadline": "2024-01-01"
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion)
	require.Equal(t, "2024-01-01", acResp.MDM.MacOSUpdates.Deadline)

	// another edited macos min version activity got created
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"2024-01-01", "minimum_version":"12.3.1", "team_id": null, "team_name": null}`, 0)

	// update something unrelated - the transparency url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)

	// clear the macos requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "",
					"deadline": ""
				}
			}
		}`), http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.MacOSUpdates.MinimumVersion)
	require.Empty(t, acResp.MDM.MacOSUpdates.Deadline)

	// edited macos min version activity got created with empty requirement
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"", "minimum_version":"", "team_id": null, "team_name": null}`, 0)

	// update again with empty macos requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "",
					"deadline": ""
				}
			}
		}`), http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.MacOSUpdates.MinimumVersion)
	require.Empty(t, acResp.MDM.MacOSUpdates.Deadline)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)
}

func (s *integrationEnterpriseTestSuite) TestSSOJITProvisioning() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)
	require.False(t, acResp.SSOSettings.EnableJITRoleSync)

	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": false
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)
	require.False(t, acResp.SSOSettings.EnableJITRoleSync)

	// users can't be created if SSO is disabled
	auth, body := s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")
	// ensure theresn't a user in the DB
	_, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// enable JIT provisioning
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": true,
			"enable_jit_role_sync": false
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.True(t, acResp.SSOSettings.EnableJITProvisioning)
	require.False(t, acResp.SSOSettings.EnableJITRoleSync)

	// a new user is created and redirected accordingly
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	// a successful redirect has this content
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	require.NoError(t, err)
	require.Equal(t, auth.UserID(), user.Email)

	// a new activity item is created
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.NotEmpty(t, activitiesResp.Activities)
	require.Condition(t, func() bool {
		for _, a := range activitiesResp.Activities {
			if (a.Type == fleet.ActivityTypeUserAddedBySSO{}.ActivityName()) && *a.ActorEmail == auth.UserID() {
				return true
			}
		}
		return false
	})

	// Test that roles are not updated for an existing user because enable_jit_role_sync is false.

	// Change role to global admin first.
	user.GlobalRole = ptr.String("admin")
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)
	// Login should NOT change the role to the default (global observer).
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	assert.Equal(t, "sso_user@example.com", auth.UserID())
	assert.Equal(t, "SSO User 1", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err = s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	require.NoError(t, err)
	require.NotNil(t, user.GlobalRole)
	require.Equal(t, *user.GlobalRole, "admin")

	// Test that roles are updated for an existing user because enable_jit_role_sync is true.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": true,
			"enable_jit_role_sync": true
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.True(t, acResp.SSOSettings.EnableJITProvisioning)
	require.True(t, acResp.SSOSettings.EnableJITRoleSync)
	// Login should change the role to the default role (global observer).
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	assert.Equal(t, "sso_user@example.com", auth.UserID())
	assert.Equal(t, "SSO User 1", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err = s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	require.NoError(t, err)
	require.NotNil(t, user.GlobalRole)
	require.Equal(t, *user.GlobalRole, "observer")

	// A user with pre-configured roles can be created
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	assert.Equal(t, "sso_user_3_global_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 3", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "admin",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// We cannot use NewTeam and must use adhoc SQL because the teams.id is
	// auto-incremented and other tests cause it to be different than what we need (ID=1).
	var execErr error
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, execErr = db.ExecContext(context.Background(), `INSERT INTO teams (id, name) VALUES (1, 'Foobar') ON DUPLICATE KEY UPDATE name = VALUES(name);`)
		return execErr
	})
	require.NoError(t, execErr)

	// Create a team for the test below.
	_, err = s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team_" + t.Name(),
		Description: "desc team_" + t.Name(),
	})
	require.NoError(t, err)

	// A user with pre-configured roles can be created
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_4_team_maintainer", "user123#")
	assert.Equal(t, "sso_user_4_team_maintainer@example.com", auth.UserID())
	assert.Equal(t, "SSO User 4", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "maintainer",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")
}

func (s *integrationEnterpriseTestSuite) TestDistributedReadWithFeatures() {
	t := s.T()

	// Global config has both features enabled
	spec := []byte(`
  features:
    additional_queries: null
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)

	// Team config has only additional queries enabled
	a := json.RawMessage(`{"time": "SELECT * FROM time"}`)
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          8324,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: false,
				AdditionalQueries:       &a,
			},
		},
	})
	require.NoError(t, err)

	// Create a host without a team
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "select 1 from osquery;"}, nil)

	// ensure we can read distributed queries for the host
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// get distributed queries for the host
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.NotContains(t, dqResp.Queries, "fleet_additional_query_time")

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries, "fleet_additional_query_time")
}

func (s *integrationEnterpriseTestSuite) TestListHosts() {
	t := s.T()

	// create a couple of hosts
	host1, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "2"),
		NodeKey:         ptr.String(t.Name() + "2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbar.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	host3, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "3"),
		NodeKey:         ptr.String(t.Name() + "3"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbaz.local", t.Name()),
		Platform:        "windows",
	})
	require.NoError(t, err)
	require.NotNil(t, host3)

	// set disk space information for some hosts (none provided for host3)
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host1.ID, 10.0, 2.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host2.ID, 40.0, 4.0))

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, 3)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host1.ID, resp.Hosts[0].ID)
	assert.Equal(t, 10.0, resp.Hosts[0].GigsDiskSpaceAvailable)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "100")
	require.Len(t, resp.Hosts, 2)

	// returns an error when the criteria is invalid (outside 1-100)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusInternalServerError, &resp, "low_disk_space", "101") // TODO: status code to be fixed with #4406
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusInternalServerError, &resp, "low_disk_space", "0")   // TODO: status code to be fixed with #4406

	// counting hosts works with and without the filter too
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp)
	require.Equal(t, 3, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "32")
	require.Equal(t, 1, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "100")
	require.Equal(t, 2, countResp.Count)

	// host summary returns counts for low disk space
	var summaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "low_disk_space", "32")
	require.Equal(t, uint(3), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(1), *summaryResp.LowDiskSpaceCount)

	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "platform", "windows", "low_disk_space", "32")
	require.Equal(t, uint(1), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// all possible filters
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux", "low_disk_space", "32")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// without low_disk_space, does not return the count
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.Nil(t, summaryResp.LowDiskSpaceCount)
}

func (s *integrationEnterpriseTestSuite) TestAppleMDMNotConfigured() {
	t := s.T()

	for _, route := range mdmAppleConfigurationRequiredEndpoints() {
		res := s.Do(route[0], route[1], nil, fleet.ErrMDMNotConfigured.StatusCode())
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())
	}

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	t.Cleanup(fleetdmSrv.Close)

	// Always accessible
	var reqCSRResp requestMDMAppleCSRResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	s.Do("POST", "/api/latest/fleet/mdm/apple/dep/key_pair", nil, http.StatusOK)
}

func (s *integrationEnterpriseTestSuite) TestGlobalPolicyCreateReadPatch() {
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	createPol1 := &globalPolicyResponse{}
	createPol1Req := &globalPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, fields...)

	createPol2 := &globalPolicyResponse{}
	createPol2Req := &globalPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, fields...)

	listPol := &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(s.T(), createPol1.Policy, listPol.Policies[0])
	require.Equal(s.T(), createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName1"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(false),
		},
	}
	patchPol1 := &modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName2"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(true),
		},
	}
	patchPol2 := &modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)

	listPol = &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(s.T(), listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicyCreateReadPatch() {
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &teamPolicyResponse{}
	createPol1Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, fields...)

	createPol2 := &teamPolicyResponse{}
	createPol2Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, fields...)

	listPol := &listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(s.T(), createPol1.Policy, listPol.Policies[0])
	require.Equal(s.T(), createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName1"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(false),
		},
	}
	patchPol1 := &modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName2"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(true),
		},
	}
	patchPol2 := &modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)

	listPol = &listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(s.T(), listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestResetAutomation() {
	ctx := context.Background()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &teamPolicyResponse{}
	createPol1Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)

	createPol2 := &teamPolicyResponse{}
	createPol2Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)

	createPol3 := &teamPolicyResponse{}
	createPol3Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name3",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol3Req, http.StatusOK, &createPol3)

	var tmResp teamResponse
	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team1.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://127/",
			PolicyIDs:      []uint{createPol1.Policy.ID, createPol2.Policy.ID},
			HostBatchSize:  12345,
		},
	}}, http.StatusOK, &tmResp)

	h1, err := s.ds.NewHost(ctx, &fleet.Host{})
	require.NoError(s.T(), err)

	err = s.ds.RecordPolicyQueryExecutions(ctx, h1, map[uint]*bool{
		createPol1.Policy.ID: ptr.Bool(false),
		createPol2.Policy.ID: ptr.Bool(false),
		createPol3.Policy.ID: ptr.Bool(false), // This policy is not activated for automation in config.
	}, time.Now(), false)
	require.NoError(s.T(), err)

	pfs, err := s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol1.Policy.ID, createPol2.Policy.ID, createPol3.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   []uint{team1.ID},
		PolicyIDs: nil,
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol2.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 1)
}

func (s *integrationEnterpriseTestSuite) TestOrbitConfigNudgeSettings() {
	t := s.T()

	// ensure the config is empty before starting
	s.applyConfig([]byte(`
  mdm:
    macos_updates:
      deadline: ""
      minimum_version: ""
 `))

	var resp orbitGetConfigResponse
	// missing orbit key
	s.DoJSON("POST", "/api/fleet/orbit/config", nil, http.StatusUnauthorized, &resp)

	// nudge config is empty if macos_updates is not set
	h := createOrbitEnrolledHost(t, "darwin", "h", s.ds)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)

	// set macos_updates
	s.applyConfig([]byte(`
  mdm:
    macos_updates:
      deadline: 2022-01-04
      minimum_version: 12.1.3
 `))

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err := fleet.NewNudgeConfig(fleet.MacOSUpdates{Deadline: "2022-01-04", MinimumVersion: "12.1.3"})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 04:00:00 +0000 UTC")

	// create a team with an empty macos_updates config
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	// add the host to the team
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{h.ID})
	require.NoError(t, err)

	// NudgeConfig should be empty
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 04:00:00 +0000 UTC")

	// modify the team config, add macos_updates config
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				Deadline:       "1992-01-01",
				MinimumVersion: "13.1.1",
			},
		},
	}, http.StatusOK, &tmResp)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.MacOSUpdates{Deadline: "1992-01-01", MinimumVersion: "13.1.1"})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "1992-01-01 04:00:00 +0000 UTC")

	// create a new host, still receives the global config
	h2 := createOrbitEnrolledHost(t, "darwin", "h2", s.ds)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h2.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.MacOSUpdates{Deadline: "2022-01-04", MinimumVersion: "12.1.3"})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 04:00:00 +0000 UTC")
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

func createHostAndDeviceToken(t *testing.T, ds *mysql.Datastore, token string) *fleet.Host {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	mysql.ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, host.ID, token)
		return err
	})
	return host
}

func (s *integrationEnterpriseTestSuite) TestListSoftware() {
	t := s.T()
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
	}
	require.NoError(t, s.ds.UpdateHostSoftware(ctx, host.ID, software))
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	bar := host.Software[0]
	if bar.Name != "bar" {
		bar = host.Software[1]
	}

	n, err := s.ds.InsertSoftwareVulnerabilities(
		ctx, []fleet.SoftwareVulnerability{
			{
				SoftwareID: bar.ID,
				CVE:        "cve-123",
			},
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.Equal(t, 1, int(n))

	require.NoError(t, s.ds.InsertCVEMeta(ctx, []fleet.CVEMeta{{
		CVE:              "cve-123",
		CVSSScore:        ptr.Float64(5.4),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        &now,
	}}))

	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now().UTC()))

	var resp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &resp)
	require.NotNil(t, resp)

	barPayload := resp.Software[0]
	if barPayload.Name != "bar" {
		barPayload = resp.Software[1]
	}

	fooPayload := resp.Software[1]
	if barPayload.Name != "bar" {
		barPayload = resp.Software[0]
	}

	require.Empty(t, fooPayload.Vulnerabilities)
	require.Len(t, barPayload.Vulnerabilities, 1)
	require.Equal(t, barPayload.Vulnerabilities[0].CVE, "cve-123")
	require.NotNil(t, barPayload.Vulnerabilities[0].CVSSScore, ptr.Float64Ptr(5.4))
	require.NotNil(t, barPayload.Vulnerabilities[0].EPSSProbability, ptr.Float64Ptr(0.5))
	require.NotNil(t, barPayload.Vulnerabilities[0].CISAKnownExploit, ptr.BoolPtr(true))
	require.Equal(t, barPayload.Vulnerabilities[0].CVEPublished, ptr.TimePtr(now))
}
