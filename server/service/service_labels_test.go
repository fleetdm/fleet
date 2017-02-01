package service

import (
	"testing"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestListLabels(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	labels, err := svc.ListLabels(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, labels, 0)

	_, err = ds.NewLabel(&kolide.Label{
		Name:  "foo",
		Query: "select * from foo;",
	})
	assert.Nil(t, err)

	labels, err = svc.ListLabels(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "foo", labels[0].Name)
}

func TestGetLabel(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	label := &kolide.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err = ds.NewLabel(label)
	assert.Nil(t, err)
	assert.NotZero(t, label.ID)

	labelVerify, err := svc.GetLabel(ctx, label.ID)
	assert.Nil(t, err)
	assert.Equal(t, label.ID, labelVerify.ID)
}

func TestNewLabel(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	name := "foo"
	query := "select * from foo;"
	label, err := svc.NewLabel(ctx, kolide.LabelPayload{
		Name:  &name,
		Query: &query,
	})
	assert.NotZero(t, label.ID)

	assert.Nil(t, err)

	labels, err := ds.ListLabels(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, labels, 1)
	assert.Equal(t, "foo", labels[0].Name)
}

func TestDeleteLabel(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	label := &kolide.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err = ds.NewLabel(label)
	assert.Nil(t, err)
	assert.NotZero(t, label.ID)

	err = svc.DeleteLabel(ctx, label.ID)
	assert.Nil(t, err)

	labels, err := ds.ListLabels(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, labels, 0)
}
