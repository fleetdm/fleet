package mysql

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestActivityUsernameChange(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	u := &fleet.User{
		Password:   []byte("asd"),
		Name:       "fullname",
		Email:      "email@asd.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	_, err := ds.NewUser(u)
	require.Nil(t, err)
	require.NoError(t, ds.NewActivity(u, "test1", &map[string]interface{}{"detail": 1, "sometext": "aaa"}))
	require.NoError(t, ds.NewActivity(u, "test2", &map[string]interface{}{"detail": 2}))

	activities, err := ds.ListActivities(fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", activities[0].ActorFullName)

	u.Name = "newname"
	err = ds.SaveUser(u)
	require.NoError(t, err)

	activities, err = ds.ListActivities(fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "newname", activities[0].ActorFullName)

	err = ds.DeleteUser(u.ID)
	require.NoError(t, err)

	activities, err = ds.ListActivities(fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", activities[0].ActorFullName)
}
