package mysql

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledQueries(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListInPackWithStats", testScheduledQueriesListInPackWithStats},
		{"ListInPack", testScheduledQueriesListInPack},
		{"New", testScheduledQueriesNew},
		{"Get", testScheduledQueriesGet},
		{"Delete", testScheduledQueriesDelete},
		{"CascadingDelete", testScheduledQueriesCascadingDelete},
		{"ScheduledQueryIDsByName", testScheduledQueriesIDsByName},
		{"AsyncBatchSaveHostsScheduledQueryStats", testScheduledQueriesAsyncBatchSaveStats},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testScheduledQueriesListInPackWithStats(t *testing.T, ds *Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo", Logging: fleet.LoggingSnapshot},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar", Logging: fleet.LoggingDifferential},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries, nil)
	require.NoError(t, err)

	specs := []*fleet.PackSpec{
		{
			Name:    "baz",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.NoError(t, err)

	gotQueries, err := ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, gotQueries, 1)
	assert.Equal(t, uint(60), gotQueries[0].Interval)
	assert.Equal(t, "test_foo", gotQueries[0].Description)
	assert.Equal(t, "select * from foo", gotQueries[0].Query)

	specs = []*fleet.PackSpec{
		{
			Name:    "baz",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
				{
					QueryName:   queries[1].Name,
					Name:        "test bar",
					Description: "test_bar",
					Interval:    60,
				},
				{
					QueryName:   queries[1].Name,
					Name:        "test bar snapshot",
					Description: "test_bar",
					Interval:    60,
					Snapshot:    ptr.Bool(true),
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.NoError(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, gotQueries, 3)

	idWithAgg := gotQueries[0].ID

	_, err = ds.writer(context.Background()).Exec(
		`INSERT INTO aggregated_stats(id,global_stats,type,json_value) VALUES (?,?,?,?)`,
		idWithAgg, false, fleet.AggregatedStatsTypeScheduledQuery,
		`{"user_time_p50": 10.5777, "user_time_p95": 111.7308, "system_time_p50": 0.6936, "system_time_p95": 95.8654, "total_executions": 5038}`,
	)
	require.NoError(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, gotQueries, 3)

	foundAgg := false
	for _, sq := range gotQueries {
		if sq.ID == idWithAgg {
			foundAgg = true
			require.NotNil(t, sq.SystemTimeP50)
			require.NotNil(t, sq.SystemTimeP95)
			assert.Equal(t, 0.6936, *sq.SystemTimeP50)
			assert.Equal(t, 95.8654, *sq.SystemTimeP95)
		}
	}
	require.True(t, foundAgg)
}

func testScheduledQueriesListInPack(t *testing.T, ds *Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo", Logging: fleet.LoggingSnapshot},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar", Logging: fleet.LoggingSnapshot},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries, nil)
	require.NoError(t, err)

	specs := []*fleet.PackSpec{
		{
			Name:    "baz",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.NoError(t, err)

	gotQueries, err := ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, gotQueries, 1)
	assert.Equal(t, uint(60), gotQueries[0].Interval)
	assert.Equal(t, "test_foo", gotQueries[0].Description)
	assert.Equal(t, "select * from foo", gotQueries[0].Query)

	specs = []*fleet.PackSpec{
		{
			Name:    "baz",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName: queries[0].Name,
					// If Name is not specified, QueryName is used.
					Description: "test_foo",
					Interval:    60,
					Snapshot:    nil,
					Removed:     nil,
					Shard:       nil,
					Platform:    nil,
					Version:     nil,
					Denylist:    nil,
				},
				{
					QueryName:   queries[1].Name,
					Name:        "test bar",
					Description: "test_bar",
					Interval:    50,
					Snapshot:    ptr.Bool(false),
					Removed:     ptr.Bool(false),
					Shard:       ptr.Uint(1),
					Platform:    ptr.String("linux"),
					Version:     ptr.String("5.0.1"),
					Denylist:    ptr.Bool(false),
				},
				{
					QueryName:   queries[1].Name,
					Name:        "test bar snapshot",
					Description: "test_bar",
					Interval:    40,
					Snapshot:    ptr.Bool(true),
					Removed:     ptr.Bool(true),
					Shard:       ptr.Uint(2),
					Platform:    ptr.String("darwin"),
					Version:     ptr.String("5.0.0"),
					Denylist:    ptr.Bool(true),
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.NoError(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, gotQueries, 3)

	sort.Slice(gotQueries, func(i, j int) bool {
		return gotQueries[i].ID < gotQueries[j].ID
	})

	require.Equal(t, "foo", gotQueries[0].Name)
	require.Equal(t, "foo", gotQueries[0].QueryName)
	require.Equal(t, "select * from foo", gotQueries[0].Query)
	require.Equal(t, "test_foo", gotQueries[0].Description)
	require.Equal(t, uint(60), gotQueries[0].Interval)
	require.Nil(t, gotQueries[0].Snapshot)
	require.Nil(t, gotQueries[0].Removed)
	require.Nil(t, gotQueries[0].Shard)
	require.Nil(t, gotQueries[0].Platform)
	require.Nil(t, gotQueries[0].Version)
	require.Nil(t, gotQueries[0].Denylist)

	require.Equal(t, "test bar", gotQueries[1].Name)
	require.Equal(t, "bar", gotQueries[1].QueryName)
	require.Equal(t, "select baz from bar", gotQueries[1].Query)
	require.Equal(t, "test_bar", gotQueries[1].Description)
	require.Equal(t, uint(50), gotQueries[1].Interval)
	require.NotNil(t, gotQueries[1].Snapshot)
	require.False(t, *gotQueries[1].Snapshot)
	require.NotNil(t, gotQueries[1].Removed)
	require.False(t, *gotQueries[1].Removed)
	require.NotNil(t, gotQueries[1].Shard)
	require.Equal(t, uint(1), *gotQueries[1].Shard)
	require.NotNil(t, gotQueries[1].Platform)
	require.Equal(t, "linux", *gotQueries[1].Platform)
	require.NotNil(t, gotQueries[1].Version)
	require.Equal(t, "5.0.1", *gotQueries[1].Version)
	require.NotNil(t, gotQueries[1].Denylist)
	require.False(t, *gotQueries[1].Denylist)

	require.Equal(t, "test bar snapshot", gotQueries[2].Name)
	require.Equal(t, "bar", gotQueries[2].QueryName)
	require.Equal(t, "select baz from bar", gotQueries[2].Query)
	require.Equal(t, "test_bar", gotQueries[2].Description)
	require.Equal(t, uint(40), gotQueries[2].Interval)
	require.NotNil(t, gotQueries[2].Snapshot)
	require.True(t, *gotQueries[2].Snapshot)
	require.NotNil(t, gotQueries[2].Removed)
	require.True(t, *gotQueries[2].Removed)
	require.NotNil(t, gotQueries[2].Shard)
	require.Equal(t, uint(2), *gotQueries[2].Shard)
	require.NotNil(t, gotQueries[2].Platform)
	require.Equal(t, "darwin", *gotQueries[2].Platform)
	require.NotNil(t, gotQueries[2].Version)
	require.Equal(t, "5.0.0", *gotQueries[2].Version)
	require.NotNil(t, gotQueries[2].Denylist)
	require.True(t, *gotQueries[2].Denylist)
}

func testScheduledQueriesNew(t *testing.T, ds *Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, nil, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")

	query, err := ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		PackID:  p1.ID,
		QueryID: q1.ID,
		Name:    "foo-scheduled",
	})
	require.Nil(t, err)
	assert.Equal(t, "foo", query.QueryName)
	assert.Equal(t, "foo-scheduled", query.Name)
	assert.Equal(t, "select * from time;", query.Query)
}

func testScheduledQueriesGet(t *testing.T, ds *Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, nil, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "")

	query, err := ds.ScheduledQuery(context.Background(), sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)
	assert.Nil(t, query.Denylist)

	denylist := false
	query.Denylist = &denylist

	_, err = ds.SaveScheduledQuery(context.Background(), query)
	require.Nil(t, err)

	query, err = ds.ScheduledQuery(context.Background(), sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)
	require.NotNil(t, query.Denylist)
	assert.False(t, *query.Denylist)
}

func testScheduledQueriesDelete(t *testing.T, ds *Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, nil, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "")

	query, err := ds.ScheduledQuery(context.Background(), sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	err = ds.DeleteScheduledQuery(context.Background(), sq1.ID)
	require.Nil(t, err)

	_, err = ds.ScheduledQuery(context.Background(), sq1.ID)
	require.NotNil(t, err)
}

func testScheduledQueriesCascadingDelete(t *testing.T, ds *Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo", Logging: fleet.LoggingSnapshot},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar", Logging: fleet.LoggingSnapshot},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries, nil)
	require.Nil(t, err)

	specs := []*fleet.PackSpec{
		{
			Name:    "baz",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
				{
					QueryName:   queries[1].Name,
					Name:        "test bar",
					Description: "test_bar",
					Interval:    60,
				},
				{
					QueryName:   queries[1].Name,
					Name:        "test bar snapshot",
					Description: "test_bar",
					Interval:    60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(context.Background(), specs)
	require.Nil(t, err)

	gotQueries, err := ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)

	err = ds.DeleteQuery(context.Background(), nil, queries[1].Name)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 1)
}

