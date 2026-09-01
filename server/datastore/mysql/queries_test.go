package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"testing"
	"time"

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
		{"Save", testQueriesSave},
		{"List", testQueriesList},
		{"LoadPacksForQueries", testQueriesLoadPacksForQueries},
		{"DuplicateNew", testQueriesDuplicateNew},
		{"ObserverCanRunQuery", testObserverCanRunQuery},
		{"ListQueriesFiltersByTeamID", testListQueriesFiltersByTeamID},
		{"ListQueriesFiltersByIsScheduled", testListQueriesFiltersByIsScheduled},
		{"ListScheduledQueriesForAgents", testListScheduledQueriesForAgents},
		{"IsSavedQuery", testIsSavedQuery},
		{"SaveQueryLabels", testSaveQueryLabels},
		{"ListScheduledQueriesForAgentsWithLabels", testListScheduledQueriesForAgentsWithLabels},
		{"QueriesPerHost", testQueriesPerHost},
		{"HasLabelScopedScheduledQueries", testHasLabelScopedScheduledQueries},
		{"LabelScopedScheduledQueryScopes", testLabelScopedScheduledQueryScopes},
		{"QueryLabelsAtomic", testQueryLabelsAtomic},
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

	// Add a user-defined label
	fooLabel, err := ds.NewLabel(
		context.Background(),
		&fleet.Label{
			Name:                "Foo",
			Query:               "select 1",
			LabelType:           fleet.LabelTypeRegular,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
	)
	require.NoError(t, err)

	barLabel, err := ds.NewLabel(
		context.Background(),
		&fleet.Label{
			Name:                "Bar",
			Query:               "select 1",
			LabelType:           fleet.LabelTypeRegular,
			LabelMembershipType: fleet.LabelMembershipTypeManual,
		},
	)
	require.NoError(t, err)

	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	groob := test.NewUser(t, ds, "Victor", "victor@fleet.co", true)

	expectedQueries := []*fleet.Query{
		{
			Name:               "foo",
			Description:        "get the foos",
			Query:              "select * from foo",
			ObserverCanRun:     true,
			Interval:           10,
			Platform:           "darwin",
			MinOsqueryVersion:  "5.2.1",
			AutomationsEnabled: true,
			Logging:            fleet.LoggingDifferential,
			DiscardData:        true,
			LabelsIncludeAny:   []fleet.LabelIdent{{LabelID: fooLabel.ID, LabelName: fooLabel.Name}},
		},
		{
			Name:             "bar",
			Description:      "do some bars",
			Query:            "select baz from bar",
			Logging:          fleet.LoggingSnapshot,
			DiscardData:      true,
			LabelsIncludeAny: []fleet.LabelIdent{{LabelID: fooLabel.ID, LabelName: fooLabel.Name}},
		},
	}

	// Zach creates some queries
	err = ds.ApplyQueries(context.Background(), zwass.ID, expectedQueries, nil)
	require.NoError(t, err)

	queries, count, _, _, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))
	require.Equal(t, count, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	// Check all queries were authored by zwass
	for _, q := range queries {
		require.Equal(t, &zwass.ID, q.AuthorID)
		require.Equal(t, zwass.Email, q.AuthorEmail)
		require.Equal(t, zwass.Name, q.AuthorName)
		require.True(t, q.Saved)
	}

	// Update the first query to have a different label
	expectedQueries[0].LabelsIncludeAny = []fleet.LabelIdent{{LabelID: barLabel.ID, LabelName: barLabel.Name}}

	err = ds.ApplyQueries(context.Background(), zwass.ID, expectedQueries, nil)
	require.NoError(t, err)

	queries, count, _, _, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))
	require.Equal(t, count, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	// Victor modifies a query (but also pushes the same version of the
	// first query)
	expectedQueries[1].Query = "not really a valid query ;)"
	err = ds.ApplyQueries(context.Background(), groob.ID, expectedQueries, nil)
	require.NoError(t, err)

	queries, count, _, _, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))
	require.Equal(t, count, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	// Check queries were authored by groob
	for _, q := range queries {
		assert.Equal(t, &groob.ID, q.AuthorID)
		require.Equal(t, groob.Email, q.AuthorEmail)
		require.Equal(t, groob.Name, q.AuthorName)
		require.True(t, q.Saved)
	}

	// Zach adds a third query (but does not re-apply the others)
	expectedQueries = append(expectedQueries,
		&fleet.Query{
			Name:        "trouble",
			Description: "Look out!",
			Query:       "select * from time",
			DiscardData: true,
			Logging:     fleet.LoggingDifferential,
		},
	)
	err = ds.ApplyQueries(context.Background(), zwass.ID, []*fleet.Query{expectedQueries[2]}, nil)
	require.NoError(t, err)

	queries, count, _, _, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	require.Len(t, queries, len(expectedQueries))
	require.Equal(t, count, len(expectedQueries))

	test.QueryElementsMatch(t, expectedQueries, queries)

	for _, q := range queries {
		require.True(t, q.Saved)
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

	// Zach tries to add a query with an invalid platform string
	invalidQueries := []*fleet.Query{
		{
			Name:               "foo",
			Description:        "get the foos",
			Query:              "select * from foo",
			ObserverCanRun:     true,
			Interval:           10,
			Platform:           "not valid",
			MinOsqueryVersion:  "5.2.1",
			AutomationsEnabled: true,
			Logging:            fleet.LoggingDifferential,
		},
	}
	err = ds.ApplyQueries(context.Background(), zwass.ID, invalidQueries, nil)
	require.ErrorIs(t, err, fleet.ErrQueryInvalidPlatform)
}

func testQueriesDelete(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	hostID := uint(1)
	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
		Logging:  fleet.LoggingDifferential,
	}
	query, err := ds.NewQuery(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, query.ID, 0)
	lastExecuted := time.Now().Add(-time.Hour).Round(time.Second) // TIMESTAMP precision is seconds by default in MySQL
	err = ds.UpdateLiveQueryStats(
		context.Background(), query.ID, []*fleet.LiveQueryStats{
			{
				HostID:       hostID,
				Executions:   1,
				LastExecuted: lastExecuted,
			},
		},
	)
	require.NoError(t, err)
	// Check that the stats were saved correctly
	stats, err := ds.GetLiveQueryStats(context.Background(), query.ID, []uint{hostID})
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, hostID, stats[0].HostID)
	assert.Equal(t, uint64(1), stats[0].Executions)
	assert.Equal(t, lastExecuted.UTC(), stats[0].LastExecuted.UTC())

	err = ds.CalculateAggregatedPerfStatsPercentiles(context.Background(), fleet.AggregatedStatsTypeScheduledQuery, query.ID)
	require.NoError(t, err)

	err = ds.DeleteQuery(context.Background(), query.TeamID, query.Name)
	require.NoError(t, err)

	require.NotEqual(t, query.ID, 0)
	_, err = ds.Query(context.Background(), query.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Ensure stats were deleted.
	// The actual delete occurs asynchronously, so we for-loop.
	statsGone := make(chan bool)
	go func() {
		for {
			stats, err := ds.GetLiveQueryStats(context.Background(), query.ID, []uint{hostID})
			require.NoError(t, err)
			if len(stats) == 0 {
				_, err = GetAggregatedStats(context.Background(), ds, fleet.AggregatedStatsTypeScheduledQuery, query.ID)
				if errors.Is(err, sql.ErrNoRows) {
					statsGone <- true
					break
				}
			}
		}
	}()
	select {
	case <-statsGone:
	case <-time.After(10 * time.Second):
		t.Error("Timeout: stats not deleted for testQueriesDelete")
	}
}

