package service

import (
	"testing"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/inmem"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/fleetdm/fleet/server/ptr"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListHosts(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc := newTestService(ds, nil, nil)

	hosts, err := svc.ListHosts(test.UserContext(test.UserAdmin), kolide.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	_, err = ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)

	hosts, err = svc.ListHosts(test.UserContext(test.UserAdmin), kolide.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestDeleteHost(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc := newTestService(ds, nil, nil)

	host, err := ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)
	assert.NotZero(t, host.ID)

	err = svc.DeleteHost(test.UserContext(test.UserAdmin), host.ID)
	assert.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

}

func TestHostDetails(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{ds: ds}

	host := &kolide.Host{ID: 3}
	expectedLabels := []*kolide.Label{
		{
			Name:        "foobar",
			Description: "the foobar label",
		},
	}
	ds.ListLabelsForHostFunc = func(hid uint) ([]*kolide.Label, error) {
		return expectedLabels, nil
	}
	expectedPacks := []*kolide.Pack{
		{
			Name: "pack1",
		},
		{
			Name: "pack2",
		},
	}
	ds.ListPacksForHostFunc = func(hid uint) ([]*kolide.Pack, error) {
		return expectedPacks, nil
	}
	ds.LoadHostSoftwareFunc = func(host *kolide.Host) error {
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

	host := &kolide.Host{ID: 3}

	ds.HostFunc = func(hid uint) (*kolide.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(host *kolide.Host) error {
		assert.True(t, host.RefetchRequested)
		return nil
	}

	require.NoError(t, svc.RefetchHost(test.UserContext(test.UserAdmin), host.ID))
}

func TestAddHostsToTeamByFilter(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedHostIDs := []uint{1, 2, 4}
	expectedTeam := (*uint)(nil)

	ds.ListHostsFunc = func(opt kolide.HostListOptions) ([]*kolide.Host, error) {
		var hosts []*kolide.Host
		for _, id := range expectedHostIDs {
			hosts = append(hosts, &kolide.Host{ID: id})
		}
		return hosts, nil
	}
	ds.AddHostsToTeamFunc = func(teamID *uint, hostIDs []uint) error {
		assert.Equal(t, expectedTeam, teamID)
		assert.Equal(t, expectedHostIDs, hostIDs)
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), expectedTeam, kolide.HostListOptions{}, nil))
}

func TestAddHostsToTeamByFilterLabel(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	expectedHostIDs := []uint{6}
	expectedTeam := ptr.Uint(1)
	expectedLabel := ptr.Uint(2)

	ds.ListHostsInLabelFunc = func(lid uint, opt kolide.HostListOptions) ([]*kolide.Host, error) {
		assert.Equal(t, *expectedLabel, lid)
		var hosts []*kolide.Host
		for _, id := range expectedHostIDs {
			hosts = append(hosts, &kolide.Host{ID: id})
		}
		return hosts, nil
	}
	ds.AddHostsToTeamFunc = func(teamID *uint, hostIDs []uint) error {
		assert.Equal(t, expectedHostIDs, hostIDs)
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), expectedTeam, kolide.HostListOptions{}, expectedLabel))
}

func TestAddHostsToTeamByFilterEmptyHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListHostsFunc = func(opt kolide.HostListOptions) ([]*kolide.Host, error) {
		return []*kolide.Host{}, nil
	}
	ds.AddHostsToTeamFunc = func(teamID *uint, hostIDs []uint) error {
		t.Error("add hosts func should not have been called")
		return nil
	}

	require.NoError(t, svc.AddHostsToTeamByFilter(test.UserContext(test.UserAdmin), nil, kolide.HostListOptions{}, nil))
}