func testScheduledQueriesIDsByName(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "User", "user@example.com", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo", Logging: fleet.LoggingSnapshot},
		{Name: "bar", Description: "do some bars", Query: "select * from bar", Logging: fleet.LoggingSnapshot},
		{Name: "foo2", Description: "get the foos", Query: "select * from foo2", Logging: fleet.LoggingSnapshot},
		{Name: "bar2", Description: "do some bars", Query: "select * from bar2", Logging: fleet.LoggingSnapshot},
	}
	err := ds.ApplyQueries(ctx, user.ID, queries, nil)
	require.NoError(t, err)

	specs := []*fleet.PackSpec{
		{
			Name:    "baz",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
				{
					QueryName:   queries[2].Name,
					Description: "test_foo2",
					Interval:    60,
				},
			},
		},
		{
			Name:    "qux",
			Targets: fleet.PackSpecTargets{Labels: []string{}},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[1].Name,
					Description: "test_bar",
					Interval:    60,
				},
				{
					QueryName:   queries[3].Name,
					Description: "test_bar2",
					Interval:    60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(ctx, specs)
	require.NoError(t, err)

	// load the scheduled query IDs as that is what we want to test
	bazPack, _, err := ds.PackByName(ctx, "baz")
	require.NoError(t, err)
	quxPack, _, err := ds.PackByName(ctx, "qux")
	require.NoError(t, err)

	sqsBaz, err := ds.ListScheduledQueriesInPack(ctx, bazPack.ID)
	require.NoError(t, err)
	require.Len(t, sqsBaz, 2)
	sqsQux, err := ds.ListScheduledQueriesInPack(ctx, quxPack.ID)
	require.NoError(t, err)
	require.Len(t, sqsQux, 2)

	const scheduledQueryIDsByNameBatchSize = 2

	// without any name
	ids, err := ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize)
	require.NoError(t, err)
	require.Len(t, ids, 0)

	// single query name
	ids, err = ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize, [2]string{"baz", "foo"})
	require.NoError(t, err)
	require.Equal(t, []uint{sqsBaz[0].ID}, ids)

	// invalid query name (mismatch pack with query)
	ids, err = ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize, [2]string{"qux", "foo"})
	require.NoError(t, err)
	require.Equal(t, []uint{0}, ids)

	// invalid query name (unknown pack)
	ids, err = ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize, [2]string{"nope", "foo"})
	require.NoError(t, err)
	require.Equal(t, []uint{0}, ids)

	// invalid query name (unknown query)
	ids, err = ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize, [2]string{"qux", "nope"})
	require.NoError(t, err)
	require.Equal(t, []uint{0}, ids)

	// multiple query names > batch size
	ids, err = ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize,
		[2]string{"qux", "nope"}, [2]string{"baz", "foo"},
		[2]string{"qux", "bar"}, [2]string{"nope", "nope"},
	)
	require.NoError(t, err)
	require.Equal(t, []uint{0, sqsBaz[0].ID, sqsQux[0].ID, 0}, ids)

	// multiple query names (many times batch size)
	ids, err = ds.ScheduledQueryIDsByName(ctx, scheduledQueryIDsByNameBatchSize,
		[2]string{"qux", "nope"}, [2]string{"baz", "foo"},
		[2]string{"qux", "bar"}, [2]string{"nope", "nope"},
		[2]string{"qux", "bar2"}, [2]string{"nope", "foo2"},
		[2]string{"qux", "foo2"}, [2]string{"baz", "foo2"},
	)
	require.NoError(t, err)
	require.Equal(t, []uint{0, sqsBaz[0].ID, sqsQux[0].ID, 0, sqsQux[1].ID, 0, 0, sqsBaz[1].ID}, ids)
}

