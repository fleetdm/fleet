package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestRotateWindowsManagedLocalAccountPassword covers the Windows branch of the rotate endpoint: it records a request
// for the next orbit config check-in rather than enqueuing an MDM command, so the only failures to map are the
// datastore's typed ones.
func TestRotateWindowsManagedLocalAccountPassword(t *testing.T) {
	t.Parallel()

	verified := string(fleet.MDMDeliveryVerified)
	admin := &fleet.User{ID: 42, GlobalRole: new(fleet.RoleAdmin)}

	setup := func(t *testing.T, platform string, acct *fleet.HostMDMManagedLocalAccount) (*mock.Store, *Service, *svcmock.Service, context.Context) {
		ds := new(mock.Store)
		svc, baseSvc := newTestServiceWithMock(t, ds)
		ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: "test-uuid", Platform: platform}, nil
		}
		ds.GetHostManagedLocalAccountStatusFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMManagedLocalAccount, error) {
			return acct, nil
		}
		ds.InitiateWindowsManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID string) error { return nil }
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error { return nil }
		return ds, svc, baseSvc, viewer.NewContext(context.Background(), viewer.Viewer{User: admin})
	}

	available := &fleet.HostMDMManagedLocalAccount{Status: &verified, PasswordAvailable: true}

	t.Run("records the request and logs the activity against the requesting user", func(t *testing.T) {
		ds, svc, baseSvc, ctx := setup(t, "windows", available)
		var actor *fleet.User
		var logged fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(_ context.Context, u *fleet.User, a fleet.ActivityDetails) error {
			actor, logged = u, a
			return nil
		}

		require.NoError(t, svc.RotateManagedLocalAccountPassword(ctx, 1))

		require.True(t, ds.InitiateWindowsManagedLocalAccountRotationFuncInvoked)
		// No MDM command is involved on Windows, so nothing should reach the Apple rotation path.
		require.False(t, ds.GetManagedLocalAccountUUIDFuncInvoked)
		rotated, ok := logged.(fleet.ActivityTypeRotatedManagedLocalAccountPassword)
		require.True(t, ok)
		assert.False(t, rotated.FleetInitiated, "a manual rotation is the user's, not Fleet's")
		require.NotNil(t, actor)
		assert.Equal(t, admin.ID, actor.ID)
	})

	t.Run("maps the datastore's typed errors to bad requests", func(t *testing.T) {
		cases := []struct {
			name    string
			dsErr   error
			message string
		}{
			{
				"already outstanding",
				fleet.ErrManagedLocalAccountRotationPending,
				"Cannot rotate managed local account password while an operation is pending.",
			},
			{
				"no windows mdm enrollment to carry the notification",
				&testNotFoundError{},
				"Host does not have MDM turned on.",
			},
			{
				"row not eligible",
				fleet.ErrManagedLocalAccountNotEligible,
				"Couldn’t rotate managed local account password. Please try again.",
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ds, svc, _, ctx := setup(t, "windows", available)
				ds.InitiateWindowsManagedLocalAccountRotationFunc = func(ctx context.Context, hostUUID string) error {
					return c.dsErr
				}

				err := svc.RotateManagedLocalAccountPassword(ctx, 1)
				require.Error(t, err)
				var badReq *fleet.BadRequestError
				require.ErrorAs(t, err, &badReq)
				assert.Equal(t, c.message, badReq.Message)
			})
		}
	})

	t.Run("rejects a platform that has no managed local account", func(t *testing.T) {
		_, svc, _, ctx := setup(t, "ubuntu", available)
		err := svc.RotateManagedLocalAccountPassword(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Equal(t, "Host is not a macOS or Windows device.", badReq.Message)
	})

	t.Run("rejects while a rotation is already in flight", func(t *testing.T) {
		ds, svc, _, ctx := setup(t, "windows", &fleet.HostMDMManagedLocalAccount{
			Status: &verified, PasswordAvailable: true, PendingRotation: true,
		})
		err := svc.RotateManagedLocalAccountPassword(ctx, 1)
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		assert.Equal(t, "Managed local account password rotation is already in progress for this host.", badReq.Message)
		assert.False(t, ds.InitiateWindowsManagedLocalAccountRotationFuncInvoked)
	})
}
