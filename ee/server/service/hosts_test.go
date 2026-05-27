package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

// notFoundErr is a minimal error type that satisfies fleet.IsNotFound (used by
// tests below to simulate a missing host_mdm_apple_enrollment_permissions row).
type notFoundErr struct{}

func (notFoundErr) Error() string    { return "not found" }
func (notFoundErr) IsNotFound() bool { return true }

func TestEffectiveAppleAccessRights(t *testing.T) {
	const hostID = uint(42)
	const hostUUID = "uuid-42"
	mkHost := func(teamID *uint) *fleet.Host {
		return &fleet.Host{ID: hostID, UUID: hostUUID, TeamID: teamID, Platform: "darwin"}
	}

	t.Run("global config, all allowed, no stored row -> ceiling = all", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
		ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMApplePermissions, error) {
			require.Equal(t, hostUUID, uuid)
			return nil, notFoundErr{}
		}
		got, err := s.effectiveAppleAccessRights(t.Context(), mkHost(nil))
		require.NoError(t, err)
		require.Equal(t, apple_mdm.MDMAccessRightAll, got)
	})

	t.Run("global config narrows wipe, no stored row -> ceiling = no-wipe", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.AllowBYODWipe = optjson.SetBool(false)
			ac.MDM.AllowBYODLock = optjson.SetBool(true)
			return ac, nil
		}
		ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMApplePermissions, error) {
			return nil, notFoundErr{}
		}
		got, err := s.effectiveAppleAccessRights(t.Context(), mkHost(nil))
		require.NoError(t, err)
		require.Equal(t, apple_mdm.AppleEnrollmentAccessRights(false, true), got)
	})

	t.Run("team config + stored rights: returns AND", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		tid := uint(7)
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, tid, teamID)
			return &fleet.TeamMDM{AllowBYODWipe: true, AllowBYODLock: false}, nil // ceiling no-lock
		}
		ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMApplePermissions, error) {
			return &fleet.HostMDMApplePermissions{HostUUID: uuid, AccessRights: apple_mdm.MDMAccessRightAll}, nil
		}
		got, err := s.effectiveAppleAccessRights(t.Context(), mkHost(&tid))
		require.NoError(t, err)
		require.Equal(t, apple_mdm.AppleEnrollmentAccessRights(true, false), got, "stored=all AND ceiling=no-lock = no-lock")
	})

	t.Run("monotonic narrowing: stored already lacks wipe, fleet allows both -> still no-wipe", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
		ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMApplePermissions, error) {
			return &fleet.HostMDMApplePermissions{HostUUID: uuid, AccessRights: apple_mdm.AppleEnrollmentAccessRights(false, true)}, nil
		}
		got, err := s.effectiveAppleAccessRights(t.Context(), mkHost(nil))
		require.NoError(t, err)
		require.Equal(t, apple_mdm.AppleEnrollmentAccessRights(false, true), got, "stored=no-wipe AND ceiling=all = no-wipe; cannot widen")
	})

	t.Run("AppConfig error propagates", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		boom := errors.New("appconfig boom")
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return nil, boom }
		_, err := s.effectiveAppleAccessRights(t.Context(), mkHost(nil))
		require.ErrorIs(t, err, boom)
	})

	t.Run("TeamMDMConfig error propagates", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		tid := uint(7)
		boom := errors.New("team boom")
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) { return nil, boom }
		_, err := s.effectiveAppleAccessRights(t.Context(), mkHost(&tid))
		require.ErrorIs(t, err, boom)
	})

	t.Run("permissions DB error (not NotFound) propagates", func(t *testing.T) {
		ds := new(mock.Store)
		s := &Service{ds: ds}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
		boom := errors.New("perms boom")
		ds.GetHostMDMAppleEnrollmentPermissionsFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMApplePermissions, error) {
			return nil, boom
		}
		_, err := s.effectiveAppleAccessRights(t.Context(), mkHost(nil))
		require.ErrorIs(t, err, boom)
	})
}

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