func testQueriesGetByName(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	// Test we can get global queries by name
	globalQ := test.NewQuery(t, ds, nil, "q1", "select * from time", user.ID, true)

	actual, err := ds.QueryByName(context.Background(), nil, globalQ.Name)
	require.NoError(t, err)
	require.Nil(t, actual.TeamID)
	require.Equal(t, "q1", actual.Name)
	require.Equal(t, "select * from time", actual.Query)

	_, err = ds.QueryByName(context.Background(), nil, "xxx")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Test we can get queries in a team
	teamRocket, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "Team Rocket",
		Description: "Something cheesy",
	})
	require.NoError(t, err)

	teamRocketQ := test.NewQuery(t, ds, &teamRocket.ID, "q1", "select * from time", user.ID, true)

	actual, err = ds.QueryByName(context.Background(), &teamRocket.ID, teamRocketQ.Name)
	require.NoError(t, err)
	require.Equal(t, "q1", actual.Name)
	require.Equal(t, teamRocket.ID, *actual.TeamID)
	require.Equal(t, "select * from time", actual.Query)

	_, err = ds.QueryByName(context.Background(), &teamRocket.ID, "xxx")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// QueriesByName resolves many (team, name) pairs in one lookup, keeps global
	// and same-named team queries distinct, and omits names that don't exist.
	got, err := ds.QueriesByName(context.Background(), []fleet.TeamScopedQueryName{
		{TeamID: nil, Name: "q1"},              // global q1
		{TeamID: &teamRocket.ID, Name: "q1"},   // team q1, same name
		{TeamID: nil, Name: "nope"},            // missing
		{TeamID: &teamRocket.ID, Name: "nope"}, // missing
		{TeamID: nil, Name: "q1"},              // duplicate of the first
	})
	require.NoError(t, err)
	require.Len(t, got, 2)

	globalHit := got[fleet.TeamScopedQueryName{TeamID: nil, Name: "q1"}.Key()]
	require.NotNil(t, globalHit)
	require.Equal(t, globalQ.ID, globalHit.ID)
	require.Nil(t, globalHit.TeamID)

	teamHit := got[fleet.TeamScopedQueryName{TeamID: &teamRocket.ID, Name: "q1"}.Key()]
	require.NotNil(t, teamHit)
	require.Equal(t, teamRocketQ.ID, teamHit.ID)
	require.Equal(t, teamRocket.ID, *teamHit.TeamID)

	// A name that exists on one team must not resolve when requested for another
	// team: the tuple scoping must be exact, not name-only.
	teamB, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "Team B", Description: "b"})
	require.NoError(t, err)
	crossTeam, err := ds.QueriesByName(context.Background(), []fleet.TeamScopedQueryName{
		{TeamID: &teamB.ID, Name: "q1"}, // q1 exists globally and on teamRocket, not teamB
	})
	require.NoError(t, err)
	require.Empty(t, crossTeam)

	// Forcing a tiny batch size exercises the multi-chunk path: every existing
	// pair must still resolve across chunk boundaries.
	defer func(orig int) { queriesByNameBatchSize = orig }(queriesByNameBatchSize)
	queriesByNameBatchSize = 1
	chunked, err := ds.QueriesByName(context.Background(), []fleet.TeamScopedQueryName{
		{TeamID: nil, Name: "q1"},
		{TeamID: &teamRocket.ID, Name: "q1"},
		{TeamID: nil, Name: "nope"},
	})
	require.NoError(t, err)
	require.Len(t, chunked, 2)
	require.NotNil(t, chunked[fleet.TeamScopedQueryName{TeamID: nil, Name: "q1"}.Key()])
	require.NotNil(t, chunked[fleet.TeamScopedQueryName{TeamID: &teamRocket.ID, Name: "q1"}.Key()])

	// Empty input must not hit the DB and returns an empty map.
	empty, err := ds.QueriesByName(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, empty)
}

