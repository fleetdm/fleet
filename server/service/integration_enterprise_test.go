package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

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
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{Name: "TestQuery2", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true})
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
