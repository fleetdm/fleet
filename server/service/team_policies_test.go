package service

import (
	"context"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
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
	ds.ListTeamPoliciesFunc = func(ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions, automationFilter string) (tpol, ipol []*fleet.Policy, err error) {
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
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{ID: 1}, nil
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

			_, _, err = svc.ListTeamPolicies(ctx, 1, fleet.ListOptions{}, fleet.ListOptions{}, false, "")
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetTeamPolicyByID(ctx, 1, 1)
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

// TestTeamPolicyAutomationsPopulated verifies that every endpoint that
// returns a team policy populates the install_software, run_script, and
// patch_software automation fields by exercising the
// populateAutomationsForTeamPolicy helper.
func TestTeamPolicyAutomationsPopulated(t *testing.T) {
	const (
		teamID                 = uint(1)
		policyID               = uint(42)
		softwareInstallerID    = uint(101)
		softwareInstallerTitle = uint(201)
		scriptID               = uint(301)
		patchSoftwareTitleID   = uint(401)
		patchInstallerTitleID  = uint(501)
		installerSoftwareTitle = "Cool Installer"
		installerDisplayName   = "Cool Installer.app"
		scriptName             = "remediate.sh"
		patchSoftwareTitleName = "Patchable App"
		patchSoftwareDisplay   = "Patchable App.app"
	)

	// Returns a fresh team-scoped policy with all three automation IDs set.
	// Each test gets a separate copy to prevent cross-test mutation.
	freshPolicy := func() *fleet.Policy {
		tID := teamID
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:                   policyID,
				TeamID:               &tID,
				Name:                 "policy-with-automations",
				Query:                "SELECT 1;",
				SoftwareInstallerID:  ptr.Uint(softwareInstallerID),
				ScriptID:             ptr.Uint(scriptID),
				PatchSoftwareTitleID: ptr.Uint(patchSoftwareTitleID),
			},
		}
	}

	setupDS := func() *mock.Store {
		ds := new(mock.Store)
		ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
			return freshPolicy(), nil
		}
		ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
			return freshPolicy(), nil
		}
		ds.TeamPolicyFunc = func(ctx context.Context, tID uint, id uint) (*fleet.Policy, error) {
			return freshPolicy(), nil
		}
		ds.ListTeamPoliciesFunc = func(ctx context.Context, tID uint, opts fleet.ListOptions, iopts fleet.ListOptions, automationFilter string) ([]*fleet.Policy, []*fleet.Policy, error) {
			return []*fleet.Policy{freshPolicy()}, nil, nil
		}
		ds.ListMergedTeamPoliciesFunc = func(ctx context.Context, tID uint, opts fleet.ListOptions, automationFilter string) ([]*fleet.Policy, error) {
			return []*fleet.Policy{freshPolicy()}, nil
		}
		ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, _ bool, _ bool) error {
			return nil
		}
		ds.TeamLiteFunc = func(ctx context.Context, tID uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{ID: tID}, nil
		}
		ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
			require.Equal(t, softwareInstallerID, id)
			return &fleet.SoftwareInstaller{
				TitleID:       ptr.Uint(softwareInstallerTitle),
				SoftwareTitle: installerSoftwareTitle,
				DisplayName:   installerDisplayName,
			}, nil
		}
		ds.ScriptFunc = func(ctx context.Context, id uint) (*fleet.Script, error) {
			require.Equal(t, scriptID, id)
			return &fleet.Script{ID: id, Name: scriptName}, nil
		}
		ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, tID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
			require.Equal(t, patchSoftwareTitleID, titleID)
			return &fleet.SoftwareInstaller{
				TitleID:       ptr.Uint(patchInstallerTitleID),
				SoftwareTitle: patchSoftwareTitleName,
				DisplayName:   patchSoftwareDisplay,
			}, nil
		}
		return ds
	}

	requireAutomationsPopulated := func(t *testing.T, p *fleet.Policy) {
		t.Helper()
		require.NotNil(t, p)
		require.NotNil(t, p.InstallSoftware, "install_software should be populated")
		assert.Equal(t, softwareInstallerTitle, p.InstallSoftware.SoftwareTitleID)
		assert.Equal(t, installerSoftwareTitle, p.InstallSoftware.Name)
		assert.Equal(t, installerDisplayName, p.InstallSoftware.DisplayName)

		require.NotNil(t, p.RunScript, "run_script should be populated")
		assert.Equal(t, scriptID, p.RunScript.ID)
		assert.Equal(t, scriptName, p.RunScript.Name)

		require.NotNil(t, p.PatchSoftware, "patch_software should be populated")
		assert.Equal(t, patchInstallerTitleID, p.PatchSoftware.SoftwareTitleID)
		assert.Equal(t, patchSoftwareTitleName, p.PatchSoftware.Name)
		assert.Equal(t, patchSoftwareDisplay, p.PatchSoftware.DisplayName)
	}

	adminCtx := func(ctx context.Context) context.Context {
		return viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
			ID:         1,
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}})
	}

	t.Run("NewTeamPolicy", func(t *testing.T) {
		ds := setupDS()
		opts := &TestServerOpts{}
		svc, baseCtx := newTestService(t, ds, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}
		ctx := adminCtx(baseCtx)

		policy, err := svc.NewTeamPolicy(ctx, teamID, fleet.NewTeamPolicyPayload{
			Name:  "policy-with-automations",
			Query: "SELECT 1;",
		})
		require.NoError(t, err)
		requireAutomationsPopulated(t, policy)
	})

	t.Run("GetTeamPolicyByID", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		policy, err := svc.GetTeamPolicyByID(ctx, teamID, policyID)
		require.NoError(t, err)
		requireAutomationsPopulated(t, policy)
	})

	t.Run("GetPolicyByID", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		policy, err := svc.GetPolicyByID(ctx, policyID)
		require.NoError(t, err)
		requireAutomationsPopulated(t, policy)
	})

	t.Run("ListTeamPolicies", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		teamPols, _, err := svc.ListTeamPolicies(ctx, teamID, fleet.ListOptions{}, fleet.ListOptions{}, false, "")
		require.NoError(t, err)
		require.Len(t, teamPols, 1)
		requireAutomationsPopulated(t, teamPols[0])
	})

	t.Run("ListTeamPolicies_mergeInherited", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		merged, _, err := svc.ListTeamPolicies(ctx, teamID, fleet.ListOptions{}, fleet.ListOptions{}, true, "")
		require.NoError(t, err)
		require.Len(t, merged, 1)
		requireAutomationsPopulated(t, merged[0])
	})

	t.Run("ModifyTeamPolicy", func(t *testing.T) {
		ds := setupDS()
		opts := &TestServerOpts{}
		svc, baseCtx := newTestService(t, ds, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}
		ctx := adminCtx(baseCtx)

		// Empty payload — no field changes; we only care that the helper runs
		// after SavePolicy and the returned policy has its automations populated.
		policy, err := svc.ModifyTeamPolicy(ctx, teamID, policyID, fleet.ModifyPolicyPayload{})
		require.NoError(t, err)
		requireAutomationsPopulated(t, policy)
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
