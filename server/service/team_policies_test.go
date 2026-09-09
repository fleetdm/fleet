package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
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
	ds.ListTeamPoliciesFunc = func(ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions, automationType fleet.PolicyAutomationType, platform string) (tpol, ipol []*fleet.Policy, err error) {
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

			_, _, err = svc.ListTeamPolicies(ctx, 1, fleet.ListOptions{}, fleet.ListOptions{}, false, "", "")
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

func TestTeamPolicyPatchWhenClosed(t *testing.T) {
	const (
		teamID               = uint(1)
		policyID             = uint(42)
		patchSoftwareTitleID = uint(401)
	)
	patchType := fleet.PolicyTypePatch

	freshPatchPolicy := func() *fleet.Policy {
		tID := teamID
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:                   policyID,
				TeamID:               &tID,
				Name:                 "macOS - App up to date",
				Type:                 fleet.PolicyTypePatch,
				PatchSoftwareTitleID: new(patchSoftwareTitleID),
			},
		}
	}

	adminCtx := func(ctx context.Context) context.Context {
		return viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}})
	}

	setupDS := func() *mock.Store {
		ds := new(mock.Store)
		ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
			return freshPatchPolicy(), nil
		}
		ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, tID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
			return &fleet.SoftwareInstaller{TitleID: new(patchSoftwareTitleID), SoftwareTitle: "App", DisplayName: "App"}, nil
		}
		ds.ClearPreInstallQueryForTitleFunc = func(ctx context.Context, teamID uint, titleID uint) error {
			return nil
		}
		return ds
	}

	// Creating a patch-when-closed policy with continuous automations on succeeds and clears the
	// title's managed pre-install query.
	t.Run("create patch-when-closed policy", func(t *testing.T) {
		ds := setupDS()
		var captured fleet.PolicyPayload
		ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
			captured = args
			created := freshPatchPolicy()
			created.PatchWhenClosed = true
			return created, nil
		}
		opts := &TestServerOpts{}
		svc, baseCtx := newTestService(t, ds, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}

		_, err := svc.NewTeamPolicy(adminCtx(baseCtx), teamID, fleet.NewTeamPolicyPayload{
			Type:                         &patchType,
			PatchSoftwareTitleID:         new(patchSoftwareTitleID),
			PatchWhenClosed:              true,
			ContinuousAutomationsEnabled: true,
		})
		require.NoError(t, err)
		assert.True(t, captured.PatchWhenClosed)
		assert.True(t, captured.ContinuousAutomationsEnabled)
		// enabling patch_when_closed cancels the title's pending installs so they re-evaluate
		assert.True(t, ds.ClearPreInstallQueryForTitleFuncInvoked)
	})

	// continuous_automations_enabled=false with patch_when_closed=true is rejected on create too.
	t.Run("create rejects disabling continuous automations", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		_, err := svc.NewTeamPolicy(adminCtx(baseCtx), teamID, fleet.NewTeamPolicyPayload{
			Type:                         &patchType,
			PatchSoftwareTitleID:         new(patchSoftwareTitleID),
			PatchWhenClosed:              true,
			ContinuousAutomationsEnabled: false,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "continuous_automations_enabled")
	})

	// patch_when_closed only applies to patch policies.
	t.Run("create rejects patch_when_closed on non-patch policy", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		_, err := svc.NewTeamPolicy(adminCtx(baseCtx), teamID, fleet.NewTeamPolicyPayload{
			Name:  "dynamic policy",
			Query: "SELECT 1;",
			// Continuous automations must be on, otherwise that check rejects the payload first.
			PatchWhenClosed:              true,
			ContinuousAutomationsEnabled: true,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "only supported for patch policies")
	})

	// An explicit continuous_automations_enabled=false alongside patch_when_closed=true is rejected;
	// omitting it (see next case) still auto-sets it to true.
	t.Run("modify rejects disabling continuous automations", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		_, err := svc.ModifyTeamPolicy(adminCtx(baseCtx), teamID, policyID, fleet.ModifyPolicyPayload{
			PatchWhenClosed:              new(true),
			ContinuousAutomationsEnabled: new(false),
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "continuous_automations_enabled")
	})

	// Enabling patch_when_closed on modify forces continuous automations on.
	t.Run("modify auto-sets continuous automations", func(t *testing.T) {
		ds := setupDS()
		var saved *fleet.Policy
		ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, _ bool, _ bool) error {
			saved = p
			return nil
		}
		opts := &TestServerOpts{}
		svc, baseCtx := newTestService(t, ds, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}

		_, err := svc.ModifyTeamPolicy(adminCtx(baseCtx), teamID, policyID, fleet.ModifyPolicyPayload{
			PatchWhenClosed: new(true),
		})
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.True(t, saved.PatchWhenClosed)
		assert.True(t, saved.ContinuousAutomationsEnabled)
		// enabling patch_when_closed cancels the title's pending installs so they re-evaluate
		assert.True(t, ds.ClearPreInstallQueryForTitleFuncInvoked)
	})
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

	wantInstallIconURL := fmt.Sprintf(
		"/api/latest/fleet/software/titles/%d/icon?fleet_id=%d",
		softwareInstallerTitle, teamID,
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
		ds.ListTeamPoliciesFunc = func(ctx context.Context, tID uint, opts fleet.ListOptions, iopts fleet.ListOptions, automationType fleet.PolicyAutomationType, platform string) ([]*fleet.Policy, []*fleet.Policy, error) {
			return []*fleet.Policy{freshPolicy()}, nil, nil
		}
		ds.ListMergedTeamPoliciesFunc = func(ctx context.Context, tID uint, opts fleet.ListOptions, automationType fleet.PolicyAutomationType, platform string) ([]*fleet.Policy, error) {
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
				InstallerID:   softwareInstallerID,
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
		ds.GetSoftwareIconsByTeamAndTitleIdsFunc = func(ctx context.Context, tID uint, titleIDs []uint) (map[uint]fleet.SoftwareTitleIcon, error) {
			require.Equal(t, teamID, tID)
			// Only the install-software title has a custom icon uploaded.
			return map[uint]fleet.SoftwareTitleIcon{
				softwareInstallerTitle: {
					TeamID:          teamID,
					SoftwareTitleID: softwareInstallerTitle,
				},
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
		// SoftwareInstallerID lets the FE pre-fill the "Select package" pin
		// on reload instead of always re-deriving first-added.
		require.NotNil(t, p.InstallSoftware.SoftwareInstallerID, "install_software.software_package_id should be populated")
		assert.Equal(t, softwareInstallerID, *p.InstallSoftware.SoftwareInstallerID)

		require.NotNil(t, p.RunScript, "run_script should be populated")
		assert.Equal(t, scriptID, p.RunScript.ID)
		assert.Equal(t, scriptName, p.RunScript.Name)

		require.NotNil(t, p.PatchSoftware, "patch_software should be populated")
		assert.Equal(t, patchInstallerTitleID, p.PatchSoftware.SoftwareTitleID)
		assert.Equal(t, patchSoftwareTitleName, p.PatchSoftware.Name)
		assert.Equal(t, patchSoftwareDisplay, p.PatchSoftware.DisplayName)
		// Patch policies target FMA titles (single package per title), so
		// per-package pinning doesn't apply and the field stays nil.
		assert.Nil(t, p.PatchSoftware.SoftwareInstallerID, "patch_software.software_package_id should stay nil")
	}

	// requireSoftwareIconURLs verifies that install_software.icon_url is set to the
	// custom icon path when one exists, and that patch_software.icon_url stays nil
	// when the title has no custom icon.
	requireSoftwareIconURLs := func(t *testing.T, p *fleet.Policy) {
		t.Helper()
		require.NotNil(t, p.InstallSoftware, "install_software should be populated")
		require.NotNil(t, p.InstallSoftware.IconURL, "install_software icon_url should be set when a custom icon exists")
		assert.Equal(t, wantInstallIconURL, *p.InstallSoftware.IconURL)

		require.NotNil(t, p.PatchSoftware, "patch_software should be populated")
		assert.Nil(t, p.PatchSoftware.IconURL, "patch_software icon_url should be nil when no custom icon exists")
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
		requireSoftwareIconURLs(t, policy)
	})

	t.Run("GetPolicyByID", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		policy, err := svc.GetPolicyByID(ctx, policyID)
		require.NoError(t, err)
		requireAutomationsPopulated(t, policy)
		requireSoftwareIconURLs(t, policy)
	})

	t.Run("ListTeamPolicies", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		teamPols, _, err := svc.ListTeamPolicies(ctx, teamID, fleet.ListOptions{}, fleet.ListOptions{}, false, "", "")
		require.NoError(t, err)
		require.Len(t, teamPols, 1)
		requireAutomationsPopulated(t, teamPols[0])
		requireSoftwareIconURLs(t, teamPols[0])
	})

	t.Run("ListTeamPolicies_mergeInherited", func(t *testing.T) {
		ds := setupDS()
		svc, baseCtx := newTestService(t, ds, nil, nil)
		ctx := adminCtx(baseCtx)

		merged, _, err := svc.ListTeamPolicies(ctx, teamID, fleet.ListOptions{}, fleet.ListOptions{}, true, "", "")
		require.NoError(t, err)
		require.Len(t, merged, 1)
		requireAutomationsPopulated(t, merged[0])
		requireSoftwareIconURLs(t, merged[0])
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

// TestPopulateSoftwareIconURLs verifies how the software automation icon_url is derived:
// - Titles with a custom uploaded icon and VPP apps both get the icon endpoint URL (which serves the custom icon or redirects to the App Store icon).
// - Package installers with no custom icon get a nil URL (so clients use the default icon and avoid a 404).
// - Inherited policies (nil team, no software, present in merged lists) are skipped.
func TestPopulateSoftwareIconURLs(t *testing.T) {
	const (
		teamID               = uint(7)
		customInstallerTitle = uint(11)
		plainInstallerTitle  = uint(12)
		vppPlainTitle        = uint(13)
		vppCustomTitle       = uint(14)
		patchCustomTitle     = uint(15)
		vppAppsTeamsID       = uint(900)
	)

	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	// populateSoftwareIconURLs is unexported, so reach the concrete service.
	svcImpl := svc.(validationMiddleware).Service.(*Service)

	var requestedTitleIDs []uint
	ds.GetSoftwareIconsByTeamAndTitleIdsFunc = func(ctx context.Context, tID uint, titleIDs []uint) (map[uint]fleet.SoftwareTitleIcon, error) {
		require.Equal(t, teamID, tID)
		requestedTitleIDs = titleIDs
		// Custom icons exist for one installer, one VPP app, and one patch title.
		return map[uint]fleet.SoftwareTitleIcon{
			customInstallerTitle: {TeamID: teamID, SoftwareTitleID: customInstallerTitle},
			vppCustomTitle:       {TeamID: teamID, SoftwareTitleID: vppCustomTitle},
			patchCustomTitle:     {TeamID: teamID, SoftwareTitleID: patchCustomTitle},
		}, nil
	}

	tID := teamID
	teamPolicy := func(install *fleet.PolicySoftwareTitle, vppAppsTeamsID *uint, patch *fleet.PolicySoftwareTitle) *fleet.Policy {
		return &fleet.Policy{
			PolicyData:      fleet.PolicyData{TeamID: &tID, VPPAppsTeamsID: vppAppsTeamsID},
			InstallSoftware: install,
			PatchSoftware:   patch,
		}
	}

	customInstaller := teamPolicy(&fleet.PolicySoftwareTitle{SoftwareTitleID: customInstallerTitle}, nil, nil)
	plainInstaller := teamPolicy(&fleet.PolicySoftwareTitle{SoftwareTitleID: plainInstallerTitle}, nil, nil)
	vppPlain := teamPolicy(&fleet.PolicySoftwareTitle{SoftwareTitleID: vppPlainTitle}, new(vppAppsTeamsID), nil)
	vppCustom := teamPolicy(&fleet.PolicySoftwareTitle{SoftwareTitleID: vppCustomTitle}, new(vppAppsTeamsID), nil)
	patchPolicy := teamPolicy(nil, nil, &fleet.PolicySoftwareTitle{SoftwareTitleID: patchCustomTitle})
	inheritedPolicy := &fleet.Policy{PolicyData: fleet.PolicyData{TeamID: nil}}

	policies := []*fleet.Policy{customInstaller, plainInstaller, vppPlain, vppCustom, patchPolicy, inheritedPolicy}
	require.NoError(t, svcImpl.populateSoftwareIconURLs(ctx, policies))

	require.True(t, ds.GetSoftwareIconsByTeamAndTitleIdsFuncInvoked)
	// The inherited policy has no team and no software, so it contributes nothing.
	require.ElementsMatch(t,
		[]uint{customInstallerTitle, plainInstallerTitle, vppPlainTitle, vppCustomTitle, patchCustomTitle},
		requestedTitleIDs,
	)

	// Custom installer icon → canonical endpoint URL.
	require.NotNil(t, customInstaller.InstallSoftware.IconURL)
	assert.Equal(t,
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", customInstallerTitle, teamID),
		*customInstaller.InstallSoftware.IconURL,
	)

	// Plain installer, no custom icon → nil (client uses default icon, no 404).
	assert.Nil(t, plainInstaller.InstallSoftware.IconURL)

	// VPP app, no custom icon → endpoint URL (redirects to App Store icon).
	require.NotNil(t, vppPlain.InstallSoftware.IconURL)
	assert.Equal(t,
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", vppPlainTitle, teamID),
		*vppPlain.InstallSoftware.IconURL,
	)

	// VPP app with a custom icon → still gets the endpoint URL (custom icon served).
	require.NotNil(t, vppCustom.InstallSoftware.IconURL)
	assert.Equal(t,
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", vppCustomTitle, teamID),
		*vppCustom.InstallSoftware.IconURL,
	)

	// Patch software with a custom icon → endpoint URL.
	require.NotNil(t, patchPolicy.PatchSoftware.IconURL)
	assert.Equal(t,
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", patchCustomTitle, teamID),
		*patchPolicy.PatchSoftware.IconURL,
	)
}

func TestNewTeamPolicyQueryIDAuth(t *testing.T) {
	const (
		callerTeamID = uint(1)
		otherTeamID  = uint(2)
		queryID      = uint(99)
		secretSQL    = "SELECT secret FROM restricted;"
	)

	otherTeam := otherTeamID
	callerTeam := callerTeamID

	testCases := []struct {
		name        string
		user        *fleet.User
		queryTeamID *uint
		shouldFail  bool
	}{
		{
			name:        "team admin references another team's query",
			user:        &fleet.User{ID: 1, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: callerTeamID}, Role: fleet.RoleAdmin}}},
			queryTeamID: &otherTeam,
			shouldFail:  true,
		},
		{
			name:        "team gitops references a global query",
			user:        &fleet.User{ID: 1, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: callerTeamID}, Role: fleet.RoleGitOps}}},
			queryTeamID: nil,
			shouldFail:  true,
		},
		{
			name:        "team admin references a global query",
			user:        &fleet.User{ID: 1, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: callerTeamID}, Role: fleet.RoleAdmin}}},
			queryTeamID: nil,
			shouldFail:  false,
		},
		{
			name:        "team admin references their own team's query",
			user:        &fleet.User{ID: 1, Teams: []fleet.UserTeam{{Team: fleet.Team{ID: callerTeamID}, Role: fleet.RoleAdmin}}},
			queryTeamID: &callerTeam,
			shouldFail:  false,
		},
		{
			name:        "global admin references another team's query",
			user:        &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)},
			queryTeamID: &otherTeam,
			shouldFail:  false,
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
				return &fleet.Query{
					ID:     id,
					TeamID: tc.queryTeamID,
					Name:   "referenced query",
					Query:  secretSQL,
				}, nil
			}
			ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
				return &fleet.Policy{
					PolicyData: fleet.PolicyData{ID: 1, TeamID: &callerTeam, Name: "referenced query", Query: secretSQL},
				}, nil
			}
			ds.TeamLiteFunc = func(ctx context.Context, tID uint) (*fleet.TeamLite, error) {
				return &fleet.TeamLite{ID: tID}, nil
			}

			ctx := viewer.NewContext(baseCtx, viewer.Viewer{User: tc.user})

			_, err := svc.NewTeamPolicy(ctx, callerTeamID, fleet.NewTeamPolicyPayload{
				QueryID: new(queryID),
			})

			if tc.shouldFail {
				require.Error(t, err)
				var forbiddenError *authz.Forbidden
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
			require.True(t, ds.QueryFuncInvoked, "expected the referenced query to be loaded for a read authorization check")
		})
	}
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

