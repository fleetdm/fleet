package service

// Report (query), schedule and pack tests for the core (no-license) suite.
//
// Belongs here: report CRUD and the query spec endpoints, pagination and platform
// filtering, the global schedule, packs and scheduled queries within them, and
// stored report results.
//
// Does not belong here: running a live query against targets
// (integration_core_targets_test.go), and the osquery protocol used to deliver
// results (integration_core_osquery_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestQueryCreationLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(context.Background(), &admin1)
	require.NoError(t, err)

	params := fleet.QueryPayload{
		Name:  new("user1"),
		Query: new("select * from time;"),
	}
	var createQueryResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer s.cleanupQuery(createQueryResp.Query.ID)
	assert.False(t, createQueryResp.Query.CreatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), createQueryResp.Query.CreatedAt, time.Minute)
	assert.False(t, createQueryResp.Query.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), createQueryResp.Query.UpdatedAt, time.Minute)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "created_saved_query" {
			found = true
			assert.Equal(t, "Test Name admin1@example.com", *activity.ActorFullName)
			require.NotNil(t, activity.ActorGravatar)
			assert.Equal(t, "http://iii.com", *activity.ActorGravatar)
		}
	}
	require.True(t, found)
}

func (s *integrationTestSuite) TestQueryLabelsIncludeAnyRequiresPremium() {
	// POST /api/v1/fleet/queries with labels_include_any should fail with 402 on free tier
	var createResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.QueryPayload{
		Name:             new("test-labels-query"),
		Query:            new("SELECT 1"),
		LabelsIncludeAny: []string{"some-label"},
	}, http.StatusPaymentRequired, &createResp)

	// Create a query without labels_include_any to use for the PATCH test
	var createOKResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.QueryPayload{
		Name:  new("test-labels-query-for-patch"),
		Query: new("SELECT 1"),
	}, http.StatusOK, &createOKResp)
	defer s.cleanupQuery(createOKResp.Query.ID)

	// PATCH with labels_include_any should also fail with 402 on free tier
	var modifyResp fleet.ModifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", createOKResp.Query.ID), fleet.QueryPayload{
		LabelsIncludeAny: []string{"some-label"},
	}, http.StatusPaymentRequired, &modifyResp)

	// POST /api/latest/fleet/spec/queries with labels_include_any should fail with 402 on free tier
	var applyResp fleet.ApplyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: "test-labels-spec-query", Query: "SELECT 1", LabelsIncludeAny: []string{"some-label"}},
		},
	}, http.StatusPaymentRequired, &applyResp)
}

func (s *integrationTestSuite) TestGlobalSchedule() {
	t := s.T()

	// list the existing global schedules (none yet)
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Empty(t, gs.GlobalSchedule)

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery1",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Saved:          true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// schedule that query
	gsParams := fleet.ScheduledQueryPayload{QueryID: &qr.ID, Interval: new(uint(42))}
	r := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/schedule", gsParams, http.StatusOK, &r)

	// list the scheduled queries, get the one just created
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(42), gs.GlobalSchedule[0].Interval)
	assert.Contains(t, gs.GlobalSchedule[0].Name, "Copy of TestQuery1 (")
	id := gs.GlobalSchedule[0].ID

	// list page 2, should be empty
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs, "page", "2", "per_page", "4")
	require.Empty(t, gs.GlobalSchedule)

	// update the scheduled query
	gs = fleet.GlobalSchedulePayload{}
	gsParams = fleet.ScheduledQueryPayload{Interval: new(uint(55))}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/schedule/%d", id), gsParams, http.StatusOK, &gs)

	// update a non-existing schedule
	gsParams = fleet.ScheduledQueryPayload{Interval: new(uint(66))}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/schedule/%d", id+1), gsParams, http.StatusNotFound, &gs)

	// read back that updated scheduled query
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, id, gs.GlobalSchedule[0].ID)
	assert.Equal(t, uint(55), gs.GlobalSchedule[0].Interval)

	// delete the scheduled query
	r = globalScheduleQueryResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/schedule/%d", id), nil, http.StatusOK, &r)

	// delete a non-existing schedule
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/schedule/%d", id+1), nil, http.StatusNotFound, &r)

	// list the scheduled queries, back to none
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Empty(t, gs.GlobalSchedule)
}

