package datastore

import (
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/test"
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

func testGetPackByName(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}
	_, err := ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotEqual(t, uint(0), pack.ID)

	pack, ok, err := ds.PackByName(pack.Name)
	require.Nil(t, err)
	assert.True(t, ok)
	assert.NotNil(t, pack)
	assert.Equal(t, "foo", pack.Name)

	pack, ok, err = ds.PackByName("bar")
	require.Nil(t, err)
	assert.False(t, ok)
	assert.Nil(t, pack)

}

func testGetHostsInPack(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is deprecated")
	}

	user := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)

	mockClock := clock.NewMockClock()

	p1, err := ds.NewPack(&kolide.Pack{
		Name: "foo",
	})
	require.Nil(t, err)

	q1, err := ds.NewQuery(&kolide.Query{
		Name:     "foo",
		Query:    "foo",
		AuthorID: user.ID,
	})
	require.Nil(t, err)

	q2, err := ds.NewQuery(&kolide.Query{
		Name:     "bar",
		Query:    "bar",
		AuthorID: user.ID,
	})
	require.Nil(t, err)

	test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false)
	test.NewScheduledQuery(t, ds, p1.ID, q2.ID, 60, false, false)

	l1, err := ds.NewLabel(&kolide.Label{
		Name: "foo",
	})
	require.Nil(t, err)

	err = ds.AddLabelToPack(l1.ID, p1.ID)
	require.Nil(t, err)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", mockClock.Now())

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l1.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	hostsInPack, err := ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hostsInPack, 1)

	explicitHostsInPack, err := ds.ListExplicitHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, explicitHostsInPack, 0)

	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", mockClock.Now())

	err = ds.RecordLabelQueryExecutions(
		h2,
		map[uint]bool{l1.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	hostsInPack, err = ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hostsInPack, 2)

	h3 := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", mockClock.Now())

	err = ds.AddHostToPack(h3.ID, p1.ID)
	require.Nil(t, err)

	hostsInPack, err = ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hostsInPack, 3)

	explicitHostsInPack, err = ds.ListExplicitHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, explicitHostsInPack, 1)
}

func testAddLabelToPackTwice(t *testing.T, ds kolide.Datastore) {
	l1 := test.NewLabel(t, ds, "l1", "select 1;")
	p1 := test.NewPack(t, ds, "p1")

	err := ds.AddLabelToPack(l1.ID, p1.ID)
	assert.Nil(t, err)

	labels, err := ds.ListLabelsForPack(p1.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)

	err = ds.AddLabelToPack(l1.ID, p1.ID)
	assert.Nil(t, err)

	labels, err = ds.ListLabelsForPack(p1.ID)
	assert.Nil(t, err)
	assert.Len(t, labels, 1)
}
