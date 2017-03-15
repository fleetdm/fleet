package service

import (
	"context"
	"testing"

	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/inmem"
	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
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
