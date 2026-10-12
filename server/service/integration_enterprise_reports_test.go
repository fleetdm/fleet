package service

// Report tests for the enterprise (premium) suite.
//
// Belongs here: fleet reports and schedules, report label scoping, and a host's report list.
//
// Does not belong here: label CRUD (integration_enterprise_labels_test.go) and policies
// (integration_enterprise_policies_test.go).

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	require.Empty(t, ts.Scheduled)

	qr, err := s.ds.NewQuery(
		context.Background(),
		&fleet.Query{
			Name:           "TestQueryTeamPolicy",
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Saved:          true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{
		QueryID:  &qr.ID,
		Interval: new(uint(42))}
	r := teamScheduleQueryResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), gsParams, http.StatusOK, &r)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Contains(t, ts.Scheduled[0].Name, "Copy of TestQueryTeamPolicy")
	assert.NotEqual(t, qr.ID, ts.Scheduled[0].QueryID) // it creates a new query (copy)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{Interval: new(uint(55))}
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
	require.Empty(t, ts.Scheduled)
}

func (s *integrationEnterpriseTestSuite) TestTeamQueries() {
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

	// create global query
	params := fleet.QueryPayload{
		Name:  new("global1"),
		Query: new("select * from time;"),
	}
	var createQueryResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer s.cleanupQuery(createQueryResp.Query.ID)

	// create team query
	params = fleet.QueryPayload{
		Name:   new("team1"),
		Query:  new("select * from time;"),
		TeamID: new(team1.ID),
	}
	createQueryResp = fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer s.cleanupQuery(createQueryResp.Query.ID)

	// list team queries
	var listQueriesResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQueriesResp, "team_id", fmt.Sprint(team1.ID))
	require.Len(t, listQueriesResp.Queries, 1)
	assert.Equal(t, "team1", listQueriesResp.Queries[0].Name)
	assert.Equal(t, 1, listQueriesResp.Count)
	assert.Equal(t, 0, listQueriesResp.InheritedQueryCount)

	// list merged team queries
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQueriesResp, "team_id", fmt.Sprint(team1.ID), "merge_inherited", "true", "order_key", "team_id", "order_direction", "desc")
	require.Len(t, listQueriesResp.Queries, 2)
	assert.Equal(t, "team1", listQueriesResp.Queries[0].Name)
	assert.Equal(t, "global1", listQueriesResp.Queries[1].Name)
	assert.Equal(t, 2, listQueriesResp.Count)
	assert.Equal(t, 1, listQueriesResp.InheritedQueryCount)
}

func (s *integrationEnterpriseTestSuite) cleanupQuery(queryID uint) {
	var delResp fleet.DeleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", queryID), nil, http.StatusOK, &delResp)
}

