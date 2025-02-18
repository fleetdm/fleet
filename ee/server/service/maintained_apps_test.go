package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestListMaintainedAppsAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ListAvailableFleetMaintainedAppsFunc = func(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
		return []fleet.MaintainedApp{}, &fleet.PaginationMetadata{}, nil
	}
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{authz: authorizer, ds: ds}

	testCases := []struct {
		name                        string
		user                        *fleet.User
		shouldFailWithNoTeam        bool
		shouldFailWithMatchingTeam  bool
		shouldFailWithDifferentTeam bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
			false,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
			false,
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
		},
	}

	var forbiddenError *authz.Forbidden
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, _, err := svc.ListFleetMaintainedApps(ctx, nil, fleet.ListOptions{})
			if tt.shouldFailWithNoTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, _, err = svc.ListFleetMaintainedApps(ctx, ptr.Uint(1), fleet.ListOptions{})
			if tt.shouldFailWithMatchingTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, _, err = svc.ListFleetMaintainedApps(ctx, ptr.Uint(2), fleet.ListOptions{})
			if tt.shouldFailWithDifferentTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetMaintainedAppAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{}, nil
	}
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{authz: authorizer, ds: ds}

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
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
	}

	var forbiddenError *authz.Forbidden
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			_, err := svc.GetFleetMaintainedApp(ctx, 123)

			if tt.shouldFail {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
