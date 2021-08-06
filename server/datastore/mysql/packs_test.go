package mysql

import (
	"fmt"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletePack(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestSavePack(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	expectedPack := &fleet.Pack{
		Name:     "foo",
		HostIDs:  []uint{1},
		LabelIDs: []uint{1},
		TeamIDs:  []uint{1},
	}

	pack, err := ds.NewPack(expectedPack)
	require.NoError(t, err)
	assert.NotEqual(t, uint(0), pack.ID)
	test.EqualSkipTimestampsID(t, expectedPack, pack)

	pack, err = ds.Pack(pack.ID)
	require.NoError(t, err)
	test.EqualSkipTimestampsID(t, expectedPack, pack)

	expectedPack = &fleet.Pack{
		ID:       pack.ID,
		Name:     "bar",
		HostIDs:  []uint{3},
		LabelIDs: []uint{4, 6},
		TeamIDs:  []uint{},
	}

	err = ds.SavePack(expectedPack)
	require.NoError(t, err)

	pack, err = ds.Pack(pack.ID)
	require.NoError(t, err)
	assert.Equal(t, "bar", pack.Name)
	test.EqualSkipTimestampsID(t, expectedPack, pack)
}

func TestGetPackByName(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

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

func TestListPacks(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	p1 := &fleet.PackSpec{
		ID:   1,
		Name: "foo_pack",
	}
	p2 := &fleet.PackSpec{
		ID:   2,
		Name: "bar_pack",
	}
	err := ds.ApplyPackSpecs([]*fleet.PackSpec{p1})
	require.Nil(t, err)

	packs, err := ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 1)

	err = ds.ApplyPackSpecs([]*fleet.PackSpec{p1, p2})
	require.Nil(t, err)

	packs, err = ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 2)
}

func setupPackSpecsTest(t *testing.T, ds fleet.Datastore) []*fleet.PackSpec {
	zwass := test.NewUser(t, ds, "Zach", "zwass@example.com", true)
	queries := []*fleet.Query{
		{Name: "foo", Description: "get the foos", Query: "select * from foo"},
		{Name: "bar", Description: "do some bars", Query: "select baz from bar"},
	}
	// Zach creates some queries
	err := ds.ApplyQueries(zwass.ID, queries)
	require.Nil(t, err)

	labels := []*fleet.LabelSpec{
		{
			Name:  "foo",
			Query: "select * from foo",
		},
		{
			Name:  "bar",
			Query: "select * from bar",
		},
		{
			Name:  "bing",
			Query: "select * from bing",
		},
	}
	err = ds.ApplyLabelSpecs(labels)
	require.Nil(t, err)

	expectedSpecs := []*fleet.PackSpec{
		{
			ID:   1,
			Name: "test_pack",
			Targets: fleet.PackSpecTargets{
				Labels: []string{
					"foo",
					"bar",
					"bing",
				},
			},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Name:        "q0",
					Description: "test_foo",
					Interval:    42,
				},
				{
					QueryName: queries[0].Name,
					Name:      "foo_snapshot",
					Interval:  600,
					Snapshot:  ptr.Bool(true),
					Denylist:  ptr.Bool(false),
				},
				{
					Name:      "q2",
					QueryName: queries[1].Name,
					Interval:  600,
					Removed:   ptr.Bool(false),
					Shard:     ptr.Uint(73),
					Platform:  ptr.String("foobar"),
					Version:   ptr.String("0.0.0.0.0.1"),
					Denylist:  ptr.Bool(true),
				},
			},
		},
		{
			ID:       2,
			Name:     "test_pack_disabled",
			Disabled: true,
			Targets: fleet.PackSpecTargets{
				Labels: []string{
					"foo",
					"bar",
					"bing",
				},
			},
			Queries: []fleet.PackSpecQuery{
				{
					QueryName:   queries[0].Name,
					Name:        "q0",
					Description: "test_foo",
					Interval:    42,
				},
				{
					QueryName: queries[0].Name,
					Name:      "foo_snapshot",
					Interval:  600,
					Snapshot:  ptr.Bool(true),
				},
				{
					Name:      "q2",
					QueryName: queries[1].Name,
					Interval:  600,
					Removed:   ptr.Bool(false),
					Shard:     ptr.Uint(73),
					Platform:  ptr.String("foobar"),
					Version:   ptr.String("0.0.0.0.0.1"),
				},
			},
		},
	}

	err = ds.ApplyPackSpecs(expectedSpecs)
	require.Nil(t, err)
	return expectedSpecs
}

func TestApplyPackSpecRoundtrip(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	expectedSpecs := setupPackSpecsTest(t, ds)

	gotSpec, err := ds.GetPackSpecs()
	require.Nil(t, err)
	assert.Equal(t, expectedSpecs, gotSpec)
}

