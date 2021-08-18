package service

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListHosts(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)

	hosts, err := svc.ListHosts(test.UserContext(test.UserAdmin), fleet.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	storedTime := time.Now().UTC()

	_, err = ds.NewHost(&fleet.Host{
		Hostname:        "foo",
		SeenTime:        storedTime,
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
	})
	require.NoError(t, err)

	hosts, err = svc.ListHosts(test.UserContext(test.UserAdmin), fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	format := "%Y-%m-%d %HH:%MM:%SS %Z"
	assert.Equal(t, storedTime.Format(format), hosts[0].SeenTime.Format(format))
}

func TestDeleteHost(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)

	mockClock := clock.NewMockClock()
	host := test.NewHost(t, ds, "foo", "192.168.1.10", "1", "1", mockClock.Now())
	assert.NotZero(t, host.ID)

	err := svc.DeleteHost(test.UserContext(test.UserAdmin), host.ID)
	assert.Nil(t, err)

	filter := fleet.TeamFilter{User: test.UserAdmin}
	hosts, err := ds.ListHosts(filter, fleet.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)
}

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
	ds.ListLabelsForHostFunc = func(hid uint) ([]*fleet.Label, error) {
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
	ds.ListPacksForHostFunc = func(hid uint) ([]*fleet.Pack, error) {
		return expectedPacks, nil
	}
	ds.LoadHostSoftwareFunc = func(host *fleet.Host) error {
		return nil
	}

	hostDetail, err := svc.getHostDetails(test.UserContext(test.UserAdmin), host)
	require.NoError(t, err)
	assert.Equal(t, expectedLabels, hostDetail.Labels)
	assert.Equal(t, expectedPacks, hostDetail.Packs)
}

func TestRefetchHost(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	host := &fleet.Host{ID: 3}

	ds.HostFunc = func(hid uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(host *fleet.Host) error {
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

	ds.HostFunc = func(hid uint) (*fleet.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(host *fleet.Host) error {
		assert.True(t, host.RefetchRequested)
		return nil
	}

	maintainer := &fleet.User{
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 4},
				Role: fleet.RoleMaintainer,
			},
		}}
	require.NoError(t, svc.RefetchHost(test.UserContext(maintainer), host.ID))

	observer := &fleet.User{
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 4},
				Role: fleet.RoleObserver,
			},
		}}
	require.NoError(t, svc.RefetchHost(test.UserContext(observer), host.ID))
}

func TestAddHostsToTeamByFilter(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedHostIDs := []uint{1, 2, 4}
	expectedTeam := (*uint)(nil)

	ds.ListHostsFunc = func(filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		var hosts []*fleet.Host
		for _, id := range expectedHostIDs {
			hosts = append(hosts, &fleet.Host{ID: id})
		}
		return hosts, nil
	}
	ds.AddHostsToTeamFunc = func(teamID *uint, hostIDs []uint) error {
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

	ds.ListHostsInLabelFunc = func(filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		assert.Equal(t, *expectedLabel, lid)
		var hosts []*fleet.Host
		for _, id := range expectedHostIDs {
			hosts = append(hosts, &fleet.Host{ID: id})
		}
		return hosts, nil
	}
	ds.AddHostsToTeamFunc = func(teamID *uint, hostIDs []uint) error {
		assert.Equal(t, expectedHostIDs, hostIDs)
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), expectedTeam, fleet.HostListOptions{}, expectedLabel))
}

func TestAddHostsToTeamByFilterEmptyHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListHostsFunc = func(filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return []*fleet.Host{}, nil
	}
	ds.AddHostsToTeamFunc = func(teamID *uint, hostIDs []uint) error {
		t.Error("add hosts func should not have been called")
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), nil, fleet.HostListOptions{}, nil))
}
