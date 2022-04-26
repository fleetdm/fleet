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
	svc := newTestService(t, ds, nil, nil)

	globalUsers := []*fleet.User{test.UserAdmin, test.UserMaintainer, test.UserObserver}
	teamUsers := []*fleet.User{test.UserTeamAdminTeam1, test.UserTeamMaintainerTeam1, test.UserTeamObserverTeam1}

	ds.ListActivitiesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Activity, error) {
		return []*fleet.Activity{
			{ID: 1},
			{ID: 2},
		}, nil
	}

	// any global user can read activities
	for _, u := range globalUsers {
		activities, err := svc.ListActivities(test.UserContext(u), fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, activities, 2)
	}

	// team users cannot read activities
	for _, u := range teamUsers {
		_, err := svc.ListActivities(test.UserContext(u), fleet.ListOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
	}

	// user with no roles cannot read activities
	_, err := svc.ListActivities(test.UserContext(test.UserNoRoles), fleet.ListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, err = svc.ListActivities(context.Background(), fleet.ListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}
