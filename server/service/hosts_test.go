package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostDetails(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{ds: ds}

	host := &fleet.Host{ID: 3}
	expectedLabels := []*fleet.Label{
		{
			Name:        "foobar",
			Description: "the foobar label",
		},
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return expectedLabels, nil
	}
	expectedPacks := []*fleet.Pack{
		{
			Name: "pack1",
		},
		{
			Name: "pack2",
		},
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return expectedPacks, nil
	}
	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	dsBats := []*fleet.HostBattery{{HostID: host.ID, SerialNumber: "a", CycleCount: 999, Health: "Check Battery"}, {HostID: host.ID, SerialNumber: "b", CycleCount: 1001, Health: "Good"}}
	ds.ListHostBatteriesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostBattery, error) {
		return dsBats, nil
	}
	// Health should be replaced at the service layer with custom values determined by the cycle count. See https://github.com/fleetdm/fleet/issues/6763.
	expectedBats := []*fleet.HostBattery{{HostID: host.ID, SerialNumber: "a", CycleCount: 999, Health: "Normal"}, {HostID: host.ID, SerialNumber: "b", CycleCount: 1001, Health: "Replacement recommended"}}

	opts := fleet.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  false,
	}
	hostDetail, err := svc.getHostDetails(test.UserContext(test.UserAdmin), host, opts)
	require.NoError(t, err)
	assert.Equal(t, expectedLabels, hostDetail.Labels)
	assert.Equal(t, expectedPacks, hostDetail.Packs)
	require.NotNil(t, hostDetail.Batteries)
	assert.Equal(t, expectedBats, *hostDetail.Batteries)
}

func TestHostAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	teamHost := &fleet.Host{TeamID: ptr.Uint(1)}
	globalHost := &fleet.Host{}

	ds.DeleteHostFunc = func(ctx context.Context, hid uint) error {
		return nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id == 1 {
			return teamHost, nil
		}
		return globalHost, nil
	}
	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id == 1 {
			return teamHost, nil
		}
		return globalHost, nil
	}
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		if identifier == "1" {
			return teamHost, nil
		}
		return globalHost, nil
	}
	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return nil, nil
	}
	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
		return nil, nil
	}
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		return nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostBattery, error) {
		return nil, nil
	}
	ds.DeleteHostsFunc = func(ctx context.Context, ids []uint) error {
		return nil
	}
	ds.UpdateHostRefetchRequestedFunc = func(ctx context.Context, id uint, value bool) error {
		if id == 1 {
			teamHost.RefetchRequested = true
		} else {
			globalHost.RefetchRequested = true
		}
		return nil
	}

	testCases := []struct {
		name                  string
		user                  *fleet.User
		shouldFailGlobalWrite bool
		shouldFailGlobalRead  bool
		shouldFailTeamWrite   bool
		shouldFailTeamRead    bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			false,
			true,
			false,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			false,
			false,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			opts := fleet.HostDetailOptions{
				IncludeCVEScores: false,
				IncludePolicies:  false,
			}

			_, err := svc.GetHost(ctx, 1, opts)
			checkAuthErr(t, tt.shouldFailTeamRead, err)

			_, err = svc.HostByIdentifier(ctx, "1", opts)
			checkAuthErr(t, tt.shouldFailTeamRead, err)

			_, err = svc.GetHost(ctx, 2, opts)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			_, err = svc.HostByIdentifier(ctx, "2", opts)
			checkAuthErr(t, tt.shouldFailGlobalRead, err)

			err = svc.DeleteHost(ctx, 1)
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			err = svc.DeleteHost(ctx, 2)
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			err = svc.DeleteHosts(ctx, []uint{1}, fleet.HostListOptions{}, nil)
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			err = svc.DeleteHosts(ctx, []uint{2}, fleet.HostListOptions{}, nil)
			checkAuthErr(t, tt.shouldFailGlobalWrite, err)

			err = svc.AddHostsToTeam(ctx, ptr.Uint(1), []uint{1})
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			err = svc.AddHostsToTeamByFilter(ctx, ptr.Uint(1), fleet.HostListOptions{}, nil)
			checkAuthErr(t, tt.shouldFailTeamWrite, err)

			err = svc.RefetchHost(ctx, 1)
			checkAuthErr(t, tt.shouldFailTeamRead, err)
		})
	}

	// List, GetHostSummary work for all
}

func TestListHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return []*fleet.Host{
			{ID: 1},
		}, nil
	}

	hosts, err := svc.ListHosts(test.UserContext(test.UserAdmin), fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	// anyone can list hosts
	hosts, err = svc.ListHosts(test.UserContext(test.UserNoRoles), fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	// a user is required
	_, err = svc.ListHosts(context.Background(), fleet.HostListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestGetHostSummary(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	ds.GenerateHostStatusStatisticsFunc = func(ctx context.Context, filter fleet.TeamFilter, now time.Time, platform *string) (*fleet.HostSummary, error) {
		return &fleet.HostSummary{
			OnlineCount:      1,
			OfflineCount:     5, // offline hosts also includes mia hosts as of Fleet 4.15
			MIACount:         3,
			NewCount:         4,
			TotalsHostsCount: 5,
			Platforms:        []*fleet.HostSummaryPlatform{{Platform: "darwin", HostsCount: 1}, {Platform: "debian", HostsCount: 2}, {Platform: "centos", HostsCount: 3}, {Platform: "ubuntu", HostsCount: 4}},
		}, nil
	}
	ds.LabelsSummaryFunc = func(ctx context.Context) ([]*fleet.LabelSummary, error) {
		return []*fleet.LabelSummary{{ID: 1, Name: "All hosts", Description: "All hosts enrolled in Fleet", LabelType: fleet.LabelTypeBuiltIn}, {ID: 10, Name: "Other label", Description: "Not a builtin label", LabelType: fleet.LabelTypeRegular}}, nil
	}

	summary, err := svc.GetHostSummary(test.UserContext(test.UserAdmin), nil, nil)
	require.NoError(t, err)
	require.Nil(t, summary.TeamID)
	require.Equal(t, uint(1), summary.OnlineCount)
	require.Equal(t, uint(5), summary.OfflineCount)
	require.Equal(t, uint(3), summary.MIACount)
	require.Equal(t, uint(4), summary.NewCount)
	require.Equal(t, uint(5), summary.TotalsHostsCount)
	require.Len(t, summary.Platforms, 4)
	require.Equal(t, uint(9), summary.AllLinuxCount)
	require.Len(t, summary.BuiltinLabels, 1)
	require.Equal(t, "All hosts", summary.BuiltinLabels[0].Name)

	_, err = svc.GetHostSummary(test.UserContext(test.UserNoRoles), nil, nil)
	require.NoError(t, err)

	// a user is required
	_, err = svc.GetHostSummary(context.Background(), nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestDeleteHost(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(t, ds, nil, nil)

	mockClock := clock.NewMockClock()
	host := test.NewHost(t, ds, "foo", "192.168.1.10", "1", "1", mockClock.Now())
	assert.NotZero(t, host.ID)

	err := svc.DeleteHost(test.UserContext(test.UserAdmin), host.ID)
	assert.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	hosts, err := ds.ListHosts(context.Background(), filter, fleet.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)
}

func TestAddHostsToTeamByFilter(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	expectedHostIDs := []uint{1, 2, 4}
	expectedTeam := (*uint)(nil)

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		var hosts []*fleet.Host
		for _, id := range expectedHostIDs {
			hosts = append(hosts, &fleet.Host{ID: id})
		}
		return hosts, nil
	}
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		assert.Equal(t, expectedTeam, teamID)
		assert.Equal(t, expectedHostIDs, hostIDs)
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), expectedTeam, fleet.HostListOptions{}, nil))
	assert.True(t, ds.ListHostsFuncInvoked)
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
}

func TestAddHostsToTeamByFilterLabel(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	expectedHostIDs := []uint{6}
	expectedTeam := ptr.Uint(1)
	expectedLabel := ptr.Uint(2)

	ds.ListHostsInLabelFunc = func(ctx context.Context, filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		assert.Equal(t, *expectedLabel, lid)
		var hosts []*fleet.Host
		for _, id := range expectedHostIDs {
			hosts = append(hosts, &fleet.Host{ID: id})
		}
		return hosts, nil
	}
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		assert.Equal(t, expectedHostIDs, hostIDs)
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), expectedTeam, fleet.HostListOptions{}, expectedLabel))
	assert.True(t, ds.ListHostsInLabelFuncInvoked)
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
}

func TestAddHostsToTeamByFilterEmptyHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return []*fleet.Host{}, nil
	}
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), nil, fleet.HostListOptions{}, nil))
	assert.True(t, ds.ListHostsFuncInvoked)
	assert.False(t, ds.AddHostsToTeamFuncInvoked)
}

func TestRefetchHost(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	host := &fleet.Host{ID: 3}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.UpdateHostRefetchRequestedFunc = func(ctx context.Context, id uint, value bool) error {
		assert.Equal(t, host.ID, id)
		assert.True(t, value)
		return nil
	}

	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserAdmin), host.ID))
	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserObserver), host.ID))
	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserMaintainer), host.ID))
	assert.True(t, ds.HostLiteFuncInvoked)
	assert.True(t, ds.UpdateHostRefetchRequestedFuncInvoked)
}

func TestRefetchHostUserInTeams(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	host := &fleet.Host{ID: 3, TeamID: ptr.Uint(4)}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.UpdateHostRefetchRequestedFunc = func(ctx context.Context, id uint, value bool) error {
		assert.Equal(t, host.ID, id)
		assert.True(t, value)
		return nil
	}

	maintainer := &fleet.User{
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 4},
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, svc.RefetchHost(test.UserContext(maintainer), host.ID))
	assert.True(t, ds.HostLiteFuncInvoked)
	assert.True(t, ds.UpdateHostRefetchRequestedFuncInvoked)
	ds.HostLiteFuncInvoked, ds.UpdateHostRefetchRequestedFuncInvoked = false, false

	observer := &fleet.User{
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 4},
				Role: fleet.RoleObserver,
			},
		},
	}
	require.NoError(t, svc.RefetchHost(test.UserContext(observer), host.ID))
	assert.True(t, ds.HostLiteFuncInvoked)
	assert.True(t, ds.UpdateHostRefetchRequestedFuncInvoked)
}

func TestEmptyTeamOSVersions(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	testVersions := []fleet.OSVersion{{HostsCount: 1, Name: "macOS 12.1", Platform: "darwin"}}

	ds.TeamFunc = func(ctx context.Context, teamID uint) (*fleet.Team, error) {
		if teamID == 1 {
			return &fleet.Team{
				Name: "team1",
			}, nil
		}
		if teamID == 2 {
			return &fleet.Team{
				Name: "team2",
			}, nil
		}

		return nil, notFoundError{}
	}

	ds.OSVersionsFunc = func(ctx context.Context, teamID *uint, platform *string, osID *uint) (*fleet.OSVersions, error) {
		if *teamID == 1 {
			return &fleet.OSVersions{CountsUpdatedAt: time.Now(), OSVersions: testVersions}, nil
		}
		if *teamID == 4 {
			return nil, errors.New("some unknown error")
		}

		return nil, notFoundError{}
	}

	// team exists with stats
	vers, err := svc.OSVersions(test.UserContext(test.UserAdmin), ptr.Uint(1), ptr.String("darwin"), nil)
	require.NoError(t, err)
	assert.Len(t, vers.OSVersions, 1)

	// team exists but no stats
	vers, err = svc.OSVersions(test.UserContext(test.UserAdmin), ptr.Uint(2), ptr.String("darwin"), nil)
	require.NoError(t, err)
	assert.Empty(t, vers.OSVersions)

	// team does not exist
	_, err = svc.OSVersions(test.UserContext(test.UserAdmin), ptr.Uint(3), ptr.String("darwin"), nil)
	require.Error(t, err)
	require.Equal(t, "not found", fmt.Sprint(err))

	// some unknown error
	_, err = svc.OSVersions(test.UserContext(test.UserAdmin), ptr.Uint(4), ptr.String("darwin"), nil)
	require.Error(t, err)
	require.Equal(t, "some unknown error", fmt.Sprint(err))
}