func testScheduledQueriesAsyncBatchSaveStats(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	lastExec := time.Now() // don't care about that value in the test, it just needs to be set

	user := test.NewUser(t, ds, "user", "user@example.com", true)

	h1 := test.NewHost(t, ds, "foo1.local", "192.168.1.1", "1", "1", time.Now())
	h2 := test.NewHost(t, ds, "foo2.local", "192.168.1.2", "2", "2", time.Now())
	h3 := test.NewHost(t, ds, "foo3.local", "192.168.1.3", "3", "3", time.Now())

	p1 := test.NewPack(t, ds, "p1")
	p2 := test.NewPack(t, ds, "p2")
	p3 := test.NewPack(t, ds, "p3")

	q1 := test.NewQuery(t, ds, nil, "q1", "select 1", user.ID, true)
	q2 := test.NewQuery(t, ds, nil, "q2", "select 2", user.ID, true)
	q3 := test.NewQuery(t, ds, nil, "q3", "select 3", user.ID, true)
	q4 := test.NewQuery(t, ds, nil, "q4", "select 4", user.ID, true)

	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "sq1")
	sq2 := test.NewScheduledQuery(t, ds, p2.ID, q2.ID, 60, false, false, "sq2")
	sq3 := test.NewScheduledQuery(t, ds, p3.ID, q3.ID, 60, false, false, "sq3")
	sq4 := test.NewScheduledQuery(t, ds, p3.ID, q4.ID, 60, false, false, "sq4") // pack 3 has two scheduled queries

	assertStats := func(m map[uint][]fleet.ScheduledQueryStats) {
		// checks that the stats are as expected (only the Executions field is
		// checked for the provided host ID/scheduled query ID).
		for hid, stats := range m {
			for _, st := range stats {
				ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
					var got uint64
					err := sqlx.GetContext(ctx, tx, &got, `SELECT executions FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`, hid, st.ScheduledQueryID)
					if err != nil {
						return err
					}
					require.Equal(t, st.Executions, got)
					return nil
				})
			}
		}
	}

	const batchSize = 2
	// save without any stats
	execs, err := ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, nil, batchSize)
	require.NoError(t, err)
	require.Equal(t, 0, execs)

	// single host, single stat
	m := map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         1,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(m)

	// single host, stats == batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         2,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         3,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(m)

	// single host, stats > batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         4,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         5,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
			{
				ScheduledQueryID:   sq3.ID,
				Executions:         6,
				LastExecuted:       lastExec,
				PackName:           p3.Name,
				ScheduledQueryName: sq3.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 2, execs)
	assertStats(m)

	// multi host, stats == batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         7,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
		},
		h2.ID: {
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         8,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 1, execs)
	assertStats(m)

	// multi host, stats > batch size
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         9,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
		},
		h2.ID: {
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         10,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
			{
				ScheduledQueryID:   sq3.ID,
				Executions:         11,
				LastExecuted:       lastExec,
				PackName:           p3.Name,
				ScheduledQueryName: sq3.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 2, execs)
	assertStats(m)

	// multi host, stats > (N * batch size)
	m = map[uint][]fleet.ScheduledQueryStats{
		h1.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         12,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         13,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
		},
		h2.ID: {
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         14,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
			{
				ScheduledQueryID:   sq4.ID,
				Executions:         15,
				LastExecuted:       lastExec,
				PackName:           p3.Name,
				ScheduledQueryName: sq4.Name,
			},
		},
		h3.ID: {
			{
				ScheduledQueryID:   sq1.ID,
				Executions:         16,
				LastExecuted:       lastExec,
				PackName:           p1.Name,
				ScheduledQueryName: sq1.Name,
			},
			{
				ScheduledQueryID:   sq2.ID,
				Executions:         17,
				LastExecuted:       lastExec,
				PackName:           p2.Name,
				ScheduledQueryName: sq2.Name,
			},
			{
				ScheduledQueryID:   sq3.ID,
				Executions:         18,
				LastExecuted:       lastExec,
				PackName:           p3.Name,
				ScheduledQueryName: sq3.Name,
			},
		},
	}
	execs, err = ds.AsyncBatchSaveHostsScheduledQueryStats(ctx, m, batchSize)
	require.NoError(t, err)
	require.Equal(t, 4, execs)
	assertStats(m)
}
