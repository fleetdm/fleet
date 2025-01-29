package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestTeamPoliciesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewTeamPolicyFunc = func(ctx context.Context, teamID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:     1,
				TeamID: ptr.Uint(1),
			},
		}, nil
	}
	ds.ListTeamPoliciesFunc = func(ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions) (tpol, ipol []*fleet.Policy, err error) {
		return nil, nil, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		return nil, nil
	}
	ds.TeamPolicyFunc = func(ctx context.Context, teamID uint, policyID uint) (*fleet.Policy, error) {
		return &fleet.Policy{}, nil
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
	ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, shouldDeleteAll bool, removePolicyStats bool) error {
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
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: 1}, nil
	}
	ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{}, nil
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
			false,
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
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.NewTeamPolicy(ctx, 1, fleet.NewTeamPolicyPayload{
				Name:  "query1",
				Query: "select 1;",
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, _, err = svc.ListTeamPolicies(ctx, 1, fleet.ListOptions{}, fleet.ListOptions{}, false)
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

func TestTeamPolicyVPPAutomationRejectsNonMacOS(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	appID := fleet.VPPAppID{AdamID: "123456", Platform: fleet.IOSPlatform}
	ds.TeamExistsFunc = func(ctx context.Context, id uint) (bool, error) {
		return true, nil
	}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{
			AppStoreApp: &fleet.VPPAppStoreApp{
				VPPAppID: appID,
			},
		}, nil
	}

	_, err := svc.NewTeamPolicy(ctx, 1, fleet.NewTeamPolicyPayload{
		Name:            "query1",
		Query:           "select 1;",
		SoftwareTitleID: ptr.Uint(123),
	})
	require.ErrorContains(t, err, "is associated to an iOS or iPadOS VPP app")
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
