package service

import (
	"context"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
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

			_, err = svc.GetPolicyByID(ctx, 1)
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

// TestGetPolicyByIDCrossTeamAuth verifies that the global "get policy
// by ID" endpoint refuses to return a team policy to a user who has no role
// on that team. This guards against the regression described in the
// "Cross-Team Policy Data Exposure" disclosure.
func TestGetPolicyByIDCrossTeamAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// The fetched policy belongs to team 2.
	const policyTeamID = uint(2)
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		teamID := policyTeamID
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:     id,
				TeamID: &teamID,
			},
		}, nil
	}

	testCases := []struct {
		name           string
		user           *fleet.User
		shouldFailRead bool
	}{
		{
			"global admin can read any team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global observer can read any team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
		},
		{
			"team observer of the policy's team can read it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: policyTeamID}, Role: fleet.RoleObserver}}},
			false,
		},
		{
			"team gitops of the policy's team can read it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: policyTeamID}, Role: fleet.RoleGitOps}}},
			false,
		},
		{
			"team observer of a different team cannot read it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"team admin of a different team cannot read it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team gitops of a different team cannot read it",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			_, err := svc.GetPolicyByID(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

// TestGetPolicyByIDGlobalPolicyAuth verifies authorization for reading a
// global policy (TeamID == nil) via the "get policy by ID" endpoint.
func TestGetPolicyByIDGlobalPolicyAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// The fetched policy is global (TeamID is nil).
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:     id,
				TeamID: nil,
			},
		}, nil
	}

	testCases := []struct {
		name           string
		user           *fleet.User
		shouldFailRead bool
	}{
		{
			"global admin can read a global policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer can read a global policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer can read a global policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
		},
		{
			"global gitops can read a global policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
		},
		{
			"team admin can read a global policy",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team observer can read a global policy",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			false,
		},
		{
			"team gitops cannot read a global policy",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"user with no role cannot read a global policy",
			&fleet.User{ID: 999},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			_, err := svc.GetPolicyByID(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

// TestGetPolicyByIDNoTeamPolicyAuth verifies authorization for reading a
// "No team" policy (TeamID == 0) via the "get policy by ID" endpoint. Team-only
// users without a role on team 0 must not be able to read it.
func TestGetPolicyByIDNoTeamPolicyAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// The fetched policy belongs to "No team" (TeamID == 0).
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:     id,
				TeamID: ptr.Uint(0),
			},
		}, nil
	}

	testCases := []struct {
		name           string
		user           *fleet.User
		shouldFailRead bool
	}{
		{
			"global admin can read a no-team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer can read a no-team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer can read a no-team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
		},
		{
			"global gitops can read a no-team policy",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
		},
		{
			"team admin of a regular team cannot read a no-team policy",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team observer of a regular team cannot read a no-team policy",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"user with no role cannot read a no-team policy",
			&fleet.User{ID: 999},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			_, err := svc.GetPolicyByID(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)
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

func TestApplyPolicySpecsLabelsValidation(t *testing.T) {
	ds := new(mock.Store)
	ds.NewGlobalPolicyFunc = func(ctx context.Context, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
		return &fleet.Policy{}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		return nil
	}
	ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
		labels := make(map[string]*fleet.Label, len(names))
		for _, name := range names {
			if name == "foo" {
				labels["foo"] = &fleet.Label{
					Name: "foo",
					ID:   1,
				}
			}
		}
		return labels, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

	testAdmin := fleet.User{
		ID:         1,
		Teams:      []fleet.UserTeam{},
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})

	// Test that a query spec with a label that exists doesn't return an error
	err := svc.ApplyPolicySpecs(viewerCtx, []*fleet.PolicySpec{
		{
			Name:             "test query",
			Query:            "select 1",
			LabelsIncludeAny: []string{"foo"},
			Platform:         "darwin,windows",
		},
	})
	// Check that no error is returned
	require.NoError(t, err)

	// Test that a query spec with a label that doesn't exist returns an error.
	err = svc.ApplyPolicySpecs(viewerCtx, []*fleet.PolicySpec{
		{
			Name:             "test query",
			Query:            "select 1",
			LabelsIncludeAny: []string{"nope"},
			Platform:         "darwin,windows",
		},
	})

	require.Error(t, err)
}

func TestApplyPolicySpecsLabelScopeRequiresPremium(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		return nil
	}
	ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
		return map[string]*fleet.Label{"foo": {Name: "foo", ID: 1}}, nil
	}

	// Free license.
	svc, ctx := newTestService(t, ds, nil, nil)

	testAdmin := fleet.User{
		ID:         1,
		Teams:      []fleet.UserTeam{},
		GlobalRole: new(fleet.RoleAdmin),
	}
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})

	for name, spec := range map[string]*fleet.PolicySpec{
		"labels_include_any": {Name: "p", Query: "SELECT 1", LabelsIncludeAny: []string{"foo"}},
		"labels_include_all": {Name: "p", Query: "SELECT 1", LabelsIncludeAll: []string{"foo"}},
		"labels_exclude_any": {Name: "p", Query: "SELECT 1", LabelsExcludeAny: []string{"foo"}},
		"labels_exclude_all": {Name: "p", Query: "SELECT 1", LabelsExcludeAll: []string{"foo"}},
	} {
		t.Run(name, func(t *testing.T) {
			err := svc.ApplyPolicySpecs(viewerCtx, []*fleet.PolicySpec{spec})
			require.ErrorIs(t, err, fleet.ErrMissingLicense)
		})
	}
	require.False(t, ds.ApplyPolicySpecsFuncInvoked)
}

