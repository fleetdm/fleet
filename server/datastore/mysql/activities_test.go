package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"ListHostUpcomingActivities", testListHostUpcomingActivities},
		{"ListHostPastActivities", testListHostPastActivities},
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
	hostIDs []uint
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

func (d dummyActivity) HostIDs() []uint {
	return d.hostIDs
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

func testListHostUpcomingActivities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	// create three hosts
	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	h3 := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now())

	// create a couple of named scripts
	scr1, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "A",
		ScriptContents: "A",
	})
	require.NoError(t, err)
	scr2, err := ds.NewScript(ctx, &fleet.Script{
		Name:           "B",
		ScriptContents: "B",
	})
	require.NoError(t, err)

	// create some script requests for h1
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptID: &scr1.ID, ScriptContents: scr1.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h1A := hsr.ExecutionID
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptID: &scr2.ID, ScriptContents: scr2.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h1B := hsr.ExecutionID
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "C", UserID: &u.ID})
	require.NoError(t, err)
	h1C := hsr.ExecutionID
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "D"})
	require.NoError(t, err)
	h1D := hsr.ExecutionID
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "E"})
	require.NoError(t, err)
	h1E := hsr.ExecutionID

	// create a single pending request for h2, as well as a non-pending one
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h2.ID, ScriptID: &scr1.ID, ScriptContents: scr1.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h2A := hsr.ExecutionID
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h2.ID, ScriptContents: "F", UserID: &u.ID})
	require.NoError(t, err)
	_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{HostID: h2.ID, ExecutionID: hsr.ExecutionID, Output: "ok", ExitCode: 0})
	require.NoError(t, err)
	h2F := hsr.ExecutionID

	// no script request for h3

	execIDsWithUser := map[string]bool{
		h1A: true,
		h1B: true,
		h1C: true,
		h1D: false,
		h1E: false,
		h2A: true,
		h2F: true,
	}
	execIDsScriptName := map[string]string{
		h1A: scr1.Name,
		h1B: scr2.Name,
		h2A: scr1.Name,
	}

	cases := []struct {
		opts      fleet.ListOptions
		hostID    uint
		wantExecs []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			opts:      fleet.ListOptions{PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1A, h1B},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 5},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1C, h1D},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 5},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 5},
		},
		{
			opts:      fleet.ListOptions{PerPage: 3},
			hostID:    h1.ID,
			wantExecs: []string{h1A, h1B, h1C},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 5},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 3},
			hostID:    h1.ID,
			wantExecs: []string{h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 5},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 3},
			hostID:    h1.ID,
			wantExecs: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 5},
		},
		{
			opts:      fleet.ListOptions{PerPage: 3},
			hostID:    h2.ID,
			wantExecs: []string{h2A},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false, TotalResults: 1},
		},
		{
			opts:      fleet.ListOptions{},
			hostID:    h3.ID,
			wantExecs: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false, TotalResults: 0},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %#v", c.hostID, c.opts), func(t *testing.T) {
			// always include metadata
			c.opts.IncludeMetadata = true
			c.opts.OrderKey = "created_at"
			c.opts.OrderDirection = fleet.OrderAscending

			acts, meta, err := ds.ListHostUpcomingActivities(ctx, c.hostID, c.opts)
			require.NoError(t, err)

			require.Equal(t, len(c.wantExecs), len(acts))
			require.Equal(t, c.wantMeta, meta)

			for i, a := range acts {
				wantExec := c.wantExecs[i]

				var details map[string]any
				require.NotNil(t, a.Details, "result %d", i)
				require.NoError(t, json.Unmarshal([]byte(*a.Details), &details), "result %d", i)

				require.Equal(t, wantExec, details["script_execution_id"], "result %d", i)
				require.Equal(t, c.hostID, uint(details["host_id"].(float64)), "result %d", i)
				require.Equal(t, execIDsScriptName[wantExec], details["script_name"], "result %d", i)
				if execIDsWithUser[wantExec] {
					require.NotNil(t, a.ActorID, "result %d", i)
					require.Equal(t, u.ID, *a.ActorID, "result %d", i)
					require.NotNil(t, a.ActorFullName, "result %d", i)
					require.Equal(t, u.Name, *a.ActorFullName, "result %d", i)
					require.NotNil(t, a.ActorEmail, "result %d", i)
					require.Equal(t, u.Email, *a.ActorEmail, "result %d", i)
				} else {
					require.Nil(t, a.ActorID, "result %d", i)
					require.Nil(t, a.ActorFullName, "result %d", i)
					require.Nil(t, a.ActorEmail, "result %d", i)
				}
			}
		})
	}
}

func testListHostPastActivities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	getDetails := func(a *fleet.Activity) map[string]any {
		details := make(map[string]any)
		err := json.Unmarshal([]byte(*a.Details), &details)
		require.NoError(t, err)

		return details
	}

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)
	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	activities := []dummyActivity{
		{
			name:    "ran_script",
			details: map[string]any{"host_id": float64(h1.ID), "host_display_name": h1.DisplayName(), "script_execution_id": "exec_1", "script_name": "script_1.sh", "async": true},
			hostIDs: []uint{h1.ID},
		},

		{
			name:    "ran_script",
			details: map[string]any{"host_id": float64(h1.ID), "host_display_name": h1.DisplayName(), "script_execution_id": "exec_2", "async": false},
			hostIDs: []uint{h1.ID},
		},
	}

	for _, a := range activities {
		require.NoError(t, ds.NewActivity(context.Background(), u, a))
	}

	cases := []struct {
		name    string
		expActs []dummyActivity
		opts    fleet.ListActivitiesOptions
		expMeta *fleet.PaginationMetadata
	}{
		{
			name:    "fetch page one",
			expActs: []dummyActivity{activities[0]},
			expMeta: &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
			opts: fleet.ListActivitiesOptions{
				ListOptions: fleet.ListOptions{
					Page:    0,
					PerPage: 1,
				},
			},
		},
		{
			name:    "fetch page two",
			expActs: []dummyActivity{activities[1]},
			expMeta: &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
			opts: fleet.ListActivitiesOptions{
				ListOptions: fleet.ListOptions{
					Page:    1,
					PerPage: 1,
				},
			},
		},
		{
			name:    "fetch all activities",
			expActs: activities,
			expMeta: &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
			opts: fleet.ListActivitiesOptions{
				ListOptions: fleet.ListOptions{
					Page:    0,
					PerPage: 2,
				},
			},
		},
	}

	for _, c := range cases {
		c.opts.ListOptions.IncludeMetadata = true
		acts, meta, err := ds.ListHostPastActivities(ctx, h1.ID, c.opts.ListOptions)
		require.NoError(t, err)
		require.Len(t, acts, len(c.expActs))
		require.Equal(t, c.expMeta, meta)

		// check fields in activities
		for i, ra := range acts {
			require.Equal(t, u.Email, *ra.ActorEmail)
			require.Equal(t, u.Name, *ra.ActorFullName)
			require.Equal(t, "ran_script", ra.Type)
			require.Equal(t, u.GravatarURL, *ra.ActorGravatar)
			require.Equal(t, u.ID, *ra.ActorID)
			details := getDetails(ra)
			for k, v := range details {
				require.Equal(t, c.expActs[i].details[k], v)
			}
		}
	}
}