func testQueriesDeleteMany(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	q1 := test.NewQuery(t, ds, nil, "q1", "select * from time", user.ID, true)
	q2 := test.NewQuery(t, ds, nil, "q2", "select * from processes", user.ID, true)
	q3 := test.NewQuery(t, ds, nil, "q3", "select 1", user.ID, true)
	q4 := test.NewQuery(t, ds, nil, "q4", "select * from osquery_info", user.ID, true)

	queries, count, _, _, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 4)
	require.Equal(t, count, 4)

	// Add query stats
	hostIDs := []uint{10, 20}
	lastExecuted := time.Now().Add(-time.Hour).Round(time.Second)
	err = ds.UpdateLiveQueryStats(
		context.Background(), q1.ID, []*fleet.LiveQueryStats{
			{
				HostID:       hostIDs[0],
				Executions:   1,
				LastExecuted: lastExecuted,
			},
			{
				HostID:       hostIDs[1],
				Executions:   1,
				LastExecuted: lastExecuted,
			},
		},
	)
	require.NoError(t, err)
	err = ds.UpdateLiveQueryStats(
		context.Background(), q3.ID, []*fleet.LiveQueryStats{
			{
				HostID:       hostIDs[0],
				Executions:   1,
				LastExecuted: lastExecuted,
			},
		},
	)
	require.NoError(t, err)
	err = ds.CalculateAggregatedPerfStatsPercentiles(context.Background(), fleet.AggregatedStatsTypeScheduledQuery, q1.ID)
	require.NoError(t, err)
	err = ds.CalculateAggregatedPerfStatsPercentiles(context.Background(), fleet.AggregatedStatsTypeScheduledQuery, q3.ID)
	require.NoError(t, err)

	deleted, err := ds.DeleteQueries(context.Background(), []uint{q1.ID, q3.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(2), deleted)

	queries, count, _, _, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 2)
	assert.Equal(t, count, 2)

	// Ensure stats were deleted.
	// The actual delete occurs asynchronously, so we for-loop.
	statsGone := make(chan bool)
	go func() {
		for {
			stats, err := ds.GetLiveQueryStats(context.Background(), q1.ID, hostIDs)
			require.NoError(t, err)
			if len(stats) == 0 {
				_, err = GetAggregatedStats(context.Background(), ds, fleet.AggregatedStatsTypeScheduledQuery, q1.ID)
				if errors.Is(err, sql.ErrNoRows) {
					statsGone <- true
					break
				}
			}
			time.Sleep(100 * time.Millisecond) // Add a small delay between checks
		}
	}()
	select {
	case <-statsGone:
	case <-time.After(10 * time.Second):
		t.Error("Timeout: stats not deleted for testQueriesDeleteMany")
	}
	stats, err := ds.GetLiveQueryStats(context.Background(), q3.ID, hostIDs)
	require.NoError(t, err)
	require.Equal(t, 0, len(stats))
	_, err = GetAggregatedStats(context.Background(), ds, fleet.AggregatedStatsTypeScheduledQuery, q3.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	deleted, err = ds.DeleteQueries(context.Background(), []uint{q2.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, count, _, _, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, count, 1)

	deleted, err = ds.DeleteQueries(context.Background(), []uint{q2.ID, q4.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, count, _, _, err = ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 0)
	assert.Equal(t, count, 0)
}

func testQueriesSave(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
		Logging:  fleet.LoggingSnapshot,
	}
	query, err := ds.NewQuery(context.Background(), query)
	require.NoError(t, err)
	require.NotNil(t, query)
	require.NotEqual(t, 0, query.ID)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "some kind of nature",
		Description: "some kind of goal",
	})
	require.NoError(t, err)

	query.Query = "baz"
	query.ObserverCanRun = true
	query.TeamID = &team.ID
	query.Interval = 10
	query.Platform = "darwin"
	query.MinOsqueryVersion = "5.2.1"
	query.AutomationsEnabled = true
	query.Logging = fleet.LoggingDifferential
	query.DiscardData = true

	err = ds.SaveQuery(context.Background(), query, true, false)
	require.NoError(t, err)

	actual, err := ds.Query(context.Background(), query.ID)
	require.NoError(t, err)
	require.NotNil(t, actual)

	test.QueriesMatch(t, actual, query)

	require.Equal(t, "baz", actual.Query)
	require.Equal(t, "Zach", actual.AuthorName)
	require.Equal(t, "zwass@fleet.co", actual.AuthorEmail)

	// Now save again and delete old stats.
	// First we create stats which will be deleted.
	const hostID = 1
	err = ds.UpdateLiveQueryStats(
		context.Background(), query.ID, []*fleet.LiveQueryStats{
			{
				HostID:     hostID,
				Executions: 1,
			},
		},
	)
	require.NoError(t, err)
	err = ds.CalculateAggregatedPerfStatsPercentiles(context.Background(), fleet.AggregatedStatsTypeScheduledQuery, query.ID)
	require.NoError(t, err)
	// Update/save query.
	query.Query = "baz2"
	err = ds.SaveQuery(context.Background(), query, true, true)
	require.NoError(t, err)
	// Ensure stats were deleted.
	// The actual delete occurs asynchronously, so we for-loop.
	aggStatsGone := make(chan bool)
	go func() {
		for {
			actual, err = ds.Query(context.Background(), query.ID)
			require.NoError(t, err)
			require.NotNil(t, actual)
			if actual.AggregatedStats.TotalExecutions == nil {
				aggStatsGone <- true
				break
			}
		}
	}()
	select {
	case <-aggStatsGone:
	case <-time.After(10 * time.Second):
		t.Error("Timeout: aggregated stats not deleted for query")
	}
	test.QueriesMatch(t, query, actual)
	stats, err := ds.GetLiveQueryStats(context.Background(), query.ID, []uint{hostID})
	require.NoError(t, err)
	require.Equal(t, 0, len(stats))
	_, err = GetAggregatedStats(context.Background(), ds, fleet.AggregatedStatsTypeScheduledQuery, query.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func testQueriesList(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	for i := 0; i < 10; i++ {
		// populate platform field of first 4 queries
		var p string
		switch i {
		case 0:
			p = "darwin"
		case 1:
			p = "windows"
		case 2:
			p = "linux"
		case 3:
			p = "darwin,windows,linux"
		}

		_, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:           fmt.Sprintf("name%02d", i),
			Query:          fmt.Sprintf("query%02d", i),
			Saved:          true,
			AuthorID:       &user.ID,
			DiscardData:    true,
			ObserverCanRun: rand.Intn(2) == 0, //nolint:gosec
			Logging:        fleet.LoggingSnapshot,
			Platform:       p,
		})
		require.Nil(t, err)
	}

	// One unsaved query should not be returned
	_, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:     "unsaved",
		Query:    "select * from time",
		Saved:    false,
		AuthorID: &user.ID,
		Logging:  fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	opts := fleet.ListQueryOptions{}
	opts.IncludeMetadata = true

	opts.Platform = ptr.String("darwin")
	// filtered by platform
	results, count, _, meta, err := ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 8, len(results))
	assert.Equal(t, count, 8)
	assert.False(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)
	require.Equal(t, "darwin", results[0].Platform)
	require.Equal(t, "darwin,windows,linux", results[1].Platform)

	opts.Platform = ptr.String("windows")
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 8, len(results))
	assert.Equal(t, count, 8)
	assert.False(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)
	require.Equal(t, "windows", results[0].Platform)
	require.Equal(t, "darwin,windows,linux", results[1].Platform)

	opts.Platform = ptr.String("linux")
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 8, len(results))
	assert.Equal(t, count, 8)
	assert.False(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)
	require.Equal(t, "linux", results[0].Platform)
	require.Equal(t, "darwin,windows,linux", results[1].Platform)

	opts.Platform = ptr.String("lucas")
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	// only returns queries set to run on all platforms with platform == ""
	require.Equal(t, 6, len(results))
	assert.Equal(t, count, 6)
	assert.False(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)

	opts.Platform = nil

	// paginated - beginning
	opts.PerPage = 3
	opts.Page = 0
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 3, len(results))
	require.Equal(t, "Zach", results[0].AuthorName)
	require.Equal(t, "zwass@fleet.co", results[0].AuthorEmail)
	require.True(t, results[0].DiscardData)
	assert.Equal(t, count, 10)
	assert.False(t, meta.HasPreviousResults)
	assert.True(t, meta.HasNextResults)

	// paginated - middle
	opts.Page = 1
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 3, len(results))
	require.Equal(t, "Zach", results[0].AuthorName)
	require.Equal(t, "zwass@fleet.co", results[0].AuthorEmail)
	require.True(t, results[0].DiscardData)
	assert.Equal(t, count, 10)
	assert.True(t, meta.HasPreviousResults)
	assert.True(t, meta.HasNextResults)

	// paginated - end
	opts.Page = 3
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 1, len(results))
	require.Equal(t, "Zach", results[0].AuthorName)
	require.Equal(t, "zwass@fleet.co", results[0].AuthorEmail)
	require.True(t, results[0].DiscardData)
	assert.Equal(t, count, 10)
	assert.True(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)

	// paginated - past end
	opts.Page = 4
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 0, len(results))
	assert.Equal(t, count, 10)
	assert.True(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)

	opts.PerPage = 0
	opts.Page = 0
	results, count, _, meta, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 10, len(results))
	require.Equal(t, "Zach", results[0].AuthorName)
	require.Equal(t, "zwass@fleet.co", results[0].AuthorEmail)
	require.True(t, results[0].DiscardData)
	assert.Equal(t, count, 10)
	assert.False(t, meta.HasPreviousResults)
	assert.False(t, meta.HasNextResults)

	idWithAgg := results[0].ID

	_, err = ds.writer(context.Background()).Exec(
		`INSERT INTO aggregated_stats(id,global_stats,type,json_value) VALUES (?,?,?,?)`,
		idWithAgg, false, fleet.AggregatedStatsTypeScheduledQuery,
		`{"user_time_p50": 10.5777, "user_time_p95": 111.7308, "system_time_p50": 0.6936, "system_time_p95": 95.8654, "total_executions": 5038}`,
	)
	require.NoError(t, err)

	results, _, _, _, err = ds.ListQueries(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, 10, len(results))

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
		{Name: "q1", Query: "select * from time", Logging: fleet.LoggingSnapshot},
		{Name: "q2", Query: "select * from osquery_info", Logging: fleet.LoggingDifferential},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries, nil)
	require.NoError(t, err)

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
		Logging:  fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	require.NotZero(t, globalQ1.ID)
	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "foo",
		Query:   "select * from osquery_info;",
		Logging: fleet.LoggingSnapshot,
	})
	require.Contains(t, err.Error(), "already exists")

	// Check uniqueness constraint on queries that belong to a team
	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "some kind of nature",
		Description: "some kind of goal",
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "foo",
		Query:   "select * from osquery_info;",
		TeamID:  &team.ID,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "foo",
		Query:   "select * from osquery_info;",
		TeamID:  &team.ID,
		Logging: fleet.LoggingSnapshot,
	})
	require.Contains(t, err.Error(), "already exists")
}

