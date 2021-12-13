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

	ds.NewTeamPolicyFunc = func(ctx context.Context, teamID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:     1,
				TeamID: ptr.Uint(1),
			},
		}, nil
	}
	ds.ListTeamPoliciesFunc = func(ctx context.Context, teamID uint) ([]*fleet.Policy, error) {
		return nil, nil
	}
	ds.TeamPolicyFunc = func(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
		return nil, nil
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		if id == 1 {
			return &fleet.Policy{
				PolicyData: fleet.PolicyData{
					ID:     1,
					TeamID: ptr.Uint(1),
				},
			}, nil
		}
		return nil, nil
	}
	ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy) error {
		return nil
	}
	ds.DeleteTeamPoliciesFunc = func(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

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
			"team observer, and team admin of another team",
			&fleet.User{Teams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleObserver,
				},
				{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleAdmin,
				},
			}},
			true,
			false,
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

			_, err := svc.NewTeamPolicy(ctx, 1, fleet.PolicyPayload{
				Name:  "query1",
				Query: "select 1;",
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ListTeamPolicies(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetTeamPolicyByIDQueries(ctx, 1, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyTeamPolicy(ctx, 1, 1, fleet.ModifyPolicyPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteTeamPolicies(ctx, 1, []uint{1})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.ApplyPolicySpecs(ctx, []*fleet.PolicySpec{
				{
					Name:  "query1",
					Query: "select 1;",
					Team:  "team1",
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
