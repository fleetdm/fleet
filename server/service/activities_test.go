package service

import (
	"context"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func Test_logRoleChangeActivities(t *testing.T) {
	tests := []struct {
		name             string
		oldRole          *string
		newRole          *string
		oldTeamRoles     map[uint]string
		newTeamRoles     map[uint]string
		expectActivities []string
	}{
		{
			name: "Empty",
		}, {
			name:             "AddGlobal",
			newRole:          ptr.String("role"),
			expectActivities: []string{"changed_user_global_role"},
		}, {
			name:             "NoChangeGlobal",
			oldRole:          ptr.String("role"),
			newRole:          ptr.String("role"),
			expectActivities: []string{},
		}, {
			name:             "ChangeGlobal",
			oldRole:          ptr.String("old"),
			newRole:          ptr.String("role"),
			expectActivities: []string{"changed_user_global_role"},
		}, {
			name:             "Delete",
			oldRole:          ptr.String("old"),
			newRole:          nil,
			expectActivities: []string{"deleted_user_global_role"},
		}, {
			name:    "SwitchGlobalToTeams",
			oldRole: ptr.String("old"),
			newTeamRoles: map[uint]string{
				1: "foo",
				2: "bar",
				3: "baz",
			},
			expectActivities: []string{"deleted_user_global_role", "changed_user_team_role", "changed_user_team_role", "changed_user_team_role"},
		}, {
			name: "DeleteModifyTeam",
			oldTeamRoles: map[uint]string{
				1: "foo",
				2: "bar",
				3: "baz",
			},
			newTeamRoles: map[uint]string{
				2: "newRole",
				3: "baz",
			},
			expectActivities: []string{"changed_user_team_role", "deleted_user_team_role"},
		},
	}
	ds := new(mock.Store)
	opts := &TestServerOpts{}
	svc, ctx := newTestService(t, ds, nil, nil, opts)
	var activities []string
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
		activities = append(activities, activity.ActivityName())
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activities = activities[:0]
			oldTeams := make([]fleet.UserTeam, 0, len(tt.oldTeamRoles))
			for id, r := range tt.oldTeamRoles {
				oldTeams = append(oldTeams, fleet.UserTeam{
					Team: fleet.Team{ID: id},
					Role: r,
				})
			}
			newTeams := make([]fleet.UserTeam, 0, len(tt.newTeamRoles))
			for id, r := range tt.newTeamRoles {
				newTeams = append(newTeams, fleet.UserTeam{
					Team: fleet.Team{ID: id},
					Role: r,
				})
			}
			newUser := &fleet.User{
				GlobalRole: tt.newRole,
				Teams:      newTeams,
			}
			require.NoError(t, fleet.LogRoleChangeActivities(ctx, svc, &fleet.User{}, tt.oldRole, oldTeams, newUser))
			require.Equal(t, tt.expectActivities, activities)
		})
	}
}

func TestCancelHostUpcomingActivityAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

	const (
		teamHostID   = 1
		globalHostID = 2
	)

	teamHost := &fleet.Host{TeamID: ptr.Uint(1), Platform: "darwin"}
	globalHost := &fleet.Host{Platform: "darwin"}

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		if hostID == teamHostID {
			return teamHost, nil
		}
		return globalHost, nil
	}
	ds.CancelHostUpcomingActivityFunc = func(ctx context.Context, hostID uint, execID string) (fleet.ActivityDetails, error) {
		return nil, nil
	}
	ds.GetHostUpcomingActivityMetaFunc = func(ctx context.Context, hostID uint, execID string) (*fleet.UpcomingActivityMeta, error) {
		return &fleet.UpcomingActivityMeta{}, nil
	}
	// svc.CancelHostUpcomingActivity now looks up VPP release info before the
	// cancel runs (so pre-activation cancels can still release the reserved
	// seat). This test doesn't exercise VPP installs, so return notFound to
	// skip the release path.
	ds.GetVPPInstallReleaseInfoForCancelFunc = func(ctx context.Context, hostID uint, execID string) (*fleet.VPPInstallReleaseInfo, error) {
		return nil, &notFoundError{}
	}

	cases := []struct {
		name             string
		user             *fleet.User
		shouldFailGlobal bool
		shouldFailTeam   bool
	}{
		{
			name:             "global observer",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team observer",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "global observer plus",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team observer plus",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "global admin",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			shouldFailGlobal: false,
			shouldFailTeam:   false,
		},
		{
			name:             "team admin",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			shouldFailGlobal: true,
			shouldFailTeam:   false,
		},
		{
			name:             "global maintainer",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			shouldFailGlobal: false,
			shouldFailTeam:   false,
		},
		{
			name:             "team maintainer",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			shouldFailGlobal: true,
			shouldFailTeam:   false,
		},
		{
			name:             "team admin wrong team",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 42}, Role: fleet.RoleAdmin}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team maintainer wrong team",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 42}, Role: fleet.RoleMaintainer}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "global gitops",
			user:             &fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
		{
			name:             "team gitops",
			user:             &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			shouldFailGlobal: true,
			shouldFailTeam:   true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.CancelHostUpcomingActivity(ctx, globalHostID, "abc")
			checkAuthErr(t, tt.shouldFailGlobal, err)
			err = svc.CancelHostUpcomingActivity(ctx, teamHostID, "abc")
			checkAuthErr(t, tt.shouldFailTeam, err)
		})
	}
}

