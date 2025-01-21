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

	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_mysql "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/mysql"
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
		{"ActivateNextActivity", testActivateNextActivity},
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

	timestamp := time.Now()
	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test1",
				details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
			}, nil, timestamp,
		),
	)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test2",
				details: map[string]interface{}{"detail": 2},
			}, nil, timestamp,
		),
	)

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
	timestamp := time.Now()

	activity := dummyActivity{
		name:    "test0",
		details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
	}
	// If we don't set the ActivityWebhookContextKey context value, the activity will not be created
	assert.Error(t, ds.NewActivity(context.Background(), u, activity, nil, timestamp))
	// If we set the context value to the wrong thing, the activity will not be created
	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, "bozo")
	assert.Error(t, ds.NewActivity(ctx, u, activity, nil, timestamp))

	ctx = context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test1",
				details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
			}, nil, timestamp,
		),
	)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test2",
				details: map[string]interface{}{"detail": 2},
			}, nil, timestamp,
		),
	)

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

	timestamp := time.Now()
	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test1",
				details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
			}, nil, timestamp,
		),
	)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test2",
				details: map[string]interface{}{"detail": 2},
			}, nil, timestamp,
		),
	)
	require.NoError(
		t, ds.NewActivity(
			ctx, u, dummyActivity{
				name:    "test3",
				details: map[string]interface{}{"detail": 3},
			}, nil, timestamp,
		),
	)

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
	timestamp := time.Now()
	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	require.NoError(
		t, ds.NewActivity(
			ctx, nil, dummyActivity{
				name:    "test1",
				details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
			}, nil, timestamp,
		),
	)

	require.NoError(
		t, ds.NewActivity(
			ctx, nil, fleet.ActivityInstalledAppStoreApp{
				HostID:          1,
				HostDisplayName: "A Host",
				SoftwareTitle:   "Trello",
				AppStoreID:      "123456",
				CommandUUID:     "some uuid",
				Status:          string(fleet.SoftwareInstalled),
				SelfService:     false,
				PolicyID:        ptr.Uint(1),
				PolicyName:      ptr.String("Sample Policy"),
			}, nil, timestamp,
		),
	)

	activities, _, err := ds.ListActivities(context.Background(), fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	assert.Equal(t, "Fleet", *activities[1].ActorFullName)
}

