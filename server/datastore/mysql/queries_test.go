package mysql

import (
	"fmt"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	test.AddAllHostsLabel(t, ds)

	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	groob := test.NewUser(t, ds, "Victor", "victor@fleet.co", true)
	expectedQueries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo", ObserverCanRun: true},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}

	// Zach creates some queries
	err := ds.ApplyQueries(zwass.ID, expectedQueries)
	require.Nil(t, err)

	queries, err := ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, len(expectedQueries))
	for i, q := range queries {
		comp := expectedQueries[i]
		assert.Equal(t, comp.Name, q.Name)
		assert.Equal(t, comp.Description, q.Description)
		assert.Equal(t, comp.Query, q.Query)
		assert.Equal(t, &zwass.ID, q.AuthorID)
		assert.Equal(t, comp.ObserverCanRun, q.ObserverCanRun)
	}

	// Victor modifies a query (but also pushes the same version of the
	// first query)
	expectedQueries[1].Query = "not really a valid query ;)"
	err = ds.ApplyQueries(groob.ID, expectedQueries)
	require.Nil(t, err)

	queries, err = ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, len(expectedQueries))
	for i, q := range queries {
		comp := expectedQueries[i]
		assert.Equal(t, comp.Name, q.Name)
		assert.Equal(t, comp.Description, q.Description)
		assert.Equal(t, comp.Query, q.Query)
		assert.Equal(t, &groob.ID, q.AuthorID)
	}

	// Zach adds a third query (but does not re-apply the others)
	expectedQueries = append(expectedQueries,
		&fleet.Query{Name: "trouble", Description: "Look out!", Query: "select * from time"},
	)
	err = ds.ApplyQueries(zwass.ID, []*fleet.Query{expectedQueries[2]})
	require.Nil(t, err)

	queries, err = ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, len(expectedQueries))
	for i, q := range queries {
		comp := expectedQueries[i]
		assert.Equal(t, comp.Name, q.Name)
		assert.Equal(t, comp.Description, q.Description)
		assert.Equal(t, comp.Query, q.Query)
	}
	assert.Equal(t, &groob.ID, queries[0].AuthorID)
	assert.Equal(t, &groob.ID, queries[1].AuthorID)
	assert.Equal(t, &zwass.ID, queries[2].AuthorID)
}

func TestDeleteQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
	}
	query, err := ds.NewQuery(query)
	require.Nil(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, query.ID, 0)

	err = ds.DeleteQuery(query.Name)
	require.Nil(t, err)

	assert.NotEqual(t, query.ID, 0)
	_, err = ds.Query(query.ID)
	assert.NotNil(t, err)
}

func TestGetQueryByName(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	test.NewQuery(t, ds, "q1", "select * from time", user.ID, true)
	actual, err := ds.QueryByName("q1")
	require.Nil(t, err)
	assert.Equal(t, "q1", actual.Name)
	assert.Equal(t, "select * from time", actual.Query)

	actual, err = ds.QueryByName("xxx")
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func TestDeleteQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	q1 := test.NewQuery(t, ds, "q1", "select * from time", user.ID, true)
	q2 := test.NewQuery(t, ds, "q2", "select * from processes", user.ID, true)
	q3 := test.NewQuery(t, ds, "q3", "select 1", user.ID, true)
	q4 := test.NewQuery(t, ds, "q4", "select * from osquery_info", user.ID, true)

	queries, err := ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 4)

	deleted, err := ds.DeleteQueries([]uint{q1.ID, q3.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(2), deleted)

	queries, err = ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 2)

	deleted, err = ds.DeleteQueries([]uint{q2.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, err = ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 1)

	deleted, err = ds.DeleteQueries([]uint{q2.ID, q4.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, err = ds.ListQueries(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 0)

}

func TestSaveQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	query := &fleet.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: &user.ID,
	}
	query, err := ds.NewQuery(query)
	require.Nil(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, 0, query.ID)

	query.Query = "baz"
	query.ObserverCanRun = true
	err = ds.SaveQuery(query)

	require.Nil(t, err)

	queryVerify, err := ds.Query(query.ID)
	require.Nil(t, err)
	require.NotNil(t, queryVerify)
	assert.Equal(t, "baz", queryVerify.Query)
	assert.Equal(t, "Zach", queryVerify.AuthorName)
	assert.True(t, queryVerify.ObserverCanRun)
}

func TestListQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)

	for i := 0; i < 10; i++ {
		_, err := ds.NewQuery(&fleet.Query{
			Name:     fmt.Sprintf("name%02d", i),
			Query:    fmt.Sprintf("query%02d", i),
			Saved:    true,
			AuthorID: &user.ID,
		})
		require.Nil(t, err)
	}

	// One unsaved query should not be returned
	_, err := ds.NewQuery(&fleet.Query{
		Name:     "unsaved",
		Query:    "select * from time",
		Saved:    false,
		AuthorID: &user.ID,
	})
	require.Nil(t, err)

	opts := fleet.ListOptions{}
	results, err := ds.ListQueries(opts)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(results))
}

func TestLoadPacksForQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	zwass := test.NewUser(t, ds, "Zach", "zwass@fleet.co", true)
	queries := []*fleet.Query{
		{Name: "q1", Query: "select * from time"},
		{Name: "q2", Query: "select * from osquery_info"},
	}
	err := ds.ApplyQueries(zwass.ID, queries)
	require.Nil(t, err)

	specs := []*fleet.PackSpec{
		&fleet.PackSpec{Name: "p1"},
		&fleet.PackSpec{Name: "p2"},
		&fleet.PackSpec{Name: "p3"},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	q0, err := ds.QueryByName(queries[0].Name)
	require.Nil(t, err)
	assert.Empty(t, q0.Packs)

	q1, err := ds.QueryByName(queries[1].Name)
	require.Nil(t, err)
	assert.Empty(t, q1.Packs)

	specs = []*fleet.PackSpec{
		&fleet.PackSpec{
			Name: "p2",
			Queries: []fleet.PackSpecQuery{
				fleet.PackSpecQuery{
					Name:      "q0",
					QueryName: queries[0].Name,
					Interval:  60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	q0, err = ds.QueryByName(queries[0].Name)
	require.Nil(t, err)
	if assert.Len(t, q0.Packs, 1) {
		assert.Equal(t, "p2", q0.Packs[0].Name)
	}

	q1, err = ds.QueryByName(queries[1].Name)
	require.Nil(t, err)
	assert.Empty(t, q1.Packs)

	specs = []*fleet.PackSpec{
		&fleet.PackSpec{
			Name: "p1",
			Queries: []fleet.PackSpecQuery{
				fleet.PackSpecQuery{
					QueryName: queries[1].Name,
					Interval:  60,
				},
			},
		},
		&fleet.PackSpec{
			Name: "p3",
			Queries: []fleet.PackSpecQuery{
				fleet.PackSpecQuery{
					QueryName: queries[1].Name,
					Interval:  60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	q0, err = ds.QueryByName(queries[0].Name)
	require.Nil(t, err)
	if assert.Len(t, q0.Packs, 1) {
		assert.Equal(t, "p2", q0.Packs[0].Name)
	}

	q1, err = ds.QueryByName(queries[1].Name)
	require.Nil(t, err)
	if assert.Len(t, q1.Packs, 2) {
		sort.Slice(q1.Packs, func(i, j int) bool { return q1.Packs[i].Name < q1.Packs[j].Name })
		assert.Equal(t, "p1", q1.Packs[0].Name)
		assert.Equal(t, "p3", q1.Packs[1].Name)
	}

	specs = []*fleet.PackSpec{
		&fleet.PackSpec{
			Name: "p3",
			Queries: []fleet.PackSpecQuery{
				fleet.PackSpecQuery{
					Name:      "q0",
					QueryName: queries[0].Name,
					Interval:  60,
				},
				fleet.PackSpecQuery{
					Name:      "q1",
					QueryName: queries[1].Name,
					Interval:  60,
				},
			},
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	q0, err = ds.QueryByName(queries[0].Name)
	require.Nil(t, err)
	if assert.Len(t, q0.Packs, 2) {
		sort.Slice(q0.Packs, func(i, j int) bool { return q0.Packs[i].Name < q0.Packs[j].Name })
		assert.Equal(t, "p2", q0.Packs[0].Name)
		assert.Equal(t, "p3", q0.Packs[1].Name)
	}

	q1, err = ds.QueryByName(queries[1].Name)
	require.Nil(t, err)
	if assert.Len(t, q1.Packs, 2) {
		sort.Slice(q1.Packs, func(i, j int) bool { return q1.Packs[i].Name < q1.Packs[j].Name })
		assert.Equal(t, "p1", q1.Packs[0].Name)
		assert.Equal(t, "p3", q1.Packs[1].Name)
	}
}

func TestDuplicateNewQuery(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	user := test.NewUser(t, ds, "Mike Arpaia", "mike@fleet.co", true)
	q1, err := ds.NewQuery(&fleet.Query{
		Name:     "foo",
		Query:    "select * from time;",
		AuthorID: &user.ID,
	})
	require.Nil(t, err)
	assert.NotZero(t, q1.ID)

	_, err = ds.NewQuery(&fleet.Query{
		Name:  "foo",
		Query: "select * from osquery_info;",
	})

	// Note that we can't do the actual type assertion here because existsError
	// is private to the individual datastore implementations
	assert.Contains(t, err.Error(), "already exists")
}
