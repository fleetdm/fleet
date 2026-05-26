package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestResetPolicyStatusAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium},
	})

	const policyTeamID = uint(2)

	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		teamID := policyTeamID
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{ID: id, TeamID: &teamID},
		}, nil
	}
	ds.ClearPolicyRunsFunc = func(ctx context.Context, policyID uint) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
	}{
		{
			"global admin can reset team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer can reset team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer cannot reset team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
		},
		{
			"team admin of the policy's team can reset it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: policyTeamID}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team maintainer of the policy's team can reset it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: policyTeamID}, Role: fleet.RoleMaintainer}}},
			false,
		},
		{
			"team observer of the policy's team cannot reset it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: policyTeamID}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"team admin of a different team cannot reset it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tCtx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			err := svc.ResetPolicyStatus(tCtx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestResetPolicyStatusGlobalPolicy(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium},
	})

	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{PolicyData: fleet.PolicyData{ID: id}}, nil
	}
	adminCtx := viewer.NewContext(ctx, viewer.Viewer{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	})

	t.Run("no runs returns not found", func(t *testing.T) {
		ds.ClearPolicyRunsFunc = func(ctx context.Context, policyID uint) error {
			return &notFoundError{}
		}
		err := svc.ResetPolicyStatus(adminCtx, 1)
		require.True(t, fleet.IsNotFound(err))
	})

	t.Run("runs exist clears successfully", func(t *testing.T) {
		ds.ClearPolicyRunsFuncInvoked = false
		ds.ClearPolicyRunsFunc = func(ctx context.Context, policyID uint) error {
			return nil
		}
		err := svc.ResetPolicyStatus(adminCtx, 1)
		require.NoError(t, err)
		require.True(t, ds.ClearPolicyRunsFuncInvoked)
	})
}

func TestResetPolicyStatusNonPremium(t *testing.T) {
	ds := new(mock.Store)
	// Default license is TierFree.
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{PolicyData: fleet.PolicyData{ID: id}}, nil
	}
	ds.ClearPolicyRunsFunc = func(ctx context.Context, policyID uint) error {
		return nil
	}

	adminCtx := viewer.NewContext(ctx, viewer.Viewer{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	})
	err := svc.ResetPolicyStatus(adminCtx, 1)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)
	require.False(t, ds.ClearPolicyRunsFuncInvoked)
}
