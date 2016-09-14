package server

import (
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestGetAllHosts(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := svc.GetAllHosts(ctx)
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	_, err = ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)

	hosts, err = svc.GetAllHosts(ctx)
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestGetHost(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
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

func TestNewHost(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
	assert.Nil(t, err)

	ctx := context.Background()

	hostname := "foo"
	_, err = svc.NewHost(ctx, kolide.HostPayload{
		HostName: &hostname,
	})

	assert.Nil(t, err)

	hosts, err := ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestModifyHost(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
	assert.Nil(t, err)

	ctx := context.Background()

	host, err := ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)
	assert.NotZero(t, host.ID)

	newHostname := "bar"
	hostVerify, err := svc.ModifyHost(ctx, host.ID, kolide.HostPayload{
		HostName: &newHostname,
	})
	assert.Nil(t, err)

	assert.Equal(t, host.ID, hostVerify.ID)
	assert.Equal(t, "bar", hostVerify.HostName)
}

func TestDeleteHost(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
	assert.Nil(t, err)

	ctx := context.Background()

	host, err := ds.NewHost(&kolide.Host{
		HostName: "foo",
	})
	assert.Nil(t, err)
	assert.NotZero(t, host.ID)

	err = svc.DeleteHost(ctx, host.ID)
	assert.Nil(t, err)

	hosts, err := ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

}
