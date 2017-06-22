package datastore

import (
	"testing"

	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func testListScheduledQueriesInPack(t *testing.T, ds kolide.Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	q2 := test.NewQuery(t, ds, "bar", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")

	test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	queries, err := ds.ListScheduledQueriesInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, uint(60), queries[0].Interval)

	test.NewScheduledQuery(t, ds, p1.ID, q2.ID, 60, false, false)
	test.NewScheduledQuery(t, ds, p1.ID, q2.ID, 60, true, false)

	queries, err = ds.ListScheduledQueriesInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, queries, 3)
}

func testSaveScheduledQuery(t *testing.T, ds kolide.Datastore) {
	u1 := test.NewUser(t, ds, "Admin", "admin", "admin@kolide.co", true)
	q1 := test.NewQuery(t, ds, "foo", "select * from time;", u1.ID, true)
	p1 := test.NewPack(t, ds, "baz")
	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)

	query, err := ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(60), query.Interval)

	query.Interval = uint(120)
	query, err = ds.SaveScheduledQuery(query)
	require.Nil(t, err)
	assert.Equal(t, uint(120), query.Interval)

	queryVerify, err := ds.ScheduledQuery(sq1.ID)
	require.Nil(t, err)
	assert.Equal(t, uint(120), queryVerify.Interval)
}

func testScheduledQueryWithDeletedPack(t *testing.T, ds kolide.Datastore) {
	// When a pack is soft-deleted, it should not appear in the list of
	// packs associated with a query.

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)
	query := test.NewQuery(t, ds, "q1", "select 1", user.ID, true)
	pack := test.NewPack(t, ds, "foobar_pack")
	test.NewScheduledQuery(t, ds, pack.ID, query.ID, 60, false, false)

	actual, err := ds.Query(query.ID)
	require.Nil(t, err)
	assert.Equal(t, "q1", actual.Name)
	assert.Equal(t, "select 1", actual.Query)
	assert.Equal(t, []kolide.Pack{*pack}, actual.Packs)

	require.Nil(t, ds.DeletePack(pack.ID))

	actual, err = ds.Query(query.ID)
	require.Nil(t, err)
	assert.Equal(t, "q1", actual.Name)
	assert.Equal(t, "select 1", actual.Query)
	assert.Empty(t, actual.Packs)
}
