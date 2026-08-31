package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestServiceSoftwareTitlesAuth(t *testing.T) {
	ds := new(mock.Store)

	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmf fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return []fleet.SoftwareTitleListResult{}, 0, &fleet.PaginationMetadata{}, nil
	}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{}, nil
	}
	ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) { return true, nil }
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{BundleIdentifier: ptr.String("foo")}, nil
	}
	ds.UpdateSoftwareTitleNameFunc = func(ctx context.Context, id uint, name string) error {
		return nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	for _, tc := range []struct {
		name                 string
		user                 *fleet.User
		shouldFailGlobalRead bool
		shouldFailTeamRead   bool
		shouldFailGetByID    bool
		shouldFailWrite      bool
	}{
		{
			name: "global-admin",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    false,
			shouldFailWrite:      false,
		},
		{
			name: "global-maintainer",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleMaintainer),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    false,
			shouldFailWrite:      true,
		},
		{
			name: "global-observer",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleObserver),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    false,
			shouldFailWrite:      true,
		},
		{
			name: "team-admin-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleAdmin,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    false,
			shouldFailWrite:      true,
		},
		{
			name: "team-maintainer-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleMaintainer,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    false,
			shouldFailWrite:      true,
		},
		{
			name: "team-observer-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleObserver,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    false,
			shouldFailWrite:      true,
		},
		{
			name: "team-admin-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleAdmin,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
			shouldFailGetByID:    true,
			shouldFailWrite:      true,
		},
		{
			name: "team-maintainer-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleMaintainer,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
			shouldFailGetByID:    true,
			shouldFailWrite:      true,
		},
		{
			name: "team-observer-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleObserver,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
			shouldFailGetByID:    true,
			shouldFailWrite:      true,
		},
		{
			// GitOps can list software titles but cannot fetch a single title
			// because SoftwareTitleByID also requires Host:list permission.
			name: "global-gitops",
			user: &fleet.User{
				ID:         1,
				GlobalRole: ptr.String(fleet.RoleGitOps),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    true,
			shouldFailWrite:      true,
		},
		{
			name: "team-gitops-belongs-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 1},
					Role: fleet.RoleGitOps,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
			shouldFailGetByID:    true,
			shouldFailWrite:      true,
		},
		{
			name: "team-gitops-does-not-belong-to-team",
			user: &fleet.User{
				ID: 1,
				Teams: []fleet.UserTeam{{
					Team: fleet.Team{ID: 2},
					Role: fleet.RoleGitOps,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
			shouldFailGetByID:    true,
			shouldFailWrite:      true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tc.user})
			premiumCtx := license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

			// List all software titles.
			_, _, _, err := svc.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// List software for a team.
			_, _, _, err = svc.ListSoftwareTitles(premiumCtx, fleet.SoftwareTitleListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			// List software for a team should fail no matter what
			// with a non-premium context
			if !tc.shouldFailTeamRead {
				_, _, _, err = svc.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
					TeamID: ptr.Uint(1),
				})
				require.ErrorContains(t, err, "Requires Fleet Premium license")
			}

			// Get a software title for a team
			_, err = svc.SoftwareTitleByID(ctx, 1, ptr.Uint(1))
			checkAuthErr(t, tc.shouldFailGetByID, err)

			// Update a software title's name
			err = svc.UpdateSoftwareName(ctx, 1, "2 Chrome 2 Furious")
			checkAuthErr(t, tc.shouldFailWrite, err)
		})
	}
}

