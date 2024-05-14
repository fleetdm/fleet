package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
		{"CleanupActivitiesAndAssociatedData", testCleanupActivitiesAndAssociatedData},
		{"CleanupActivitiesAndAssociatedDataBatch", testCleanupActivitiesAndAssociatedDataBatch},
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
	noUserCtx := context.Background()

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)
	u2 := test.NewUser(t, ds, "user2", "user2@example.com", false)
	ctx := viewer.NewContext(noUserCtx, viewer.Viewer{User: u2})

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

	// create a couple of software installers
	installer := strings.NewReader("echo")
	sw1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install foo",
		InstallerFile: installer,
		StorageID:     uuid.NewString(),
		Filename:      "foo.pkg",
		Title:         "foo",
		Source:        "apps",
		Version:       "0.0.1",
	})
	require.NoError(t, err)
	sw2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install bar",
		InstallerFile: installer,
		StorageID:     uuid.NewString(),
		Filename:      "bar.pkg",
		Title:         "bar",
		Source:        "apps",
		Version:       "0.0.2",
	})
	require.NoError(t, err)
	sw1Meta, err := ds.GetSoftwareInstallerMetadataByID(ctx, sw1)
	require.NoError(t, err)
	sw2Meta, err := ds.GetSoftwareInstallerMetadataByID(ctx, sw2)
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
	// create some software installs requests for h1, make some complete
	h1FooFailed, err := ds.InsertSoftwareInstallRequest(ctx, h1.ID, sw1Meta.InstallerID)
	require.NoError(t, err)
	h1Bar, err := ds.InsertSoftwareInstallRequest(ctx, h1.ID, sw2Meta.InstallerID)
	require.NoError(t, err)
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    h1.ID,
		InstallUUID:               h1FooFailed,
		PreInstallConditionOutput: ptr.String(""), // pre-install failed
	})
	require.NoError(t, err)
	h1FooInstalled, err := ds.InsertSoftwareInstallRequest(ctx, h1.ID, sw1Meta.InstallerID)
	require.NoError(t, err)
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    h1.ID,
		InstallUUID:               h1FooInstalled,
		PreInstallConditionOutput: ptr.String("ok"),
		InstallScriptExitCode:     ptr.Int(0),
	})
	require.NoError(t, err)
	h1Foo, err := ds.InsertSoftwareInstallRequest(noUserCtx, h1.ID, sw1Meta.InstallerID) // no user for this one
	require.NoError(t, err)

	// create a single pending request for h2, as well as a non-pending one
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h2.ID, ScriptID: &scr1.ID, ScriptContents: scr1.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h2A := hsr.ExecutionID
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h2.ID, ScriptContents: "F", UserID: &u.ID})
	require.NoError(t, err)
	_, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{HostID: h2.ID, ExecutionID: hsr.ExecutionID, Output: "ok", ExitCode: 0})
	require.NoError(t, err)
	h2F := hsr.ExecutionID
	// add a pending software install request for h2
	h2Bar, err := ds.InsertSoftwareInstallRequest(ctx, h2.ID, sw2Meta.InstallerID)
	require.NoError(t, err)

	// nothing for h3

	// force-set the order of the created_at timestamps
	endTime := SetOrderedCreatedAtTimestamps(t, ds, time.Now(), "host_script_results", "execution_id", h1A, h1B)
	endTime = SetOrderedCreatedAtTimestamps(t, ds, endTime, "host_software_installs", "execution_id", h1FooFailed, h1Bar)
	endTime = SetOrderedCreatedAtTimestamps(t, ds, endTime, "host_script_results", "execution_id", h1C, h1D, h1E)
	endTime = SetOrderedCreatedAtTimestamps(t, ds, endTime, "host_software_installs", "execution_id", h1FooInstalled, h1Foo)
	endTime = SetOrderedCreatedAtTimestamps(t, ds, endTime, "host_software_installs", "execution_id", h2Bar)
	SetOrderedCreatedAtTimestamps(t, ds, endTime, "host_script_results", "execution_id", h2A, h2F)

	execIDsWithUser := map[string]bool{
		h1A:   true,
		h1B:   true,
		h1C:   true,
		h1D:   false,
		h1E:   false,
		h2A:   true,
		h2F:   true,
		h1Foo: false,
		h1Bar: true,
		h2Bar: true,
	}
	execIDsScriptName := map[string]string{
		h1A: scr1.Name,
		h1B: scr2.Name,
		h2A: scr1.Name,
	}
	execIDsSoftwareTitle := map[string]string{
		h1Foo: "foo",
		h1Bar: "bar",
		h2Bar: "bar",
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
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1Bar, h1C},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{Page: 3, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1Foo},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{h1A, h1B, h1Bar, h1C},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{h1D, h1E, h1Foo},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 7},
		},
		{
			opts:      fleet.ListOptions{PerPage: 3},
			hostID:    h2.ID,
			wantExecs: []string{h2Bar, h2A},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false, TotalResults: 2},
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

				require.Equal(t, c.hostID, uint(details["host_id"].(float64)), "result %d", i)

				var wantUser *fleet.User
				switch a.Type {
				case fleet.ActivityTypeRanScript{}.ActivityName():
					require.Equal(t, wantExec, details["script_execution_id"], "result %d", i)
					require.Equal(t, execIDsScriptName[wantExec], details["script_name"], "result %d", i)
					wantUser = u

				case fleet.ActivityTypeInstalledSoftware{}.ActivityName():
					require.Equal(t, wantExec, details["install_uuid"], "result %d", i)
					require.Equal(t, execIDsSoftwareTitle[wantExec], details["software_title"], "result %d", i)
					wantUser = u2

				default:
					t.Fatalf("unknown activity type %s", a.Type)
				}

				if execIDsWithUser[wantExec] {
					require.NotNil(t, a.ActorID, "result %d", i)
					require.Equal(t, wantUser.ID, *a.ActorID, "result %d", i)
					require.NotNil(t, a.ActorFullName, "result %d", i)
					require.Equal(t, wantUser.Name, *a.ActorFullName, "result %d", i)
					require.NotNil(t, a.ActorEmail, "result %d", i)
					require.Equal(t, wantUser.Email, *a.ActorEmail, "result %d", i)
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

func testCleanupActivitiesAndAssociatedData(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user1 := &fleet.User{
		Password:   []byte("p4ssw0rd.123"),
		Name:       "user1",
		Email:      "user1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	user1, err := ds.NewUser(ctx, user1)
	require.NoError(t, err)

	// Nothing to delete.
	err = ds.CleanupActivitiesAndAssociatedData(ctx, 500, 1)
	require.NoError(t, err)

	nonSavedQuery1, err := ds.NewQuery(ctx, &fleet.Query{
		Name:    "nonSavedQuery1",
		Saved:   false,
		Query:   "SELECT 1;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	savedQuery1, err := ds.NewQuery(ctx, &fleet.Query{
		Name:    "savedQuery1",
		Saved:   true,
		Query:   "SELECT 2;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	distributedQueryCampaign1, err := ds.NewDistributedQueryCampaign(ctx, &fleet.DistributedQueryCampaign{
		QueryID: nonSavedQuery1.ID,
		Status:  fleet.QueryComplete,
		UserID:  user1.ID,
	})
	require.NoError(t, err)
	_, err = ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
		DistributedQueryCampaignID: distributedQueryCampaign1.ID,
		TargetID:                   1,
		Type:                       fleet.TargetHost,
	})
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "other activity",
		details: map[string]interface{}{"detail": 0, "foo": "zoo"},
	})
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "live query",
		details: map[string]interface{}{"detail": 1, "foo": "bar"},
	})
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "some host activity",
		details: map[string]interface{}{"detail": 0, "foo": "zoo"},
		hostIDs: []uint{1},
	})
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "some host activity 2",
		details: map[string]interface{}{"detail": 0, "foo": "bar"},
		hostIDs: []uint{2},
	})
	require.NoError(t, err)

	// Nothing is deleted, as the activities and associated data is recent.
	const maxCount = 500
	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err := ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 4)
	nonExpiredActivityID := activities[0].ID
	expiredActivityID := activities[1].ID
	nonExpiredHostActivityID := activities[2].ID
	expiredHostActivityID := activities[3].ID
	_, err = ds.Query(ctx, nonSavedQuery1.ID)
	require.NoError(t, err)
	_, err = ds.DistributedQueryCampaign(ctx, distributedQueryCampaign1.ID)
	require.NoError(t, err)
	targets, err := ds.DistributedQueryCampaignTargetIDs(ctx, distributedQueryCampaign1.ID)
	require.NoError(t, err)
	require.Len(t, targets.HostIDs, 1)

	// Make some of the activity and associated data older.
	_, err = ds.writer(context.Background()).Exec(`
		UPDATE activities SET created_at = ? WHERE id = ? OR id = ?`,
		time.Now().Add(-48*time.Hour), expiredActivityID, expiredHostActivityID,
	)
	require.NoError(t, err)
	_, err = ds.writer(context.Background()).Exec(`
		UPDATE queries SET created_at = ? WHERE id = ? OR id = ?`,
		time.Now().Add(-48*time.Hour), nonSavedQuery1.ID, savedQuery1.ID,
	)
	require.NoError(t, err)

	// Expired activity and associated data should be cleaned up.
	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 3)
	require.Equal(t, nonExpiredActivityID, activities[0].ID)
	require.Equal(t, nonExpiredHostActivityID, activities[1].ID)
	require.Equal(t, expiredHostActivityID, activities[2].ID)
	_, err = ds.Query(ctx, nonSavedQuery1.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = ds.DistributedQueryCampaign(ctx, distributedQueryCampaign1.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	targets, err = ds.DistributedQueryCampaignTargetIDs(ctx, distributedQueryCampaign1.ID)
	require.NoError(t, err)
	require.Empty(t, targets.HostIDs)
	require.Empty(t, targets.LabelIDs)
	require.Empty(t, targets.TeamIDs)

	// Saved query should not be cleaned up.
	savedQuery1, err = ds.Query(ctx, savedQuery1.ID)
	require.NoError(t, err)
	require.NotNil(t, savedQuery1)
}

func testCleanupActivitiesAndAssociatedDataBatch(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user1 := &fleet.User{
		Password:   []byte("p4ssw0rd.123"),
		Name:       "user1",
		Email:      "user1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}
	user1, err := ds.NewUser(ctx, user1)
	require.NoError(t, err)

	const maxCount = 500

	// Create 1500 activities.
	insertActivitiesStmt := `
		INSERT INTO activities
		(user_id, user_name, activity_type, details, user_email)
		VALUES `
	var insertActivitiesArgs []interface{}
	for i := 0; i < 1500; i++ {
		insertActivitiesArgs = append(insertActivitiesArgs,
			user1.ID, user1.Name, "foobar", `{"foo": "bar"}`, user1.Email,
		)
	}
	insertActivitiesStmt += strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?),", 1500), ",")
	_, err = ds.writer(ctx).ExecContext(ctx, insertActivitiesStmt, insertActivitiesArgs...)
	require.NoError(t, err)

	// Create 1500 non-saved queries.
	insertQueriesStmt := `
		INSERT INTO queries
		(name, description, query)
		VALUES `
	var insertQueriesArgs []interface{}
	for i := 0; i < 1500; i++ {
		insertQueriesArgs = append(insertQueriesArgs,
			fmt.Sprintf("foobar%d", i), "foobar", "SELECT 1;",
		)
	}
	insertQueriesStmt += strings.TrimSuffix(strings.Repeat("(?, ?, ?),", 1500), ",")
	_, err = ds.writer(ctx).ExecContext(ctx, insertQueriesStmt, insertQueriesArgs...)
	require.NoError(t, err)

	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err := ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 1500)
	var queriesLen int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 1500, queriesLen)

	// Make 1250 activities as expired.
	_, err = ds.writer(context.Background()).Exec(`
		UPDATE activities SET created_at = ? WHERE id <= 1250`,
		time.Now().Add(-48*time.Hour),
	)
	require.NoError(t, err)

	// Make 1250 queries as expired.
	_, err = ds.writer(context.Background()).Exec(`
		UPDATE queries SET created_at = ? WHERE id <= 1250`,
		time.Now().Add(-48*time.Hour),
	)
	require.NoError(t, err)

	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 1000)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 1000, queriesLen)

	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 500)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 500, queriesLen)

	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 250)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 250, queriesLen)

	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 250)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 250, queriesLen)
}