func (s *integrationTestSuite) TestPacks() {
	t := s.T()

	var packResp getPackResponse
	// get non-existing pack
	s.Do("GET", "/api/latest/fleet/packs/999", nil, http.StatusNotFound)

	// create some packs
	packs := make([]fleet.Pack, 3)
	for i := range packs {
		req := &createPackRequest{
			PackPayload: fleet.PackPayload{
				Name: new(fmt.Sprintf("%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), i)),
			},
		}

		var createResp createPackResponse
		s.DoJSON("POST", "/api/latest/fleet/packs", req, http.StatusOK, &createResp)
		packs[i] = createResp.Pack.Pack
	}

	// get existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[0].ID), nil, http.StatusOK, &packResp)
	require.Equal(t, packs[0].ID, packResp.Pack.ID)

	// list packs
	var listResp listPacksResponse
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 2)
	assert.Equal(t, packs[0].ID, listResp.Packs[0].ID)
	assert.Equal(t, packs[1].ID, listResp.Packs[1].ID)

	// get page 1
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)

	// get page 2, empty
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "page", "2", "per_page", "2", "order_key", "name")
	require.Empty(t, listResp.Packs)

	var delResp deletePackResponse
	// delete non-existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/%s", "zzz"), nil, http.StatusNotFound, &delResp)

	// delete existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/%s", url.PathEscape(packs[0].Name)), nil, http.StatusOK, &delResp)

	// delete non-existing pack by id
	var delIDResp deletePackByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", packs[2].ID+1), nil, http.StatusNotFound, &delIDResp)

	// delete existing pack by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", packs[1].ID), nil, http.StatusOK, &delIDResp)

	var modResp modifyPackResponse
	// modify non-existing pack
	req := &fleet.PackPayload{Name: new("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[2].ID+1), req, http.StatusNotFound, &modResp)

	// modify existing pack
	req = &fleet.PackPayload{Name: new("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[2].ID), req, http.StatusOK, &modResp)
	require.Equal(t, packs[2].ID, modResp.Pack.ID)
	require.Contains(t, modResp.Pack.Name, "updated_")

	// list packs, only packs[2] remains
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)
}

func (s *integrationTestSuite) TestScheduledQueries() {
	t := s.T()

	// create a pack
	var createPackResp createPackResponse
	reqPack := &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new(strings.ReplaceAll(t.Name(), "/", "_")),
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/packs", reqPack, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	// try a non existent query
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", 9999), nil, http.StatusNotFound)

	// list queries
	var listQryResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp)
	assert.Empty(t, listQryResp.Queries)
	assert.Equal(t, 0, listQryResp.Count)
	assert.Equal(t, 0, listQryResp.InheritedQueryCount)

	// create a query
	sql := "select * from time;"
	var createQueryResp fleet.CreateQueryResponse
	reqQuery := &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_")),
		Query: new(sql),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query := createQueryResp.Query
	assert.Equal(t, query.Query, sql)
	assert.Equal(t, createQueryResp.Report.Query, sql)

	// listing returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)
	assert.Equal(t, 1, listQryResp.Count)
	assert.Equal(t, 0, listQryResp.InheritedQueryCount)

	// listing with matching name returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", query.Name)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// listing with matching name plus whitespace returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", "  "+query.Name+" ")
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// listing with non-matching name returns nothing
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", "  nomatch")
	require.Empty(t, listQryResp.Queries)

	// Return that query by name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries?query=%s", query.Name), nil, http.StatusOK, &listQryResp)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// next page returns nothing, count and meta are correct
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "per_page", "2", "page", "1")
	require.Empty(t, listQryResp.Queries)
	require.Equal(t, 1, listQryResp.Count)
	require.Equal(t, 0, listQryResp.InheritedQueryCount)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// getting that query works
	var getQryResp fleet.GetQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID), nil, http.StatusOK, &getQryResp)
	assert.Equal(t, query.ID, getQryResp.Query.ID)
	assert.Equal(t, query.ID, getQryResp.Report.ID)
	assert.Equal(t, sql, getQryResp.Query.Query)
	assert.Equal(t, sql, getQryResp.Report.Query)

	// list scheduled queries in pack, none yet
	var getInPackResp fleet.GetScheduledQueriesInPackResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	assert.Empty(t, getInPackResp.Scheduled)

	// list scheduled queries in non-existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID+1), nil, http.StatusOK, &getInPackResp)
	assert.Empty(t, getInPackResp.Scheduled)

	// create scheduled query
	var createResp fleet.ScheduleQueryResponse
	reqSQ := &fleet.ScheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID,
		Interval: 1,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusOK, &createResp)
	sq1 := createResp.Scheduled.ScheduledQuery
	assert.NotZero(t, sq1.ID)
	assert.Equal(t, uint(1), sq1.Interval)

	// create scheduled query with invalid pack
	reqSQ = &fleet.ScheduleQueryRequest{
		PackID:   pack.ID + 1,
		QueryID:  query.ID,
		Interval: 2,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusUnprocessableEntity, &createResp)

	// create scheduled query with invalid query
	reqSQ = &fleet.ScheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID + 1,
		Interval: 3,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusNotFound, &createResp)

	// list scheduled queries in pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	require.Len(t, getInPackResp.Scheduled, 1)
	assert.Equal(t, sq1.ID, getInPackResp.Scheduled[0].ID)

	// list scheduled queries in pack, next page
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp, "page", "1", "per_page", "2")
	require.Empty(t, getInPackResp.Scheduled)

	// get non-existing scheduled query
	var getResp fleet.GetScheduledQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &getResp)

	// get existing scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, sq1.ID, getResp.Scheduled.ID)
	assert.Equal(t, sq1.Interval, getResp.Scheduled.Interval)

	// modify scheduled query
	var modResp fleet.ModifyScheduledQueryResponse
	reqMod := fleet.ScheduledQueryPayload{
		Interval: new(uint(4)),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID), reqMod, http.StatusOK, &modResp)
	assert.Equal(t, sq1.ID, modResp.Scheduled.ID)
	assert.Equal(t, uint(4), modResp.Scheduled.Interval)

	// modify non-existing scheduled query
	reqMod = fleet.ScheduledQueryPayload{
		Interval: new(uint(5)),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID+1), reqMod, http.StatusNotFound, &modResp)

	// delete non-existing scheduled query
	var delResp fleet.DeleteScheduledQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &delResp)

	// delete existing scheduled query
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID), nil, http.StatusOK, &delResp)

	// get the now-deleted scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID), nil, http.StatusNotFound, &getResp)

	// modify the query
	var modQryResp fleet.ModifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID), fleet.QueryPayload{Description: new("updated")}, http.StatusOK, &modQryResp)
	assert.Equal(t, "updated", modQryResp.Query.Description)
	assert.Equal(t, sql, modQryResp.Query.Query)
	assert.Equal(t, sql, modQryResp.Report.Query)

	// TODO(jahziel): check that the query results were deleted

	// TODO(jahziel): check that the query results were deleted after setting `discard_data`

	// modify a non-existing query
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID+1), fleet.QueryPayload{Description: new("updated")}, http.StatusNotFound, &modQryResp)

	// delete the query by name
	var delByNameResp fleet.DeleteQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", query.Name), nil, http.StatusOK, &delByNameResp)

	// delete unknown query by name (i.e. the same, now deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", query.Name), nil, http.StatusNotFound, &delByNameResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_") + "_2"),
		Query: new("select 2"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query2 := createQueryResp.Query

	// delete it by id
	var delByIDResp fleet.DeleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", query2.ID), nil, http.StatusOK, &delByIDResp)

	// delete unknown query by id (same id just deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", query2.ID), nil, http.StatusNotFound, &delByIDResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_") + "_3"),
		Query: new("select 3"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query3 := createQueryResp.Query

	// batch-delete by id, 3 ids, only one exists
	var delBatchResp fleet.DeleteQueriesResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]any{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(1), delBatchResp.Deleted)

	// batch-delete by id, none exist
	delBatchResp.Deleted = 0
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]any{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusNotFound, &delBatchResp)
	assert.Equal(t, uint(0), delBatchResp.Deleted)
}

