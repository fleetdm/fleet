package datastore

import (
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeletePack(t *testing.T, ds kolide.Datastore) {
	pack := test.NewPack(t, ds, "foo")
	assert.NotEqual(t, uint(0), pack.ID)

	pack, err := ds.Pack(pack.ID)
	require.Nil(t, err)

	err = ds.DeletePack(pack.Name)
	assert.Nil(t, err)

	assert.NotEqual(t, uint(0), pack.ID)
	pack, err = ds.Pack(pack.ID)
	assert.NotNil(t, err)
}

func testNewPack(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}

	pack, err := ds.NewPack(pack)
	require.NoError(t, err)
	assert.NotEqual(t, uint(0), pack.ID)

	pack, err = ds.Pack(pack.ID)
	require.NoError(t, err)
	assert.Equal(t, "foo", pack.Name)
}

func testGetPackByName(t *testing.T, ds kolide.Datastore) {
	pack := test.NewPack(t, ds, "foo")
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

func testListPacks(t *testing.T, ds kolide.Datastore) {
	p1 := &kolide.PackSpec{
		ID:   1,
		Name: "foo_pack",
	}
	p2 := &kolide.PackSpec{
		ID:   2,
		Name: "bar_pack",
	}
	err := ds.ApplyPackSpecs([]*kolide.PackSpec{p1})
	require.Nil(t, err)

	packs, err := ds.ListPacks(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 1)

	err = ds.ApplyPackSpecs([]*kolide.PackSpec{p1, p2})
	require.Nil(t, err)

	packs, err = ds.ListPacks(kolide.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 2)
}

func testListHostsInPack(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is deprecated")
	}

	mockClock := clock.NewMockClock()

	l1 := kolide.LabelSpec{
		ID:   1,
		Name: "foo",
	}
	err := ds.ApplyLabelSpecs([]*kolide.LabelSpec{&l1})
	require.Nil(t, err)

	p1 := &kolide.PackSpec{
		ID:   1,
		Name: "foo_pack",
		Targets: kolide.PackSpecTargets{
			Labels: []string{
				l1.Name,
			},
		},
	}
	err = ds.ApplyPackSpecs([]*kolide.PackSpec{p1})
	require.Nil(t, err)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", mockClock.Now())

	hostsInPack, err := ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
	require.Nil(t, err)
	require.Len(t, hostsInPack, 0)

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l1.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	hostsInPack, err = ds.ListHostsInPack(p1.ID, kolide.ListOptions{})
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
}

func testAddLabelToPackTwice(t *testing.T, ds kolide.Datastore) {
	l1 := kolide.LabelSpec{
		ID:    1,
		Name:  "l1",
		Query: "select 1",
	}
	err := ds.ApplyLabelSpecs([]*kolide.LabelSpec{&l1})
	require.Nil(t, err)

	p1 := &kolide.PackSpec{
		ID:   1,
		Name: "pack1",
		Targets: kolide.PackSpecTargets{
			Labels: []string{
				l1.Name,
				l1.Name,
			},
		},
	}
	err = ds.ApplyPackSpecs([]*kolide.PackSpec{p1})
	require.NotNil(t, err)
}

func setupPackSpecsTest(t *testing.T, ds kolide.Datastore) []*kolide.PackSpec {
	zwass := test.NewUser(t, ds, "Zach", "zwass", "zwass@kolide.co", true)
	queries := []*kolide.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	// Zach creates some queries
	err := ds.ApplyQueries(zwass.ID, queries)
	require.Nil(t, err)

	labels := []*kolide.LabelSpec{
		&kolide.LabelSpec{
			Name:  "foo",
			Query: "select * from foo",
		},
		&kolide.LabelSpec{
			Name:  "bar",
			Query: "select * from bar",
		},
		&kolide.LabelSpec{
			Name:  "bing",
			Query: "select * from bing",
		},
	}
	err = ds.ApplyLabelSpecs(labels)
	require.Nil(t, err)

	boolPtr := func(b bool) *bool { return &b }
	uintPtr := func(x uint) *uint { return &x }
	stringPtr := func(s string) *string { return &s }
	expectedSpecs := []*kolide.PackSpec{
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
					Name:        "q0",
					Description: "test_foo",
					Interval:    42,
				},
				kolide.PackSpecQuery{
					QueryName: queries[0].Name,
					Name:      "foo_snapshot",
					Interval:  600,
					Snapshot:  boolPtr(true),
					Denylist:  boolPtr(false),
				},
				kolide.PackSpecQuery{
					Name:      "q2",
					QueryName: queries[1].Name,
					Interval:  600,
					Removed:   boolPtr(false),
					Shard:     uintPtr(73),
					Platform:  stringPtr("foobar"),
					Version:   stringPtr("0.0.0.0.0.1"),
					Denylist:  boolPtr(true),
				},
			},
		},
		&kolide.PackSpec{
			ID:       2,
			Name:     "test_pack_disabled",
			Disabled: true,
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
					Name:        "q0",
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
					Name:      "q2",
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

	err = ds.ApplyPackSpecs(expectedSpecs)
	require.Nil(t, err)
	return expectedSpecs
}

