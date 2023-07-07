package mysql

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueries(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Apply", testQueriesApply},
		{"Delete", testQueriesDelete},
		{"GetByName", testQueriesGetByName},
		{"DeleteMany", testQueriesDeleteMany},
		// {"Save", testQueriesSave},
		// {"List", testQueriesList},
		// {"LoadPacksForQueries", testQueriesLoadPacksForQueries},
		// {"DuplicateNew", testQueriesDuplicateNew},
		// {"ListFiltersObservers", testQueriesListFiltersObservers},
		// {"ObserverCanRunQuery", testObserverCanRunQuery},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testQueriesApply(t *testing.T, ds *Datastore) {
	test.AddAllHostsLabel(t, ds)

	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	groob := test.NewUser(t, ds, "Victor", "victor@fleet.co", true)

	expectedQueries := []*fleet.Query{
		{
			Name:               "foo",
			Description:        "get the foos",
			Query:              "select * from foo",
			ObserverCanRun:     true,
			ScheduleInterval:   10,
			Platform:           ptr.String("macos"),
			MinOsqueryVersion:  ptr.String("5.2.1"),
			AutomationsEnabled: true,
			LoggingType:        "differential",
		},
		{
			Name:        "bar",
			Description: "do some bars",
			Query:       "select baz from bar",
		},
	}

	// Zach creates some queries
	err := ds.ApplyQueries(context.Background(), zwass.ID, expectedQueries)
	require.NoError(t, err)

	queries, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	// Check all queries were authored by zwass
	for _, q := range queries {
		require.Equal(t, &zwass.ID, q.AuthorID)
		require.Equal(t, zwass.Email, q.AuthorEmail)
		require.Equal(t, zwass.Name, q.AuthorName)
	}

	// Victor modifies a query (but also pushes the same version of the
	// first query)
	expectedQueries[1].Query = "not really a valid query ;)"
	err = ds.ApplyQueries(context.Background(), groob.ID, expectedQueries)
	require.NoError(t, err)

	queries, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	// Check queries were authored by groob
	for _, q := range queries {
		assert.Equal(t, &groob.ID, q.AuthorID)
		require.Equal(t, groob.Email, q.AuthorEmail)
		require.Equal(t, groob.Name, q.AuthorName)
	}

	// Zach adds a third query (but does not re-apply the others)
	expectedQueries = append(expectedQueries,
		&fleet.Query{
			Name:        "trouble",
			Description: "Look out!",
			Query:       "select * from time",
		},
	)
	err = ds.ApplyQueries(context.Background(), zwass.ID, []*fleet.Query{expectedQueries[2]})
	require.NoError(t, err)

	queries, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	for _, q := range queries {
		switch q.Name {
		case "foo", "bar":
			require.Equal(t, &groob.ID, q.AuthorID)
			require.Equal(t, groob.Email, q.AuthorEmail)
			require.Equal(t, groob.Name, q.AuthorName)
		default:
			require.Equal(t, &zwass.ID, q.AuthorID)
			require.Equal(t, zwass.Email, q.AuthorEmail)
			require.Equal(t, zwass.Name, q.AuthorName)
		}
	}
}

func testQueriesDelete(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
	}
	query, err := ds.NewQuery(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, query.ID, 0)

	err = ds.DeleteQuery(context.Background(), query.TeamID, query.Name)
	require.NoError(t, err)

	require.NotEqual(t, query.ID, 0)
	_, err = ds.Query(context.Background(), query.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
}

func testQueriesGetByName(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	q := test.NewQuery(t, ds, nil, "q1", "select * from time", user.ID, true)

	actual, err := ds.QueryByName(context.Background(), q.TeamID, q.Name)
	require.NoError(t, err)
	require.Equal(t, "q1", actual.Name)
	require.Equal(t, "select * from time", actual.Query)

	actual, err = ds.QueryByName(context.Background(), q.TeamID, "xxx")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
}

