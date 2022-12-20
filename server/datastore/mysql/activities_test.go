package mysql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivity(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UsernameChange", testActivityUsernameChange},
		{"New", testActivityNew},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

type dummyActivity struct {
	name    string `json:"-"`
	details map[string]interface{}
}

func (d dummyActivity) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(d.details)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (d dummyActivity) ActivityName() string {
	return d.name
}

func (d dummyActivity) Documentation() (activity string, details string, detailsExample string) {
	return "", "", ""
}

func testActivityUsernameChange(t *testing.T, ds *Datastore) {
	u := &fleet.User{
		Password:    []byte("asd"),
		Name:        "fullname",
		Email:       "email@asd.com",
		GravatarURL: "http://asd.com",
		GlobalRole:  ptr.String(fleet.RoleObserver),
	}
	_, err := ds.NewUser(context.Background(), u)
	require.Nil(t, err)
	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test1",
		details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
	}))
	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test2",
		details: map[string]interface{}{"detail": 2},
	}))

	activities, err := ds.ListActivities(context.Background(), fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", activities[0].ActorFullName)

	u.Name = "newname"
	err = ds.SaveUser(context.Background(), u)
	require.NoError(t, err)

	activities, err = ds.ListActivities(context.Background(), fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "newname", activities[0].ActorFullName)
	assert.Equal(t, "http://asd.com", *activities[0].ActorGravatar)
	assert.Equal(t, "email@asd.com", *activities[0].ActorEmail)

	err = ds.DeleteUser(context.Background(), u.ID)
	require.NoError(t, err)

	activities, err = ds.ListActivities(context.Background(), fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", activities[0].ActorFullName)
	assert.Nil(t, activities[0].ActorGravatar)
}

func testActivityNew(t *testing.T, ds *Datastore) {
	u := &fleet.User{
		Password:   []byte("asd"),
		Name:       "fullname",
		Email:      "email@asd.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	_, err := ds.NewUser(context.Background(), u)
	require.Nil(t, err)
	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test1",
		details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
	}))
	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test2",
		details: map[string]interface{}{"detail": 2},
	}))

	opt := fleet.ListOptions{
		Page:    0,
		PerPage: 1,
	}
	activities, err := ds.ListActivities(context.Background(), opt)
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "fullname", activities[0].ActorFullName)
	assert.Equal(t, "test1", activities[0].Type)

	opt = fleet.ListOptions{
		Page:    1,
		PerPage: 1,
	}
	activities, err = ds.ListActivities(context.Background(), opt)
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "fullname", activities[0].ActorFullName)
	assert.Equal(t, "test2", activities[0].Type)

	opt = fleet.ListOptions{
		Page:    0,
		PerPage: 10,
	}
	activities, err = ds.ListActivities(context.Background(), opt)
	require.NoError(t, err)
	assert.Len(t, activities, 2)
}
