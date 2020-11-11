package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/inmem"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
)

func TestGetLabel(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
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