func TestTeamPolicyResendConfigProfile(t *testing.T) {
	const (
		teamID     = uint(1)
		policyID   = uint(42)
		appleUUID  = fleet.MDMAppleProfileUUIDPrefix + "1111"
		winUUID    = fleet.MDMWindowsProfileUUIDPrefix + "2222"
		otherApple = fleet.MDMAppleProfileUUIDPrefix + "3333"
		appleName  = "Apple Profile"
		winName    = "Windows Profile"
	)

	adminCtx := func(ctx context.Context) context.Context {
		return viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}})
	}

	// policy returns a fresh team policy whose resend columns are set as given, so
	// each subtest gets its own copy and cannot mutate another's.
	policy := func(apple, windows *string) *fleet.Policy {
		tID := teamID
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				ID:                       policyID,
				TeamID:                   &tID,
				Name:                     "resend-policy",
				Query:                    "SELECT 1;",
				Platform:                 "darwin,windows",
				ResendAppleProfileUUID:   apple,
				ResendWindowsProfileUUID: windows,
			},
		}
	}

	setupDS := func(existing *fleet.Policy) *mock.Store {
		ds := new(mock.Store)
		ds.TeamExistsFunc = func(ctx context.Context, id uint) (bool, error) { return true, nil }
		ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) { return existing, nil }
		ds.TeamPolicyFunc = func(ctx context.Context, tID uint, id uint) (*fleet.Policy, error) { return existing, nil }
		ds.TeamLiteFunc = func(ctx context.Context, id uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{ID: id}, nil
		}
		ds.TeamWithExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
			return &fleet.Team{ID: id, Name: "team1"}, nil
		}
		ds.GetMDMAppleConfigProfileFunc = func(ctx context.Context, profileUUID string) (*fleet.MDMAppleConfigProfile, error) {
			return &fleet.MDMAppleConfigProfile{ProfileUUID: profileUUID, Name: appleName}, nil
		}
		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, profileUUID string) (*fleet.MDMWindowsConfigProfile, error) {
			return &fleet.MDMWindowsConfigProfile{ProfileUUID: profileUUID, Name: winName}, nil
		}
		return ds
	}

	newPremiumSvc := func(t *testing.T, ds *mock.Store) (fleet.Service, context.Context) {
		opts := &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}}
		svc, baseCtx := newTestService(t, ds, nil, nil, opts)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}
		return svc, adminCtx(baseCtx)
	}

	// Create passes profile_uuid straight through to the datastore payload. This is
	// the plumbing the datastore tests cannot see, since they call the store directly.
	t.Run("create plumbs profile_uuid to the payload", func(t *testing.T) {
		for _, uuid := range []string{appleUUID, winUUID} {
			t.Run(uuid, func(t *testing.T) {
				ds := setupDS(nil)
				var captured fleet.PolicyPayload
				ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
					captured = args
					return policy(nil, nil), nil
				}
				svc, ctx := newPremiumSvc(t, ds)

				_, err := svc.NewTeamPolicy(ctx, teamID, fleet.NewTeamPolicyPayload{
					Name:        "resend-policy",
					Query:       "SELECT 1;",
					Platform:    "darwin,windows",
					ProfileUUID: new(uuid),
				})
				require.NoError(t, err)
				require.True(t, ds.NewTeamPolicyFuncInvoked)
				require.NotNil(t, captured.ProfileUUID)
				require.Equal(t, uuid, *captured.ProfileUUID)
			})
		}
	})

	t.Run("platform gate", func(t *testing.T) {
		t.Run("create on a policy targeting neither darwin nor windows is rejected", func(t *testing.T) {
			for _, platform := range []string{"linux", "chrome", "linux,chrome"} {
				t.Run("platform="+platform, func(t *testing.T) {
					ds := setupDS(nil)
					ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
						return policy(nil, nil), nil
					}
					svc, ctx := newPremiumSvc(t, ds)

					_, err := svc.NewTeamPolicy(ctx, teamID, fleet.NewTeamPolicyPayload{
						Name:        "resend-policy",
						Query:       "SELECT 1;",
						Platform:    platform,
						ProfileUUID: new(appleUUID),
					})
					require.Error(t, err)
					require.Contains(t, err.Error(), `"profile_uuid" is only valid on "darwin" and "windows" policies`)
					require.False(t, ds.NewTeamPolicyFuncInvoked)
				})
			}
		})

		t.Run("create is allowed whenever darwin or windows is targeted", func(t *testing.T) {
			// Including the cross-platform pairings, which the automation handles per host.
			for _, platform := range []string{"darwin", "windows", "darwin,windows", "linux,darwin", ""} {
				t.Run("platform="+platform, func(t *testing.T) {
					ds := setupDS(nil)
					ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
						return policy(new(appleUUID), nil), nil
					}
					svc, ctx := newPremiumSvc(t, ds)

					_, err := svc.NewTeamPolicy(ctx, teamID, fleet.NewTeamPolicyPayload{
						Name:        "resend-policy",
						Query:       "SELECT 1;",
						Platform:    platform,
						ProfileUUID: new(appleUUID),
					})
					require.NoError(t, err)
					require.True(t, ds.NewTeamPolicyFuncInvoked)
				})
			}
		})

		t.Run("modify rejects adding a profile to a linux-only policy", func(t *testing.T) {
			existing := policy(nil, nil)
			existing.Platform = "linux"
			ds := setupDS(existing)
			ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, removeAllMemberships, removeStats bool) error {
				return nil
			}
			svc, ctx := newPremiumSvc(t, ds)

			_, err := svc.ModifyTeamPolicy(ctx, teamID, policyID, fleet.ModifyPolicyPayload{
				ProfileUUID: optjson.SetString(appleUUID),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), `"profile_uuid" is only valid on "darwin" and "windows" policies`)
			require.False(t, ds.SavePolicyFuncInvoked)
		})

		t.Run("modify rejects narrowing the platform away from a policy that resends", func(t *testing.T) {
			// The profile stays as it was; the platform change is what invalidates the pairing.
			ds := setupDS(policy(new(appleUUID), nil))
			ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, removeAllMemberships, removeStats bool) error {
				return nil
			}
			svc, ctx := newPremiumSvc(t, ds)

			_, err := svc.ModifyTeamPolicy(ctx, teamID, policyID, fleet.ModifyPolicyPayload{
				Platform: new("linux"),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), `"profile_uuid" is only valid on "darwin" and "windows" policies`)
			require.False(t, ds.SavePolicyFuncInvoked)
		})

		t.Run("modify allows clearing the profile while narrowing the platform", func(t *testing.T) {
			ds := setupDS(policy(new(appleUUID), nil))
			var saved *fleet.Policy
			ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, removeAllMemberships, removeStats bool) error {
				saved = p
				return nil
			}
			svc, ctx := newPremiumSvc(t, ds)

			_, err := svc.ModifyTeamPolicy(ctx, teamID, policyID, fleet.ModifyPolicyPayload{
				Platform:    new("linux"),
				ProfileUUID: optjson.SetString(""),
			})
			require.NoError(t, err)
			require.True(t, ds.SavePolicyFuncInvoked)
			require.Nil(t, saved.ResendAppleProfileUUID)
			require.Nil(t, saved.ResendWindowsProfileUUID)
		})
	})

	t.Run("create with no profile_uuid leaves the payload nil", func(t *testing.T) {
		ds := setupDS(nil)
		var captured fleet.PolicyPayload
		ds.NewTeamPolicyFunc = func(ctx context.Context, tID uint, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
			captured = args
			return policy(nil, nil), nil
		}
		svc, ctx := newPremiumSvc(t, ds)

		_, err := svc.NewTeamPolicy(ctx, teamID, fleet.NewTeamPolicyPayload{Name: "p", Query: "SELECT 1;"})
		require.NoError(t, err)
		require.Nil(t, captured.ProfileUUID)
	})

	// modifyPolicy routes the single profile_uuid onto the right column, clears the
	// other, and mirrors the script_id reset asymmetry: setting or changing a profile
	// resets memberships and stats, unsetting does not.
	t.Run("modify routes columns and resets stats", func(t *testing.T) {
		cases := []struct {
			name        string
			prevApple   *string
			prevWindows *string
			newValue    string
			wantApple   *string
			wantWindows *string
			wantReset   bool
		}{
			{
				name:      "set where there was none",
				newValue:  appleUUID,
				wantApple: new(appleUUID),
				wantReset: true,
			},
			{
				name:        "set windows where there was none",
				newValue:    winUUID,
				wantWindows: new(winUUID),
				wantReset:   true,
			},
			{
				name:      "same apple profile re-applied",
				prevApple: new(appleUUID),
				newValue:  appleUUID,
				wantApple: new(appleUUID),
				wantReset: false,
			},
			{
				name:      "different apple profile",
				prevApple: new(appleUUID),
				newValue:  otherApple,
				wantApple: new(otherApple),
				wantReset: true,
			},
			{
				name:        "apple switched to windows clears the apple column",
				prevApple:   new(appleUUID),
				newValue:    winUUID,
				wantWindows: new(winUUID),
				wantReset:   true,
			},
			{
				name:        "windows switched to apple clears the windows column",
				prevWindows: new(winUUID),
				newValue:    appleUUID,
				wantApple:   new(appleUUID),
				wantReset:   true,
			},
			{
				name:        "same windows profile re-applied",
				prevWindows: new(winUUID),
				newValue:    winUUID,
				wantWindows: new(winUUID),
				wantReset:   false,
			},
			{
				name:      "unsetting clears both without resetting",
				prevApple: new(appleUUID),
				newValue:  "",
				wantReset: false,
			},
			{
				name:        "unsetting a windows profile clears both without resetting",
				prevWindows: new(winUUID),
				newValue:    "",
				wantReset:   false,
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ds := setupDS(policy(c.prevApple, c.prevWindows))
				var (
					saved                *fleet.Policy
					gotRemoveMemberships bool
					gotRemoveStats       bool
				)
				ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, removeAllMemberships bool, removeStats bool) error {
					saved, gotRemoveMemberships, gotRemoveStats = p, removeAllMemberships, removeStats
					return nil
				}
				svc, ctx := newPremiumSvc(t, ds)

				tID := teamID
				_, err := svc.ModifyTeamPolicy(ctx, tID, policyID, fleet.ModifyPolicyPayload{
					ProfileUUID: optjson.SetString(c.newValue),
				})
				require.NoError(t, err)
				require.True(t, ds.SavePolicyFuncInvoked)

				require.Equal(t, c.wantApple, saved.ResendAppleProfileUUID)
				require.Equal(t, c.wantWindows, saved.ResendWindowsProfileUUID)
				// At most one column may ever be set.
				require.False(t, saved.ResendAppleProfileUUID != nil && saved.ResendWindowsProfileUUID != nil)

				assert.Equal(t, c.wantReset, gotRemoveMemberships, "removeAllMemberships")
				assert.Equal(t, c.wantReset, gotRemoveStats, "removeStats")
			})
		}
	})

	// An absent profile_uuid must leave the existing association untouched, since
	// savePolicy writes both columns unconditionally.
	t.Run("absent profile_uuid keeps the existing profile", func(t *testing.T) {
		ds := setupDS(policy(new(appleUUID), nil))
		var saved *fleet.Policy
		var gotRemoveMemberships, gotRemoveStats bool
		ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, removeAllMemberships bool, removeStats bool) error {
			saved, gotRemoveMemberships, gotRemoveStats = p, removeAllMemberships, removeStats
			return nil
		}
		svc, ctx := newPremiumSvc(t, ds)

		_, err := svc.ModifyTeamPolicy(ctx, teamID, policyID, fleet.ModifyPolicyPayload{
			Name: new("renamed"),
		})
		require.NoError(t, err)
		require.Equal(t, new(appleUUID), saved.ResendAppleProfileUUID)
		require.Nil(t, saved.ResendWindowsProfileUUID)
		assert.False(t, gotRemoveMemberships)
		assert.False(t, gotRemoveStats)
	})

	// Prefix validation happens before the save.
	t.Run("rejected prefixes", func(t *testing.T) {
		cases := []struct {
			name       string
			uuid       string
			wantErrMsg string
		}{
			{
				name:       "apple declaration",
				uuid:       fleet.MDMAppleDeclarationUUIDPrefix + "4444",
				wantErrMsg: fleet.CantResendAppleDeclarationProfilesMessage,
			},
			{
				name:       "android profile",
				uuid:       fleet.MDMAndroidProfileUUIDPrefix + "5555",
				wantErrMsg: "has an invalid prefix",
			},
			{
				name:       "unknown prefix",
				uuid:       "z5555",
				wantErrMsg: "has an invalid prefix",
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ds := setupDS(policy(nil, nil))
				ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, _ bool, _ bool) error {
					return nil
				}
				svc, ctx := newPremiumSvc(t, ds)

				_, err := svc.ModifyTeamPolicy(ctx, teamID, policyID, fleet.ModifyPolicyPayload{
					ProfileUUID: optjson.SetString(c.uuid),
				})
				require.Error(t, err)
				var bre *fleet.BadRequestError
				require.ErrorAs(t, err, &bre)
				require.Contains(t, bre.Message, c.wantErrMsg)
				// Nothing must reach the datastore.
				require.False(t, ds.SavePolicyFuncInvoked)
			})
		}
	})

	// "All fleets" (global) policies cannot carry a resend profile.
	t.Run("global policy rejects profile_uuid", func(t *testing.T) {
		globalPolicy := &fleet.Policy{
			PolicyData: fleet.PolicyData{ID: policyID, Name: "global", Query: "SELECT 1;"},
		}
		ds := setupDS(globalPolicy)
		ds.SavePolicyFunc = func(ctx context.Context, p *fleet.Policy, _ bool, _ bool) error { return nil }
		svc, ctx := newPremiumSvc(t, ds)

		_, err := svc.ModifyGlobalPolicy(ctx, policyID, fleet.ModifyPolicyPayload{
			ProfileUUID: optjson.SetString(appleUUID),
		})
		require.Error(t, err)
		require.ErrorContains(t, err, errPolicyAllFleetsForProfiles)
		require.False(t, ds.SavePolicyFuncInvoked)
	})

	// The response object is populated from whichever column is set.
	t.Run("response object is populated", func(t *testing.T) {
		t.Run("apple", func(t *testing.T) {
			ds := setupDS(policy(new(appleUUID), nil))
			svc, ctx := newPremiumSvc(t, ds)

			got, err := svc.GetTeamPolicyByID(ctx, teamID, policyID)
			require.NoError(t, err)
			require.True(t, ds.GetMDMAppleConfigProfileFuncInvoked)
			require.False(t, ds.GetMDMWindowsConfigProfileFuncInvoked)
			require.NotNil(t, got.ResendConfigurationProfile)
			assert.Equal(t, appleUUID, got.ResendConfigurationProfile.UUID)
			assert.Equal(t, appleName, got.ResendConfigurationProfile.Name)
		})

		t.Run("windows", func(t *testing.T) {
			ds := setupDS(policy(nil, new(winUUID)))
			svc, ctx := newPremiumSvc(t, ds)

			got, err := svc.GetTeamPolicyByID(ctx, teamID, policyID)
			require.NoError(t, err)
			require.True(t, ds.GetMDMWindowsConfigProfileFuncInvoked)
			require.False(t, ds.GetMDMAppleConfigProfileFuncInvoked)
			require.NotNil(t, got.ResendConfigurationProfile)
			assert.Equal(t, winUUID, got.ResendConfigurationProfile.UUID)
			assert.Equal(t, winName, got.ResendConfigurationProfile.Name)
		})

		t.Run("neither column set", func(t *testing.T) {
			ds := setupDS(policy(nil, nil))
			svc, ctx := newPremiumSvc(t, ds)

			got, err := svc.GetTeamPolicyByID(ctx, teamID, policyID)
			require.NoError(t, err)
			require.Nil(t, got.ResendConfigurationProfile)
			require.False(t, ds.GetMDMAppleConfigProfileFuncInvoked)
			require.False(t, ds.GetMDMWindowsConfigProfileFuncInvoked)
		})
	})
}
