package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticatedUser(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	createTestUsers(t, ds)
	svc := newTestService(ds, nil, nil)
	admin1, err := ds.UserByEmail(context.Background(), "admin1@example.com")
	assert.Nil(t, err)
	admin1Session, err := ds.NewSession(context.Background(), &fleet.Session{
		UserID: admin1.ID,
		Key:    "admin1",
	})
	assert.Nil(t, err)

	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin1, Session: admin1Session})
	user, err := svc.AuthenticatedUser(ctx)
	assert.Nil(t, err)
	assert.Equal(t, user, admin1)
}
