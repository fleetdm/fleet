package server

import (
	"context"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/stretchr/testify/assert"
)

func TestEnrollAgent(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	config := ServiceConfig{
		Datastore:           ds,
		Logger:              kitlog.NewNopLogger(),
		BcryptCost:          12,
		SaltKeySize:         24,
		SessionCookieName:   "KolideSession",
		OsqueryNodeKeySize:  12,
		OsqueryEnrollSecret: "foobar",
	}
	svc, err := NewService(config)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "foobar", "host123")
	assert.Nil(t, err)
	assert.NotEmpty(t, nodeKey)

	hosts, err = ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 1)
}

func TestEnrollAgentIncorrectEnrollSecret(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	config := ServiceConfig{
		Datastore:           ds,
		Logger:              kitlog.NewNopLogger(),
		BcryptCost:          12,
		SaltKeySize:         24,
		SessionCookieName:   "KolideSession",
		OsqueryNodeKeySize:  12,
		OsqueryEnrollSecret: "foobar",
	}
	svc, err := NewService(config)
	assert.Nil(t, err)

	ctx := context.Background()

	hosts, err := ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)

	nodeKey, err := svc.EnrollAgent(ctx, "not_correct", "host123")
	assert.NotNil(t, err)
	assert.Empty(t, nodeKey)

	hosts, err = ds.Hosts()
	assert.Nil(t, err)
	assert.Len(t, hosts, 0)
}
