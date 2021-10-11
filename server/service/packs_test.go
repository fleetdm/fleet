package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestGetPack(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.PackFunc = func(ctx context.Context, id uint) (*fleet.Pack, error) {
		return &fleet.Pack{
			ID:      1,
			TeamIDs: []uint{1},
		}, nil
	}

	pack, err := svc.GetPack(test.UserContext(test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), pack.ID)

	_, err = svc.GetPack(test.UserContext(test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}
