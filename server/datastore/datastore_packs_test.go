package datastore

import (
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/test"
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

	mockClock := clock.NewMockClock()

	p1, err := ds.NewPack(&kolide.Pack{
		Name: "foo",
	})
	require.Nil(t, err)

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

func testApplyPackSpecRoundtrip(t *testing.T, ds kolide.Datastore) {
	zwass := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)
	queries := []*kolide.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	// Zach creates some queries
	err := ds.ApplyQueries(zwass.ID, queries)
	require.Nil(t, err)

	test.NewLabel(t, ds, "foo", "select * from foo")
	test.NewLabel(t, ds, "bar", "select * from bar")
	test.NewLabel(t, ds, "bing", "select * from bing")

	boolPtr := func(b bool) *bool { return &b }
	uintPtr := func(x uint) *uint { return &x }
	stringPtr := func(s string) *string { return &s }
	specs := []*kolide.PackSpec{
		&kolide.PackSpec{
			ID:   1,
			Name: "test_pack",
			Targets: kolide.PackSpecTargets{
				Labels: []string{
					"foo",
					"bar",
					"bing",
				},
			},
			Queries: []kolide.PackSpecQuery{
				kolide.PackSpecQuery{
					QueryName:   queries[0].Name,
					Description: "test_foo",
					Interval:    42,
				},
				kolide.PackSpecQuery{
					QueryName: queries[0].Name,
					Name:      "foo_snapshot",
					Interval:  600,
					Snapshot:  boolPtr(true),
				},
				kolide.PackSpecQuery{
					QueryName: queries[1].Name,
					Interval:  600,
					Removed:   boolPtr(false),
					Shard:     uintPtr(73),
					Platform:  stringPtr("foobar"),
					Version:   stringPtr("0.0.0.0.0.1"),
				},
			},
		},
	}

	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	gotSpec, err := ds.GetPackSpecs()

	require.Nil(t, err)
	assert.Equal(t, specs, gotSpec)
}

func testApplyPackSpecMissingQueries(t *testing.T, ds kolide.Datastore) {
	// Do not define queries mentioned in spec

	specs := []*kolide.PackSpec{
		&kolide.PackSpec{
			ID:   1,
			Name: "test_pack",
			Targets: kolide.PackSpecTargets{
				Labels: []string{},
			},
			Queries: []kolide.PackSpecQuery{
				kolide.PackSpecQuery{
					QueryName: "bar",
					Interval:  600,
				},
			},
		},
	}

	// Should error due to unkown query
	err := ds.ApplyPackSpecs(specs)
	if assert.NotNil(t, err) {
		assert.Contains(t, err.Error(), "unknown query 'bar'")
	}
}
