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

func TestListActivities(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListActivitiesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Activity, error) {
		return []*fleet.Activity{
			{ID: 1},
			{ID: 2},
		}, nil
	}

	// admin user
	activities, err := svc.ListActivities(test.UserContext(test.UserAdmin), fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 2)

	// anyone can read activities
	activities, err = svc.ListActivities(test.UserContext(test.UserNoRoles), fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 2)

	// no user in context
	_, err = svc.ListActivities(context.Background(), fleet.ListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}
