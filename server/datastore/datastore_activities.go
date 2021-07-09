package datastore

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
	"testing"
)

func testNewActivity(t *testing.T, ds fleet.Datastore) {
	u := &fleet.User{
		Password:   []byte("asd"),
		Email:      "email@asd.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	_, err := ds.NewUser(u)
	require.Nil(t, err)
	ds.NewActivity(u, "test", &map[string]interface{}{"detail": 1, "sometext": "aaa"})
}