func testObserverCanRunQuery(t *testing.T, ds *Datastore) {
	_, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "canRunTrue",
		Query:          "select 1;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "canRunFalse",
		Query:          "select 1;",
		ObserverCanRun: false,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	_, err = ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "canRunOmitted",
		Query:   "select 1;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	queries, _, _, _, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)

	for _, q := range queries {
		canRun, err := ds.ObserverCanRunQuery(context.Background(), q.ID)
		require.NoError(t, err)
		require.Equal(t, q.ObserverCanRun, canRun)
	}
}

func testListQueriesFiltersByTeamID(t *testing.T, ds *Datastore) {
	globalQ1, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "query1",
		Query:   "select 1;",
		Saved:   true,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	globalQ2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "query2",
		Query:   "select 1;",
		Saved:   true,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	globalQ3, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "query3",
		Query:   "select 1;",
		Saved:   true,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	queries, count, _, _, err := ds.ListQueries(context.Background(), fleet.ListQueryOptions{})
	require.NoError(t, err)
	test.QueryElementsMatch(t, queries, []*fleet.Query{globalQ1, globalQ2, globalQ3})
	assert.Equal(t, count, 3)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "some kind of nature",
		Description: "some kind of goal",
	})
	require.NoError(t, err)

	teamQ1, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "query1",
		Query:   "select 1;",
		Saved:   true,
		TeamID:  &team.ID,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	teamQ2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "query2",
		Query:   "select 1;",
		Saved:   true,
		TeamID:  &team.ID,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	teamQ3, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:    "query3",
		Query:   "select 1;",
		Saved:   true,
		TeamID:  &team.ID,
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	queries, count, _, _, err = ds.ListQueries(
		context.Background(),
		fleet.ListQueryOptions{
			TeamID: &team.ID,
		},
	)
	require.NoError(t, err)
	test.QueryElementsMatch(t, queries, []*fleet.Query{teamQ1, teamQ2, teamQ3})
	assert.Equal(t, count, 3)

	// test merge inherited
	queries, count, _, _, err = ds.ListQueries(
		context.Background(),
		fleet.ListQueryOptions{
			TeamID:         &team.ID,
			MergeInherited: true,
		},
	)
	require.NoError(t, err)
	test.QueryElementsMatch(t, queries, []*fleet.Query{globalQ1, globalQ2, globalQ3, teamQ1, teamQ2, teamQ3})
	assert.Equal(t, count, 6)

	// merge inherited ignored for global queries
	queries, count, _, _, err = ds.ListQueries(
		context.Background(),
		fleet.ListQueryOptions{
			MergeInherited: true,
		},
	)
	require.NoError(t, err)
	test.QueryElementsMatch(t, queries, []*fleet.Query{globalQ1, globalQ2, globalQ3})
	assert.Equal(t, count, 3)
}

func testListQueriesFiltersByIsScheduled(t *testing.T, ds *Datastore) {
	q1, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:     "query1",
		Query:    "select 1;",
		Saved:    true,
		Interval: 0,
		Logging:  fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "query2",
		Query:              "select 1;",
		Saved:              true,
		Interval:           10,
		AutomationsEnabled: false,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	q3, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "query3",
		Query:              "select 1;",
		Saved:              true,
		Interval:           20,
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	testCases := []struct {
		opts     fleet.ListQueryOptions
		expected []*fleet.Query
	}{
		{
			opts: fleet.ListQueryOptions{},

			expected: []*fleet.Query{q1, q2, q3},
		},
		{
			opts:     fleet.ListQueryOptions{IsScheduled: ptr.Bool(true)},
			expected: []*fleet.Query{q3},
		},
		{
			opts:     fleet.ListQueryOptions{IsScheduled: ptr.Bool(false)},
			expected: []*fleet.Query{q1, q2},
		},
	}

	for i, tCase := range testCases {
		queries, count, _, _, err := ds.ListQueries(
			context.Background(),
			tCase.opts,
		)
		require.NoError(t, err)
		test.QueryElementsMatch(t, queries, tCase.expected, i)
		assert.Equal(t, count, len(tCase.expected))

	}
}

