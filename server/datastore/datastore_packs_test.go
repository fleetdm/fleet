package datastore

import (
	"fmt"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeletePack(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}
	_, err := ds.NewPack(pack)
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
	_, err := ds.NewPack(pack)
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

func testGetHostsInPack(t *testing.T, ds kolide.Datastore) {
	mockClock := clock.NewMockClock()

	p1, err := ds.NewPack(&kolide.Pack{
		Name: "foo",
	})
	require.Nil(t, err)

	q1, err := ds.NewQuery(&kolide.Query{
		Name:  "foo",
		Query: "foo",
	})
	require.Nil(t, err)

	q2, err := ds.NewQuery(&kolide.Query{
		Name:  "bar",
		Query: "bar",
	})
	require.Nil(t, err)

	err = ds.AddQueryToPack(q1.ID, p1.ID)
	require.Nil(t, err)

	err = ds.AddQueryToPack(q2.ID, p1.ID)
	require.Nil(t, err)

	l1, err := ds.NewLabel(&kolide.Label{
		Name: "foo",
	})
	require.Nil(t, err)

	err = ds.AddLabelToPack(l1.ID, p1.ID)
	require.Nil(t, err)

	h1, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: mockClock.Now(),
		HostName:         "foobar.local",
		OsqueryHostID:    "1",
		NodeKey:          "1",
		UUID:             "1",
	})
	require.Nil(t, err)

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[string]bool{fmt.Sprintf("%d", l1.ID): true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	hostsInPack, err := ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hostsInPack, 1)

	h2, err := ds.NewHost(&kolide.Host{
		DetailUpdateTime: mockClock.Now(),
		HostName:         "foobaz.local",
		OsqueryHostID:    "2",
		NodeKey:          "2",
		UUID:             "2",
	})
	require.Nil(t, err)

	err = ds.RecordLabelQueryExecutions(
		h2,
		map[string]bool{fmt.Sprintf("%d", l1.ID): true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	hostsInPack, err = ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hostsInPack, 2)
}
