package service

import (
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/inmem"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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

func TestNewSavesTargets(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.NewPackFunc = func(pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
		return pack, nil
	}
	ds.NewActivityFunc = func(user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	packPayload := fleet.PackPayload{
		Name:     ptr.String("foo"),
		HostIDs:  &[]uint{123},
		LabelIDs: &[]uint{456},
		TeamIDs:  &[]uint{789},
	}
	pack, _ := svc.NewPack(test.UserContext(test.UserAdmin), packPayload)

	require.Len(t, pack.HostIDs, 1)
	require.Len(t, pack.LabelIDs, 1)
	require.Len(t, pack.TeamIDs, 1)
	assert.Equal(t, uint(123), pack.HostIDs[0])
	assert.Equal(t, uint(456), pack.LabelIDs[0])
	assert.Equal(t, uint(789), pack.TeamIDs[0])
	assert.True(t, ds.NewActivityFuncInvoked)
}