func TestGetPackSpec(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	expectedSpecs := setupPackSpecsTest(t, ds)

	for _, s := range expectedSpecs {
		spec, err := ds.GetPackSpec(s.Name)
		require.Nil(t, err)
		assert.Equal(t, s, spec)
	}
}

func TestApplyPackSpecMissingQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	// Do not define queries mentioned in spec
	specs := []*fleet.PackSpec{
		{
			ID:   1,
			Name: "test_pack",
			Targets: fleet.PackSpecTargets{
				Labels: []string{},
			},
			Queries: []fleet.PackSpecQuery{
				{
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

func TestApplyPackSpecMissingName(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	setupPackSpecsTest(t, ds)

	specs := []*fleet.PackSpec{
		{
			Name: "test2",
			Targets: fleet.PackSpecTargets{
				Labels: []string{},
			},
			Queries: []fleet.PackSpecQuery{
				{
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

func TestListPacksForHost(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	if ds.Name() == "inmem" {
		t.Skip("inmem is deprecated")
	}

	mockClock := clock.NewMockClock()

	l1 := &fleet.LabelSpec{
		ID:   1,
		Name: "foo",
	}
	l2 := &fleet.LabelSpec{
		ID:   2,
		Name: "bar",
	}
	err := ds.ApplyLabelSpecs([]*fleet.LabelSpec{l1, l2})
	require.Nil(t, err)

	p1 := &fleet.PackSpec{
		ID:   1,
		Name: "foo_pack",
		Targets: fleet.PackSpecTargets{
			Labels: []string{
				l1.Name,
				l2.Name,
			},
		},
	}
	p2 := &fleet.PackSpec{
		ID:   2,
		Name: "shmoo_pack",
		Targets: fleet.PackSpecTargets{
			Labels: []string{
				l2.Name,
			},
		},
	}
	err = ds.ApplyPackSpecs([]*fleet.PackSpec{p1, p2})
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
}

func TestEnsureGlobalPack(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	test.AddAllHostsLabel(t, ds)

	packs, err := ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 0)

	gp, err := ds.EnsureGlobalPack()
	require.Nil(t, err)

	packs, err = ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 1)
	assert.Equal(t, gp.ID, packs[0].ID)
	assert.Equal(t, "global", *gp.Type)

	labels, err := ds.LabelIDsByName([]string{"All Hosts"})
	require.Nil(t, err)

	assert.Equal(t, []uint{labels[0]}, gp.LabelIDs)

	_, err = ds.EnsureGlobalPack()
	require.Nil(t, err)

	packs, err = ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 1)
	assert.Equal(t, gp.ID, packs[0].ID)
	assert.Equal(t, "global", *gp.Type)
}

func TestEnsureTeamPack(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	packs, err := ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 0)

	_, err = ds.EnsureTeamPack(12)
	require.Error(t, err)

	team1, err := ds.NewTeam(&fleet.Team{Name: "team1"})
	require.NoError(t, err)

	tp, err := ds.EnsureTeamPack(team1.ID)
	require.NoError(t, err)

	packs, err = ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 1)
	assert.Equal(t, tp.ID, packs[0].ID)
	assert.Equal(t, fmt.Sprintf("team-%d", team1.ID), *tp.Type)
	assert.Equal(t, []uint{team1.ID}, tp.TeamIDs)

	_, err = ds.EnsureTeamPack(team1.ID)
	require.NoError(t, err)

	packs, err = ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 1)
	assert.Equal(t, tp.ID, packs[0].ID)

	team2, err := ds.NewTeam(&fleet.Team{Name: "team2"})
	require.NoError(t, err)

	tp2, err := ds.EnsureTeamPack(team2.ID)
	require.NoError(t, err)

	packs, err = ds.ListPacks(fleet.ListOptions{})
	require.Nil(t, err)
	assert.Len(t, packs, 2)
	assert.Equal(t, tp.ID, packs[0].ID)
	assert.Equal(t, tp2.ID, packs[1].ID)

	assert.Equal(t, fmt.Sprintf("team-%d", team2.ID), *tp2.Type)
	assert.Equal(t, []uint{team2.ID}, tp2.TeamIDs)
}

func TestApplyPackSpecFailsOnTargetIDNull(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	// Do not define queries mentioned in spec
	specs := []*fleet.PackSpec{
		{
			ID:   1,
			Name: "test_pack",
			Targets: fleet.PackSpecTargets{
				Labels: []string{"UnexistentLabel"},
			},
		},
	}

	// Should error due to unkown label target id
	err := ds.ApplyPackSpecs(specs)
	require.Error(t, err)
}
