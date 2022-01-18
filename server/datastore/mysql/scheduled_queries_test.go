package mysql

import (
	"context"
	"sort"
	"testing"

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
		{"ListInPackWithStats", testScheduledQueriesListInPackWithStats},
		{"ListInPack", testScheduledQueriesListInPack},
		{"New", testScheduledQueriesNew},
		{"Get", testScheduledQueriesGet},
		{"Delete", testScheduledQueriesDelete},
		{"CascadingDelete", testScheduledQueriesCascadingDelete},
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

	_, err = ds.writer.Exec(
		`INSERT INTO aggregated_stats(id,type,json_value) VALUES (?,?,?)`,
		idWithAgg, "scheduled_query", `{"user_time_p50": 10.5777, "user_time_p95": 111.7308, "system_time_p50": 0.6936, "system_time_p95": 95.8654, "total_executions": 5038}`,
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

	gotQueries, err := ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)

	err = ds.DeleteQuery(context.Background(), queries[1].Name)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPackWithStats(context.Background(), 1, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 1)
}