func TestGetHostActivitiesWebhookSettings(t *testing.T) {
	newLicenseCtx := func(t *testing.T, tier string) context.Context {
		return license.NewContext(t.Context(), &fleet.LicenseInfo{Tier: tier})
	}

	teamID1, teamID2 := uint(1), uint(2)
	hostInTeam := func(id uint, teamID *uint) *fleet.Host {
		return &fleet.Host{ID: id, TeamID: teamID}
	}

	newDS := func(hosts []*fleet.Host, teamWebhooks map[uint]*fleet.HostActivitiesWebhookSettings, noTeamWebhook *fleet.HostActivitiesWebhookSettings) *mock.Store {
		ds := new(mock.Store)
		ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
			return hosts, nil
		}
		ds.TeamLitesByIDsFunc = func(ctx context.Context, ids []uint) ([]*fleet.TeamLite, error) {
			teams := make([]*fleet.TeamLite, 0, len(ids))
			for _, tid := range ids {
				// ID 0 ("Unassigned") is always present, synthesized from the
				// default team config like the real bulk query.
				if tid == 0 {
					teams = append(teams, &fleet.TeamLite{
						ID:     0,
						Config: fleet.TeamConfigLite{WebhookSettings: fleet.TeamWebhookSettings{HostActivitiesWebhook: noTeamWebhook}},
					})
					continue
				}
				// Fleets absent from teamWebhooks are "deleted": omitted from
				// the result.
				webhook, ok := teamWebhooks[tid]
				if !ok {
					continue
				}
				teams = append(teams, &fleet.TeamLite{
					ID:     tid,
					Config: fleet.TeamConfigLite{WebhookSettings: fleet.TeamWebhookSettings{HostActivitiesWebhook: webhook}},
				})
			}
			return teams, nil
		}
		return ds
	}

	enabled := func(url string) *fleet.HostActivitiesWebhookSettings {
		return &fleet.HostActivitiesWebhookSettings{Enable: true, DestinationURL: url}
	}

	t.Run("free tier returns nil without touching the datastore", func(t *testing.T) {
		ds := newDS([]*fleet.Host{hostInTeam(1, &teamID1)}, map[uint]*fleet.HostActivitiesWebhookSettings{teamID1: enabled("https://example.com")}, nil)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierFree), []uint{1})
		require.NoError(t, err)
		require.Nil(t, settings)
		require.False(t, ds.ListHostsLiteByIDsFuncInvoked)
	})

	t.Run("returns the host's fleet webhook", func(t *testing.T) {
		ds := newDS([]*fleet.Host{hostInTeam(1, &teamID1)}, map[uint]*fleet.HostActivitiesWebhookSettings{teamID1: enabled("https://example.com/a")}, nil)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1})
		require.NoError(t, err)
		require.Len(t, settings, 1)
		require.Equal(t, "https://example.com/a", settings[0].DestinationURL)
	})

	t.Run("dedups hosts in the same fleet", func(t *testing.T) {
		ds := newDS(
			[]*fleet.Host{hostInTeam(1, &teamID1), hostInTeam(2, &teamID1)},
			map[uint]*fleet.HostActivitiesWebhookSettings{teamID1: enabled("https://example.com/a")},
			nil,
		)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1, 2})
		require.NoError(t, err)
		require.Len(t, settings, 1)
		require.Equal(t, []uint{1, 2}, settings[0].HostIDs)
	})

	t.Run("hosts across fleets return one webhook per enabled fleet", func(t *testing.T) {
		ds := newDS(
			[]*fleet.Host{hostInTeam(1, &teamID1), hostInTeam(2, &teamID2), hostInTeam(3, nil)},
			map[uint]*fleet.HostActivitiesWebhookSettings{
				teamID1: enabled("https://example.com/a"),
				teamID2: {Enable: false, DestinationURL: "https://example.com/disabled"},
			},
			enabled("https://example.com/no-team"),
		)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1, 2, 3})
		require.NoError(t, err)
		// Each delivery carries only its own fleet's hosts (the fleet-2 host is
		// absent entirely: that fleet's webhook is disabled).
		hostsByURL := make(map[string][]uint, len(settings))
		for _, s := range settings {
			hostsByURL[s.DestinationURL] = s.HostIDs
		}
		require.Equal(t, map[string][]uint{
			"https://example.com/a":       {1},
			"https://example.com/no-team": {3},
		}, hostsByURL)
	})

	t.Run("fleets sharing a destination URL yield separate scoped deliveries", func(t *testing.T) {
		ds := newDS(
			[]*fleet.Host{hostInTeam(1, &teamID1), hostInTeam(2, &teamID2)},
			map[uint]*fleet.HostActivitiesWebhookSettings{
				teamID1: enabled("https://example.com/shared"),
				teamID2: enabled("https://example.com/shared"),
			},
			nil,
		)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1, 2})
		require.NoError(t, err)
		// A delivery is one fleet's subscription: sharing a URL must not merge
		// fleets' host IDs into one payload.
		require.Len(t, settings, 2)
		require.Equal(t, "https://example.com/shared", settings[0].DestinationURL)
		require.Equal(t, []uint{1}, settings[0].HostIDs)
		require.Equal(t, "https://example.com/shared", settings[1].DestinationURL)
		require.Equal(t, []uint{2}, settings[1].HostIDs)
	})

	t.Run("a fleet and no-fleet sharing a destination stay separate deliveries", func(t *testing.T) {
		ds := newDS(
			[]*fleet.Host{hostInTeam(1, &teamID1), hostInTeam(2, nil)},
			map[uint]*fleet.HostActivitiesWebhookSettings{teamID1: enabled("https://example.com/shared")},
			enabled("https://example.com/shared"),
		)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1, 2})
		require.NoError(t, err)
		require.Len(t, settings, 2)
		require.Equal(t, []uint{1}, settings[0].HostIDs)
		require.Equal(t, []uint{2}, settings[1].HostIDs)
	})

	t.Run("no-team host uses the default team config", func(t *testing.T) {
		ds := newDS([]*fleet.Host{hostInTeam(1, nil)}, nil, enabled("https://example.com/no-team"))
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1})
		require.NoError(t, err)
		require.Len(t, settings, 1)
		require.Equal(t, "https://example.com/no-team", settings[0].DestinationURL)
	})

	t.Run("deleted fleet is skipped", func(t *testing.T) {
		ds := newDS([]*fleet.Host{hostInTeam(1, &teamID1)}, nil, nil) // fleet absent from the bulk lookup
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1})
		require.NoError(t, err)
		require.Empty(t, settings)
	})

	t.Run("nil or empty-URL webhooks are filtered", func(t *testing.T) {
		ds := newDS(
			[]*fleet.Host{hostInTeam(1, &teamID1), hostInTeam(2, &teamID2)},
			map[uint]*fleet.HostActivitiesWebhookSettings{
				teamID1: nil,
				teamID2: {Enable: true, DestinationURL: ""},
			},
			nil,
		)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), []uint{1, 2})
		require.NoError(t, err)
		require.Empty(t, settings)
	})

	t.Run("no host IDs short-circuits", func(t *testing.T) {
		ds := new(mock.Store)
		svc := &Service{ds: ds}
		settings, err := svc.GetHostActivitiesWebhookSettings(newLicenseCtx(t, fleet.TierPremium), nil)
		require.NoError(t, err)
		require.Nil(t, settings)
		require.False(t, ds.ListHostsLiteByIDsFuncInvoked)
	})
}