func testListScheduledQueriesForAgents(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "Team 1",
		Description: "Team 1",
	})
	require.NoError(t, err)

	for i, teamID := range []*uint{nil, &team.ID} {
		var teamIDStr string
		if teamID != nil {
			teamIDStr = fmt.Sprintf("%d", *teamID)
		}

		// Non saved queries should not be returned here.
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query1", teamIDStr),
			Query:              "select 1;",
			Saved:              false,
			Interval:           10,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=0, DiscardData=0, Snapshot=0
		_, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query2", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			TeamID:             teamID,
			AutomationsEnabled: false,
			DiscardData:        false,
			Logging:            fleet.LoggingDifferential,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=0, DiscardData=0, Snapshot=1
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query3", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			TeamID:             teamID,
			AutomationsEnabled: false,
			DiscardData:        false,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=0, DiscardData=1, Snapshot=0
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query4", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingDifferential,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=0, DiscardData=1, Snapshot=1
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query5", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=1, DiscardData=0, Snapshot=0
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query6", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        false,
			Logging:            fleet.LoggingDifferential,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=1, DiscardData=0, Snapshot=1
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query7", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        false,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=1, DiscardData=1, Snapshot=0
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query8", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingDifferential,
		})
		require.NoError(t, err)

		// Interval=0, AutomationsEnabled=1, DiscardData=1, Snapshot=1
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query9", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           0,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=0, DiscardData=0, Snapshot=0
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query10", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        false,
			Logging:            fleet.LoggingDifferentialIgnoreRemovals,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=0, DiscardData=0, Snapshot=1
		q11, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query11", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        false,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=0, DiscardData=1, Snapshot=0
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query12", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingDifferentialIgnoreRemovals,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=0, DiscardData=1, Snapshot=1
		_, err = ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query13", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: false,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=1, DiscardData=0, Snapshot=0
		q14, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query14", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        false,
			Logging:            fleet.LoggingDifferential,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=1, DiscardData=0, Snapshot=1
		q15, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query15", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        false,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=1, DiscardData=1, Snapshot=0
		q16, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query16", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingDifferential,
		})
		require.NoError(t, err)

		// Interval=1, AutomationsEnabled=1, DiscardData=1, Snapshot=1
		q17, err := ds.NewQuery(context.Background(), &fleet.Query{
			Name:               fmt.Sprintf("%s query17", teamIDStr),
			Query:              "select 1;",
			Saved:              true,
			Interval:           10,
			AutomationsEnabled: true,
			TeamID:             teamID,
			DiscardData:        true,
			Logging:            fleet.LoggingSnapshot,
		})
		require.NoError(t, err)

		queryReportsDisabled := false
		result, err := ds.ListScheduledQueriesForAgents(ctx, teamID, nil, queryReportsDisabled)
		require.NoError(t, err)
		sort.Slice(result, func(i, j int) bool {
			return result[i].ID < result[j].ID
		})
		test.QueryElementsMatch(t, result, []*fleet.Query{q11, q14, q15, q16, q17}, i)

		queryReportsDisabled = true
		result, err = ds.ListScheduledQueriesForAgents(ctx, teamID, nil, queryReportsDisabled)
		require.NoError(t, err)
		sort.Slice(result, func(i, j int) bool {
			return result[i].ID < result[j].ID
		})
		test.QueryElementsMatch(t, result, []*fleet.Query{q14, q15, q16, q17}, i)
	}
}

func testIsSavedQuery(t *testing.T, ds *Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	// NOT saved query
	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
		Logging:  fleet.LoggingSnapshot,
		Saved:    false,
	}
	query, err := ds.NewQuery(context.Background(), query)
	require.NoError(t, err)
	isSaved, err := ds.IsSavedQuery(context.Background(), query.ID)
	require.NoError(t, err)
	assert.False(t, isSaved)

	// Saved query
	query = &fleet.Query{
		Name:     "foo2",
		Query:    "bar",
		AuthorID: &user.ID,
		Logging:  fleet.LoggingSnapshot,
		Saved:    true,
	}
	query, err = ds.NewQuery(context.Background(), query)
	require.NoError(t, err)
	isSaved, err = ds.IsSavedQuery(context.Background(), query.ID)
	require.NoError(t, err)
	assert.True(t, isSaved)

	// error case
	_, err = ds.IsSavedQuery(context.Background(), math.MaxUint)
	require.Error(t, err)
}

func testSaveQueryLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
	require.NoError(t, err)
	label2, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2"})
	require.NoError(t, err)

	// Create query with label
	query1, err := ds.NewQuery(ctx, &fleet.Query{
		Name:     "query1",
		Query:    "SELECT 1",
		AuthorID: &user.ID,
		Logging:  fleet.LoggingSnapshot,
		Saved:    true,
		LabelsIncludeAny: []fleet.LabelIdent{
			{LabelName: label1.Name},
		},
	})
	require.NoError(t, err)
	require.Len(t, query1.LabelsIncludeAny, 1)
	require.Equal(t, label1.Name, query1.LabelsIncludeAny[0].LabelName)
	require.Equal(t, label1.ID, query1.LabelsIncludeAny[0].LabelID)

	// Change the label
	query1.LabelsIncludeAny = []fleet.LabelIdent{{LabelName: label2.Name}}
	err = ds.SaveQuery(ctx, query1, true, true)
	require.NoError(t, err)
	require.Len(t, query1.LabelsIncludeAny, 1)
	require.Equal(t, label2.Name, query1.LabelsIncludeAny[0].LabelName)
	require.Equal(t, label2.ID, query1.LabelsIncludeAny[0].LabelID)

	// Two labels
	query1.LabelsIncludeAny = []fleet.LabelIdent{{LabelName: label1.Name}, {LabelName: label2.Name}}
	err = ds.SaveQuery(ctx, query1, true, true)
	require.NoError(t, err)
	require.Len(t, query1.LabelsIncludeAny, 2)
	require.Equal(t, label1.Name, query1.LabelsIncludeAny[0].LabelName)
	require.Equal(t, label1.ID, query1.LabelsIncludeAny[0].LabelID)
	require.Equal(t, label2.Name, query1.LabelsIncludeAny[1].LabelName)
	require.Equal(t, label2.ID, query1.LabelsIncludeAny[1].LabelID)

	// Remove all labels
	query1.LabelsIncludeAny = []fleet.LabelIdent{}
	err = ds.SaveQuery(ctx, query1, true, true)
	require.NoError(t, err)
	require.Len(t, query1.LabelsIncludeAny, 0)

	// Switch scope to include_all.
	query1.LabelsIncludeAny = nil
	query1.LabelsIncludeAll = []fleet.LabelIdent{{LabelName: label1.Name}, {LabelName: label2.Name}}
	err = ds.SaveQuery(ctx, query1, true, true)
	require.NoError(t, err)
	require.Empty(t, query1.LabelsIncludeAny)
	require.Len(t, query1.LabelsIncludeAll, 2)

	// Round-trip from DB.
	reloaded, err := ds.Query(ctx, query1.ID)
	require.NoError(t, err)
	require.Empty(t, reloaded.LabelsIncludeAny)
	require.Len(t, reloaded.LabelsIncludeAll, 2)

	// Mutual exclusion is rejected at the datastore boundary.
	query1.LabelsIncludeAny = []fleet.LabelIdent{{LabelName: label1.Name}}
	query1.LabelsIncludeAll = []fleet.LabelIdent{{LabelName: label2.Name}}
	err = ds.SaveQuery(ctx, query1, true, true)
	require.Error(t, err)
}

