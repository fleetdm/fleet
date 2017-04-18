package datastore

import (
	"fmt"
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/test"
	"github.com/patrickmn/sortutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeleteQuery(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	query := &kolide.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: user.ID,
	}
	query, err := ds.NewQuery(query)
	require.Nil(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, query.ID, 0)

	err = ds.DeleteQuery(query.ID)
	require.Nil(t, err)

	assert.NotEqual(t, query.ID, 0)
	_, err = ds.Query(query.ID)
	assert.NotNil(t, err)
}

func testGetQueryByName(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)
	test.NewQuery(t, ds, "q1", "select * from time", user.ID, true)
	actual, ok, err := ds.QueryByName("q1")
	require.Nil(t, err)
	assert.True(t, ok)
	assert.Equal(t, "q1", actual.Name)
	assert.Equal(t, "select * from time", actual.Query)

	actual, ok, err = ds.QueryByName("xxx")
	assert.Nil(t, err)
	assert.False(t, ok)
}

func testDeleteQueries(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	q1 := test.NewQuery(t, ds, "q1", "select * from time", user.ID, true)
	q2 := test.NewQuery(t, ds, "q2", "select * from processes", user.ID, true)
	q3 := test.NewQuery(t, ds, "q3", "select 1", user.ID, true)
	q4 := test.NewQuery(t, ds, "q4", "select * from osquery_info", user.ID, true)

	queries, err := ds.ListQueries(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 4)

	deleted, err := ds.DeleteQueries([]uint{q1.ID, q3.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(2), deleted)

	queries, err = ds.ListQueries(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 2)

	deleted, err = ds.DeleteQueries([]uint{q2.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, err = ds.ListQueries(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 1)

	deleted, err = ds.DeleteQueries([]uint{q2.ID, q4.ID})
	require.Nil(t, err)
	assert.Equal(t, uint(1), deleted)

	queries, err = ds.ListQueries(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, queries, 0)

}

func testSaveQuery(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	query := &kolide.Query{
		Name:     "foo",
		Query:    "bar",
		AuthorID: user.ID,
	}
	query, err := ds.NewQuery(query)
	require.Nil(t, err)
	require.NotNil(t, query)
	assert.NotEqual(t, 0, query.ID)

	query.Query = "baz"
	err = ds.SaveQuery(query)

	require.Nil(t, err)

	queryVerify, err := ds.Query(query.ID)
	require.Nil(t, err)
	require.NotNil(t, queryVerify)
	assert.Equal(t, "baz", queryVerify.Query)
	assert.Equal(t, "Zach", queryVerify.AuthorName)
}

func testListQuery(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	for i := 0; i < 10; i++ {
		_, err := ds.NewQuery(&kolide.Query{
			Name:     fmt.Sprintf("name%02d", i),
			Query:    fmt.Sprintf("query%02d", i),
			Saved:    true,
			AuthorID: user.ID,
		})
		require.Nil(t, err)
	}

	// One unsaved query should not be returned
	_, err := ds.NewQuery(&kolide.Query{
		Name:     "unsaved",
		Query:    "select * from time",
		Saved:    false,
		AuthorID: user.ID,
	})
	require.Nil(t, err)

	opts := kolide.ListOptions{}
	results, err := ds.ListQueries(opts)
	assert.Nil(t, err)
	assert.Equal(t, 10, len(results))
}

func checkPacks(t *testing.T, expected []kolide.Pack, actual []kolide.Pack) {
	sortutil.AscByField(expected, "ID")
	sortutil.AscByField(actual, "ID")
	assert.Equal(t, expected, actual)
}

func testLoadPacksForQueries(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	q1 := test.NewQuery(t, ds, "q1", "select * from time", user.ID, true)
	q2 := test.NewQuery(t, ds, "q2", "select * from osquery_info", user.ID, true)

	p1 := test.NewPack(t, ds, "p1")
	p2 := test.NewPack(t, ds, "p2")
	p3 := test.NewPack(t, ds, "p3")

	var err error

	test.NewScheduledQuery(t, ds, p2.ID, q1.ID, 60, false, false)

	q1, err = ds.Query(q1.ID)
	require.Nil(t, err)
	q2, err = ds.Query(q2.ID)
	require.Nil(t, err)
	checkPacks(t, []kolide.Pack{*p2}, q1.Packs)
	checkPacks(t, []kolide.Pack{}, q2.Packs)

	test.NewScheduledQuery(t, ds, p1.ID, q2.ID, 60, false, false)
	test.NewScheduledQuery(t, ds, p3.ID, q2.ID, 60, false, false)

	q1, err = ds.Query(q1.ID)
	require.Nil(t, err)
	q2, err = ds.Query(q2.ID)
	require.Nil(t, err)
	checkPacks(t, []kolide.Pack{*p2}, q1.Packs)
	checkPacks(t, []kolide.Pack{*p1, *p3}, q2.Packs)

	test.NewScheduledQuery(t, ds, p3.ID, q1.ID, 60, false, false)

	q1, err = ds.Query(q1.ID)
	require.Nil(t, err)
	q2, err = ds.Query(q2.ID)
	require.Nil(t, err)
	checkPacks(t, []kolide.Pack{*p2, *p3}, q1.Packs)
	checkPacks(t, []kolide.Pack{*p1, *p3}, q2.Packs)
}

func testDuplicateNewQuery(t *testing.T, ds kolide.Datastore) {
	user := test.NewUser(t, ds, "Mike Arpaia", "marpaia", "mike@kolide.co", true)
	q1, err := ds.NewQuery(&kolide.Query{
		Name:     "foo",
		Query:    "select * from time;",
		AuthorID: user.ID,
	})
	require.Nil(t, err)
	assert.NotZero(t, q1.ID)

	_, err = ds.NewQuery(&kolide.Query{
		Name:  "foo",
		Query: "select * from osquery_info;",
	})

	// Note that we can't do the actual type assertion here because existsError
	// is private to the individual datastore implementations
	assert.Contains(t, err.Error(), "already exists in the datastore")
}
