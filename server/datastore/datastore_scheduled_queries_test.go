package datastore

import (
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListScheduledQueriesInPack(t *testing.T, ds kolide.Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)
	queries := []*kolide.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	err := ds.ApplyQueries(zwass.ID, queries)
	require.Nil(t, err)

	specs := []*kolide.PackSpec{
		&kolide.PackSpec{
			Name:    "baz",
			Targets: kolide.PackSpecTargets{Labels: []string{}},
			Queries: []kolide.PackSpecQuery{
				kolide.PackSpecQuery{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	gotQueries, err := ds.ListScheduledQueriesInPack(1, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 1)
	assert.Equal(t, uint(60), gotQueries[0].Interval)
	assert.Equal(t, "test_foo", gotQueries[0].Description)
	assert.Equal(t, "select * from foo", gotQueries[0].Query)

	boolPtr := func(b bool) *bool { return &b }
	specs = []*kolide.PackSpec{
		&kolide.PackSpec{
			Name:    "baz",
			Targets: kolide.PackSpecTargets{Labels: []string{}},
			Queries: []kolide.PackSpecQuery{
				kolide.PackSpecQuery{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
				kolide.PackSpecQuery{
					QueryName:   queries[1].Name,
					Name:        "test bar",
					Description: "test_bar",
					Interval:    60,
				},
				kolide.PackSpecQuery{
					QueryName:   queries[1].Name,
					Name:        "test bar snapshot",
					Description: "test_bar",
					Interval:    60,
					Snapshot:    boolPtr(true),
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(1, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)
}

func testNewScheduledQuery(t *testing.T, ds kolide.Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")

	query, err := ds.NewScheduledQuery(&kolide.ScheduledQuery{
		PackID:  p1.ID,
		QueryID: q1.ID,
	})
	require.Nil(t, err)
	assert.Equal(t, "foo", query.Name)
	assert.Equal(t, "select * from time;", query.Query)
}

func testScheduledQuery(t *testing.T, ds kolide.Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

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

func testDeleteScheduledQuery(t *testing.T, ds kolide.Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	query, err := ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	err = ds.DeleteScheduledQuery(sq1.ID)
	require.Nil(t, err)

	_, err = ds.ScheduledQuery(sq1.ID)
	require.NotNil(t, err)
}

func testCascadingDeletionOfQueries(t *testing.T, ds kolide.Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)
	queries := []*kolide.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	err := ds.ApplyQueries(zwass.ID, queries)
	require.Nil(t, err)

	specs := []*kolide.PackSpec{
		&kolide.PackSpec{
			Name:    "baz",
			Targets: kolide.PackSpecTargets{Labels: []string{}},
			Queries: []kolide.PackSpecQuery{
				kolide.PackSpecQuery{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    60,
				},
				kolide.PackSpecQuery{
					QueryName:   queries[1].Name,
					Name:        "test bar",
					Description: "test_bar",
					Interval:    60,
				},
				kolide.PackSpecQuery{
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

	gotQueries, err := ds.ListScheduledQueriesInPack(1, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 3)

	err = ds.DeleteQuery(queries[1].Name)
	require.Nil(t, err)

	gotQueries, err = ds.ListScheduledQueriesInPack(1, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, gotQueries, 1)

}
