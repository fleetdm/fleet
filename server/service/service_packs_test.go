package service

import (
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestListPacks(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	queries, err := svc.ListPacks(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(&kolide.Pack{
		Name: "foo",
	})
	assert.Nil(t, err)

	queries, err = svc.ListPacks(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestGetPack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	pack := &kolide.Pack{
		Name: "foo",
	}
	_, err = ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotZero(t, pack.ID)

	packVerify, err := svc.GetPack(ctx, pack.ID)
	assert.Nil(t, err)

	assert.Equal(t, pack.ID, packVerify.ID)
}

func TestNewPack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	labelName := "label"
	labelQuery := "select 1"
	label, err := svc.NewLabel(ctx, kolide.LabelPayload{
		Name:  &labelName,
		Query: &labelQuery,
	})

	packName := "foo"
	packLabelIDs := []uint{label.ID}
	pack, err := svc.NewPack(ctx, kolide.PackPayload{
		Name:     &packName,
		LabelIDs: &packLabelIDs,
	})

	assert.Nil(t, err)

	packs, err := ds.ListPacks(kolide.ListOptions{})
	assert.Nil(t, err)
	require.Len(t, packs, 1)
	assert.Equal(t, pack.ID, packs[0].ID)

	labels, err := ds.ListLabelsForPack(pack.ID)
	assert.Nil(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, label.ID, labels[0].ID)
}

func TestModifyPack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	label := &kolide.Label{
		Name:  "label",
		Query: "select 1",
	}
	label, err = ds.NewLabel(label)
	assert.Nil(t, err)
	assert.NotZero(t, label.ID)

	pack := &kolide.Pack{
		Name: "foo",
	}
	pack, err = ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotZero(t, pack.ID)

	newName := "bar"
	labelIDs := []uint{label.ID}
	packVerify, err := svc.ModifyPack(ctx, pack.ID, kolide.PackPayload{
		Name:     &newName,
		LabelIDs: &labelIDs,
	})
	assert.Nil(t, err)

	assert.Equal(t, pack.ID, packVerify.ID)
	assert.Equal(t, "bar", packVerify.Name)

	labels, err := ds.ListLabelsForPack(pack.ID)
	assert.Nil(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, label.ID, labels[0].ID)

	newLabelIDs := []uint{}
	packVerify2, err := svc.ModifyPack(ctx, pack.ID, kolide.PackPayload{
		LabelIDs: &newLabelIDs,
	})
	assert.Nil(t, err)

	assert.Equal(t, pack.ID, packVerify2.ID)

	labels, err = ds.ListLabelsForPack(pack.ID)
	assert.Nil(t, err)
	require.Len(t, labels, 0)
}

func TestDeletePack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	pack := &kolide.Pack{
		Name: "foo",
	}
	_, err = ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotZero(t, pack.ID)

	err = svc.DeletePack(ctx, pack.ID)
	assert.Nil(t, err)

	queries, err := ds.ListPacks(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)
}

func TestListPacksForHost(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	mockClock := clock.NewMockClock()

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	h1 := test.NewHost(t, ds, "h1", "10.10.10.1", "1", "1", mockClock.Now())
	h2 := test.NewHost(t, ds, "h2", "10.10.10.2", "2", "2", mockClock.Now())

	p1 := test.NewPack(t, ds, "p1")
	p2 := test.NewPack(t, ds, "p2")

	require.Nil(t, svc.AddHostToPack(ctx, h1.ID, p1.ID))
	require.Nil(t, svc.AddHostToPack(ctx, h2.ID, p1.ID))

	require.Nil(t, svc.AddHostToPack(ctx, h1.ID, p2.ID))

	{
		packs, err := svc.ListPacksForHost(ctx, h1.ID)
		require.Nil(t, err)
		require.Len(t, packs, 2)
	}
	{
		packs, err := svc.ListPacksForHost(ctx, h2.ID)
		require.Nil(t, err)
		require.Len(t, packs, 1)
	}

}
