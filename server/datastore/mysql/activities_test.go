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

func testActivityUsernameChange(t *testing.T, ds *Datastore) {
	u := &fleet.User{
		Password:    []byte("asd"),
		Name:        "fullname",
		Email:       "email@asd.com",
		GravatarURL: "http://asd.com",
		GlobalRole:  ptr.String(fleet.RoleObserver),
	}
	u, err := ds.NewUser(context.Background(), u)
	require.Nil(t, err)

	u2 := &fleet.User{
		Password:    []byte("asd2"),
		Name:        "fullname2",
		Email:       "email2@asd.com",
		GravatarURL: "http://asd2.com",
		GlobalRole:  ptr.String(fleet.RoleObserver),
	}
	u2, err = ds.NewUser(context.Background(), u2)
	require.Nil(t, err)

	details1 := map[string]interface{}{"detail": 1.0, "sometext": "aaa"}
	activity1, err := ds.NewActivity(context.Background(), u, "test1", &details1)
	require.NoError(t, err)
	checkActivity(t, "test1", activity1, u, details1)

	details2 := map[string]interface{}{"detail": 2.0}
	activity2, err := ds.NewActivity(context.Background(), u2, "test2", &details2)
	require.NoError(t, err)
	checkActivity(t, "test2", activity2, u2, details2)

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
	err = ds.DeleteUser(context.Background(), u2.ID)
	require.NoError(t, err)

	activities, err = ds.ListActivities(context.Background(), fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Contains(t, activities[0].ActorFullName, "fullname")
	assert.Nil(t, activities[0].ActorGravatar)
}

func checkActivity(
	t *testing.T,
	actType string,
	activity *fleet.Activity,
	author *fleet.User,
	details map[string]interface{},
) {
	require.Equal(t, actType, activity.Type)
	require.NotZero(t, activity.ID)
	require.NotZero(t, activity.CreatedAt)
	require.Equal(t, author.Name, activity.ActorFullName)
	require.NotNil(t, activity.ActorID)
	require.Equal(t, author.ID, *activity.ActorID)
	require.NotNil(t, activity.ActorGravatar)
	require.Equal(t, author.GravatarURL, *activity.ActorGravatar)
	require.NotNil(t, activity.ActorEmail)
	require.Equal(t, author.Email, *activity.ActorEmail)
	require.NotNil(t, activity.Details)
	var activity1Details map[string]interface{}
	err := json.Unmarshal([]byte(*activity.Details), &activity1Details)
	require.NoError(t, err)
	require.Equal(t, details, activity1Details)
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
	details1 := map[string]interface{}{"detail": 1.0, "sometext": "aaa"}
	activity1, err := ds.NewActivity(context.Background(), u, "test1", &details1)
	require.NoError(t, err)
	checkActivity(t, "test1", activity1, u, details1)
	details2 := map[string]interface{}{"detail": 2.0}
	activity2, err := ds.NewActivity(context.Background(), u, "test2", &details2)
	require.NoError(t, err)
	checkActivity(t, "test2", activity2, u, details2)

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
