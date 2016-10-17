package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeletePack(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}
	err := ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotEqual(t, uint(0), pack.ID)

	pack, err = ds.Pack(pack.ID)
	require.Nil(t, err)

	err = ds.DeletePack(pack.ID)
	assert.Nil(t, err)

	assert.NotEqual(t, uint(0), pack.ID)
	pack, err = ds.Pack(pack.ID)
	assert.NotNil(t, err)
}

func testAddAndRemoveQueryFromPack(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}
	err := ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotEqual(t, uint(0), pack.ID)

	q1 := &kolide.Query{
		Name:  "bar",
		Query: "bar",
	}
	q1, err = ds.NewQuery(q1)
	assert.Nil(t, err)
	assert.NotEqual(t, uint(0), q1.ID)

	err = ds.AddQueryToPack(q1.ID, pack.ID)
	assert.Nil(t, err)

	q2 := &kolide.Query{
		Name:  "baz",
		Query: "baz",
	}
	q2, err = ds.NewQuery(q2)
	assert.Nil(t, err)
	assert.NotEqual(t, uint(0), q2.ID)

	assert.NotEqual(t, q1.ID, q2.ID)

	err = ds.AddQueryToPack(q2.ID, pack.ID)
	assert.Nil(t, err)

	queries, err := ds.ListQueriesInPack(pack)
	assert.Nil(t, err)
	assert.Len(t, queries, 2)

	err = ds.RemoveQueryFromPack(q1, pack)
	assert.Nil(t, err)

	queries, err = ds.ListQueriesInPack(pack)
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}
