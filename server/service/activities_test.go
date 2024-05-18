package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestListActivities(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	globalUsers := []*fleet.User{test.UserAdmin, test.UserMaintainer, test.UserObserver, test.UserObserverPlus}
	teamUsers := []*fleet.User{test.UserTeamAdminTeam1, test.UserTeamMaintainerTeam1, test.UserTeamObserverTeam1}

	ds.ListActivitiesFunc = func(ctx context.Context, opts fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
		return []*fleet.Activity{
			{ID: 1},
			{ID: 2},
		}, nil, nil
	}

	// any global user can read activities
	for _, u := range globalUsers {
		activities, _, err := svc.ListActivities(test.UserContext(ctx, u), fleet.ListActivitiesOptions{})
		require.NoError(t, err)
		require.Len(t, activities, 2)
	}

	// team users cannot read activities
	for _, u := range teamUsers {
		_, _, err := svc.ListActivities(test.UserContext(ctx, u), fleet.ListActivitiesOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
	}

	// user with no roles cannot read activities
	_, _, err := svc.ListActivities(test.UserContext(ctx, test.UserNoRoles), fleet.ListActivitiesOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, _, err = svc.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func Test_logRoleChangeActivities(t *testing.T) {
	tests := []struct {
		name             string
		oldRole          *string
		newRole          *string
		oldTeamRoles     map[uint]string
		newTeamRoles     map[uint]string
		expectActivities []string
	}{
		{
			name: "Empty",
		}, {
			name:             "AddGlobal",
			newRole:          ptr.String("role"),
			expectActivities: []string{"changed_user_global_role"},
		}, {
			name:             "NoChangeGlobal",
			oldRole:          ptr.String("role"),
			newRole:          ptr.String("role"),
			expectActivities: []string{},
		}, {
			name:             "ChangeGlobal",
			oldRole:          ptr.String("old"),
			newRole:          ptr.String("role"),
			expectActivities: []string{"changed_user_global_role"},
		}, {
			name:             "Delete",
			oldRole:          ptr.String("old"),
			newRole:          nil,
			expectActivities: []string{"deleted_user_global_role"},
		}, {
			name:    "SwitchGlobalToTeams",
			oldRole: ptr.String("old"),
			newTeamRoles: map[uint]string{
				1: "foo",
				2: "bar",
				3: "baz",
			},
			expectActivities: []string{"deleted_user_global_role", "changed_user_team_role", "changed_user_team_role", "changed_user_team_role"},
		}, {
			name: "DeleteModifyTeam",
			oldTeamRoles: map[uint]string{
				1: "foo",
				2: "bar",
				3: "baz",
			},
			newTeamRoles: map[uint]string{
				2: "newRole",
				3: "baz",
			},
			expectActivities: []string{"changed_user_team_role", "deleted_user_team_role"},
		},
	}
	ctx := context.Background()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	var activities []string
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		activities = append(activities, activity.ActivityName())
		return nil
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activities = activities[:0]
			oldTeams := make([]fleet.UserTeam, 0, len(tt.oldTeamRoles))
			for id, r := range tt.oldTeamRoles {
				oldTeams = append(oldTeams, fleet.UserTeam{
					Team: fleet.Team{ID: id},
					Role: r,
				})
			}
			newTeams := make([]fleet.UserTeam, 0, len(tt.newTeamRoles))
			for id, r := range tt.newTeamRoles {
				newTeams = append(newTeams, fleet.UserTeam{
					Team: fleet.Team{ID: id},
					Role: r,
				})
			}
			newUser := &fleet.User{
				GlobalRole: tt.newRole,
				Teams:      newTeams,
			}
			require.NoError(t, fleet.LogRoleChangeActivities(ctx, svc, &fleet.User{}, tt.oldRole, oldTeams, newUser))
			require.Equal(t, tt.expectActivities, activities)
		})
	}
}