func TestApplyPolicySpecsDefaultType(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	var capturedSpecs []*fleet.PolicySpec
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		capturedSpecs = specs
		return nil
	}

	opts := &TestServerOpts{}
	svc, ctx := newTestService(t, ds, nil, nil, opts)
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
		return nil
	}

	testAdmin := fleet.User{
		ID:         1,
		Teams:      []fleet.UserTeam{},
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})

	// Test that an omitted type defaults to "dynamic".
	err := svc.ApplyPolicySpecs(viewerCtx, []*fleet.PolicySpec{
		{
			Name:  "no-type-policy",
			Query: "SELECT 1;",
		},
	})
	require.NoError(t, err)
	require.Len(t, capturedSpecs, 1)
	require.Equal(t, fleet.PolicyTypeDynamic, capturedSpecs[0].Type)

	// Test that an explicit type is not overridden.
	err = svc.ApplyPolicySpecs(viewerCtx, []*fleet.PolicySpec{
		{
			Name:  "explicit-dynamic-policy",
			Query: "SELECT 1;",
			Type:  fleet.PolicyTypeDynamic,
		},
	})
	require.NoError(t, err)
	require.Len(t, capturedSpecs, 1)
	require.Equal(t, fleet.PolicyTypeDynamic, capturedSpecs[0].Type)
}

func TestResetPolicyAuth(t *testing.T) {
	const policyID = uint(42)
	teamID := uint(1)

	testCases := []struct {
		name            string
		user            *fleet.User
		policyTeamID    *uint
		shouldFailWrite bool
	}{
		{
			name:            "global admin can reset global policy",
			user:            &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
			policyTeamID:    nil,
			shouldFailWrite: false,
		},
		{
			name:            "global maintainer can reset global policy",
			user:            &fleet.User{GlobalRole: new(fleet.RoleMaintainer)},
			policyTeamID:    nil,
			shouldFailWrite: false,
		},
		{
			name:            "global observer cannot reset policy",
			user:            &fleet.User{GlobalRole: new(fleet.RoleObserver)},
			policyTeamID:    nil,
			shouldFailWrite: true,
		},
		{
			name:            "global gitops can reset global policy",
			user:            &fleet.User{GlobalRole: new(fleet.RoleGitOps)},
			policyTeamID:    nil,
			shouldFailWrite: false,
		},
		{
			name:            "team admin can reset own team policy",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleAdmin}}},
			policyTeamID:    &teamID,
			shouldFailWrite: false,
		},
		{
			name:            "team maintainer can reset own team policy",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleMaintainer}}},
			policyTeamID:    &teamID,
			shouldFailWrite: false,
		},
		{
			name:            "team observer cannot reset policy",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamID}, Role: fleet.RoleObserver}}},
			policyTeamID:    &teamID,
			shouldFailWrite: true,
		},
		{
			name:            "team admin of different team cannot reset policy",
			user:            &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			policyTeamID:    &teamID,
			shouldFailWrite: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.PolicyFunc = func(_ context.Context, id uint) (*fleet.Policy, error) {
				return &fleet.Policy{PolicyData: fleet.PolicyData{ID: id, TeamID: tt.policyTeamID}}, nil
			}
			ds.ResetPolicyFunc = func(_ context.Context, _ uint) error { return nil }

			opts := &TestServerOpts{}
			svc, baseCtx := newTestService(t, ds, nil, nil, opts)
			opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
				return nil
			}
			ctx := viewer.NewContext(baseCtx, viewer.Viewer{User: tt.user})

			err := svc.ResetPolicy(ctx, policyID)
			checkAuthErr(t, tt.shouldFailWrite, err)
			if !tt.shouldFailWrite {
				require.True(t, ds.ResetPolicyFuncInvoked)
			}
		})
	}
}

