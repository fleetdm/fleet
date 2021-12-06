package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

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
	svc := newTestService(ds, nil, nil)

	ds.GenerateHostStatusStatisticsFunc = func(ctx context.Context, filter fleet.TeamFilter, now time.Time) (*fleet.HostSummary, error) {
		return &fleet.HostSummary{
			OnlineCount:      1,
			OfflineCount:     2,
			MIACount:         3,
			NewCount:         4,
			TotalsHostsCount: 5,
		}, nil
	}

	summary, err := svc.GetHostSummary(test.UserContext(test.UserAdmin), nil)
	require.NoError(t, err)
	require.Nil(t, summary.TeamID)
	require.Equal(t, uint(1), summary.OnlineCount)
	require.Equal(t, uint(2), summary.OfflineCount)
	require.Equal(t, uint(3), summary.MIACount)
	require.Equal(t, uint(4), summary.NewCount)
	require.Equal(t, uint(5), summary.TotalsHostsCount)

	_, err = svc.GetHostSummary(test.UserContext(test.UserNoRoles), nil)
	require.NoError(t, err)

	// a user is required
	_, err = svc.GetHostSummary(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestAddHostsToTeamByFilter(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

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
}

func TestAddHostsToTeamByFilterLabel(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

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
}

func TestAddHostsToTeamByFilterEmptyHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return []*fleet.Host{}, nil
	}
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		t.Error("add hosts func should not have been called")
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), nil, fleet.HostListOptions{}, nil))
}

func TestRefetchHost(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	host := &fleet.Host{ID: 3}

	ds.HostFunc = func(ctx context.Context, hid uint, skipLoadingExtras bool) (*fleet.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		assert.True(t, host.RefetchRequested)
		return nil
	}

	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserAdmin), host.ID))
	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserObserver), host.ID))
	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserMaintainer), host.ID))
}

func TestRefetchHostUserInTeams(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	host := &fleet.Host{ID: 3, TeamID: ptr.Uint(4)}

	ds.HostFunc = func(ctx context.Context, hid uint, skipLoadingExtras bool) (*fleet.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		assert.True(t, host.RefetchRequested)
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

	observer := &fleet.User{
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 4},
				Role: fleet.RoleObserver,
			},
		},
	}
	require.NoError(t, svc.RefetchHost(test.UserContext(observer), host.ID))
}