func (s *integrationTestSuite) TestScheduledQueriesInPackOrderKey() {
	t := s.T()

	// create a pack
	var createPackResp createPackResponse
	s.DoJSON("POST", "/api/latest/fleet/packs", &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new(strings.ReplaceAll(t.Name(), "/", "_")),
		},
	}, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	// create a query
	var createQueryResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_")),
		Query: new("select 1"),
	}, http.StatusOK, &createQueryResp)
	query := createQueryResp.Query

	// schedule the query in the pack so the listing has at least one row
	var createSchedResp fleet.ScheduleQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", &fleet.ScheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID,
		Interval: 60,
	}, http.StatusOK, &createSchedResp)

	// every key in scheduledQueriesAllowedOrderKeys must work end-to-end with cursor pagination.
	allowedOrderKeys := []string{
		"id",
		"pack_id",
		"name",
		"query_name",
		"description",
		"interval",
		"snapshot",
		"removed",
		"platform",
		"version",
		"shard",
		"denylist",
		"query",
		"query_id",
		"user_time_p50",
		"user_time_p95",
		"system_time_p50",
		"system_time_p95",
		"total_executions",
	}
	for _, orderKey := range allowedOrderKeys {
		t.Run(orderKey, func(t *testing.T) {
			var getInPackResp fleet.GetScheduledQueriesInPackResponse
			s.DoJSON(
				"GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID),
				nil, http.StatusOK, &getInPackResp,
				"order_key", orderKey,
				"after", "0",
			)
		})
	}
}

func (s *integrationTestSuite) TestScheduledQueriesInPackInvalidOrderKey() {
	t := s.T()

	// create a pack so the endpoint has a real id to operate on
	var createPackResp createPackResponse
	name := strings.ReplaceAll(t.Name(), "/", "_")
	s.DoJSON("POST", "/api/latest/fleet/packs", &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: &name,
		},
	}, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	var getInPackResp fleet.GetScheduledQueriesInPackResponse
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID),
		nil, http.StatusUnprocessableEntity, &getInPackResp,
		"order_key", "not_a_real_column",
		"after", "0",
	)
}

func (s *integrationTestSuite) TestQueriesPaginationAndPlatformFilter() {
	t := s.T()

	// make a few queries
	testQueries := []*fleet.Query{
		{Name: "PPTestQuery1", Query: "select 1", Platform: "darwin"},
		{Name: "PPTestQuery2", Query: "select 2", Platform: "linux"},
		{Name: "PPTestQuery3", Query: "select 3", Platform: "windows"},
		{Name: "PPTestQuery4", Query: "select 4", Platform: "darwin,windows,linux"},
		{Name: "PPTestQuery5", Query: "select 5"},
		{Name: "PPTestQuery6", Query: "select 6"},
		{Name: "PPTestQuery7", Query: "select 7"},
		{Name: "PPTestQuery8", Query: "select 8"},
		{Name: "PPTestQuery9", Query: "select 9"},
		{Name: "PPTestQuery10", Query: "select 10"},
	}
	var createQueryResp fleet.CreateQueryResponse
	for _, q := range testQueries {
		reqQuery := &fleet.QueryPayload{
			Name:     &q.Name,
			Query:    &q.Query,
			Platform: &q.Platform,
		}
		s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
		require.Equal(t, createQueryResp.Query.Name, q.Name)
		require.Equal(t, createQueryResp.Query.Platform, q.Platform)
	}

	var listQryResp fleet.ListQueriesResponse
	queryNameToMatch := "TestQuery"

	// Test pagination, no filter

	// middle of the pages
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "2", "page", "1")
	require.Len(t, listQryResp.Queries, 2)
	require.Equal(t, 10, listQryResp.Count)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)

	// first and only page
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "10", "page", "0")
	require.Len(t, listQryResp.Queries, 10)
	require.Equal(t, 10, listQryResp.Count)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// first of a few pages
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "2", "page", "0")
	require.Len(t, listQryResp.Queries, 2)
	require.Equal(t, 10, listQryResp.Count)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)

	// last page
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "5", "page", "1")
	require.Len(t, listQryResp.Queries, 5)
	require.Equal(t, 10, listQryResp.Count)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// after last page
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "2", "page", "5")
	require.Empty(t, listQryResp.Queries)
	require.Equal(t, 10, listQryResp.Count)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// invalid order_key returns 422
	listQryResp = fleet.ListQueriesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusUnprocessableEntity, &listQryResp, "order_key", "invalid")

	// test platform filtering

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "macos")
	require.Len(t, listQryResp.Queries, 8)
	require.Equal(t, 8, listQryResp.Count)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "darwin", listQryResp.Queries[0].Platform)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[1].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "windows")
	require.Len(t, listQryResp.Queries, 8)
	require.Equal(t, 8, listQryResp.Count)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "windows", listQryResp.Queries[0].Platform)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[1].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "linux")
	require.Len(t, listQryResp.Queries, 8)
	require.Equal(t, 8, listQryResp.Count)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "linux", listQryResp.Queries[0].Platform)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[1].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "linux", "per_page", "1", "page", "0")
	require.Len(t, listQryResp.Queries, 1)
	require.Equal(t, 8, listQryResp.Count)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "linux", listQryResp.Queries[0].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "linux", "per_page", "1", "page", "1")
	require.Len(t, listQryResp.Queries, 1)
	require.Equal(t, 8, listQryResp.Count)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[0].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusBadRequest, &listQryResp, "platform", "lucas", "per_page", "1", "page", "1")

	// delete them by name
	var delByNameResp fleet.DeleteQueryResponse
	// for _, qId := range testQueryIds {
	for _, q := range testQueries {
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", q.Name), nil, http.StatusOK, &delByNameResp)
	}
}