func testActivityPaginationMetadata(t *testing.T, ds *Datastore) {
	timestamp := time.Now()
	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	for i := 0; i < 3; i++ {
		require.NoError(
			t, ds.NewActivity(
				ctx, nil, dummyActivity{
					name:    fmt.Sprintf("test-%d", i),
					details: map[string]interface{}{},
				}, nil, timestamp,
			),
		)
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

	test.CreateInsertGlobalVPPToken(t, ds)

	// create three hosts
	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h1, false)
	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h2, false)
	h3 := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h3, false)

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
	installer1, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	sw1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install foo",
		InstallerFile:   installer1,
		StorageID:       uuid.NewString(),
		Filename:        "foo.pkg",
		Title:           "foo",
		Source:          "apps",
		Version:         "0.0.1",
		UserID:          u.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	installer2, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	sw2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install bar",
		InstallerFile:   installer2,
		StorageID:       uuid.NewString(),
		Filename:        "bar.pkg",
		Title:           "bar",
		Source:          "apps",
		Version:         "0.0.2",
		UserID:          u.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	installer3, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	sw3, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install to delete",
		InstallerFile:   installer3,
		StorageID:       uuid.NewString(),
		Filename:        "todelete.pkg",
		Title:           "todelete",
		Source:          "apps",
		Version:         "0.0.3",
		UserID:          u.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	sw1Meta, err := ds.GetSoftwareInstallerMetadataByID(ctx, sw1)
	require.NoError(t, err)
	sw2Meta, err := ds.GetSoftwareInstallerMetadataByID(ctx, sw2)
	require.NoError(t, err)
	sw3Meta, err := ds.GetSoftwareInstallerMetadataByID(ctx, sw3)
	require.NoError(t, err)

	// insert a VPP app
	vppCommand1, vppCommand2 := "vpp-command-1", "vpp-command-2"
	vppApp := &fleet.VPPApp{
		Name: "vpp_no_team_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}},
		BundleIdentifier: "b3",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppApp, nil)
	require.NoError(t, err)

	// install the VPP app on h1
	err = ds.InsertHostVPPSoftwareInstall(ctx, h1.ID, vppApp.VPPAppID, vppCommand1, "event-id-1", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	// vppCommand1 is now active for h1

	// install the VPP app on h2, self-service
	err = ds.InsertHostVPPSoftwareInstall(noUserCtx, h2.ID, vppApp.VPPAppID, vppCommand2, "event-id-2", fleet.HostSoftwareInstallOptions{SelfService: true})
	require.NoError(t, err)
	// vppCommand2 is now active for h2

	// create a sync script request for h1 that has been pending for >
	// MaxWaitTime, will still show up (sync scripts go through the upcoming
	// queue as any script)
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "sync", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	hSyncExpired := hsr.ExecutionID
	t.Log("hSyncExpired", hSyncExpired)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE upcoming_activities SET created_at = ? WHERE execution_id = ?", time.Now().Add(-(scripts.MaxServerWaitTime + time.Minute)), hSyncExpired)
		return err
	})

	// create some script requests for h1
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptID: &scr1.ID, ScriptContents: scr1.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h1A := hsr.ExecutionID
	t.Log("h1A", h1A)

	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptID: &scr2.ID, ScriptContents: scr2.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h1B := hsr.ExecutionID
	t.Log("h1B", h1B)

	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "C", UserID: &u.ID})
	require.NoError(t, err)
	h1C := hsr.ExecutionID
	t.Log("h1C", h1C)

	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "D"})
	require.NoError(t, err)
	h1D := hsr.ExecutionID
	t.Log("h1D", h1D)

	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h1.ID, ScriptContents: "E"})
	require.NoError(t, err)
	h1E := hsr.ExecutionID
	t.Log("h1E", h1E)

	// create some software installs requests for h1
	h1Bar, err := ds.InsertSoftwareInstallRequest(ctx, h1.ID, sw2Meta.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	t.Log("h1Bar", h1Bar)

	// No user for this one and not Self-service, means it was installed by Fleet
	policy, err := ds.NewTeamPolicy(ctx, 0, &u.ID, fleet.PolicyPayload{
		Name:  "Test Policy",
		Query: "SELECT 1",
	})
	require.NoError(t, err)
	h1Fleet, err := ds.InsertSoftwareInstallRequest(noUserCtx, h1.ID, sw1Meta.InstallerID, fleet.HostSoftwareInstallOptions{PolicyID: &policy.ID})
	require.NoError(t, err)
	t.Log("h1Fleet", h1Fleet)

	// create a single pending request for h2
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h2.ID, ScriptID: &scr1.ID, ScriptContents: scr1.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	h2A := hsr.ExecutionID
	t.Log("h2A", h2A)
	// add a pending software install request for h2
	h2Bar, err := ds.InsertSoftwareInstallRequest(ctx, h2.ID, sw2Meta.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	t.Log("h2Bar", h2Bar)
	// No user for this one and Self-service, means it was installed by the end user, so the user_id should be null/nil.
	h2SelfService, err := ds.InsertSoftwareInstallRequest(noUserCtx, h2.ID, sw1Meta.InstallerID, fleet.HostSoftwareInstallOptions{SelfService: true})
	require.NoError(t, err)
	t.Log("h2SelfService", h2SelfService)

	setupExpScript := &fleet.Script{Name: "setup_experience_script", ScriptContents: "setup_experience"}
	err = ds.SetSetupExperienceScript(ctx, setupExpScript)
	require.NoError(t, err)
	ses, err := ds.GetSetupExperienceScript(ctx, h2.TeamID)
	require.NoError(t, err)
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h2.ID, ScriptContents: "setup_experience", SetupExperienceScriptID: &ses.ID})
	require.NoError(t, err)
	h2SetupExp := hsr.ExecutionID
	t.Log("h2SetupExp", h2SetupExp)

	// create pending install and uninstall requests for h3 that will be deleted
	_, err = ds.InsertSoftwareInstallRequest(ctx, h3.ID, sw3Meta.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	err = ds.InsertSoftwareUninstallRequest(ctx, "uninstallRun", h3.ID, sw3Meta.InstallerID)
	require.NoError(t, err)

	// delete installer (should clear pending requests)
	err = ds.DeleteSoftwareInstaller(ctx, sw3Meta.InstallerID)
	require.NoError(t, err)

	// force-set the order of the created_at timestamps
	// even if vppCommand1 and 2 are later, since they are already activated
	// (because they were enqueued first) they will show up first.
	SetOrderedCreatedAtTimestamps(t, ds, time.Now(), "upcoming_activities", "execution_id",
		h1A, h1B, h1Bar, h1C, h1D, h1E, h1Fleet, h2SelfService, h2Bar, h2A, vppCommand1, vppCommand2, h2SetupExp)

	execIDsWithUser := map[string]bool{
		hSyncExpired:  true,
		h1A:           true,
		h1B:           true,
		h1C:           true,
		h1D:           false,
		h1E:           false,
		h2A:           true,
		h1Fleet:       false,
		h2SelfService: false,
		h1Bar:         true,
		h2Bar:         true,
		vppCommand1:   true,
		vppCommand2:   false,
		h2SetupExp:    false,
	}
	execIDsScriptName := map[string]string{
		h1A:        scr1.Name,
		h1B:        scr2.Name,
		h2A:        scr1.Name,
		h2SetupExp: setupExpScript.Name,
	}
	execIDsSoftwareTitle := map[string]string{
		h1Fleet:       "foo",
		h1Bar:         "bar",
		h2Bar:         "bar",
		h2SelfService: "foo",
	}
	execIDsFromPolicyAutomation := map[string]struct{}{
		h1Fleet: {},
	}
	// to simplify map, false = cancellable, true = NON-cancellable
	execIDsNonCancellable := map[string]bool{
		vppCommand1: true,
		vppCommand2: true,
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
			wantExecs: []string{vppCommand1, hSyncExpired},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1A, h1B},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1Bar, h1C},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 3, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 4, PerPage: 2},
			hostID:    h1.ID,
			wantExecs: []string{h1Fleet},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{vppCommand1, hSyncExpired, h1A, h1B},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 1, PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{h1Bar, h1C, h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 2, PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{h1Fleet},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{Page: 3, PerPage: 4},
			hostID:    h1.ID,
			wantExecs: []string{},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true, TotalResults: 9},
		},
		{
			opts:      fleet.ListOptions{PerPage: 5},
			hostID:    h2.ID,
			wantExecs: []string{vppCommand2, h2SetupExp, h2SelfService, h2Bar, h2A}, // setup experience is top-priority, but vppCommand2 was already activated
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false, TotalResults: 5},
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
			c.opts.OrderKey = ""
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

				case fleet.ActivityInstalledAppStoreApp{}.ActivityName():
					require.Equal(t, wantExec, details["command_uuid"], "result %d", i)
					require.Equal(t, "vpp_no_team_app_1", details["software_title"], "result %d", i)
					require.Equal(t, !execIDsWithUser[wantExec], details["self_service"], "result %d", i)
					wantUser = u2

				default:
					t.Fatalf("unknown activity type %s", a.Type)
				}

				require.Equal(t, !execIDsNonCancellable[wantExec], a.Cancellable, "result %d", i)

				if _, ok := execIDsFromPolicyAutomation[wantExec]; ok {
					require.Nil(t, a.ActorID, "result %d", i)
					require.NotNil(t, a.ActorFullName, "result %d", i)
					require.Equal(t, "Fleet", *a.ActorFullName, "result %d", i)
					require.Nil(t, a.ActorEmail, "result %d", i)
					require.NotNil(t, details["policy_id"])
					require.Equal(t, float64(policy.ID), details["policy_id"], "result %d", i)
					require.NotNil(t, details["policy_name"])
					require.Equal(t, policy.Name, details["policy_name"], "result %d", i)
				} else if execIDsWithUser[wantExec] {
					require.NotNil(t, a.ActorID, "result %d", i)
					require.Equal(t, wantUser.ID, *a.ActorID, "result %d", i)
					require.NotNil(t, a.ActorFullName, "result %d", i)
					require.Equal(t, wantUser.Name, *a.ActorFullName, "result %d", i)
					require.NotNil(t, a.ActorEmail, "result %d", i)
					require.Equal(t, wantUser.Email, *a.ActorEmail, "result %d", i)
				} else {
					require.Nil(t, a.ActorID, "result %d", i)
					if a.FleetInitiated {
						require.NotNil(t, a.ActorFullName, "result %d", i)
						require.Equal(t, "Fleet", *a.ActorFullName, "result %d", i)
					} else {
						require.Nil(t, a.ActorFullName, "result %d", i)
					}
					require.Nil(t, a.ActorEmail, "result %d", i)
				}
			}
		})
	}
}

