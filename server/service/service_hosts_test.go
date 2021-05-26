package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/inmem"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListHosts(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := svc.ListHosts(ctx, kolide.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	_, err = ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)

	hosts, err = svc.ListHosts(ctx, kolide.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestDeleteHost(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	host, err := ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)
	assert.NotZero(t, host.ID)

	err = svc.DeleteHost(ctx, host.ID)
	assert.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.HostListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

}

func TestHostDetails(t *testing.T) {
	ds := new(mock.Store)
	svc := service{ds: ds}

	host := &kolide.Host{ID: 3}
	ctx := context.Background()
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

	hostDetail, err := svc.getHostDetails(ctx, host)
	require.NoError(t, err)
	assert.Equal(t, expectedLabels, hostDetail.Labels)
	assert.Equal(t, expectedPacks, hostDetail.Packs)
}

func TestRefetchHost(t *testing.T) {
	ds := new(mock.Store)
	svc := service{ds: ds}

	host := &kolide.Host{ID: 3}
	ctx := context.Background()

	ds.HostFunc = func(hid uint) (*kolide.Host, error) {
		return host, nil
	}
	ds.SaveHostFunc = func(host *kolide.Host) error {
		assert.True(t, host.RefetchRequested)
		return nil
	}

	require.NoError(t, svc.RefetchHost(ctx, host.ID))
}