func (s *integrationTestSuite) TestQueriesBadRequests() {
	t := s.T()

	reqQuery := &fleet.QueryPayload{
		Name:  new("existing query"),
		Query: new("select 42;"),
	}
	createQueryResp := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	require.NotNil(t, createQueryResp.Query)
	existingQueryID := createQueryResp.Query.ID
	defer s.cleanupQuery(existingQueryID)

	for _, tc := range []struct {
		tname    string
		name     string
		query    string
		platform string
		logging  string
	}{
		{
			tname: "empty name",
			name:  " ", // #3704
			query: "select 42;",
		},
		{
			tname: "empty query",
			name:  "Some name",
			query: "",
		},
		{
			tname: "Invalid query",
			name:  "Invalid query",
			query: "",
		},
		{
			tname:    "unsupported platform",
			name:     "bad query",
			query:    "select 1",
			platform: "oops",
		},
		{
			tname:    "unsupported platform",
			name:     "bad query",
			query:    "select 1",
			platform: "charles,darwin",
		},
		{
			tname:    "missing platform comma delimeter",
			name:     "bad query",
			query:    "select 1",
			platform: "linuxdarwin",
		},
		{
			tname:    "missing platform comma delimeter",
			name:     "bad query",
			query:    "select 1",
			platform: "windows darwin",
		},
		{
			tname:   "invalid logging value",
			name:    "bad query",
			query:   "select 1",
			logging: "foobar",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.QueryPayload{
				Name:     new(tc.name),
				Query:    new(tc.query),
				Platform: new(tc.platform),
				Logging:  new(tc.logging),
			}
			createQueryResp := fleet.CreateQueryResponse{}
			s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)
			require.Nil(t, createQueryResp.Query)

			payload := fleet.QueryPayload{
				Name:     new(tc.name),
				Query:    new(tc.query),
				Platform: new(tc.platform),
				Logging:  new(tc.logging),
			}
			mResp := fleet.ModifyQueryResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", existingQueryID), &payload, http.StatusBadRequest, &mResp)
			require.Nil(t, mResp.Query)
			// TODO – add checks for specific errors
		})
	}

	for _, tc := range []struct {
		tname string
		body  string
	}{
		{
			tname: "null name",
			body:  `{"name": null, "query": "SELECT 1;"}`,
		},
		{
			tname: "null query",
			body:  `{"name": "Test", "query": null}`,
		},
		{
			tname: "null name and query",
			body:  `{"name": null, "query": null}`,
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			resp := s.DoRaw("POST", "/api/latest/fleet/queries", []byte(tc.body), http.StatusBadRequest)
			resp.Body.Close()
		})
	}

	for _, tc := range []struct {
		tname string
		body  string
	}{
		{
			tname: "spec null name",
			body:  `{"specs": [{"name": null, "query": "SELECT 1;"}]}`,
		},
		{
			tname: "spec null query",
			body:  `{"specs": [{"name": "Test", "query": null}]}`,
		},
		{
			tname: "spec null name and query",
			body:  `{"specs": [{"name": null, "query": null}]}`,
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			resp := s.DoRaw("POST", "/api/latest/fleet/spec/queries", []byte(tc.body), http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

func (s *integrationTestSuite) TestPacksBadRequests() {
	t := s.T()

	reqPacks := &fleet.PackPayload{
		Name: new("existing pack"),
	}
	createPackResp := createPackResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs", reqPacks, http.StatusOK, &createPackResp)
	existingPackID := createPackResp.Pack.ID

	for _, tc := range []struct {
		tname string
		name  string
	}{
		{
			tname: "empty name",
			name:  " ", // #3704
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.PackPayload{
				Name: new(tc.name),
			}
			createPackResp := createPackResponse{}
			s.DoJSON("POST", "/api/latest/fleet/packs", reqQuery, http.StatusBadRequest, &createPackResp)

			payload := fleet.PackPayload{
				Name: new(tc.name),
			}
			mResp := modifyPackResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", existingPackID), &payload, http.StatusBadRequest, &mResp)
		})
	}

	t.Run("null name on create", func(t *testing.T) {
		res := s.Do("POST", "/api/latest/fleet/packs", json.RawMessage(`{"name": null}`), http.StatusBadRequest)
		assertBodyContains(t, res, "pack name cannot be empty")
	})
}

func (s *integrationTestSuite) TestQuerySpecs() {
	t := s.T()

	s.lq.On("SetQueryResultsCount", mock.Anything, mock.Anything).Return(nil)

	// list specs, none yet
	var getSpecsResp fleet.GetQuerySpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	assert.Empty(t, getSpecsResp.Specs)

	// get unknown one
	var getSpecResp fleet.GetQuerySpecResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/queries/nonesuch", nil, http.StatusNotFound, &getSpecResp)

	// create some queries via apply specs
	q1 := strings.ReplaceAll(t.Name(), "/", "_")
	q2 := q1 + "_2"
	var applyResp fleet.ApplyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q1, Query: "SELECT 1"},
			{Name: q2, Query: "SELECT 2"},
		},
	}, http.StatusOK, &applyResp)

	// get the queries back
	var listQryResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 2)
	assert.Equal(t, q1, listQryResp.Queries[0].Name)
	assert.Equal(t, q2, listQryResp.Queries[1].Name)
	q1ID, q2ID := listQryResp.Queries[0].ID, listQryResp.Queries[1].ID

	// list specs
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 2)
	names := []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name}
	assert.ElementsMatch(t, []string{q1, q2}, names)

	// get specific spec
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/queries/%s", q1), nil, http.StatusOK, &getSpecResp)
	assert.Equal(t, "SELECT 1", getSpecResp.Spec.Query)

	// apply specs again - create q3 and update q2
	q3 := q1 + "_3"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q2, Query: "SELECT -2"},
			{Name: q3, Query: "SELECT 3"},
		},
	}, http.StatusOK, &applyResp)

	// try to create a query with invalid platform, fail
	q4 := q1 + "_4"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q4, Query: "SELECT 4", Platform: "not valid"},
		},
	}, http.StatusBadRequest, &applyResp)

	// try to edit a query with invalid platform, fail
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q3, Query: "SELECT 3", Platform: "charles darwin"},
		},
	}, http.StatusBadRequest, &applyResp)

	// list specs - has 3, not 4 (one was an update)
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 3)
	names = []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name, getSpecsResp.Specs[2].Name}
	assert.ElementsMatch(t, []string{q1, q2, q3}, names)

	// get the queries back again
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 3)
	assert.Equal(t, q1ID, listQryResp.Queries[0].ID)
	assert.Equal(t, q2ID, listQryResp.Queries[1].ID)
	assert.Equal(t, "SELECT -2", listQryResp.Queries[1].Query)
	q3ID := listQryResp.Queries[2].ID

	// delete all queries created
	var delBatchResp fleet.DeleteQueriesResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]any{
		"ids": []uint{q1ID, q2ID, q3ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(3), delBatchResp.Deleted)
}