// TestSoftwareTitleByIDInstallerDetails covers the two authorization rules the
// title detail response depends on: the "no team" scope is authorized even when
// no fleet is given, and script contents and managed app configuration reach
// only callers who can read the installer, not merely software inventory.
func TestSoftwareTitleByIDInstallerDetails(t *testing.T) {
	const (
		installScript     = "echo install"
		uninstallScript   = "echo uninstall"
		postInstallScript = "echo post-install"
		preInstallQuery   = "SELECT 1;"
	)
	appConfiguration := json.RawMessage(`{"key":"value"}`)

	ds := new(mock.Store)
	ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) { return true, nil }
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{
			ID:                      id,
			Name:                    "Foo",
			SoftwareInstallersCount: 1,
			VPPAppsCount:            1,
		}, nil
	}
	ds.GetSoftwarePackagesByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) ([]*fleet.SoftwareInstaller, error) {
		return []*fleet.SoftwareInstaller{{
			InstallerID:       1,
			TitleID:           &titleID,
			Name:              "foo.pkg",
			Version:           "1.0",
			Platform:          "darwin",
			StorageID:         "abc123",
			SelfService:       true,
			InstallScript:     installScript,
			UninstallScript:   uninstallScript,
			PostInstallScript: postInstallScript,
			PreInstallQuery:   preInstallQuery,
			Configuration:     appConfiguration,
		}}, nil
	}
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{InstallerID: 1, DisplayName: "Foo"}, nil
	}
	ds.GetCategoriesForSoftwareInstallersFunc = func(ctx context.Context, installerIDs []uint) (map[uint][]string, error) {
		return nil, nil
	}
	ds.GetSummaryHostSoftwareInstallsFunc = func(ctx context.Context, installerID uint) (*fleet.SoftwareInstallerStatusSummary, error) {
		return &fleet.SoftwareInstallerStatusSummary{}, nil
	}
	ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
		return &fleet.VPPAppStoreApp{Name: "Bar", Configuration: appConfiguration}, nil
	}
	ds.GetSummaryHostVPPAppInstallsFunc = func(ctx context.Context, teamID *uint, appID fleet.VPPAppID) (*fleet.VPPAppStatusSummary, error) {
		return &fleet.VPPAppStatusSummary{}, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	for _, tc := range []struct {
		name string
		user *fleet.User
		// canReadInstaller is whether the role holds installable_entity read.
		canReadInstaller bool
		// installersWithoutTeam is whether omitting the fleet still returns
		// installer data, which needs read on the "no team" scope it resolves to.
		installersWithoutTeam bool
	}{
		{
			name:                  "global-admin",
			user:                  &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)},
			canReadInstaller:      true,
			installersWithoutTeam: true,
		},
		{
			name:                  "global-maintainer",
			user:                  &fleet.User{ID: 1, GlobalRole: new(fleet.RoleMaintainer)},
			canReadInstaller:      true,
			installersWithoutTeam: true,
		},
		{
			// Technician is the role that separates an installable_entity check
			// from a hardcoded admin/maintainer list: it reads installers but is
			// neither.
			name:                  "global-technician",
			user:                  &fleet.User{ID: 1, GlobalRole: new(fleet.RoleTechnician)},
			canReadInstaller:      true,
			installersWithoutTeam: true,
		},
		{
			name:                  "global-observer",
			user:                  &fleet.User{ID: 1, GlobalRole: new(fleet.RoleObserver)},
			canReadInstaller:      false,
			installersWithoutTeam: true,
		},
		{
			name:                  "global-observer-plus",
			user:                  &fleet.User{ID: 1, GlobalRole: new(fleet.RoleObserverPlus)},
			canReadInstaller:      false,
			installersWithoutTeam: true,
		},
		{
			name: "team-admin",
			user: &fleet.User{ID: 1, Teams: []fleet.UserTeam{{
				Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin,
			}}},
			canReadInstaller:      true,
			installersWithoutTeam: false,
		},
		{
			name: "team-maintainer",
			user: &fleet.User{ID: 1, Teams: []fleet.UserTeam{{
				Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer,
			}}},
			canReadInstaller:      true,
			installersWithoutTeam: false,
		},
		{
			name: "team-technician",
			user: &fleet.User{ID: 1, Teams: []fleet.UserTeam{{
				Team: fleet.Team{ID: 1}, Role: fleet.RoleTechnician,
			}}},
			canReadInstaller:      true,
			installersWithoutTeam: false,
		},
		{
			name: "team-observer",
			user: &fleet.User{ID: 1, Teams: []fleet.UserTeam{{
				Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver,
			}}},
			canReadInstaller:      false,
			installersWithoutTeam: false,
		},
		{
			name: "team-observer-plus",
			user: &fleet.User{ID: 1, Teams: []fleet.UserTeam{{
				Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus,
			}}},
			canReadInstaller:      false,
			installersWithoutTeam: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tc.user})
			ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

			title, err := svc.SoftwareTitleByID(ctx, 1, new(uint(1)))
			require.NoError(t, err)
			require.NotNil(t, title.SoftwarePackage)
			require.Len(t, title.Packages, 1)
			require.NotNil(t, title.AppStoreApp)

			// Both the single package and the packages list are serialized.
			for _, pkg := range []*fleet.SoftwareInstaller{title.SoftwarePackage, &title.Packages[0]} {
				if tc.canReadInstaller {
					require.Equal(t, installScript, pkg.InstallScript)
					require.Equal(t, uninstallScript, pkg.UninstallScript)
					require.Equal(t, postInstallScript, pkg.PostInstallScript)
					require.Equal(t, preInstallQuery, pkg.PreInstallQuery)
					require.NotEmpty(t, pkg.Configuration)
				} else {
					require.Empty(t, pkg.InstallScript)
					require.Empty(t, pkg.UninstallScript)
					require.Empty(t, pkg.PostInstallScript)
					require.Empty(t, pkg.PreInstallQuery)
					require.Empty(t, pkg.Configuration)
				}

				// Filtering the whole package would be a regression.
				require.Equal(t, "foo.pkg", pkg.Name)
				require.Equal(t, "1.0", pkg.Version)
				require.Equal(t, "darwin", pkg.Platform)
				require.Equal(t, "abc123", pkg.StorageID)
				require.True(t, pkg.SelfService)
				require.NotNil(t, pkg.Status)
			}

			if tc.canReadInstaller {
				require.NotEmpty(t, title.AppStoreApp.Configuration)
			} else {
				require.Empty(t, title.AppStoreApp.Configuration)
			}
			require.Equal(t, "Bar", title.AppStoreApp.Name)

			// Omitting the fleet must not fail, or a title's existence becomes
			// guessable; the "no team" installer data is what gets withheld.
			ds.TeamExistsFuncInvoked = false
			noTeamTitle, err := svc.SoftwareTitleByID(ctx, 1, nil)
			require.NoError(t, err)
			require.False(t, ds.TeamExistsFuncInvoked)
			if tc.installersWithoutTeam {
				require.NotNil(t, noTeamTitle.SoftwarePackage)
				require.Len(t, noTeamTitle.Packages, 1)
			} else {
				require.Nil(t, noTeamTitle.SoftwarePackage)
				require.Empty(t, noTeamTitle.Packages)
				require.Nil(t, noTeamTitle.AppStoreApp)
			}

			// An explicit "no team" is authorized up front, so it still fails.
			_, err = svc.SoftwareTitleByID(ctx, 1, new(uint(0)))
			checkAuthErr(t, !tc.installersWithoutTeam, err)
			require.False(t, ds.TeamExistsFuncInvoked)
		})
	}
}