func TestResetPolicyNotFound(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.PolicyFunc = func(_ context.Context, _ uint) (*fleet.Policy, error) {
		return nil, &notFoundError{}
	}

	user := &fleet.User{GlobalRole: new(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	err := svc.ResetPolicy(ctx, 999)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
}

func TestResetPolicyEmitsActivity(t *testing.T) {
	const policyID = uint(7)
	const policyName = "My Policy"

	newSvc := func(teamID *uint) (*mock.Store, fleet.Service, context.Context, *TestServerOpts) {
		ds := new(mock.Store)
		ds.PolicyFunc = func(_ context.Context, id uint) (*fleet.Policy, error) {
			return &fleet.Policy{PolicyData: fleet.PolicyData{ID: id, Name: policyName, TeamID: teamID}}, nil
		}
		ds.ResetPolicyFunc = func(_ context.Context, _ uint) error { return nil }
		opts := &TestServerOpts{}
		svc, baseCtx := newTestService(t, ds, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}
		ctx := viewer.NewContext(baseCtx, viewer.Viewer{
			User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
		})
		return ds, svc, ctx, opts
	}

	t.Run("global policy emits team_id -1", func(t *testing.T) {
		ds, svc, ctx, opts := newSvc(nil)
		var capturedActivity activity_api.ActivityDetails
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, a activity_api.ActivityDetails) error {
			capturedActivity = a
			return nil
		}

		require.NoError(t, svc.ResetPolicy(ctx, policyID))
		require.True(t, ds.ResetPolicyFuncInvoked)
		require.True(t, opts.ActivityMock.NewActivityFuncInvoked)

		act, ok := capturedActivity.(fleet.ActivityTypeResetPolicy)
		require.True(t, ok)
		require.Equal(t, policyID, act.ID)
		require.Equal(t, policyName, act.Name)
		require.NotNil(t, act.TeamID)
		require.Equal(t, int64(-1), *act.TeamID)
		require.Nil(t, act.TeamName)
	})

	t.Run("no-team policy emits team_id 0", func(t *testing.T) {
		noTeamID := uint(0)
		ds, svc, ctx, opts := newSvc(&noTeamID)
		var capturedActivity activity_api.ActivityDetails
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, a activity_api.ActivityDetails) error {
			capturedActivity = a
			return nil
		}

		require.NoError(t, svc.ResetPolicy(ctx, policyID))
		require.True(t, ds.ResetPolicyFuncInvoked)
		require.True(t, opts.ActivityMock.NewActivityFuncInvoked)

		act, ok := capturedActivity.(fleet.ActivityTypeResetPolicy)
		require.True(t, ok)
		require.Equal(t, policyID, act.ID)
		require.Equal(t, policyName, act.Name)
		require.NotNil(t, act.TeamID)
		require.Equal(t, int64(0), *act.TeamID)
		require.Nil(t, act.TeamName)
	})
}

func TestNewGlobalPolicyQueryIDAuth(t *testing.T) {
	const (
		queryID   = uint(99)
		secretSQL = "SELECT secret FROM restricted;"
	)

	testCases := []struct {
		name            string
		user            *fleet.User
		payload         fleet.PolicyPayload
		queryErr        error
		wantQueryLoaded bool
		wantErr         bool
	}{
		{
			name:            "global admin from query_id loads and authorizes the query",
			user:            &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)},
			payload:         fleet.PolicyPayload{QueryID: new(queryID)},
			wantQueryLoaded: true,
		},
		{
			name:            "global maintainer from query_id loads and authorizes the query",
			user:            &fleet.User{ID: 1, GlobalRole: new(fleet.RoleMaintainer)},
			payload:         fleet.PolicyPayload{QueryID: new(queryID)},
			wantQueryLoaded: true,
		},
		{
			name:            "global gitops from query_id loads and authorizes the query",
			user:            &fleet.User{ID: 1, GlobalRole: new(fleet.RoleGitOps)},
			payload:         fleet.PolicyPayload{QueryID: new(queryID)},
			wantQueryLoaded: true,
		},
		{
			name:            "no query_id does not load any query",
			user:            &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)},
			payload:         fleet.PolicyPayload{Name: "inline", Query: "SELECT 1;"},
			wantQueryLoaded: false,
		},
		{
			name:            "missing referenced query fails",
			user:            &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)},
			payload:         fleet.PolicyPayload{QueryID: new(queryID)},
			queryErr:        &notFoundError{},
			wantQueryLoaded: true,
			wantErr:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			opts := &TestServerOpts{}
			svc, baseCtx := newTestService(t, ds, nil, nil, opts)
			opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
				return nil
			}

			ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
				require.Equal(t, queryID, id)
				if tc.queryErr != nil {
					return nil, tc.queryErr
				}
				return &fleet.Query{ID: id, Name: "referenced query", Query: secretSQL}, nil
			}
			ds.NewGlobalPolicyFunc = func(ctx context.Context, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
				return &fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "referenced query", Query: secretSQL}}, nil
			}

			ctx := viewer.NewContext(baseCtx, viewer.Viewer{User: tc.user})

			_, err := svc.NewGlobalPolicy(ctx, tc.payload)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantQueryLoaded, ds.QueryFuncInvoked)
		})
	}
}
