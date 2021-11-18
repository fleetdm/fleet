package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledQueries(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListInPack", testScheduledQueriesListInPack},
		{"New", testScheduledQueriesNew},
		{"Get", testScheduledQueriesGet},
		{"Delete", testScheduledQueriesDelete},
		{"CascadingDelete", testScheduledQueriesCascadingDelete},
		{"CleanupOrphanStats", testScheduledQueriesCleanupOrphanStats},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testScheduledQueriesListInPack(t *testing.T, ds *Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries)
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

	gotQueries, err := ds.ListScheduledQueriesInPack(context.Background(), 1, fleet.ListOptions{})
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

	gotQueries, err = ds.ListScheduledQueriesInPack(context.Background(), 1, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, gotQueries, 3)

	idWithAgg := gotQueries[0].ID

	_, err = ds.writer.Exec(
		`INSERT INTO aggregated_stats(id,type,json_value) VALUES (?,?,?)`,
		idWithAgg, "scheduled_query", `{"user_time_p50": 10.5777, "user_time_p95": 111.7308, "system_time_p50": 0.6936, "system_time_p95": 95.8654, "total_executions": 5038}`,
	)
	require.NoError(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(context.Background(), 1, fleet.ListOptions{})
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

func testScheduledQueriesNew(t *testing.T, ds *Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
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
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
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
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
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
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	err := ds.ApplyQueries(context.Background(), zwass.ID, queries)
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

	gotQueries, err := ds.ListScheduledQueriesInPack(context.Background(), 1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)

	err = ds.DeleteQuery(context.Background(), queries[1].Name)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(context.Background(), 1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 1)
}

func testScheduledQueriesCleanupOrphanStats(t *testing.T, ds *Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	h1 := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())
	p1, err := ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "baz",
		HostIDs: []uint{h1.ID},
	})
	require.NoError(t, err)
	test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "1")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "2")

	_, err = ds.writer.Exec(`INSERT INTO scheduled_query_stats (
                                   host_id, scheduled_query_id, average_memory, denylisted,
                                   executions, schedule_interval, output_size, system_time,
                                   user_time, wall_time
                                ) VALUES (?, ?, 32, false, 4, 4, 4, 4, 4, 4);`, h1.ID, sq1.ID)
	require.NoError(t, err)

	// Cleanup doesn't remove stats that are ok
	require.NoError(t, ds.CleanupOrphanScheduledQueryStats(context.Background()))

	h1, err = ds.Host(context.Background(), h1.ID, false)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)

	// now we insert a bogus stat
	_, err = ds.writer.Exec(`INSERT INTO scheduled_query_stats (
                                   host_id, scheduled_query_id, average_memory, denylisted, executions
                               ) VALUES (?, 999, 32, false, 2);`, h1.ID)
	require.NoError(t, err)
	// and also for an unknown host
	_, err = ds.writer.Exec(`INSERT INTO scheduled_query_stats (
                                   host_id, scheduled_query_id, average_memory, denylisted, executions
                               ) VALUES (888, 999, 32, true, 4);`)
	require.NoError(t, err)

	// And we don't see it in the host
	h1, err = ds.Host(context.Background(), h1.ID, false)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)

	// but there are definitely there
	var count int
	err = ds.writer.Get(&count, `SELECT count(*) FROM scheduled_query_stats`)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// now we clean it up
	require.NoError(t, ds.CleanupOrphanScheduledQueryStats(context.Background()))

	err = ds.writer.Get(&count, `SELECT count(*) FROM scheduled_query_stats`)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	h1, err = ds.Host(context.Background(), h1.ID, false)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)
}
