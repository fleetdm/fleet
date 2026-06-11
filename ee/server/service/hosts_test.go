package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func TestGetHostManagedAccountPasswordAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, baseSvc := newTestServiceWithMock(t, ds)

	teamID := uint(1)

	verified := string(fleet.MDMDeliveryVerified)

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		return &fleet.Host{ID: hostID, UUID: "test-uuid", Platform: "darwin", TeamID: &teamID}, nil
	}
	ds.GetHostManagedLocalAccountStatusFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMManagedLocalAccount, error) {
		return &fleet.HostMDMManagedLocalAccount{Status: &verified, PasswordAvailable: true}, nil
	}
	ds.GetHostManagedLocalAccountPasswordFunc = func(ctx context.Context, hostUUID string) (*fleet.HostManagedLocalAccountPassword, error) {
		return &fleet.HostManagedLocalAccountPassword{}, nil
	}
	ds.MarkManagedLocalAccountPasswordViewedFunc = func(ctx context.Context, hostUUID string) (time.Time, error) {
		return time.Now(), nil
	}
	baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			false,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			true,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			false,
		},
		{
			"team observer+, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			false,
		},
		{
			"team gitops, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			_, err := svc.GetHostManagedAccountPassword(ctx, 1)
			checkAuthErr(t, tt.shouldFail, err)
		})
	}
}