func testApplyPackSpecRoundtrip(t *testing.T, ds kolide.Datastore) {
	expectedSpecs := setupPackSpecsTest(t, ds)

	gotSpec, err := ds.GetPackSpecs()
	require.Nil(t, err)
	assert.Equal(t, expectedSpecs, gotSpec)
}

func testGetPackSpec(t *testing.T, ds kolide.Datastore) {
	expectedSpecs := setupPackSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetPackSpec(s.Name)
		require.Nil(t, err)
		assert.Equal(t, s, spec)
	}
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

func testApplyPackSpecMissingName(t *testing.T, ds kolide.Datastore) {
	setupPackSpecsTest(t, ds)

	specs := []*kolide.PackSpec{
		&kolide.PackSpec{
			Name: "test2",
			Targets: kolide.PackSpecTargets{
				Labels: []string{},
			},
			Queries: []kolide.PackSpecQuery{
				kolide.PackSpecQuery{
					QueryName: "foo",
					Interval:  600,
				},
			},
		},
	}
	err := ds.ApplyPackSpecs(specs)
	require.NoError(t, err)

	// Query name should have been copied into name field
	spec, err := ds.GetPackSpec("test2")
	require.NoError(t, err)
	assert.Equal(t, "foo", spec.Queries[0].Name)
}

func testListLabelsForPack(t *testing.T, ds kolide.Datastore) {
	labelSpecs := []*kolide.LabelSpec{
		&kolide.LabelSpec{
			Name:  "foo",
			Query: "select * from foo",
		},
		&kolide.LabelSpec{
			Name:  "bar",
			Query: "select * from bar",
		},
		&kolide.LabelSpec{
			Name:  "bing",
			Query: "select * from bing",
		},
	}
	err := ds.ApplyLabelSpecs(labelSpecs)
	require.Nil(t, err)

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
		},
		&kolide.PackSpec{
			ID:   2,
			Name: "test 2",
			Targets: kolide.PackSpecTargets{
				Labels: []string{
					"bing",
				},
			},
		},
		&kolide.PackSpec{
			ID:   3,
			Name: "test 3",
		},
	}
	err = ds.ApplyPackSpecs(specs)
	require.Nil(t, err)

	labels, err := ds.ListLabelsForPack(specs[0].ID)
	require.Nil(t, err)
	assert.Len(t, labels, 3)

	labels, err = ds.ListLabelsForPack(specs[1].ID)
	require.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "bing", labels[0].Name)

	labels, err = ds.ListLabelsForPack(specs[2].ID)
	require.Nil(t, err)
	assert.Len(t, labels, 0)
}

func testListPacksForHost(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is deprecated")
	}

	mockClock := clock.NewMockClock()

	l1 := &kolide.LabelSpec{
		ID:   1,
		Name: "foo",
	}
	l2 := &kolide.LabelSpec{
		ID:   2,
		Name: "bar",
	}
	err := ds.ApplyLabelSpecs([]*kolide.LabelSpec{l1, l2})
	require.Nil(t, err)

	p1 := &kolide.PackSpec{
		ID:   1,
		Name: "foo_pack",
		Targets: kolide.PackSpecTargets{
			Labels: []string{
				l1.Name,
				l2.Name,
			},
		},
	}
	p2 := &kolide.PackSpec{
		ID:   2,
		Name: "shmoo_pack",
		Targets: kolide.PackSpecTargets{
			Labels: []string{
				l2.Name,
			},
		},
	}
	err = ds.ApplyPackSpecs([]*kolide.PackSpec{p1, p2})
	require.Nil(t, err)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", mockClock.Now())

	packs, err := ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	require.Len(t, packs, 0)

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l1.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	if assert.Len(t, packs, 1) {
		assert.Equal(t, "foo_pack", packs[0].Name)
	}

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l1.ID: false, l2.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l1.ID: true, l2.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", mockClock.Now())

	err = ds.RecordLabelQueryExecutions(
		h2,
		map[uint]bool{l2.ID: true},
		mockClock.Now(),
	)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l2.ID: false},
		mockClock.Now(),
	)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	if assert.Len(t, packs, 1) {
		assert.Equal(t, "foo_pack", packs[0].Name)
	}

	// Add host directly to pack
	err = ds.AddHostToPack(h1.ID, p2.ID)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)

	// Remove label membership for both
	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l2.ID: false, l1.ID: false},
		mockClock.Now(),
	)
	require.Nil(t, err)

	err = ds.RecordLabelQueryExecutions(
		h1,
		map[uint]bool{l2.ID: false},
		mockClock.Now(),
	)
	require.Nil(t, err)
	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	if assert.Len(t, packs, 1) {
		assert.Equal(t, p2.Name, packs[0].Name)
	}

	// Now host is added directly to both packs
	err = ds.AddHostToPack(h1.ID, p1.ID)
	require.Nil(t, err)

	packs, err = ds.ListPacksForHost(h1.ID)
	require.Nil(t, err)
	assert.Len(t, packs, 2)
}
