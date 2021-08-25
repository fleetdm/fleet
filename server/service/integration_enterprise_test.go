package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsEnterprise(t *testing.T) {
	suite.Run(t, new(integrationEnterpriseTestSuite))
}

type integrationEnterpriseTestSuite struct {
	abstractIntegrationTestSuite
}

func (s *integrationEnterpriseTestSuite) SetupSuite() {
	s.ds = mysql.CreateMySQLDS(s.T())
	test.AddAllHostsLabel(s.T(), s.ds)

	t := s.T()

	users, server := RunServerForTestsWithDS(s.T(), s.ds, TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierBasic}})
	s.server = server
	s.users = users
	s.token = getTestAdminToken(t, s.server)
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecs() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}

	resp := s.Do("POST", "/api/v1/fleet/teams", team, http.StatusOK)
	defer resp.Body.Close()

	// updates a team
	agentOpts := json.RawMessage(`{"config": {"foo": "bar"}, "overrides": {"platforms": {"darwin": {"foo": "override"}}}}`)
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: teamName, AgentOptions: &agentOpts}}}
	respUpdateTeam := s.Do("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusOK)
	defer respUpdateTeam.Body.Close()

	team, err := s.ds.TeamByName(teamName)
	require.NoError(t, err)

	assert.Len(t, team.Secrets, 0)
	require.JSONEq(t, string(agentOpts), string(*team.AgentOptions))

	// creates a team with default agent options
	user, err := s.ds.UserByEmail("admin1@example.com")
	require.NoError(t, err)

	teams, err := s.ds.ListTeams(fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	require.True(t, len(teams) >= 1)

	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2"}}}
	respUpdateTeam2 := s.Do("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusOK)
	defer respUpdateTeam2.Body.Close()

	teams, err = s.ds.ListTeams(fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.True(t, len(teams) >= 2)

	team, err = s.ds.TeamByName("team2")
	require.NoError(t, err)

	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/v1/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 0)
	require.NotNil(t, team.AgentOptions)
	require.JSONEq(t, defaultOpts, string(*team.AgentOptions))

	// updates secrets
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2", Secrets: []fleet.EnrollSecret{{Secret: "ABC"}}}}}
	respUpdateSecrets := s.Do("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusOK)
	defer respUpdateSecrets.Body.Close()

	team, err = s.ds.TeamByName("team2")
	require.NoError(t, err)

	require.Len(t, team.Secrets, 1)
	assert.Equal(t, "ABC", team.Secrets[0].Secret)
}

func (s *integrationEnterpriseTestSuite) TestTeamSchedule() {
	t := s.T()

	team1, err := s.ds.NewTeam(&fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	ts := getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)

	qr, err := s.ds.NewQuery(&fleet.Query{Name: "TestQuery2", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true})
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{QueryID: &qr.ID, Interval: ptr.Uint(42)}}
	r := teamScheduleQueryResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), gsParams, http.StatusOK, &r)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Equal(t, "TestQuery2", ts.Scheduled[0].Name)
	assert.Equal(t, qr.ID, ts.Scheduled[0].QueryID)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/team/%d/schedule/%d", team1.ID, id), modifyParams, http.StatusOK, &modifyResp)

	// just to satisfy my paranoia, wanted to make sure the contents of the json would work
	s.DoRaw("PATCH", fmt.Sprintf("/api/v1/fleet/team/%d/schedule/%d", team1.ID, id), []byte(`{"interval": 77}`), http.StatusOK)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(77), ts.Scheduled[0].Interval)

	deleteResp := deleteTeamScheduleResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/team/%d/schedule/%d", team1.ID, id), nil, http.StatusOK, &deleteResp)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)
}
