package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestOrgLogoAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool // PUT and DELETE
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
		},
		{
			"team gitops",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"user without roles",
			&fleet.User{ID: 777},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			authedCtx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.UploadOrgLogo(authedCtx, fleet.OrgLogoModeLight, bytes.NewReader([]byte{}))
			checkOrgLogoAuth(t, tt.shouldFailWrite, err)

			err = svc.DeleteOrgLogo(authedCtx, fleet.OrgLogoModeLight)
			checkOrgLogoAuth(t, tt.shouldFailWrite, err)

			// GET is public — never an authz failure regardless of viewer.
			_, _, err = svc.GetOrgLogo(authedCtx, fleet.OrgLogoModeLight)
			checkOrgLogoAuth(t, false, err)
		})
	}

	// GET should also work without any viewer in the context (login page
	// case). It may still fail downstream because no store is wired, but
	// that's not an authz failure.
	t.Run("public GET without viewer", func(t *testing.T) {
		_, _, err := svc.GetOrgLogo(ctx, fleet.OrgLogoModeLight)
		checkOrgLogoAuth(t, false, err)
	})
}

func checkOrgLogoAuth(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	var forbidden *authz.Forbidden
	if shouldFail {
		require.Error(t, err)
		require.ErrorAs(t, err, &forbidden, "expected authz Forbidden, got %T: %v", err, err)
		return
	}
	if err != nil {
		require.NotErrorAs(t, err, &forbidden,
			"expected non-authz error, got authz Forbidden: %v", err)
	}
}