func testListHostPastActivities(t *testing.T, ds *Datastore) {
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

	timestamp := time.Now()
	ctx := context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	for _, a := range activities {
		detailsBytes, err := json.Marshal(a)
		require.NoError(t, err)
		require.NoError(t, ds.NewActivity(ctx, u, a, detailsBytes, timestamp))
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
	timestamp := time.Now()
	ctx = context.WithValue(context.Background(), fleet.ActivityWebhookContextKey, true)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "other activity",
		details: map[string]interface{}{"detail": 0, "foo": "zoo"},
	}, nil, timestamp,
	)
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "live query",
		details: map[string]interface{}{"detail": 1, "foo": "bar"},
	}, nil, timestamp,
	)
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "some host activity",
		details: map[string]interface{}{"detail": 0, "foo": "zoo"},
		hostIDs: []uint{1},
	}, nil, timestamp,
	)
	require.NoError(t, err)
	err = ds.NewActivity(ctx, user1, dummyActivity{
		name:    "some host activity 2",
		details: map[string]interface{}{"detail": 0, "foo": "bar"},
		hostIDs: []uint{2},
	}, nil, timestamp,
	)
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
	require.Equal(t, 250, queriesLen) // All expired queries should be cleaned up.

	err = ds.CleanupActivitiesAndAssociatedData(ctx, maxCount, 1)
	require.NoError(t, err)

	activities, _, err = ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 500)
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

