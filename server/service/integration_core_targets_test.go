package service

// Live query target tests for the core (no-license) suite.
//
// Belongs here: searching and counting the hosts, labels and fleets that can be
// targeted by a live query.
//
// Does not belong here: the report definitions themselves
// (integration_core_queries_test.go).

import (
	"context"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestSearchTargets() {
	t := s.T()
	hosts := s.createHosts(t)

	var builtinNames []string
	for name := range fleet.ReservedLabelNames() {
		builtinNames = append(builtinNames, name)
	}
	lblMap, err := s.ds.LabelIDsByName(context.Background(), builtinNames, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lblMap, len(builtinNames))

	// no search criteria
	var searchResp fleet.SearchTargetsResponse
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // the HostTargets.HostIDs are actually host IDs to *omit* from the search
	require.Len(t, searchResp.Targets.Labels, len(lblMap))
	require.Empty(t, searchResp.Targets.Teams)

	var lblIDs []uint
	for _, labelID := range lblMap {
		lblIDs = append(lblIDs, labelID)
	}

	searchResp = fleet.SearchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // no omitted host id
	require.Empty(t, searchResp.Targets.Labels)          // All built-in labels have been omitted (pre-selected)
	require.Empty(t, searchResp.Targets.Teams)

	searchResp = fleet.SearchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(1), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)-1) // one omitted host id
	require.Len(t, searchResp.Targets.Labels, len(lblMap)) // labels have not been omitted
	require.Empty(t, searchResp.Targets.Teams)

	searchResp = fleet.SearchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{MatchQuery: "foo.local1"}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, 1)
	require.Len(t, searchResp.Targets.Labels, 1) // with a match query, only matching label names and "All Hosts" can be returned (here, only all hosts)
	require.Empty(t, searchResp.Targets.Teams)
	require.Contains(t, searchResp.Targets.Hosts[0].Hostname, "foo.local1")
}

func (s *integrationTestSuite) TestCountTargets() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)
	require.Equal(t, "TestTeam", team.Name)

	hosts := s.createHosts(t)

	lblMap, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lblMap, 1)

	for i := range hosts {
		err = s.ds.RecordLabelQueryExecutions(context.Background(), hosts[i], map[uint]*bool{lblMap["All Hosts"]: new(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	var hostIDs []uint
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}

	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(new(team.ID), []uint{hostIDs[0]})) // nolint:nilaway // createHosts always returns at least one host
	require.NoError(t, err)

	var countResp fleet.CountTargetsResponse
	// sleep to reduce flake in last seen time so that online/offline counts can be tested
	time.Sleep(1 * time.Second)

	// none selected
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{}, http.StatusOK, &countResp)
	require.Equal(t, uint(0), countResp.TargetsCount)
	require.Equal(t, uint(0), countResp.TargetsOnline)
	require.Equal(t, uint(0), countResp.TargetsOffline)

	var lblIDs []uint
	for _, labelID := range lblMap {
		lblIDs = append(lblIDs, labelID)
	}
	// all hosts label selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &countResp)
	require.Equal(t, uint(3), countResp.TargetsCount)
	require.Equal(t, uint(1), countResp.TargetsOnline)
	require.Equal(t, uint(2), countResp.TargetsOffline)

	// team selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{TeamIDs: []uint{team.ID}}}, http.StatusOK, &countResp)
	require.Equal(t, uint(1), countResp.TargetsCount)
	require.Equal(t, uint(1), countResp.TargetsOnline)
	require.Equal(t, uint(0), countResp.TargetsOffline)

	// 'No team' selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON(
		"POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{TeamIDs: []uint{0}}},
		http.StatusOK, &countResp,
	)
	assert.Equal(t, uint(2), countResp.TargetsCount)
	assert.Equal(t, uint(0), countResp.TargetsOnline)
	assert.Equal(t, uint(2), countResp.TargetsOffline)

	// host id selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &countResp)
	require.Equal(t, uint(1), countResp.TargetsCount)
	require.Equal(t, uint(0), countResp.TargetsOnline)
	require.Equal(t, uint(1), countResp.TargetsOffline)
}
