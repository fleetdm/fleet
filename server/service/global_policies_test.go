package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCheckPolicySpecAuthorization(t *testing.T) {
	t.Run("when team not found", func(t *testing.T) {
		ds := new(mock.Store)
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			return nil, &notFoundError{}
		}

		svc, ctx := newTestService(t, ds, nil, nil)

		req := []*fleet.PolicySpec{
			{
				Team: "some_team",
			},
		}

		user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

		actual := svc.ApplyPolicySpecs(ctx, req)
		var expected fleet.NotFoundError

		require.ErrorAs(t, actual, &expected)
	})
}

func TestGlobalPoliciesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewGlobalPolicyFunc = func(ctx context.Context, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
		return &fleet.Policy{}, nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) {
		return nil, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		return nil, nil
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID: id,
			},
		}, nil
	}
	ds.DeleteGlobalPoliciesFunc = func(ctx context.Context, ids []uint) ([]uint, error) {
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		return nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, shouldDeleteAll bool, removePolicyStats bool) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
					Enable: false,
				},
			},
		}, nil
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
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.NewGlobalPolicy(ctx, fleet.PolicyPayload{
				Name:  "query1",
				Query: "select 1;",
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ListGlobalPolicies(ctx, fleet.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetPolicyByIDQueries(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyGlobalPolicy(ctx, 1, fleet.ModifyPolicyPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteGlobalPolicies(ctx, []uint{1})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.ApplyPolicySpecs(ctx, []*fleet.PolicySpec{
				{
					Name:  "query2",
					Query: "select 1;",
				},
			})
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestRemoveGlobalPoliciesFromWebhookConfig(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{ds: ds}

	var storedAppConfig fleet.AppConfig

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &storedAppConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		storedAppConfig = *info
		return nil
	}

	for _, tc := range []struct {
		name     string
		currCfg  []uint
		toDelete []uint
		expCfg   []uint
	}{
		{
			name:     "delete-one",
			currCfg:  []uint{1},
			toDelete: []uint{1},
			expCfg:   []uint{},
		},
		{
			name:     "delete-all-2",
			currCfg:  []uint{1, 2, 3},
			toDelete: []uint{1, 2, 3},
			expCfg:   []uint{},
		},
		{
			name:     "basic",
			currCfg:  []uint{1, 2, 3},
			toDelete: []uint{1, 2},
			expCfg:   []uint{3},
		},
		{
			name:     "empty-cfg",
			currCfg:  []uint{},
			toDelete: []uint{1},
			expCfg:   []uint{},
		},
		{
			name:     "no-deletion-cfg",
			currCfg:  []uint{1},
			toDelete: []uint{2, 3, 4},
			expCfg:   []uint{1},
		},
		{
			name:     "no-deletion-cfg-2",
			currCfg:  []uint{1},
			toDelete: []uint{},
			expCfg:   []uint{1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			storedAppConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs = tc.currCfg
			err := svc.removeGlobalPoliciesFromWebhookConfig(context.Background(), tc.toDelete)
			require.NoError(t, err)
			require.Equal(t, tc.expCfg, storedAppConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
		})
	}
}

// test ApplyPolicySpecsReturnsErrorOnDuplicatePolicyNamesInSpecs
func TestApplyPolicySpecsReturnsErrorOnDuplicatePolicyNamesInSpecs(t *testing.T) {
	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return nil, &notFoundError{}
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	req := []*fleet.PolicySpec{
		{
			Name:     "query1",
			Query:    "select 1;",
			Platform: "windows",
		},
		{
			Name:     "query1",
			Query:    "select 1;",
			Platform: "windows",
		},
	}

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	err := svc.ApplyPolicySpecs(ctx, req)

	badRequestError := &fleet.BadRequestError{}
	require.ErrorAs(t, err, &badRequestError)
	require.Equal(t, "duplicate policy names not allowed", badRequestError.Message)
}
