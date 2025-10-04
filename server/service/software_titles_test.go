package service

import (
	"context"
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
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			// Update a software title's name
			err = svc.UpdateSoftwareName(ctx, 1, "2 Chrome 2 Furious")
			checkAuthErr(t, tc.shouldFailWrite, err)
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

func TestListJetbrainsPlugins(t *testing.T) {
	ds := new(mock.Store)
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions, tmf fleet.TeamFilter) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return []fleet.SoftwareTitleListResult{
			{
				Name:    "Plugin Name",
				Source:  "jetbrains_plugins",
				Browser: "goland",
			},
		}, 0, &fleet.PaginationMetadata{}, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})

	results, _, _, err := svc.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "Plugin Name", results[0].Name)
	require.Equal(t, "jetbrains_plugins", results[0].Source)
	require.Equal(t, "goland", results[0].ExtensionFor)
	require.Empty(t, results[0].Browser)
}

func TestSoftwareTitleByID(t *testing.T) {
	ds := new(mock.Store)
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{
			ID:      1,
			Name:    "Google Chrome",
			IconUrl: ptr.String("https://example.com/icon.png"),
			Source:  "apps",
			// Browser: "",
			HostsCount:    10,
			VersionsCount: 1,
			Versions: []fleet.SoftwareVersion{
				{
					ID:         1,
					Version:    "89.0.4389.90",
					HostsCount: ptr.Uint(10),
					TitleID:    1,
				},
			},
		}, nil
	}
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})

	title, err := svc.SoftwareTitleByID(ctx, 1, nil)
	require.NoError(t, err)
	require.Equal(t, uint(1), title.ID)
	require.Equal(t, "Google Chrome", title.Name)
	require.Equal(t, ptr.String("https://example.com/icon.png"), title.IconUrl)
	require.Equal(t, "apps", title.Source)
	require.Equal(t, "", title.Browser)
	require.Equal(t, uint(10), title.HostsCount)
	require.Equal(t, uint(1), title.VersionsCount)
	require.Len(t, title.Versions, 1)
	require.Equal(t, uint(1), title.Versions[0].ID)
	require.Equal(t, "89.0.4389.90", title.Versions[0].Version)
	require.Equal(t, ptr.Uint(10), title.Versions[0].HostsCount)
	require.Equal(t, uint(1), title.Versions[0].TitleID)
}

func TestSoftwareTitleByIDWithJetbrainsPlugin(t *testing.T) {
	ds := new(mock.Store)
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
		return &fleet.SoftwareTitle{
			ID:            1,
			Name:          "Some Plugin",
			IconUrl:       ptr.String("https://example.com/icon.png"),
			Source:        "jetbrains_plugins",
			Browser:       "goland",
			HostsCount:    10,
			VersionsCount: 1,
			Versions: []fleet.SoftwareVersion{
				{
					ID:         1,
					Version:    "1.2.3",
					HostsCount: ptr.Uint(10),
					TitleID:    1,
				},
			},
		}, nil
	}
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{
		ID:         1,
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})

	title, err := svc.SoftwareTitleByID(ctx, 1, nil)
	require.NoError(t, err)
	require.Equal(t, "Some Plugin", title.Name)
	require.Equal(t, "jetbrains_plugins", title.Source)
	require.Equal(t, "goland", title.ExtensionFor)
	require.Empty(t, title.Browser)
}