// TestQueryLabelsIncludeAll mirrors TestPolicyLabelsIncludeAll for queries (reports),
// covering the new LabelsIncludeAll field. Reports do not support exclude_any,
// so the 2-way mutex (include_any vs include_all) is exercised here.
func (s *integrationEnterpriseTestSuite) TestQueryLabelsIncludeAll() {
	t := s.T()
	ctx := context.Background()

	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lblA := lblResp.Label
	lblResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 2"}, http.StatusOK, &lblResp)
	lblB := lblResp.Label

	// Create a query with include_all.
	autoOn := true
	var createResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.QueryPayload{
		Name:               new("q-include-all-" + t.Name()),
		Query:              new("SELECT 1"),
		Logging:            new(fleet.LoggingSnapshot),
		Interval:           new(uint(60)),
		AutomationsEnabled: &autoOn,
		LabelsIncludeAll:   []string{lblA.Name, lblB.Name},
	}, http.StatusOK, &createResp)
	require.NotNil(t, createResp.Query)
	require.Len(t, createResp.Query.LabelsIncludeAll, 2)
	require.Empty(t, createResp.Query.LabelsIncludeAny)

	// Mutex rejection on create. Assert on the specific validation error so a
	// generic 400 (e.g., name required) cannot silently satisfy the test.
	rejCreateResp := s.Do("POST", "/api/latest/fleet/queries", fleet.QueryPayload{
		Name:             new("q-rej-" + t.Name()),
		Query:            new("SELECT 1"),
		Logging:          new(fleet.LoggingSnapshot),
		LabelsIncludeAll: []string{lblA.Name},
		LabelsIncludeAny: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejCreateResp.Body), fleet.ErrQueryConflictingLabels.Error())

	// PATCH: switch from include_all to include_any (clears include_all).
	includeAny := []string{lblA.Name}
	var modifyResp fleet.ModifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", createResp.Query.ID), fleet.QueryPayload{
		LabelsIncludeAny: includeAny,
	}, http.StatusOK, &modifyResp)
	require.Empty(t, modifyResp.Query.LabelsIncludeAll)
	require.Len(t, modifyResp.Query.LabelsIncludeAny, 1)

	// PATCH mutex rejection. Assert on the specific validation error.
	rejPatchResp := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", createResp.Query.ID), fleet.QueryPayload{
		LabelsIncludeAny: []string{lblA.Name},
		LabelsIncludeAll: []string{lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rejPatchResp.Body), fleet.ErrQueryConflictingLabels.Error())

	// Switch back to include_all so the end-to-end test below uses it.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", createResp.Query.ID), fleet.QueryPayload{
		LabelsIncludeAll: []string{lblA.Name, lblB.Name},
	}, http.StatusOK, &modifyResp)
	require.Len(t, modifyResp.Query.LabelsIncludeAll, 2)
	require.Empty(t, modifyResp.Query.LabelsIncludeAny)

	// End-to-end: scheduled-query distribution should only return the query
	// for hosts that are members of BOTH labels.
	mkHost := func(suffix string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(), LabelUpdatedAt: time.Now(), PolicyUpdatedAt: time.Now(), SeenTime: time.Now(),
			OsqueryHostID: new("q" + suffix + t.Name()), NodeKey: new("q" + suffix + t.Name()),
			UUID: uuid.New().String(), Hostname: t.Name() + "-" + suffix + ".local", Platform: "linux",
		})
		require.NoError(t, err)
		return h
	}
	hostA := mkHost("a")
	hostBoth := mkHost("both")
	hostNone := mkHost("none")
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostA.ID, []uint{lblA.ID}))
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostBoth.ID, []uint{lblA.ID, lblB.ID}))

	queryName := createResp.Query.Name
	hasQueryFor := func(hostID uint) bool {
		queries, err := s.ds.ListScheduledQueriesForAgents(ctx, nil, &hostID, false)
		require.NoError(t, err)
		// ListScheduledQueriesForAgents does not project q.id, so match by name.
		for _, q := range queries {
			if q.Name == queryName {
				return true
			}
		}
		return false
	}
	require.False(t, hasQueryFor(hostNone.ID), "host with no labels should not match include_all query")
	require.False(t, hasQueryFor(hostA.ID), "host with one of two required labels should not match include_all query")
	require.True(t, hasQueryFor(hostBoth.ID), "host with both required labels should match include_all query")
}

func (s *integrationEnterpriseTestSuite) TestListHostReportsIncludeAllPremium() {
	t := s.T()
	ctx := t.Context()

	labelA, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "-A", Query: "SELECT 1"})
	require.NoError(t, err)
	labelB, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "-B", Query: "SELECT 1"})
	require.NoError(t, err)

	admin := s.users[TestAdminUserEmail]
	qIncludeAll, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:        t.Name() + "_include_all",
		Query:       "SELECT 1",
		AuthorID:    &admin.ID,
		Saved:       true,
		DiscardData: false,
		Logging:     fleet.LoggingSnapshot,
		LabelsIncludeAll: []fleet.LabelIdent{
			{LabelName: labelA.Name},
			{LabelName: labelB.Name},
		},
	})
	require.NoError(t, err)

	mkHost := func(suffix string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(t.Name() + suffix),
			NodeKey:         new(t.Name() + suffix),
			UUID:            uuid.New().String(),
			Hostname:        t.Name() + "-" + suffix + ".local",
			Platform:        "linux",
		})
		require.NoError(t, err)
		return h
	}
	hostNone := mkHost("none")
	hostOnlyA := mkHost("onlyA")
	hostBoth := mkHost("both")

	require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, hostOnlyA, map[uint]*bool{labelA.ID: new(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, hostBoth, map[uint]*bool{labelA.ID: new(true), labelB.ID: new(true)}, time.Now(), false))

	hasIncludeAllReport := func(hostID uint) bool {
		t.Helper()
		var resp listHostReportsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/reports", hostID), nil, http.StatusOK, &resp, "include_reports_dont_store_results", "true")
		for _, r := range resp.Reports {
			if r.ReportID == qIncludeAll.ID {
				return true
			}
		}
		return false
	}

	require.False(t, hasIncludeAllReport(hostNone.ID), "host with no labels must not see include_all query")
	require.False(t, hasIncludeAllReport(hostOnlyA.ID), "host with subset of labels must not see include_all query")
	require.True(t, hasIncludeAllReport(hostBoth.ID), "host with all labels must see include_all query on premium")
}