func TestSoftwareNameUpdate(t *testing.T) {
	ds := new(mock.Store)
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return nil, &notFoundError{}
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})

	// Title not found
	err := svc.UpdateSoftwareName(ctx, 1, "2 Chrome 2 Furious")
	require.ErrorContains(t, err, "not found")
	require.False(t, ds.UpdateHostSoftwareFuncInvoked)

	// Title found but doesn't have a bundle ID
	title := &fleet.SoftwareTitle{}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return title, nil
	}
	err = svc.UpdateSoftwareName(ctx, 1, "2 Chrome 2 Furious")
	require.ErrorContains(t, err, "bundle")
	require.False(t, ds.UpdateHostSoftwareFuncInvoked)

	// Title found with bundle ID but user didn't provide a name
	title = &fleet.SoftwareTitle{BundleIdentifier: ptr.String("foo")}
	err = svc.UpdateSoftwareName(ctx, 1, "")
	require.ErrorContains(t, err, "name")
	require.False(t, ds.UpdateHostSoftwareFuncInvoked)

	// Success case
	ds.UpdateSoftwareTitleNameFunc = func(ctx context.Context, id uint, name string) error {
		return nil
	}
	err = svc.UpdateSoftwareName(ctx, 1, "2 Chrome 2 Furious")
	require.NoError(t, err)
	require.True(t, ds.UpdateSoftwareTitleNameFuncInvoked)
}

func TestSoftwareTitleByIDTeamIDZero(t *testing.T) {
	ds := new(mock.Store)
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{BundleIdentifier: new("com.example.app")}, nil
	}
	ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) { return true, nil }

	svc, ctx := newTestService(t, ds, nil, nil)

	teamIDZero := new(uint) // *uint pointing to 0

	// Team-scoped user on team 1 should not be able to access software with team_id=0
	teamUser := &fleet.User{
		ID: 1,
		Teams: []fleet.UserTeam{{
			Team: fleet.Team{ID: 1},
			Role: fleet.RoleAdmin,
		}},
	}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: teamUser})

	_, err := svc.SoftwareTitleByID(ctx, 1, teamIDZero)
	checkAuthErr(t, true, err)

	// Global admin should still be able to access software with team_id=0
	globalAdmin := &fleet.User{
		ID:         2,
		GlobalRole: new(fleet.RoleAdmin),
	}
	adminCtx := viewer.NewContext(ctx, viewer.Viewer{User: globalAdmin})

	_, err = svc.SoftwareTitleByID(adminCtx, 1, teamIDZero)
	checkAuthErr(t, false, err)
}

func TestSoftwareTitleByIDNilTeamIDExistsElsewhere(t *testing.T) {
	ds := new(mock.Store)
	var filtersUsed []fleet.TeamFilter
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		filtersUsed = append(filtersUsed, tmFilter)
		return nil, newNotFoundError()
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	teamUser := &fleet.User{
		ID:    1,
		Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}},
	}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: teamUser})

	_, err := svc.SoftwareTitleByID(ctx, 1, nil)
	require.Error(t, err)
	// A title's existence on a team the caller can't see must not be
	// distinguishable (via status code) from it not existing at all.
	require.True(t, fleet.IsNotFound(err), "expected NotFound, got: %v", err)

	// Guard against reintroducing a secondary lookup: exactly one call, and
	// never with a global-role filter standing in for the real caller.
	require.Len(t, filtersUsed, 1)
	require.Equal(t, teamUser, filtersUsed[0].User)
}
