package service

import (
	"context"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

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
	ds := new(mock.Store)
	opts := &TestServerOpts{}
	svc, ctx := newTestService(t, ds, nil, nil, opts)
	var activities []string
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
		activities = append(activities, activity.ActivityName())
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
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

func TestCancelHostUpcomingActivityAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

	const (
		teamHostID   = 1
		globalHostID = 2
	)

	teamHost := &fleet.Host{TeamID: ptr.Uint(1), Platform: "darwin"}
	globalHost := &fleet.Host{Platform: "darwin"}

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		if hostID == teamHostID {
			return teamHost, nil
		}
		return globalHost, nil
	}
	ds.CancelHostUpcomingActivityFunc = func(ctx context.Context, hostID uint, execID string) (fleet.ActivityDetails, error) {
		return nil, nil
	}
	ds.GetHostUpcomingActivityMetaFunc = func(ctx context.Context, hostID uint, execID string) (*fleet.UpcomingActivityMeta, error) {
		return &fleet.UpcomingActivityMeta{}, nil
	}

	cases := []struct {
		name             string
		user             *fleet.User
		shouldFailGlobal bool
		shouldFailTeam   bool
	}{
		{
			name:             "global observer",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team observer",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "global observer plus",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team observer plus",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "global admin",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailGlobal: false,
			shouldFailTeam:   false,
		},
		{
			name:             "team admin",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailGlobal: true,
			shouldFailTeam:   false,
		},
		{
			name:             "global maintainer",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailGlobal: false,
			shouldFailTeam:   false,
		},
		{
			name:             "team maintainer",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailGlobal: true,
			shouldFailTeam:   false,
		},
		{
			name:             "team admin wrong team",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 42}, Role: fleet.RoleAdmin}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team maintainer wrong team",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 42}, Role: fleet.RoleMaintainer}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "global gitops",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team gitops",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.CancelHostUpcomingActivity(ctx, globalHostID, "abc")
			checkAuthErr(t, tt.shouldFailGlobal, err)
			err = svc.CancelHostUpcomingActivity(ctx, teamHostID, "abc")
			checkAuthErr(t, tt.shouldFailTeam, err)
		})
	}
}