func testActivateNextActivity(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	test.CreateInsertGlobalVPPToken(t, ds)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h1, false)
	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h2, false)

	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)
	nanoCtx := &mdm.Request{EnrollID: &mdm.EnrollID{ID: h1.UUID}, Context: ctx}

	// create a couple VPP apps that can be installed later
	vppApp1 := &fleet.VPPApp{
		Name: "vpp_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "vpp1", Platform: fleet.MacOSPlatform}},
		BundleIdentifier: "vpp1",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppApp1, nil)
	require.NoError(t, err)
	vppApp2 := &fleet.VPPApp{
		Name: "vpp_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "vpp2", Platform: fleet.MacOSPlatform}},
		BundleIdentifier: "vpp2",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppApp2, nil)
	require.NoError(t, err)

	// activating an empty queue is fine, nothing activated
	execIDs, err := ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, "")
	require.NoError(t, err)
	require.Empty(t, execIDs)

	// activating when empty with an unknown completed exec id is fine
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, uuid.NewString())
	require.NoError(t, err)
	require.Empty(t, execIDs)

	// create a script execution request
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         h1.ID,
		ScriptContents: "echo 'a'",
	})
	require.NoError(t, err)
	script1_1 := hsr.ExecutionID

	// add a couple install requests for vpp1 and vpp2
	vpp1_1 := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, h1.ID, vppApp1.VPPAppID, vpp1_1, "event-id-1", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	vpp1_2 := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, h1.ID, vppApp2.VPPAppID, vpp1_2, "event-id-2", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// activating does nothing because the script is still activated
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, "")
	require.NoError(t, err)
	require.Empty(t, execIDs)

	// pending activities are script1_1, vpp1_1, vpp1_2
	pendingActs, _, err := ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 3)
	require.Equal(t, script1_1, pendingActs[0].UUID)
	require.False(t, pendingActs[0].Cancellable)
	require.Equal(t, vpp1_1, pendingActs[1].UUID)
	require.True(t, pendingActs[1].Cancellable)
	require.Equal(t, vpp1_2, pendingActs[2].UUID)
	require.True(t, pendingActs[2].Cancellable)

	// set a script result, will activate both VPP apps
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID: h1.ID, ExecutionID: script1_1, Output: "a", ExitCode: 0,
	})
	require.NoError(t, err)

	// pending activities are vpp1_1, vpp1_2, both are non-cancellable because activated
	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, vpp1_1, pendingActs[0].UUID)
	require.False(t, pendingActs[0].Cancellable)
	require.Equal(t, vpp1_2, pendingActs[1].UUID)
	require.False(t, pendingActs[1].Cancellable)

	// nano commands have been inserted
	cmd, err := nanoDB.RetrieveNextCommand(nanoCtx, false)
	require.NoError(t, err)
	require.Equal(t, vpp1_1, cmd.CommandUUID)
	require.Equal(t, "InstallApplication", cmd.Command.Command.RequestType)
	rawCmd := string(cmd.Raw)
	require.Contains(t, rawCmd, ">"+vppApp1.VPPAppTeam.AdamID+"<")
	require.Contains(t, rawCmd, ">"+vpp1_1+"<")
}
