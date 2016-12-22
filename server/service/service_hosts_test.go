package service

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/datastore/inmem"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestListHosts(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := svc.ListHosts(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	_, err = ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)

	hosts, err = svc.ListHosts(ctx, kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestGetHost(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	host, err := ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)
	assert.NotZero(t, host.ID)

	hostVerify, err := svc.GetHost(ctx, host.ID)
	assert.Nil(t, err)

	assert.Equal(t, host.ID, hostVerify.ID)
}

func TestDeleteHost(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)

	svc, err := newTestService(ds, nil)
	assert.Nil(t, err)

	ctx := context.Background()

	host, err := ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)
	assert.NotZero(t, host.ID)

	err = svc.DeleteHost(ctx, host.ID)
	assert.Nil(t, err)

	hosts, err := ds.ListHosts(kolide.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

}

func TestHostStatus(t *testing.T) {
	mockClock := clock.NewMockClock()
	svc, err := newTestServiceWithClock(nil, nil, mockClock)
	require.Nil(t, err)

	assert.Nil(t, err)
	ctx := context.Background()

	host := kolide.Host{}

	host.UpdatedAt = mockClock.Now()
	assert.Equal(t, StatusOnline, svc.HostStatus(ctx, host))

	host.UpdatedAt = mockClock.Now().Add(-1 * time.Minute)
	assert.Equal(t, StatusOnline, svc.HostStatus(ctx, host))

	host.UpdatedAt = mockClock.Now().Add(-1 * time.Hour)
	assert.Equal(t, StatusOffline, svc.HostStatus(ctx, host))

	host.UpdatedAt = mockClock.Now().Add(-24 * 35 * time.Hour) // 35 days
	assert.Equal(t, StatusMIA, svc.HostStatus(ctx, host))
}