func (s *integrationTestSuite) TestQueryReports() {
	t := s.T()
	ctx := context.Background()

	// Add mock expectations to the existing mock (the server already has a reference to it)
	counts := make(map[uint]int)
	s.lq.GetQueryResultsCountsOverride = func(queryIDs []uint) (map[uint]int, error) {
		return counts, nil
	}
	s.lq.SetQueryResultsCountOverride = func(queryID uint, count int) error {
		counts[queryID] = count
		return nil
	}
	s.lq.IncrQueryResultsCountsOverride = func(queryIDsToAmounts map[uint]int) error {
		for queryID, amount := range queryIDsToAmounts {
			counts[queryID] += amount
		}
		return nil
	}
	s.lq.DeleteQueryResultsCountOverride = func(queryID uint) error {
		delete(counts, queryID)
		return nil
	}
	defer func() {
		s.lq.GetQueryResultsCountsOverride = nil
		s.lq.SetQueryResultsCountOverride = nil
		s.lq.IncrQueryResultsCountsOverride = nil
		s.lq.DeleteQueryResultsCountOverride = nil
	}()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)

	host1Global, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("1"),
		UUID:            "1",
		Hostname:        "foo.local1",
		OsqueryHostID:   new("1"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	host2Global, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("2"),
		UUID:            "2",
		Hostname:        "foo.local2",
		OsqueryHostID:   new("2"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	host2Team1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("3"),
		UUID:            "3",
		ComputerName:    "Foo Local3",
		Hostname:        "foo.local3",
		OsqueryHostID:   new("3"),
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host2Team1.ID}))
	require.NoError(t, err)

	host1Team2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("4"),
		UUID:            "4",
		ComputerName:    "Foo Local4",
		Hostname:        "foo.local4",
		OsqueryHostID:   new("4"),
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "30-65-EC-6F-C4-61",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team2.ID, []uint{host1Team2.ID}))
	require.NoError(t, err)

	osqueryInfoQuery, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:               "Osquery info",
		Description:        "osquery_info table",
		Query:              "select * from osquery_info;",
		Saved:              true,
		Interval:           30,
		AutomationsEnabled: true,
		DiscardData:        false,
		TeamID:             nil,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	usbDevicesQuery, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:               "USB devices",
		Description:        "usb_devices table",
		Query:              "select * from usb_devices;",
		Saved:              true,
		Interval:           60,
		AutomationsEnabled: true,
		DiscardData:        false,
		TeamID:             new(team1.ID),
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// Should return no results.
	var gqrr fleet.GetQueryReportResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, gqrr.QueryID)
	require.NotNil(t, gqrr.Results)
	require.Empty(t, gqrr.Results)

	var ghqrr getHostQueryReportResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, usbDevicesQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, ghqrr.QueryID)
	require.Equal(t, host1Global.ID, ghqrr.HostID)
	require.Nil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.NotNil(t, ghqrr.Results)
	require.Empty(t, ghqrr.Results)

	slreq := submitLogsRequest{
		NodeKey: *host2Team1.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [
    {
      "class": "239",
      "model": "HD Pro Webcam C920",
      "model_id": "0892",
      "protocol": "",
      "removable": "1",
      "serial": "zoobar",
      "subclass": "2",
      "usb_address": "3",
      "usb_port": "1",
      "vendor": "",
      "vendor_id": "046d",
      "version": "0.19"
    },
    {
      "class": "0",
      "model": "Apple Internal Keyboard / Trackpad",
      "model_id": "027e",
      "protocol": "",
      "removable": "0",
      "serial": "foobar",
      "subclass": "0",
      "usb_address": "8",
      "usb_port": "5",
      "vendor": "Apple Inc.",
      "vendor_id": "05ac",
      "version": "9.33"
    }
  ],
  "action": "snapshot",
  "name": "pack/team-` + usbDevicesQuery.TeamIDStr() + `/` + usbDevicesQuery.Name + `",
  "hostIdentifier": "` + *host2Team1.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 17:32:08 2023 UTC",
  "unixTime": 1696613528,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host2Team1.UUID + `",
    "hostname": "` + host2Team1.Hostname + `"
  }
}`), json.RawMessage(`{
  "snapshot": [
    {
      "build_distro": "10.14",
      "build_platform": "darwin",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
      "pid": "95637",
      "platform_mask": "21",
      "start_time": "1696611201",
      "uuid": "` + host2Team1.UUID + `",
      "version": "5.9.1",
      "watcher": "95636"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host2Team1.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:08:18 2023 UTC",
  "unixTime": 1696615698,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host2Team1.UUID + `",
    "hostname": "` + host2Team1.Hostname + `"
  }
}`),
		},
	}
	slres := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	slreq = submitLogsRequest{
		NodeKey: *host1Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [
    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "` + host1Global.UUID + `",
      "version": "5.9.2",
      "watcher": "3570"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 2, counts[osqueryInfoQuery.ID])

	slreq = submitLogsRequest{
		NodeKey: *host1Team2.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [
    {
      "build_distro": "10.14",
      "build_platform": "darwin",
      "config_hash": "ca2bc81cd5e79132cb0f842c433ad7f84c056c12",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "975f2ce1-8672-4932-85f8-340272820e79",
      "pid": "1039",
      "platform_mask": "21",
      "start_time": "1733334052",
      "uuid": "` + host1Team2.UUID + `",
      "version": "5.14.1",
      "watcher": "1037"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Team2.OsqueryHostID + `",
  "calendarTime": "Mon Dec  16 13:28:00 2024 UTC",
  "unixTime": 1734377280,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host1Team2.UUID + `",
    "hostname": "` + host1Team2.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 3, counts[osqueryInfoQuery.ID])

	emptyslreq := submitLogsRequest{
		NodeKey: *host2Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
			  "snapshot": [],
			  "action": "snapshot",
			  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
			  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
			  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
			  "unixTime": 1696615984,
			  "epoch": 0,
			  "counter": 0,
			  "numerics": false,
			  "decorations": {
				"host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
				"hostname": "` + host1Global.Hostname + `"
			  }
			}`),
		},
	}
	emptyslres := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", emptyslreq, http.StatusOK, &emptyslres)
	require.NoError(t, emptyslres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	// Count stays the same because error rows don't count against the limit.
	require.Equal(t, 3, counts[osqueryInfoQuery.ID])

	gqrr = fleet.GetQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["usb_port"] < gqrr.Results[j].Columns["usb_port"]
	})
	require.Equal(t, host2Team1.ID, gqrr.Results[0].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[0].Hostname)
	require.NotZero(t, gqrr.Results[0].LastFetched)
	require.Equal(t, map[string]string{
		"class":       "239",
		"model":       "HD Pro Webcam C920",
		"model_id":    "0892",
		"protocol":    "",
		"removable":   "1",
		"serial":      "zoobar",
		"subclass":    "2",
		"usb_address": "3",
		"usb_port":    "1",
		"vendor":      "",
		"vendor_id":   "046d",
		"version":     "0.19",
	}, gqrr.Results[0].Columns)
	require.Equal(t, host2Team1.ID, gqrr.Results[1].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[1].Hostname)
	require.NotZero(t, gqrr.Results[1].LastFetched)
	require.Equal(t, map[string]string{
		"class":       "0",
		"model":       "Apple Internal Keyboard / Trackpad",
		"model_id":    "027e",
		"protocol":    "",
		"removable":   "0",
		"serial":      "foobar",
		"subclass":    "0",
		"usb_address": "8",
		"usb_port":    "5",
		"vendor":      "Apple Inc.",
		"vendor_id":   "05ac",
		"version":     "9.33",
	}, gqrr.Results[1].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host2Team1.ID, usbDevicesQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, ghqrr.QueryID)
	require.Equal(t, host2Team1.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["usb_port"] < gqrr.Results[j].Columns["usb_port"]
	})
	require.Equal(t, map[string]string{
		"class":       "239",
		"model":       "HD Pro Webcam C920",
		"model_id":    "0892",
		"protocol":    "",
		"removable":   "1",
		"serial":      "zoobar",
		"subclass":    "2",
		"usb_address": "3",
		"usb_port":    "1",
		"vendor":      "",
		"vendor_id":   "046d",
		"version":     "0.19",
	}, ghqrr.Results[0].Columns)
	require.Equal(t, map[string]string{
		"class":       "0",
		"model":       "Apple Internal Keyboard / Trackpad",
		"model_id":    "027e",
		"protocol":    "",
		"removable":   "0",
		"serial":      "foobar",
		"subclass":    "0",
		"usb_address": "8",
		"usb_port":    "5",
		"vendor":      "Apple Inc.",
		"vendor_id":   "05ac",
		"version":     "9.33",
	}, ghqrr.Results[1].Columns)

	gqrr = fleet.GetQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 3)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["version"] > gqrr.Results[j].Columns["version"]
	})
	require.Equal(t, host1Global.ID, gqrr.Results[0].HostID)
	require.Equal(t, host1Global.DisplayName(), gqrr.Results[0].Hostname)
	require.NotZero(t, gqrr.Results[0].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "centos7",
		"build_platform": "linux",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
		"pid":            "3574",
		"platform_mask":  "9",
		"start_time":     "1696502961",
		"uuid":           host1Global.UUID,
		"version":        "5.9.2",
		"watcher":        "3570",
	}, gqrr.Results[0].Columns)
	require.Equal(t, host2Team1.ID, gqrr.Results[1].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[1].Hostname)
	require.NotZero(t, gqrr.Results[1].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "10.14",
		"build_platform": "darwin",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
		"pid":            "95637",
		"platform_mask":  "21",
		"start_time":     "1696611201",
		"uuid":           host2Team1.UUID,
		"version":        "5.9.1",
		"watcher":        "95636",
	}, gqrr.Results[1].Columns)
	require.Equal(t, host1Team2.ID, gqrr.Results[2].HostID)
	require.Equal(t, host1Team2.DisplayName(), gqrr.Results[2].Hostname)
	require.NotZero(t, gqrr.Results[2].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "10.14",
		"build_platform": "darwin",
		"config_hash":    "ca2bc81cd5e79132cb0f842c433ad7f84c056c12",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "975f2ce1-8672-4932-85f8-340272820e79",
		"pid":            "1039",
		"platform_mask":  "21",
		"start_time":     "1733334052",
		"uuid":           host1Team2.UUID,
		"version":        "5.14.1",
		"watcher":        "1037",
	}, gqrr.Results[2].Columns)

	gqrr = fleet.GetQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report?team_id=%d", osqueryInfoQuery.ID, team2.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 1)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, ghqrr.QueryID)
	require.Equal(t, host1Global.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 1)
	require.Equal(t, map[string]string{
		"build_distro":   "centos7",
		"build_platform": "linux",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
		"pid":            "3574",
		"platform_mask":  "9",
		"start_time":     "1696502961",
		"uuid":           host1Global.UUID,
		"version":        "5.9.2",
		"watcher":        "3570",
	}, ghqrr.Results[0].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host2Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, ghqrr.QueryID)
	require.Equal(t, host2Global.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Empty(t, ghqrr.Results)

	// verify that certain modifications to queries don't cause result deletion
	modifyQueryResp := fleet.ModifyQueryResponse{}
	updatedDesc := "Updated description"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{Description: &updatedDesc}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, updatedDesc, modifyQueryResp.Query.Description)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 3)

	// now update the query and verify that results are deleted
	updatedQuery := "SELECT * FROM some_new_table;"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{Query: &updatedQuery}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, updatedQuery, modifyQueryResp.Query.Query)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the platform and verify that results are deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				Platform: new("linux"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "linux", modifyQueryResp.Query.Platform)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// now update the platform to the same value and verify that results are not deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				Platform: new("linux"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "linux", modifyQueryResp.Query.Platform)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the min_osquery_version and verify that results are deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				MinOsqueryVersion: new("5.9.1"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.9.1", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// now update the min_osquery_version to another value and verify that results are deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				MinOsqueryVersion: new("5.11.0"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.11.0", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// now update the min_osquery_version to the same value and verify that results are not deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				MinOsqueryVersion: new("5.11.0"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.11.0", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the query via specs and change the min_osquery_version, results should be deleted.
	osqueryInfoQuerySpec := &fleet.QuerySpec{
		Name:               osqueryInfoQuery.Name,
		Description:        osqueryInfoQuery.Description,
		Query:              osqueryInfoQuery.Query,
		Interval:           osqueryInfoQuery.Interval,
		ObserverCanRun:     osqueryInfoQuery.ObserverCanRun,
		Platform:           osqueryInfoQuery.Platform,
		MinOsqueryVersion:  osqueryInfoQuery.MinOsqueryVersion,
		AutomationsEnabled: osqueryInfoQuery.AutomationsEnabled,
		Logging:            osqueryInfoQuery.Logging,
		DiscardData:        osqueryInfoQuery.DiscardData,
	}
	osqueryInfoQuerySpec.MinOsqueryVersion = "5.12.0"
	var applyResp fleet.ApplyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[osqueryInfoQuery.ID]) // counter reset after min_osquery_version change

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// don't change platform or min_osquery_version and results should not be deleted
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)

	// now update the platform and results should be deleted.
	osqueryInfoQuerySpec.Platform = "darwin"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[osqueryInfoQuery.ID]) // counter reset after platform change

	// Update logging type, which should cause results deletion
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", usbDevicesQuery.ID), fleet.ModifyQueryRequest{ID: usbDevicesQuery.ID, QueryPayload: fleet.QueryPayload{Logging: &fleet.LoggingDifferential}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, fleet.LoggingDifferential, modifyQueryResp.Query.Logging)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[usbDevicesQuery.ID]) // counter reset after logging type change

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	discardData := true
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{DiscardData: &discardData}}, http.StatusOK, &modifyQueryResp)
	require.True(t, modifyQueryResp.Query.DiscardData)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[osqueryInfoQuery.ID]) // counter reset after discardData=true

	// check that now that discardData is set, we don't add new results
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Empty(t, gqrr.Results)
	require.False(t, gqrr.ReportClipped)

	// Verify row limit behavior with 10% buffer.
	// The system allows up to limit+10% rows before blocking new inserts.
	// This ensures the cleanup cron always has rows to delete, enabling rotation.
	discardData = false
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{DiscardData: &discardData}}, http.StatusOK, &modifyQueryResp)
	require.False(t, modifyQueryResp.Query.DiscardData)

	// Host1 submits exactly the max rows
	slreq = submitLogsRequest{
		NodeKey: *host1Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [` + results(fleet.DefaultMaxQueryReportRows, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, fleet.DefaultMaxQueryReportRows, counts[osqueryInfoQuery.ID])

	// Host1 submits same rows again (overwrite), should still have 1000 rows
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, gqrr.ReportClipped)
	require.Equal(t, fleet.DefaultMaxQueryReportRows, counts[osqueryInfoQuery.ID]) // counter unchanged after overwrite

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Len(t, ghqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, ghqrr.ReportClipped)

	// Host2 submits 1000 rows. Since we're at the limit but within the 10% buffer,
	// these rows are allowed. Total becomes 2000 rows (1000 from each host).
	slreq = submitLogsRequest{
		NodeKey: *host2Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [` + results(fleet.DefaultMaxQueryReportRows, host2Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host2Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host2Global.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	// Now we have 2000 rows (1000 from host1, 1000 from host2)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows*2)
	require.True(t, gqrr.ReportClipped)
	require.Equal(t, fleet.DefaultMaxQueryReportRows*2, counts[osqueryInfoQuery.ID]) // counter is now 2000

	// Host1 tries to submit 1 row, but counter is now > limit+10%, so it's blocked
	slreq.NodeKey = *host1Global.NodeKey
	slreq.Data = []json.RawMessage{json.RawMessage(`{
  "snapshot": [` + results(1, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`)}

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	// Still 2000 rows since the submission was blocked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows*2)
	require.True(t, gqrr.ReportClipped)
	require.Equal(t, fleet.DefaultMaxQueryReportRows*2, counts[osqueryInfoQuery.ID]) // counter unchanged (blocked)

	// Increase the limit so we can test further submissions
	appConfigSpec := map[string]map[string]int{
		"server_settings": {"query_report_cap": fleet.DefaultMaxQueryReportRows * 3},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)

	// With limit 3000, we have 2000 rows, which is below the limit, so not clipped
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows*2)
	require.False(t, gqrr.ReportClipped)

	// Now host1 can submit again since we're under the new limit+10%
	slreq.Data = []json.RawMessage{
		json.RawMessage(`{
  "snapshot": [` + results(500, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`),
	}

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	// Host1's 1000 rows were replaced with 500 rows, so total is now 1500 (500 from host1, 1000 from host2)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1500)
	require.Equal(t, 1500, counts[osqueryInfoQuery.ID]) // counter unchanged (blocked)
	require.False(t, gqrr.ReportClipped)

	// TODO: Set global discard flag and verify that all data is gone.
}

// TestOsqueryConfigPackCacheLabelScopedQueries verifies that the per-team pack
// config cache does not serve one host's label-scoped query set to other hosts
// of the same team. Reproduces the bug described in #51033.
func (s *integrationTestSuite) TestOsqueryConfigPackCacheLabelScopedQueries() {
	t := s.T()
	ctx := t.Context()

	// Two teams so each ordering gets its own (cold) cache key.
	teamA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-A"})
	require.NoError(t, err)
	teamB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-B"})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(uuid.New().String()),
			NodeKey:         new(uuid.New().String()),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	hostA1 := newHost("a1", &teamA.ID) // label member
	hostA2 := newHost("a2", &teamA.ID) // not a member
	hostB1 := newHost("b1", &teamB.ID) // label member
	hostB2 := newHost("b2", &teamB.ID) // not a member

	label, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  t.Name() + "-label",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	for _, h := range []*fleet.Host{hostA1, hostB1} {
		err = s.ds.RecordLabelQueryExecutions(ctx, h, map[uint]*bool{label.ID: new(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	scoped := []fleet.LabelIdent{{LabelName: label.Name}}

	unscopedA, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-unscoped-A", TeamID: &teamA.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
	})
	require.NoError(t, err)
	scopedA, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-scoped-A", TeamID: &teamA.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
		LabelsIncludeAny: scoped,
	})
	require.NoError(t, err)
	unscopedB, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-unscoped-B", TeamID: &teamB.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
	})
	require.NoError(t, err)
	scopedB, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-scoped-B", TeamID: &teamB.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
		LabelsIncludeAny: scoped,
	})
	require.NoError(t, err)

	teamQueries := func(host *fleet.Host, teamID uint) map[string]any {
		req := getClientConfigRequest{NodeKey: *host.NodeKey}
		var resp getClientConfigResponse
		s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
		packs, ok := resp.Config["packs"].(map[string]any)
		require.True(t, ok, "expected a packs key in the osquery config")
		teamPack, ok := packs[fmt.Sprintf("team-%d", teamID)].(map[string]any)
		require.True(t, ok, "expected a team pack in the osquery config")
		return teamPack["queries"].(map[string]any)
	}

	// Team A: the label member calls GetClientConfig first (populates the cache).
	queries := teamQueries(hostA1, teamA.ID)
	require.Contains(t, queries, unscopedA.Name)
	require.Contains(t, queries, scopedA.Name)

	// The non-member must NOT receive the label-scoped query.
	queries = teamQueries(hostA2, teamA.ID)
	require.Contains(t, queries, unscopedA.Name)
	require.NotContains(t, queries, scopedA.Name,
		"host that is not a member of the label received the label-scoped query")

	// Team B: the non-member calls GetClientConfig first (populates the cache).
	queries = teamQueries(hostB2, teamB.ID)
	require.Contains(t, queries, unscopedB.Name)
	require.NotContains(t, queries, scopedB.Name,
		"host that is not a member of the label received the label-scoped query")

	// The label member must still receive its label-scoped query.
	queries = teamQueries(hostB1, teamB.ID)
	require.Contains(t, queries, unscopedB.Name)
	require.Contains(t, queries, scopedB.Name,
		"label member did not receive its label-scoped query")
}
