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

func TestTeamPoliciesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.NewTeamPolicyFunc = func(ctx context.Context, teamID uint, queryID uint, resolution string) (*fleet.Policy, error) {
		return &fleet.Policy{}, nil
	}
	ds.ListTeamPoliciesFunc = func(ctx context.Context, teamID uint) ([]*fleet.Policy, error) {
		return nil, nil
	}
	ds.TeamPolicyFunc = func(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
		return nil, nil
	}
	ds.DeleteTeamPoliciesFunc = func(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, specs []*fleet.PolicySpec) error {
		return nil
	}

	var testCases = []struct {
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
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			false,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
			false,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
			false,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			false,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.NewTeamPolicy(ctx, 1, 2, "")
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ListTeamPolicies(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetTeamPolicyByIDQueries(ctx, 1, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.DeleteTeamPolicies(ctx, 1, []uint{1})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.ApplyPolicySpecs(ctx, []*fleet.PolicySpec{
				{
					QueryName: "query1",
					Team:      "team1",
				},
			})
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func checkAuthErr(t *testing.T, shouldFail bool, err error) {
	if shouldFail {
		require.Error(t, err)
		require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
	} else {
		require.NoError(t, err)
	}
}
