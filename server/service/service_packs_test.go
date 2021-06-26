package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/inmem"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
)

func TestListPacks(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc := newTestService(ds, nil, nil)

	queries, err := svc.ListPacks(test.UserContext(test.UserAdmin), fleet.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(&fleet.Pack{
		Name: "foo",
	})
	assert.Nil(t, err)

	queries, err = svc.ListPacks(test.UserContext(test.UserAdmin), fleet.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestGetPack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc := newTestService(ds, nil, nil)

	pack := &fleet.Pack{
		Name: "foo",
	}
	_, err = ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotZero(t, pack.ID)

	packVerify, err := svc.GetPack(test.UserContext(test.UserAdmin), pack.ID)
	assert.Nil(t, err)

	assert.Equal(t, pack.ID, packVerify.ID)
}
