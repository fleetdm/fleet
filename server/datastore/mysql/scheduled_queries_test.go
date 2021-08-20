package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListScheduledQueriesInPack(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	err := ds.ApplyQueries(zwass.ID, queries)
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
			},
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	gotQueries, err := ds.ListScheduledQueriesInPack(1, fleet.ListOptions{})
	require.Nil(t, err)
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
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)
}

func TestNewScheduledQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")

	query, err := ds.NewScheduledQuery(&fleet.ScheduledQuery{
		PackID:  p1.ID,
		QueryID: q1.ID,
		Name:    "foo-scheduled",
	})
	require.Nil(t, err)
	assert.Equal(t, "foo", query.QueryName)
	assert.Equal(t, "foo-scheduled", query.Name)
	assert.Equal(t, "select * from time;", query.Query)
}

func TestScheduledQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "")

	query, err := ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)
	assert.Nil(t, query.Denylist)

	denylist := false
	query.Denylist = &denylist

	_, err = ds.SaveScheduledQuery(query)
	require.Nil(t, err)

	query, err = ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)
	require.NotNil(t, query.Denylist)
	assert.False(t, *query.Denylist)
}

func TestDeleteScheduledQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "")

	query, err := ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	err = ds.DeleteScheduledQuery(sq1.ID)
	require.Nil(t, err)

	_, err = ds.ScheduledQuery(sq1.ID)
	require.NotNil(t, err)
}

func TestCascadingDeletionOfQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	err := ds.ApplyQueries(zwass.ID, queries)
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
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	gotQueries, err := ds.ListScheduledQueriesInPack(1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)

	err = ds.DeleteQuery(queries[1].Name)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 1)
}

func TestCleanupOrphanScheduledQueryStats(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	h1 := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())
	test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "1")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "2")

	_, err := ds.db.Exec(`INSERT INTO scheduled_query_stats (
                                   host_id, scheduled_query_id, average_memory, denylisted, 
                                   executions, schedule_interval, output_size, system_time, 
                                   user_time, wall_time
                                ) VALUES (?, ?, 32, false, 4, 4, 4, 4, 4, 4);`, h1.ID, sq1.ID)
	require.NoError(t, err)

	// Cleanup doesn't remove stats that are ok
	require.NoError(t, ds.CleanupOrphanScheduledQueryStats())

	h1, err = ds.Host(h1.ID)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)

	// now we insert a bogus stat
	_, err = ds.db.Exec(`INSERT INTO scheduled_query_stats (
                                   host_id, scheduled_query_id, average_memory, denylisted, executions
                               ) VALUES (?, 999, 32, false, 2);`, h1.ID)
	require.NoError(t, err)
	// and also for an unknown host
	_, err = ds.db.Exec(`INSERT INTO scheduled_query_stats (
                                   host_id, scheduled_query_id, average_memory, denylisted, executions
                               ) VALUES (888, 999, 32, true, 4);`)
	require.NoError(t, err)

	// And we don't see it in the host
	h1, err = ds.Host(h1.ID)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)

	// but there are definitely there
	var count int
	err = ds.db.Get(&count, `SELECT count(*) FROM scheduled_query_stats`)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// now we clean it up
	require.NoError(t, ds.CleanupOrphanScheduledQueryStats())

	err = ds.db.Get(&count, `SELECT count(*) FROM scheduled_query_stats`)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	h1, err = ds.Host(h1.ID)
	require.NoError(t, err)
	require.Len(t, h1.PackStats, 1)
}
