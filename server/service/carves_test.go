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

func TestListCarves(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListCarvesFunc = func(ctx context.Context, opts fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
		return []*fleet.CarveMetadata{
			{ID: 1},
			{ID: 2},
		}, nil
	}

	// admin user
	carves, err := svc.ListCarves(test.UserContext(test.UserAdmin), fleet.CarveListOptions{})
	require.NoError(t, err)
	require.Len(t, carves, 2)

	// only global admin can read carves
	_, err = svc.ListCarves(test.UserContext(test.UserNoRoles), fleet.CarveListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.ListCarves(context.Background(), fleet.CarveListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestGetCarve(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.CarveFunc = func(ctx context.Context, id int64) (*fleet.CarveMetadata, error) {
		return &fleet.CarveMetadata{
			ID: id,
		}, nil
	}

	// admin user
	carve, err := svc.GetCarve(test.UserContext(test.UserAdmin), 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), carve.ID)

	// only global admin can read carves
	_, err = svc.GetCarve(test.UserContext(test.UserNoRoles), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.GetCarve(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}
