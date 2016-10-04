package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func testDeleteQuery(t *testing.T, ds kolide.Datastore) {
	query := &kolide.Query{
		Name:  "foo",
		Query: "bar",
	}
	query, err := ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotEqual(t, query.ID, 0)

	err = ds.DeleteQuery(query)
	assert.Nil(t, err)

	assert.NotEqual(t, query.ID, 0)
	_, err = ds.Query(query.ID)
	assert.NotNil(t, err)
}

func testSaveQuery(t *testing.T, ds kolide.Datastore) {
	query := &kolide.Query{
		Name:  "foo",
		Query: "bar",
	}
	query, err := ds.NewQuery(query)
	assert.Nil(t, err)
	assert.NotEqual(t, 0, query.ID)

	query.Query = "baz"
	err = ds.SaveQuery(query)

	assert.Nil(t, err)

	queryVerify, err := ds.Query(query.ID)
	assert.Nil(t, err)
	assert.Equal(t, "baz", queryVerify.Query)
}
