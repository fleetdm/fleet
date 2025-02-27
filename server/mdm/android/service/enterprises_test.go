package service

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestEnterprisesAuth(t *testing.T) {
	proxy := android_mock.Proxy{}
	proxy.InitCommonMocks()
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	fleetDS := InitCommonDSMocks()
	svc, err := NewServiceWithProxy(logger, fleetDS, &proxy)
	require.NoError(t, err)

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			true,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			true,
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.GetEnterprise(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			err = svc.DeleteEnterprise(ctx)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.EnterpriseSignup(ctx)
			checkAuthErr(t, tt.shouldFailWrite, err)

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			_, err = svc.EnterpriseSignupSSE(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

		})
	}

	t.Run("unauthorized", func(t *testing.T) {
		err := svc.EnterpriseSignupCallback(context.Background(), 1, "token")
		checkAuthErr(t, false, err)
	})
}

func checkAuthErr(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	if shouldFail {
		require.Error(t, err)
		var forbiddenError *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenError)
	} else {
		require.NoError(t, err)
	}
}

func InitCommonDSMocks() *mock.Store {
	fleetDS := mock.Store{}
	ds := android_mock.Datastore{}
	ds.InitCommonMocks()

	fleetDS.GetAndroidDSFunc = func() android.Datastore {
		return &ds
	}
	fleetDS.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	fleetDS.SetAndroidEnabledAndConfiguredFunc = func(_ context.Context, configured bool) error {
		return nil
	}
	return &fleetDS
}
