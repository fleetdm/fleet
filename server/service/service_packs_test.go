package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSavesTargets(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.NewPackFunc = func(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
		return pack, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
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
}