func testQueriesDeleteMany(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	q1 := test.NewQuery(t, ds, nil, "q1", "select * from time", user.ID, true)
	q2 := test.NewQuery(t, ds, nil, "q2", "select * from processes", user.ID, true)
	q3 := test.NewQuery(t, ds, nil, "q3", "select 1", user.ID, true)
	q4 := test.NewQuery(t, ds, nil, "q4", "select * from osquery_info", user.ID, true)

	queries, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 4)

	deleted, err := ds.DeleteQueries(context.Background(), []uint{q1.ID, q3.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(2), deleted)

	queries, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 2)

	deleted, err = ds.DeleteQueries(context.Background(), []uint{q2.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 1)

	deleted, err = ds.DeleteQueries(context.Background(), []uint{q2.ID, q4.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 0)
}

func testQueriesSave(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
	}
	query, err := ds.NewQuery(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, 0, query.ID)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "some kind of nature",
		Description: "some kind of goal",
	})
	require.NoError(t, err)

	query.Query = "baz"
	query.ObserverCanRun = true
	query.TeamID = &team.ID
	query.ScheduleInterval = 10
	query.Platform = ptr.String("macos")
	query.MinOsqueryVersion = ptr.String("5.2.1")
	query.AutomationsEnabled = true
	query.LoggingType = "differential"

	err = ds.SaveQuery(context.Background(), query)
	require.NoError(t, err)

	queryVerify, err := ds.Query(context.Background(), query.ID)
	require.NoError(t, err)
	require.NotNil(t, queryVerify)

	assert.Equal(t, "baz", queryVerify.Query)
	assert.Equal(t, "Zach", queryVerify.AuthorName)
	assert.Equal(t, "zwass@fleet.co", queryVerify.AuthorEmail)
	assert.True(t, queryVerify.ObserverCanRun)
	assert.Equal(t, *query.TeamID, team.ID)
	assert.Equal(t, query.ScheduleInterval, uint(10))
	assert.Equal(t, *query.Platform, "macos")
	assert.Equal(t, *query.MinOsqueryVersion, "5.2.1")
	assert.Equal(t, query.AutomationsEnabled, true)
	assert.Equal(t, query.LoggingType, "differential")
}

func testQueriesList(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	for i := 0; i < 10; i++ {
		_, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:     fmt.Sprintf("name%02d", i),
			Query:    fmt.Sprintf("query%02d", i),
			Saved:    true,
			AuthorID: &user.ID,
		})
		require.Nil(t, err)
	}

	// One unsaved query should not be returned
	_, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:     "unsaved",
		Query:    "select * from time",
		Saved:    false,
		AuthorID: &user.ID,
	})
	require.NoError(t, err)

	opts := fleet.ListQueryOptions{}
	results, err := ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 10, len(results))
	assert.Equal(t, "Zach", results[0].AuthorName)
	assert.Equal(t, "zwass@fleet.co", results[0].AuthorEmail)

	idWithAgg := results[0].ID

	_, err = ds.writer(context.Background()).Exec(
		`INSERT INTO aggregated_stats(id,global_stats,type,json_value) VALUES (?,?,?,?)`,
		idWithAgg, false, aggregatedStatsTypeQuery, `{"user_time_p50": 10.5777, "user_time_p95": 111.7308, "system_time_p50": 0.6936, "system_time_p95": 95.8654, "total_executions": 5038}`,
	)
	require.NoError(t, err)

	results, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, 10, len(results))

	foundAgg := false
	for _, q := range results {
		if q.ID == idWithAgg {
			foundAgg = true
			require.NotNil(t, q.SystemTimeP50)
			require.NotNil(t, q.SystemTimeP95)
			assert.Equal(t, 0.6936, *q.SystemTimeP50)
			assert.Equal(t, 95.8654, *q.SystemTimeP95)
		}
	}
	require.True(t, foundAgg)
}

