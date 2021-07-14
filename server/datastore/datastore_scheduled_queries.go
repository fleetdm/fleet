package datastore

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListScheduledQueriesInPack(t *testing.T, ds fleet.Datastore) {
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

func testNewScheduledQuery(t *testing.T, ds fleet.Datastore) {
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

func testScheduledQuery(t *testing.T, ds fleet.Datastore) {
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

func testDeleteScheduledQuery(t *testing.T, ds fleet.Datastore) {
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

func testCascadingDeletionOfQueries(t *testing.T, ds fleet.Datastore) {
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

func testScheduledQueryBulkSave(t *testing.T, ds fleet.Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin@fleet.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	q2 := test.NewQuery(t, ds, "foo2", "select 1 from time;", u1.ID, true)
	q3 := test.NewQuery(t, ds, "foo3", "select 1 from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")

	sq1 := &fleet.ScheduledQuery{PackID: p1.ID, QueryID: q1.ID, Interval: 60, Name: q1.Name}
	sq2 := &fleet.ScheduledQuery{PackID: p1.ID, QueryID: q2.ID, Interval: 42, Name: q2.Name}
	sq3 := &fleet.ScheduledQuery{PackID: p1.ID, QueryID: q3.ID, Interval: 23, Name: q3.Name}

	_, err := ds.ReplaceScheduledQueriesInPack(p1.ID, []*fleet.ScheduledQuery{sq1, sq2})
	require.Nil(t, err)

	query, err := ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	// Test updates on already inserted sched queries
	sq1.Denylist = ptr.Bool(true)

	_, err = ds.ReplaceScheduledQueriesInPack(p1.ID, []*fleet.ScheduledQuery{sq1, sq2})
	require.Nil(t, err)

	query, err = ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)
	require.NotNil(t, query.Denylist)
	assert.True(t, *query.Denylist)

	query, err = ds.ScheduledQuery(sq2.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(42), query.Interval)

	// replacement
	_, err = ds.ReplaceScheduledQueriesInPack(p1.ID, []*fleet.ScheduledQuery{sq1, sq3})
	require.Nil(t, err)

	sqs, err := ds.ListScheduledQueriesInPack(p1.ID, fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, sqs, 2)

	assert.Equal(t, q1.Name, sqs[0].Name)
	assert.Equal(t, q3.Name, sqs[1].Name)
}
