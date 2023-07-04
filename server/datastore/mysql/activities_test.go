package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
		{"ListActivitiesStreamed", testListActivitiesStreamed},
		{"EmptyUser", testActivityEmptyUser},
		{"PaginationMetadata", testActivityPaginationMetadata},
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
	require.NoError(t, err)

	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test1",
		details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
	}))
	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test2",
		details: map[string]interface{}{"detail": 2},
	}))

	activities, _, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)

	u.Name = "newname"
	err = ds.SaveUser(context.Background(), u)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "newname", *activities[0].ActorFullName)
	assert.Equal(t, "http://asd.com", *activities[0].ActorGravatar)
	assert.Equal(t, "email@asd.com", *activities[0].ActorEmail)

	err = ds.DeleteUser(context.Background(), u.ID)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)
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

	opt := fleet.ListActivitiesOptions{
		ListOptions: fleet.ListOptions{
			Page:    0,
			PerPage: 1,
		},
	}
	activities, _, err := ds.ListActivities(context.Background(), opt)
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)
	assert.Equal(t, "test1", activities[0].Type)

	opt = fleet.ListActivitiesOptions{
		ListOptions: fleet.ListOptions{
			Page:    1,
			PerPage: 1,
		},
	}
	activities, _, err = ds.ListActivities(context.Background(), opt)
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)
	assert.Equal(t, "test2", activities[0].Type)

	opt = fleet.ListActivitiesOptions{
		ListOptions: fleet.ListOptions{
			Page:    0,
			PerPage: 10,
		},
	}
	activities, _, err = ds.ListActivities(context.Background(), opt)
	require.NoError(t, err)
	assert.Len(t, activities, 2)
}

func testListActivitiesStreamed(t *testing.T, ds *Datastore) {
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
	require.NoError(t, ds.NewActivity(context.Background(), u, dummyActivity{
		name:    "test3",
		details: map[string]interface{}{"detail": 3},
	}))

	activities, _, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 3)

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].ID < activities[j].ID
	})

	err = ds.MarkActivitiesAsStreamed(context.Background(), []uint{activities[0].ID})
	require.NoError(t, err)

	// Reload activities (with streamed field updated).
	activities, _, err = ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 3)
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].ID < activities[j].ID
	})

	nonStreamed, _, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{
		Streamed: ptr.Bool(false),
	})
	require.NoError(t, err)
	assert.Len(t, nonStreamed, 2)
	require.Equal(t, nonStreamed[0], activities[1])
	require.Equal(t, nonStreamed[1], activities[2])

	streamed, _, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{
		Streamed: ptr.Bool(true),
	})
	require.NoError(t, err)
	assert.Len(t, streamed, 1)
	require.Equal(t, streamed[0], activities[0])
}

func testActivityEmptyUser(t *testing.T, ds *Datastore) {
	require.NoError(t, ds.NewActivity(context.Background(), nil, dummyActivity{
		name:    "test1",
		details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
	}))
	activities, _, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 1)
}

func testActivityPaginationMetadata(t *testing.T, ds *Datastore) {
	for i := 0; i < 3; i++ {
		require.NoError(t, ds.NewActivity(context.Background(), nil, dummyActivity{
			name:    fmt.Sprintf("test-%d", i),
			details: map[string]interface{}{},
		}))
	}

	cases := []struct {
		name  string
		opts  fleet.ListOptions
		count int
		meta  *fleet.PaginationMetadata
	}{
		{
			"default options",
			fleet.ListOptions{PerPage: 0},
			3,
			&fleet.PaginationMetadata{},
		},
		{
			"per page 2",
			fleet.ListOptions{PerPage: 2},
			2,
			&fleet.PaginationMetadata{HasNextResults: true},
		},
		{
			"per page 2 - page 1",
			fleet.ListOptions{PerPage: 2, Page: 1},
			1,
			&fleet.PaginationMetadata{HasPreviousResults: true},
		},
		{
			"per page 3",
			fleet.ListOptions{PerPage: 3},
			3,
			&fleet.PaginationMetadata{},
		},
		{
			`after "0" - orderKey "a.id"`,
			fleet.ListOptions{After: "0", OrderKey: "a.id"},
			3,
			nil,
		},
		{
			"per page 4",
			fleet.ListOptions{PerPage: 4},
			3,
			&fleet.PaginationMetadata{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			activities, metadata, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{ListOptions: c.opts})
			require.NoError(t, err)
			assert.Len(t, activities, c.count)
			if c.meta == nil {
				assert.Nil(t, metadata)
			} else {
				require.NotNil(t, metadata)
				assert.Equal(t, c.meta.HasNextResults, metadata.HasNextResults)
				assert.Equal(t, c.meta.HasPreviousResults, metadata.HasPreviousResults)
			}
		})
	}
}