func testQueriesPerHost(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	otherTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	label, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
	require.NoError(t, err)
	otherLabel, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2"})
	require.NoError(t, err)

	hostInLabel := test.NewHost(t, ds, "host1", "10.0.0.1", "asdf", "host1", time.Now())
	require.NoError(t, ds.AddLabelsToHost(ctx, hostInLabel.ID, []uint{label.ID}))
	hostInOtherLabel := test.NewHost(t, ds, "host2", "10.0.0.2", "asdg", "host2", time.Now())
	require.NoError(t, ds.AddLabelsToHost(ctx, hostInOtherLabel.ID, []uint{otherLabel.ID}))
	hostInBothLabels := test.NewHost(t, ds, "host3", "10.0.0.3", "asdh", "host3", time.Now())
	require.NoError(t, ds.AddLabelsToHost(ctx, hostInBothLabels.ID, []uint{label.ID, otherLabel.ID}))
	hostNoLabels := test.NewHost(t, ds, "host4", "10.0.0.4", "asdj", "host4", time.Now())

	hostOnTeam := test.NewHost(t, ds, "host5", "10.0.0.5", "asdk", "host5", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{hostOnTeam.ID})))
	hostOnOtherTeam := test.NewHost(t, ds, "host6", "10.0.0.6", "asdl", "host6", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&otherTeam.ID, []uint{hostOnOtherTeam.ID})))

	// Read the team hosts back so their TeamID is the one the service would pass.
	hostOnTeam, err = ds.Host(ctx, hostOnTeam.ID)
	require.NoError(t, err)
	require.Equal(t, team.ID, *hostOnTeam.TeamID)
	hostOnOtherTeam, err = ds.Host(ctx, hostOnOtherTeam.ID)
	require.NoError(t, err)

	newQuery := func(t *testing.T, name string, mutate func(*fleet.Query)) *fleet.Query {
		query := &fleet.Query{
			Name:     name,
			Query:    "SELECT 1",
			AuthorID: &user.ID,
			Logging:  fleet.LoggingSnapshot,
			Interval: 10,
			Saved:    true,
		}
		if mutate != nil {
			mutate(query)
		}
		created, err := ds.NewQuery(ctx, query)
		require.NoError(t, err)
		return created
	}

	// Stores results in a report, no automations.
	global := newQuery(t, "global", nil)
	// Stores results and streams them.
	automated := newQuery(t, "automated", func(q *fleet.Query) { q.AutomationsEnabled = true })
	// Streams results without storing them, which is still a reason to run it.
	streamOnly := newQuery(t, "stream only", func(q *fleet.Query) {
		q.AutomationsEnabled = true
		q.Logging = fleet.LoggingDifferential
	})

	// Never scheduled for any host: they would neither store nor stream a result.
	noInterval := newQuery(t, "no interval", func(q *fleet.Query) { q.Interval = 0 })
	notSaved := newQuery(t, "not saved", func(q *fleet.Query) { q.Saved = false })
	discardData := newQuery(t, "discard data", func(q *fleet.Query) { q.DiscardData = true })
	differential := newQuery(t, "differential", func(q *fleet.Query) { q.Logging = fleet.LoggingDifferential })

	includeAny := newQuery(t, "include any", func(q *fleet.Query) {
		q.LabelsIncludeAny = []fleet.LabelIdent{{LabelName: label.Name}}
	})
	includeAll := newQuery(t, "include all", func(q *fleet.Query) {
		q.LabelsIncludeAll = []fleet.LabelIdent{{LabelName: label.Name}, {LabelName: otherLabel.Name}}
	})
	teamQuery := newQuery(t, "team query", func(q *fleet.Query) { q.TeamID = &team.ID })
	otherTeamQuery := newQuery(t, "other team query", func(q *fleet.Query) { q.TeamID = &otherTeam.ID })

	// Scheduled for every host, whatever its team and labels.
	unscopedGlobal := []uint{global.ID, automated.ID, streamOnly.ID}

	for _, tc := range []struct {
		name string
		host *fleet.Host
		want []uint
	}{
		{
			name: "host with no labels gets the unscoped global queries",
			host: hostNoLabels,
			want: unscopedGlobal,
		},
		{
			name: "host in an include_any label also gets the includeAny query",
			host: hostInLabel,
			want: append(slices.Clone(unscopedGlobal), includeAny.ID),
		},
		{
			name: "host in a different label gets neither label-scoped query",
			host: hostInOtherLabel,
			want: unscopedGlobal,
		},
		{
			name: "host in every include_all label gets both label-scoped queries",
			host: hostInBothLabels,
			want: append(slices.Clone(unscopedGlobal), includeAny.ID, includeAll.ID),
		},
		{
			name: "host on a team gets global queries and its own team's",
			host: hostOnTeam,
			want: append(slices.Clone(unscopedGlobal), teamQuery.ID),
		},
		{
			name: "host on another team gets that team's queries instead",
			host: hostOnOtherTeam,
			want: append(slices.Clone(unscopedGlobal), otherTeamQuery.ID),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			queryIDs, err := ds.QueriesPerHost(ctx, tc.host.ID, tc.host.TeamID)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, queryIDs)
			require.NotContains(t, queryIDs, noInterval.ID)
			require.NotContains(t, queryIDs, notSaved.ID)
			require.NotContains(t, queryIDs, discardData.ID)
			require.NotContains(t, queryIDs, differential.ID)
		})
	}
}