func testQueriesLoadPacksForQueries(t *testing.T, ds *Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "q1", Query: "select * from time"},
		{Name: "q2", Query: "select * from osquery_info"},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries)
	require.Nil(t, err)

	specs := []*fleet.PackSpec{
		{Name: "p1"},
		{Name: "p2"},
		{Name: "p3"},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.Nil(t, err)

	q0, err := ds.QueryByName(context.Background(), nil, queries[0].Name)
	require.Nil(t, err)
	assert.Empty(t, q0.Packs)

	q1, err := ds.QueryByName(context.Background(), nil, queries[1].Name)
	require.Nil(t, err)
	assert.Empty(t, q1.Packs)

	specs = []*fleet.PackSpec{
		{
			Name: "p2",
			Queries: []fleet.PackSpecQuery{
				{
					Name:      "q0",
					QueryName: queries[0].Name,
					Interval:  60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.Nil(t, err)

	q0, err = ds.QueryByName(context.Background(), nil, queries[0].Name)
	require.Nil(t, err)
	if assert.Len(t, q0.Packs, 1) {
		assert.Equal(t, "p2", q0.Packs[0].Name)
	}

	q1, err = ds.QueryByName(context.Background(), nil, queries[1].Name)
	require.Nil(t, err)
	assert.Empty(t, q1.Packs)

	specs = []*fleet.PackSpec{
		{
			Name: "p1",
			Queries: []fleet.PackSpecQuery{
				{
					QueryName: queries[1].Name,
					Interval:  60,
				},
			},
		},
		{
			Name: "p3",
			Queries: []fleet.PackSpecQuery{
				{
					QueryName: queries[1].Name,
					Interval:  60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.Nil(t, err)

	q0, err = ds.QueryByName(context.Background(), nil, queries[0].Name)
	require.Nil(t, err)
	if assert.Len(t, q0.Packs, 1) {
		assert.Equal(t, "p2", q0.Packs[0].Name)
	}

	q1, err = ds.QueryByName(context.Background(), nil, queries[1].Name)
	require.Nil(t, err)
	if assert.Len(t, q1.Packs, 2) {
		sort.Slice(q1.Packs, func(i, j int) bool { return q1.Packs[i].Name < q1.Packs[j].Name })
		assert.Equal(t, "p1", q1.Packs[0].Name)
		assert.Equal(t, "p3", q1.Packs[1].Name)
	}

	specs = []*fleet.PackSpec{
		{
			Name: "p3",
			Queries: []fleet.PackSpecQuery{
				{
					Name:      "q0",
					QueryName: queries[0].Name,
					Interval:  60,
				},
				{
					Name:      "q1",
					QueryName: queries[1].Name,
					Interval:  60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.Nil(t, err)

	q0, err = ds.QueryByName(context.Background(), nil, queries[0].Name)
	require.Nil(t, err)
	if assert.Len(t, q0.Packs, 2) {
		sort.Slice(q0.Packs, func(i, j int) bool { return q0.Packs[i].Name < q0.Packs[j].Name })
		assert.Equal(t, "p2", q0.Packs[0].Name)
		assert.Equal(t, "p3", q0.Packs[1].Name)
	}

	q1, err = ds.QueryByName(context.Background(), nil, queries[1].Name)
	require.Nil(t, err)
	if assert.Len(t, q1.Packs, 2) {
		sort.Slice(q1.Packs, func(i, j int) bool { return q1.Packs[i].Name < q1.Packs[j].Name })
		assert.Equal(t, "p1", q1.Packs[0].Name)
		assert.Equal(t, "p3", q1.Packs[1].Name)
	}
}

func testQueriesDuplicateNew(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Mike Arpaia", "mike@fleet.co", true)

	// The uniqueness of 'global' queries should be based on their name alone.
	globalQ1, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:     "foo",
		Query:    "select * from time;",
		AuthorID: &user.ID,
	})
	require.NoError(t, err)
	assert.NotZero(t, globalQ1.ID)
	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:  "foo",
		Query: "select * from osquery_info;",
	})
	assert.Contains(t, err.Error(), "already exists")

	// Check uniqueness constraint on queries that belong to a team
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "some kind of nature",
		Description: "some kind of goal",
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:   "foo",
		Query:  "select * from osquery_info;",
		TeamID: &team.ID,
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:   "foo",
		Query:  "select * from osquery_info;",
		TeamID: &team.ID,
	})
	assert.Contains(t, err.Error(), "already exists")
}

func testQueriesListFiltersObservers(t *testing.T, ds *Datastore) {
	_, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:  "query1",
		Query: "select 1;",
		Saved: true,
	})
	require.NoError(t, err)
	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:  "query2",
		Query: "select 1;",
		Saved: true,
	})
	require.NoError(t, err)
	query3, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "query3",
		Query:          "select 1;",
		Saved:          true,
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	queries, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, 3)

	queries, err = ds.ListQueries(
		context.Background(),
		fleet.ListQueryOptions{OnlyObserverCanRun: true, ListOptions: fleet.ListOptions{PerPage: 1}},
	)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, query3.ID, queries[0].ID)
}

func testObserverCanRunQuery(t *testing.T, ds *Datastore) {
	_, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "canRunTrue",
		Query:          "select 1;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "canRunFalse",
		Query:          "select 1;",
		ObserverCanRun: false,
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:  "canRunOmitted",
		Query: "select 1;",
	})
	require.NoError(t, err)

	queries, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)

	for _, q := range queries {
		canRun, err := ds.ObserverCanRunQuery(context.Background(), q.ID)
		require.NoError(t, err)
		require.Equal(t, q.ObserverCanRun, canRun)
	}
}