func testListScheduledQueriesForAgentsWithLabels(t *testing.T, ds *Datastore) {
	requireQueries := func(t *testing.T, queries []*fleet.Query, names []string) {
		require.Len(t, queries, len(names))
		for _, name := range names {
			found := false
			for _, query := range queries {
				if name == query.Name {
					found = true
					break
				}
			}
			if !found {
				foundNames := []string{}
				for _, query := range queries {
					foundNames = append(foundNames, query.Name)
				}
				require.Truef(t, found, "failed to find query %d in list %#v", name, foundNames)
			}
		}
	}

	ctx := context.Background()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
	require.NoError(t, err)
	label2, err := ds.NewLabel(ctx, &fleet.Label{Name: "label2"})
	require.NoError(t, err)

	hostLabel1 := test.NewHost(t, ds, "host1", "10.0.0.1", "asdf", "host1", time.Now())
	err = ds.AddLabelsToHost(ctx, hostLabel1.ID, []uint{label1.ID})
	require.NoError(t, err)

	hostLabel2 := test.NewHost(t, ds, "host2", "10.0.0.2", "asdg", "host2", time.Now())
	err = ds.AddLabelsToHost(ctx, hostLabel2.ID, []uint{label2.ID})
	require.NoError(t, err)

	hostLabel1And2 := test.NewHost(t, ds, "host3", "10.0.0.3", "asdh", "host3", time.Now())
	err = ds.AddLabelsToHost(ctx, hostLabel1And2.ID, []uint{label1.ID, label2.ID})
	require.NoError(t, err)

	hostNoLabels := test.NewHost(t, ds, "host4", "10.0.0.4", "asdj", "host4", time.Now())

	queryLabel1, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "query1",
		Query:              "SELECT 1",
		DiscardData:        false,
		AutomationsEnabled: true,
		AuthorID:           &user.ID,
		Logging:            fleet.LoggingSnapshot,
		Interval:           10,
		Saved:              true,
		LabelsIncludeAny: []fleet.LabelIdent{
			{LabelName: label1.Name},
		},
	})
	require.NoError(t, err)

	queryLabel2, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "query2",
		Query:              "SELECT 1",
		DiscardData:        false,
		AutomationsEnabled: true,
		AuthorID:           &user.ID,
		Logging:            fleet.LoggingSnapshot,
		Interval:           10,
		Saved:              true,
		LabelsIncludeAny: []fleet.LabelIdent{
			{LabelName: label2.Name},
		},
	})
	require.NoError(t, err)

	queryLabel1And2, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "query3",
		Query:              "SELECT 1",
		DiscardData:        false,
		AutomationsEnabled: true,
		AuthorID:           &user.ID,
		Logging:            fleet.LoggingSnapshot,
		Interval:           10,
		Saved:              true,
		LabelsIncludeAny: []fleet.LabelIdent{
			{LabelName: label1.Name},
			{LabelName: label2.Name},
		},
	})
	require.NoError(t, err)

	queryNoLabel, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "query4",
		Query:              "SELECT 1",
		DiscardData:        false,
		AutomationsEnabled: true,
		AuthorID:           &user.ID,
		Logging:            fleet.LoggingSnapshot,
		Interval:           10,
		Saved:              true,
	})
	require.NoError(t, err)

	queryIncludeAllBoth, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "queryIncludeAllBoth",
		Query:              "SELECT 1",
		DiscardData:        false,
		AutomationsEnabled: true,
		AuthorID:           &user.ID,
		Logging:            fleet.LoggingSnapshot,
		Interval:           10,
		Saved:              true,
		LabelsIncludeAll: []fleet.LabelIdent{
			{LabelName: label1.Name},
			{LabelName: label2.Name},
		},
	})
	require.NoError(t, err)

	// No host specified, list all queries on team, regardless of tag.
	queries, err := ds.ListScheduledQueriesForAgents(ctx, nil, nil, false)
	require.NoError(t, err)
	requireQueries(t, queries, []string{
		queryLabel1.Name, queryLabel2.Name, queryLabel1And2.Name, queryNoLabel.Name,
		queryIncludeAllBoth.Name,
	})

	// Host with only label1: include_any{label1} matches; include_any{label2} doesn't;
	// include_any{label1,label2} matches; include_all{label1,label2} doesn't (missing label2).
	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostLabel1.ID, false)
	require.NoError(t, err)
	requireQueries(t, queries, []string{queryLabel1.Name, queryLabel1And2.Name, queryNoLabel.Name})

	// Host with only label2: same logic — include_all{both} doesn't match.
	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostLabel2.ID, false)
	require.NoError(t, err)
	requireQueries(t, queries, []string{queryLabel2.Name, queryLabel1And2.Name, queryNoLabel.Name})

	// Host with both label1 and label2: matches include_any (both), include_all{both}, no-label.
	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostLabel1And2.ID, false)
	require.NoError(t, err)
	requireQueries(t, queries, []string{
		queryLabel1.Name, queryLabel2.Name, queryLabel1And2.Name, queryNoLabel.Name,
		queryIncludeAllBoth.Name,
	})

	// Host with no labels: only the no-label query.
	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostNoLabels.ID, false)
	require.NoError(t, err)
	requireQueries(t, queries, []string{queryNoLabel.Name})
}

func testHasLabelScopedScheduledQueries(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a team.
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test-label-scoped"})
	require.NoError(t, err)

	// Create a label.
	label, err := ds.NewLabel(ctx, &fleet.Label{Name: "test-label-scoped", Query: "SELECT 1"})
	require.NoError(t, err)

	// No queries at all -- should return false.
	has, err := ds.HasLabelScopedScheduledQueries(ctx, nil, false)
	require.NoError(t, err)
	assert.False(t, has, "no queries exist, should be false")

	has, err = ds.HasLabelScopedScheduledQueries(ctx, &team.ID, false)
	require.NoError(t, err)
	assert.False(t, has, "no queries exist for team, should be false")

	// Create a global scheduled query WITHOUT labels.
	q1, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "global-no-labels",
		Query:              "SELECT 1",
		Saved:              true,
		Interval:           60,
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	has, err = ds.HasLabelScopedScheduledQueries(ctx, nil, false)
	require.NoError(t, err)
	assert.False(t, has, "global query exists but has no labels")

	// Add a label to the query.
	_, err = ds.writer(ctx).ExecContext(ctx,
		"INSERT INTO query_labels (query_id, label_id, require_all) VALUES (?, ?, 0)", q1.ID, label.ID)
	require.NoError(t, err)

	has, err = ds.HasLabelScopedScheduledQueries(ctx, nil, false)
	require.NoError(t, err)
	assert.True(t, has, "global query with label should be true for global scope")

	has, err = ds.HasLabelScopedScheduledQueries(ctx, &team.ID, false)
	require.NoError(t, err)
	assert.True(t, has, "global query with label should be true for team scope too")

	// Create a team query with labels.
	q2, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "team-with-labels",
		Query:              "SELECT 2",
		TeamID:             &team.ID,
		Saved:              true,
		Interval:           30,
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		"INSERT INTO query_labels (query_id, label_id, require_all) VALUES (?, ?, 0)", q2.ID, label.ID)
	require.NoError(t, err)

	// Remove label from the global query to isolate team test.
	_, err = ds.writer(ctx).ExecContext(ctx,
		"DELETE FROM query_labels WHERE query_id = ?", q1.ID)
	require.NoError(t, err)

	has, err = ds.HasLabelScopedScheduledQueries(ctx, nil, false)
	require.NoError(t, err)
	assert.False(t, has, "only team query has label, global scope should be false")

	has, err = ds.HasLabelScopedScheduledQueries(ctx, &team.ID, false)
	require.NoError(t, err)
	assert.True(t, has, "team query with label should be true for that team")

	// Test the automations filter: query with discard_data=true and automations_enabled=false
	// should NOT count even if it has labels.
	_, err = ds.writer(ctx).ExecContext(ctx,
		"DELETE FROM query_labels WHERE query_id = ?", q2.ID)
	require.NoError(t, err)

	q3, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "team-discarded",
		Query:              "SELECT 3",
		TeamID:             &team.ID,
		Saved:              true,
		Interval:           30,
		AutomationsEnabled: false,
		DiscardData:        true,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		"INSERT INTO query_labels (query_id, label_id, require_all) VALUES (?, ?, 0)", q3.ID, label.ID)
	require.NoError(t, err)

	has, err = ds.HasLabelScopedScheduledQueries(ctx, &team.ID, false)
	require.NoError(t, err)
	assert.False(t, has, "query with discard_data=true and automations_enabled=false should not count")

	// Test queryReportsDisabled toggle: a snapshot query with discard_data=false,
	// automations_enabled=false, and labels should count when queryReportsDisabled=false
	// but NOT when queryReportsDisabled=true (the NOT ? bind in the SQL).
	q4, err := ds.NewQuery(ctx, &fleet.Query{
		Name:               "team-snapshot-report",
		Query:              "SELECT 4",
		TeamID:             &team.ID,
		Saved:              true,
		Interval:           30,
		AutomationsEnabled: false,
		DiscardData:        false,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		"INSERT INTO query_labels (query_id, label_id, require_all) VALUES (?, ?, 0)", q4.ID, label.ID)
	require.NoError(t, err)

	has, err = ds.HasLabelScopedScheduledQueries(ctx, &team.ID, false)
	require.NoError(t, err)
	assert.True(t, has, "snapshot report with labels should count when queryReportsDisabled=false")

	has, err = ds.HasLabelScopedScheduledQueries(ctx, &team.ID, true)
	require.NoError(t, err)
	assert.False(t, has, "snapshot report with labels should NOT count when queryReportsDisabled=true")
}

func testLabelScopedScheduledQueryScopes(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Gabriel", "gabriel@fleet.co", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "etag scopes team"})
	require.NoError(t, err)

	// empty: no label-scoped scheduled queries anywhere
	got, err := ds.LabelScopedScheduledQueryScopes(ctx)
	require.NoError(t, err)
	require.False(t, got.Global)
	require.Empty(t, got.TeamIDs)

	// a plain scheduled query (no labels) does not count
	_, err = ds.NewQuery(ctx, &fleet.Query{
		Name:     "plain scheduled",
		Query:    "SELECT 1",
		AuthorID: &user.ID,
		Logging:  fleet.LoggingSnapshot,
		Interval: 10,
		Saved:    true,
	})
	require.NoError(t, err)
	// a label-scoped query that is NOT scheduled (interval 0) does not count
	label, err := ds.NewLabel(ctx, &fleet.Label{Name: "etag gate label"})
	require.NoError(t, err)
	_, err = ds.NewQuery(ctx, &fleet.Query{
		Name:             "label scoped unscheduled",
		Query:            "SELECT 1",
		AuthorID:         &user.ID,
		Logging:          fleet.LoggingSnapshot,
		Interval:         0,
		Saved:            true,
		LabelsIncludeAny: []fleet.LabelIdent{{LabelName: label.Name}},
	})
	require.NoError(t, err)
	got, err = ds.LabelScopedScheduledQueryScopes(ctx)
	require.NoError(t, err)
	require.False(t, got.Global)
	require.Empty(t, got.TeamIDs)

	// a TEAM label-scoped scheduled query flags only that team
	teamScoped, err := ds.NewQuery(ctx, &fleet.Query{
		Name:             "team label scoped scheduled",
		Query:            "SELECT 1",
		AuthorID:         &user.ID,
		Logging:          fleet.LoggingSnapshot,
		Interval:         10,
		Saved:            true,
		TeamID:           &team.ID,
		LabelsIncludeAny: []fleet.LabelIdent{{LabelName: label.Name}},
	})
	require.NoError(t, err)
	got, err = ds.LabelScopedScheduledQueryScopes(ctx)
	require.NoError(t, err)
	require.False(t, got.Global)
	require.Equal(t, []uint{team.ID}, got.TeamIDs)

	// a GLOBAL label-scoped scheduled query flags the global scope too
	globalScoped, err := ds.NewQuery(ctx, &fleet.Query{
		Name:             "global label scoped scheduled",
		Query:            "SELECT 1",
		AuthorID:         &user.ID,
		Logging:          fleet.LoggingSnapshot,
		Interval:         10,
		Saved:            true,
		LabelsIncludeAny: []fleet.LabelIdent{{LabelName: label.Name}},
	})
	require.NoError(t, err)
	got, err = ds.LabelScopedScheduledQueryScopes(ctx)
	require.NoError(t, err)
	require.True(t, got.Global)
	require.Equal(t, []uint{team.ID}, got.TeamIDs)

	// deleting them clears the answers (query_labels rows cascade)
	err = ds.DeleteQuery(ctx, globalScoped.TeamID, globalScoped.Name)
	require.NoError(t, err)
	err = ds.DeleteQuery(ctx, teamScoped.TeamID, teamScoped.Name)
	require.NoError(t, err)
	got, err = ds.LabelScopedScheduledQueryScopes(ctx)
	require.NoError(t, err)
	require.False(t, got.Global)
	require.Empty(t, got.TeamIDs)
}

// testQueryLabelsAtomic verifies that a query row and its query_labels rows are
// written in a single transaction. A scheduled query with no query_labels rows
// targets every host, so a query row that outlives a failed label write (or
// is visible before its labels are written) leaks a label-scoped query to
// out-of-scope hosts (#49502).
func testQueryLabelsAtomic(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	label1, err := ds.NewLabel(ctx, &fleet.Label{Name: "label1"})
	require.NoError(t, err)

	hostInLabel := test.NewHost(t, ds, "host1", "10.0.0.1", "asdf", "host1", time.Now())
	require.NoError(t, ds.AddLabelsToHost(ctx, hostInLabel.ID, []uint{label1.ID}))
	hostNotInLabel := test.NewHost(t, ds, "host2", "10.0.0.2", "asdg", "host2", time.Now())

	// NewQuery: a label that does not exist must roll back the query row too,
	// otherwise an unscoped (global) scheduled query would be left behind.
	_, err = ds.NewQuery(ctx, &fleet.Query{
		Name:             "scoped",
		Query:            "SELECT 1",
		AuthorID:         &user.ID,
		Logging:          fleet.LoggingSnapshot,
		Interval:         10,
		Saved:            true,
		LabelsIncludeAny: []fleet.LabelIdent{{LabelName: "does-not-exist"}},
	})
	require.Error(t, err)
	_, err = ds.QueryByName(ctx, nil, "scoped")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "query row must not persist when its labels fail to save")

	queries, err := ds.ListScheduledQueriesForAgents(ctx, nil, &hostNotInLabel.ID, false)
	require.NoError(t, err)
	require.Empty(t, queries)

	// A valid scoped query is only delivered to hosts in the label.
	scoped, err := ds.NewQuery(ctx, &fleet.Query{
		Name:             "scoped",
		Query:            "SELECT 1",
		AuthorID:         &user.ID,
		Logging:          fleet.LoggingSnapshot,
		Interval:         10,
		Saved:            true,
		LabelsIncludeAny: []fleet.LabelIdent{{LabelName: label1.Name}},
	})
	require.NoError(t, err)

	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostInLabel.ID, false)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostNotInLabel.ID, false)
	require.NoError(t, err)
	require.Empty(t, queries)

	// SaveQuery: a failed label write must roll back the column updates and
	// keep the previous labels, so the query never becomes unscoped.
	scoped.Name = "renamed"
	scoped.LabelsIncludeAny = []fleet.LabelIdent{{LabelName: "does-not-exist"}}
	err = ds.SaveQuery(ctx, scoped, false, false)
	require.Error(t, err)

	reloaded, err := ds.Query(ctx, scoped.ID)
	require.NoError(t, err)
	require.Equal(t, "scoped", reloaded.Name)
	require.Len(t, reloaded.LabelsIncludeAny, 1)
	require.Equal(t, label1.ID, reloaded.LabelsIncludeAny[0].LabelID)

	queries, err = ds.ListScheduledQueriesForAgents(ctx, nil, &hostNotInLabel.ID, false)
	require.NoError(t, err)
	require.Empty(t, queries)
}
