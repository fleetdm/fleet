package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
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
		{"ListHostUpcomingActivities", testListHostUpcomingActivities},
		{"CleanupExpiredLiveQueries", testCleanupExpiredLiveQueries},
		{"CleanupExpiredLiveQueriesBatch", testCleanupExpiredLiveQueriesBatch},
		{"ActivateNextActivity", testActivateNextActivity},
		{"ActivateItselfOnEmptyQueue", testActivateItselfOnEmptyQueue},
		{"CancelNonActivatedUpcomingActivity", testCancelNonActivatedUpcomingActivity},
		{"CancelActivatedUpcomingActivity", testCancelActivatedUpcomingActivity},
		{"BatchCancelAllHostUpcomingActivities", testBatchCancelAllHostUpcomingActivities},
		{"SetResultAfterCancelUpcomingActivity", testSetResultAfterCancelUpcomingActivity},
		{"GetHostUpcomingActivityMeta", testGetHostUpcomingActivityMeta},
		{"UnblockHostsUpcomingActivityQueue", testUnblockHostsUpcomingActivityQueue},
		{"ReleaseFleetInitiatedUpcomingActivities", testReleaseFleetInitiatedUpcomingActivities},
		{"ReapStuckActivatedMDMInstalls", testReapStuckActivatedMDMInstalls},
		{"ActivateScriptPackageInstallWithCorruptPayload", testActivateScriptPackageInstallWithCorruptPayload},
		{"ActivateRegularPackageInstall", testActivateRegularPackageInstall},
		{"ActivateDeletedInstallerShowsPlaceholder", testActivateDeletedInstallerShowsPlaceholder},
		{"ActivateScriptPackageUninstallWithCorruptPayload", testActivateScriptPackageUninstallWithCorruptPayload},
		{"ListPolicyAutomationActivities", testListPolicyAutomationActivities},
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
	activitySvc := NewTestActivityService(t, ds)

	u := &fleet.User{
		Password:    []byte("asd"),
		Name:        "fullname",
		Email:       "email@asd.com",
		GravatarURL: "http://asd.com",
		APIOnly:     true,
		GlobalRole:  ptr.String(fleet.RoleObserver),
	}
	_, err := ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	apiUser := &activity_api.User{ID: u.ID, Name: u.Name, Email: u.Email}
	ctx := context.Background()
	require.NoError(
		t, activitySvc.NewActivity(
			ctx, apiUser, dummyActivity{
				name:    "test1",
				details: map[string]interface{}{"detail": 1, "sometext": "aaa"},
			},
		),
	)
	require.NoError(
		t, activitySvc.NewActivity(
			ctx, apiUser, dummyActivity{
				name:    "test2",
				details: map[string]interface{}{"detail": 2},
			},
		),
	)

	activities := ListActivitiesAPI(t, context.Background(), activitySvc, activity_api.ListOptions{})
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)

	u.Name = "newname"
	err = ds.SaveUser(context.Background(), u)
	require.NoError(t, err)

	activities = ListActivitiesAPI(t, context.Background(), activitySvc, activity_api.ListOptions{})
	assert.Len(t, activities, 2)
	assert.Equal(t, "newname", *activities[0].ActorFullName)
	assert.Equal(t, "http://asd.com", *activities[0].ActorGravatar)
	assert.Equal(t, "email@asd.com", *activities[0].ActorEmail)
	assert.Equal(t, true, *activities[0].ActorAPIOnly)

	err = ds.DeleteUser(context.Background(), u.ID)
	require.NoError(t, err)

	activities = ListActivitiesAPI(t, context.Background(), activitySvc, activity_api.ListOptions{})
	assert.Len(t, activities, 2)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)
	assert.Nil(t, activities[0].ActorGravatar)
}

func testListHostUpcomingActivities(t *testing.T, ds *Datastore) {
	noUserCtx := context.Background()

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)
	u2 := test.NewUser(t, ds, "user2", "user2@example.com", false)
	ctx := viewer.NewContext(noUserCtx, viewer.Viewer{User: u2})

	test.CreateInsertGlobalVPPToken(t, ds)

	// create four hosts
	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h1, false)
	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h2, false)
	h3 := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h3, false)
	h4 := test.NewHost(t, ds, "h4.local", "10.10.10.4", "4", "4", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h4, false)

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
	_, err = ds.SetSetupExperienceScript(ctx, setupExpScript)
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
	err = ds.InsertSoftwareUninstallRequest(ctx, "uninstallRun", h3.ID, sw3Meta.InstallerID, false)
	require.NoError(t, err)

	// delete installer (should clear pending requests)
	err = ds.DeleteSoftwareInstaller(ctx, sw3Meta.InstallerID)
	require.NoError(t, err)

	// Setup host 4. We will create upcoming activities, then
	// delete and "restore" the host, similar to what would happen
	// if you delete an ABM DEP host.
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: h4.ID, ScriptID: &scr1.ID, ScriptContents: scr1.ScriptContents, UserID: &u.ID})
	require.NoError(t, err)
	// h4A := hsr.ExecutionID
	// h4Bar, err := ds.InsertSoftwareInstallRequest(ctx, h4.ID, sw2Meta.InstallerID, false, nil)
	_, err = ds.InsertSoftwareInstallRequest(ctx, h4.ID, sw2Meta.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// Set LastEnrolledAt before deleting the host (simulating a DEP enrolled host)
	h4.LastEnrolledAt = time.Now()

	// Delete the host
	err = ds.DeleteHost(ctx, h4.ID)
	require.NoError(t, err)
	// DEP "restore" the host
	err = ds.RestoreMDMApplePendingDEPHost(ctx, h4)
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
		{
			opts:      fleet.ListOptions{},
			hostID:    h4.ID,
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

	t.Run("rejects_unknown_order_key", func(t *testing.T) {
		_, _, err := ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{OrderKey: "h.node_key"})
		require.Error(t, err)
	})
}

func testCleanupExpiredLiveQueries(t *testing.T, ds *Datastore) {
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
	err = ds.CleanupExpiredLiveQueries(ctx, 1)
	require.NoError(t, err)

	nonSavedQuery, err := ds.NewQuery(ctx, &fleet.Query{
		Name:    "nonSavedQuery",
		Saved:   false,
		Query:   "SELECT 1;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	savedQuery, err := ds.NewQuery(ctx, &fleet.Query{
		Name:    "savedQuery",
		Saved:   true,
		Query:   "SELECT 2;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	campaign, err := ds.NewDistributedQueryCampaign(ctx, &fleet.DistributedQueryCampaign{
		QueryID: nonSavedQuery.ID,
		Status:  fleet.QueryComplete,
		UserID:  user1.ID,
	})
	require.NoError(t, err)
	_, err = ds.NewDistributedQueryCampaignTarget(ctx, &fleet.DistributedQueryCampaignTarget{
		DistributedQueryCampaignID: campaign.ID,
		TargetID:                   1,
		Type:                       fleet.TargetHost,
	})
	require.NoError(t, err)

	// Nothing is deleted because the data is recent.
	err = ds.CleanupExpiredLiveQueries(ctx, 1)
	require.NoError(t, err)

	_, err = ds.Query(ctx, nonSavedQuery.ID)
	require.NoError(t, err)
	_, err = ds.DistributedQueryCampaign(ctx, campaign.ID)
	require.NoError(t, err)
	targets, err := ds.DistributedQueryCampaignTargetIDs(ctx, campaign.ID)
	require.NoError(t, err)
	require.Len(t, targets.HostIDs, 1)

	// Make the queries older.
	_, err = ds.writer(context.Background()).Exec(`
		UPDATE queries SET created_at = ? WHERE id = ? OR id = ?`,
		time.Now().Add(-48*time.Hour), nonSavedQuery.ID, savedQuery.ID,
	)
	require.NoError(t, err)

	// Expired unsaved query, its campaign, and campaign targets should be cleaned up.
	err = ds.CleanupExpiredLiveQueries(ctx, 1)
	require.NoError(t, err)

	_, err = ds.Query(ctx, nonSavedQuery.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = ds.DistributedQueryCampaign(ctx, campaign.ID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	targets, err = ds.DistributedQueryCampaignTargetIDs(ctx, campaign.ID)
	require.NoError(t, err)
	require.Empty(t, targets.HostIDs)
	require.Empty(t, targets.LabelIDs)
	require.Empty(t, targets.TeamIDs)

	// Saved query should not be cleaned up.
	savedQuery, err = ds.Query(ctx, savedQuery.ID)
	require.NoError(t, err)
	require.NotNil(t, savedQuery)
}

func testCleanupExpiredLiveQueriesBatch(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create 1500 non-saved queries.
	insertQueriesStmt := `
		INSERT INTO queries
		(name, description, query)
		VALUES `
	var insertQueriesArgs []any
	for i := range 1500 {
		insertQueriesArgs = append(insertQueriesArgs,
			fmt.Sprintf("foobar%d", i), "foobar", "SELECT 1;",
		)
	}
	insertQueriesStmt += strings.TrimSuffix(strings.Repeat("(?, ?, ?),", 1500), ",")
	_, err := ds.writer(ctx).ExecContext(ctx, insertQueriesStmt, insertQueriesArgs...)
	require.NoError(t, err)

	// Nothing deleted; all recent.
	err = ds.CleanupExpiredLiveQueries(ctx, 1)
	require.NoError(t, err)

	var queriesLen int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 1500, queriesLen)

	// Make 1250 queries expired.
	_, err = ds.writer(context.Background()).Exec(`
		UPDATE queries SET created_at = ? WHERE id <= 1250`,
		time.Now().Add(-48*time.Hour),
	)
	require.NoError(t, err)

	// All 1250 expired queries should be cleaned up in one call (batched internally).
	err = ds.CleanupExpiredLiveQueries(ctx, 1)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 250, queriesLen)

	// Running again should be a no-op (remaining 250 are not expired).
	err = ds.CleanupExpiredLiveQueries(ctx, 1)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queriesLen, `SELECT COUNT(*) FROM queries WHERE NOT saved;`)
	})
	require.Equal(t, 250, queriesLen)
}

func testActivateNextActivity(t *testing.T, ds *Datastore) {
	activitySvc := NewTestActivityService(t, ds)
	ctx := context.Background()

	test.CreateInsertGlobalVPPToken(t, ds)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h1, false)
	h2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h2, false)
	hIOS := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now().Add(-1*time.Second), test.WithPlatform("ios"))
	nanoEnrollAndSetHostMDMData(t, ds, hIOS, false)

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

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
	vppApp1IOS := &fleet.VPPApp{
		Name: "vpp_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "vpp1", Platform: fleet.IOSPlatform}},
		BundleIdentifier: "vpp1",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppApp1IOS, nil)
	require.NoError(t, err)

	// create a software installer that can be installed later
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
		UninstallScript: "uninstall foo",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// create an in-house app that can be installed later
	ihaID, ihaTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        uuid.NewString(),
		Filename:         "inhouse.ipa",
		Title:            "inhouse",
		Source:           "ios_apps",
		Extension:        "ipa",
		BundleIdentifier: "inhouse",
		UserID:           u.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
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

	// create a second script execution request that will not be activated yet
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         h1.ID,
		ScriptContents: "echo 'b'",
	})
	require.NoError(t, err)
	script1_2 := hsr.ExecutionID

	// host 2 is unaffected, activating results in nothing activated
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h2.ID, "")
	require.NoError(t, err)
	require.Empty(t, execIDs)

	// add a couple install requests for vpp1 and vpp2
	vpp1_1 := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, h1.ID, vppApp1.VPPAppID, vpp1_1, "event-id-1", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	vpp1_2 := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, h1.ID, vppApp2.VPPAppID, vpp1_2, "event-id-2", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// activating does nothing because the first script is still activated
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, "")
	require.NoError(t, err)
	require.Empty(t, execIDs)

	// pending activities are script1_1, script1_2, vpp1_1, vpp1_2
	pendingActs, _, err := ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 4)
	require.Equal(t, script1_1, pendingActs[0].UUID)
	require.Equal(t, script1_2, pendingActs[1].UUID)
	require.Equal(t, vpp1_1, pendingActs[2].UUID)
	require.Equal(t, vpp1_2, pendingActs[3].UUID)

	// listing scripts ready to execute returns script1_1
	pendingScripts, err := ds.ListReadyToExecuteScriptsForHost(ctx, h1.ID, false)
	require.NoError(t, err)
	require.Len(t, pendingScripts, 1)
	require.Equal(t, script1_1, pendingScripts[0].ExecutionID)

	// get host script result while there are no results yet returns the current status
	scriptRes, err := ds.GetHostScriptExecutionResult(ctx, script1_1)
	require.NoError(t, err)
	require.Nil(t, scriptRes.ExitCode)

	scriptRes, err = ds.GetHostScriptExecutionResult(ctx, script1_2)
	require.NoError(t, err)
	require.Nil(t, scriptRes.ExitCode)

	// delete the script1_2 upcoming activity as if it was cancelled
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			DELETE FROM upcoming_activities
			WHERE execution_id = ?`,
			script1_2)
		return err
	})

	// set a script result, will activate both VPP apps
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID: h1.ID, ExecutionID: script1_1, Output: "a", ExitCode: 0,
	}, nil)
	require.NoError(t, err)

	// get host script result now returns the result
	scriptRes, err = ds.GetHostScriptExecutionResult(ctx, script1_1)
	require.NoError(t, err)
	require.NotNil(t, scriptRes.ExitCode)
	require.EqualValues(t, 0, *scriptRes.ExitCode)

	// pending activities are vpp1_1, vpp1_2
	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, vpp1_1, pendingActs[0].UUID)
	require.Equal(t, vpp1_2, pendingActs[1].UUID)

	// nano commands have been inserted
	cmd, err := nanoDB.RetrieveNextCommand(nanoCtx, false)
	require.NoError(t, err)
	require.Equal(t, vpp1_1, cmd.CommandUUID)
	require.Equal(t, "InstallApplication", cmd.Command.Command.RequestType)
	rawCmd := string(cmd.Raw)
	require.Contains(t, rawCmd, ">"+vppApp1.VPPAppTeam.AdamID+"<")
	require.Contains(t, rawCmd, ">"+vpp1_1+"<")
	require.Contains(t, rawCmd, `<key>ManagementFlags</key>
        <integer>0</integer>`, "MacOS VPP app install command should have ManagementFlags 0")

	// insert a result for that command and create the past activity,
	// which triggers the next activity to be activated (should be none
	// in this scenario, as one is still active)
	cmdRes := &mdm.CommandResults{
		CommandUUID: vpp1_1,
		Status:      "Acknowledged",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      h1.ID,
		AppStoreID:  vppApp1.VPPAppTeam.AdamID,
		CommandUUID: vpp1_1,
	})
	require.NoError(t, err)

	appleCmdRes, err := ds.GetMDMAppleCommandResults(ctx, vpp1_1, "")
	require.NoError(t, err)
	require.Len(t, appleCmdRes, 1)
	require.Equal(t, "Acknowledged", appleCmdRes[0].Status)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 1)
	require.Equal(t, vpp1_2, pendingActs[0].UUID)

	// vpp1_2 is now the next nano command
	cmd, err = nanoDB.RetrieveNextCommand(nanoCtx, false)
	require.NoError(t, err)
	require.Equal(t, vpp1_2, cmd.CommandUUID)
	require.Equal(t, "InstallApplication", cmd.Command.Command.RequestType)
	rawCmd = string(cmd.Raw)
	require.Contains(t, rawCmd, ">"+vppApp2.VPPAppTeam.AdamID+"<")
	require.Contains(t, rawCmd, ">"+vpp1_2+"<")

	// create a pending software install request
	sw1_1, err := ds.InsertSoftwareInstallRequest(ctx, h1.ID, sw1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// the software install request is not active yet, so with only active, returns nothing
	pendingSw, err := ds.ListReadyToExecuteSoftwareInstalls(ctx, h1.ID)
	require.NoError(t, err)
	require.Len(t, pendingSw, 0)

	// without only active, returns it
	pendingSw, err = ds.ListPendingSoftwareInstalls(ctx, h1.ID)
	require.NoError(t, err)
	require.Len(t, pendingSw, 1)
	require.Equal(t, sw1_1, pendingSw[0])

	// activating does nothing because the VPP app 2 is still activated
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, "")
	require.NoError(t, err)
	require.Empty(t, execIDs)

	// trying to activate from a non-activated execution id (here, the software
	// install sw1_1 one) does not delete that activity - it deletes only if it
	// was activated
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, sw1_1)
	require.NoError(t, err)
	require.Empty(t, execIDs)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, vpp1_2, pendingActs[0].UUID)
	require.Equal(t, sw1_1, pendingActs[1].UUID)

	// create a pending uninstall request
	sw1_2 := uuid.NewString()
	err = ds.InsertSoftwareUninstallRequest(ctx, sw1_2, h1.ID, sw1, false)
	require.NoError(t, err)

	// still hasn't changed the pending queue
	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 3)
	require.Equal(t, vpp1_2, pendingActs[0].UUID)
	require.Equal(t, sw1_1, pendingActs[1].UUID)
	require.Equal(t, sw1_2, pendingActs[2].UUID)

	// insert a result for the vpp1_2 command
	cmdRes = &mdm.CommandResults{
		CommandUUID: vpp1_2,
		Status:      "Error",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      h1.ID,
		AppStoreID:  vppApp2.VPPAppTeam.AdamID,
		CommandUUID: vpp1_2,
	})
	require.NoError(t, err)

	appleCmdRes, err = ds.GetMDMAppleCommandResults(ctx, vpp1_2, "")
	require.NoError(t, err)
	require.Len(t, appleCmdRes, 1)
	require.Equal(t, "Error", appleCmdRes[0].Status)

	// software install activity is now activated
	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, sw1_1, pendingActs[0].UUID)
	require.Equal(t, sw1_2, pendingActs[1].UUID)

	// set a result for the software install
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                h1.ID,
		InstallUUID:           sw1_1,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	swRes, err := ds.GetSoftwareInstallResults(ctx, sw1_1)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstalled, swRes.Status)

	// activating does nothing because the sw1_2 was automatically activated
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, sw1_1)
	require.NoError(t, err)
	require.Empty(t, execIDs)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 1)
	require.Equal(t, sw1_2, pendingActs[0].UUID)

	// set a result for the software uninstall
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      h1.ID,
		ExecutionID: sw1_2,
		ExitCode:    1,
	}, nil)
	require.NoError(t, err)

	// because the install and uninstall are for the same software installer,
	// only the latest attempt is shown in the summary and it is the uninstall.
	swSummary, err := ds.GetSummaryHostSoftwareInstalls(ctx, sw1)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{
		FailedUninstall: 1,
	}, *swSummary)

	// activating does nothing because the queue is now empty
	execIDs, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), h1.ID, sw1_2)
	require.NoError(t, err)
	require.Empty(t, execIDs)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 0)

	// enqueue a VPP app request for iOS host
	vpp1_1_ios := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, hIOS.ID, vppApp1IOS.VPPAppID, vpp1_1_ios, "event-id-1-ios", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// enqueue an in-house app request for the iOS host
	ihaCmd := uuid.NewString()
	err = ds.InsertHostInHouseAppInstall(ctx, hIOS.ID, ihaID, ihaTitleID, ihaCmd, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, vpp1_1_ios, pendingActs[0].UUID)
	require.Equal(t, ihaCmd, pendingActs[1].UUID)

	// get next nano command for iOS host is the VPP app
	nanoCtx = &mdm.Request{EnrollID: &mdm.EnrollID{ID: hIOS.UUID}, Context: ctx}
	cmd, err = nanoDB.RetrieveNextCommand(nanoCtx, false)
	require.NoError(t, err)
	require.Equal(t, vpp1_1_ios, cmd.CommandUUID)
	require.Equal(t, "InstallApplication", cmd.Command.Command.RequestType)
	rawCmd = string(cmd.Raw)
	require.Contains(t, rawCmd, ">"+vppApp1IOS.VPPAppTeam.AdamID+"<")
	require.Contains(t, rawCmd, ">"+vpp1_1_ios+"<")
	require.Contains(t, rawCmd, `<key>ManagementFlags</key>
        <integer>1</integer>`)

	// record a result for the VPP app install, which will activate the in-house app
	cmdRes = &mdm.CommandResults{
		CommandUUID: vpp1_1_ios,
		Status:      "Acknowledged",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      hIOS.ID,
		AppStoreID:  vppApp1IOS.VPPAppTeam.AdamID,
		CommandUUID: vpp1_1_ios,
		Status:      "Error", // using a failure because otherwise it requires verification to activate next
	})
	require.NoError(t, err)

	// the in-house app is now activated
	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 1)
	require.Equal(t, ihaCmd, pendingActs[0].UUID)

	cmd, err = nanoDB.RetrieveNextCommand(nanoCtx, false)
	require.NoError(t, err)
	require.Equal(t, ihaCmd, cmd.CommandUUID)
	require.Equal(t, "InstallApplication", cmd.Command.Command.RequestType)
	rawCmd = string(cmd.Raw)
	require.Contains(t, rawCmd, ">"+ihaCmd+"<")
	require.Contains(t, rawCmd, `<key>ManagementFlags</key>
        <integer>1</integer>`)

	// enqueue a VPP app request for iOS host once more
	vpp1_1_ios = uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, hIOS.ID, vppApp1IOS.VPPAppID, vpp1_1_ios, "event-id-2-ios", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, ihaCmd, pendingActs[0].UUID)
	require.Equal(t, vpp1_1_ios, pendingActs[1].UUID)

	// record a result for in-house app and it should activate the next VPP app.
	cmdRes = &mdm.CommandResults{
		CommandUUID: ihaCmd,
		Status:      "Acknowledged",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, &fleet.ActivityTypeInstalledSoftware{
		HostID:      hIOS.ID,
		CommandUUID: ihaCmd,
		Status:      "Error", // using a failure because otherwise it requires verification to activate next
	})
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 1)
	require.Equal(t, vpp1_1_ios, pendingActs[0].UUID)

	// enqueue the in-house app again
	ihaCmd = uuid.NewString()
	err = ds.InsertHostInHouseAppInstall(ctx, hIOS.ID, ihaID, ihaTitleID, ihaCmd, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, vpp1_1_ios, pendingActs[0].UUID)
	require.Equal(t, ihaCmd, pendingActs[1].UUID)

	// record a successful result for the VPP app, will not activate the next until verification
	cmdRes = &mdm.CommandResults{
		CommandUUID: vpp1_1_ios,
		Status:      "Acknowledged",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, &fleet.ActivityTypeInstalledSoftware{
		HostID:      hIOS.ID,
		CommandUUID: vpp1_1_ios,
		Status:      string(fleet.SoftwareInstalled),
	})
	require.NoError(t, err)

	// both are still upcoming...
	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 2)
	require.Equal(t, vpp1_1_ios, pendingActs[0].UUID)
	require.Equal(t, ihaCmd, pendingActs[1].UUID)

	// mark the VPP app as verified, will activate the next activity
	err = ds.SetVPPInstallAsVerified(ctx, hIOS.ID, vpp1_1_ios, uuid.NewString())
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 1)
	require.Equal(t, ihaCmd, pendingActs[0].UUID)

	// record a successful result for the in-house app, will not become "past" until verification
	cmdRes = &mdm.CommandResults{
		CommandUUID: ihaCmd,
		Status:      "Acknowledged",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, nil, &fleet.ActivityTypeInstalledSoftware{
		HostID:      hIOS.ID,
		CommandUUID: ihaCmd,
		Status:      string(fleet.SoftwareInstalled),
	})
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 1)
	require.Equal(t, ihaCmd, pendingActs[0].UUID)

	// mark the in-house app as failed, will become "past"
	err = ds.SetVPPInstallAsFailed(ctx, hIOS.ID, ihaCmd, uuid.NewString())
	require.NoError(t, err)

	pendingActs, _, err = ds.ListHostUpcomingActivities(ctx, hIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 0)
}

func testActivateItselfOnEmptyQueue(t *testing.T, ds *Datastore) {
	activitySvc := NewTestActivityService(t, ds)
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	h1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, h1, false)
	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)
	nanoCtx := &mdm.Request{EnrollID: &mdm.EnrollID{ID: h1.UUID}, Context: ctx}

	vppApp1 := &fleet.VPPApp{
		Name: "vpp_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "vpp1", Platform: fleet.MacOSPlatform}},
		BundleIdentifier: "vpp1",
	}
	_, err = ds.InsertVPPAppWithTeam(ctx, vppApp1, nil)
	require.NoError(t, err)

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
		UninstallScript: "uninstall foo",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// create a pending software install request
	sw1_1, err := ds.InsertSoftwareInstallRequest(ctx, h1.ID, sw1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// set a result for the software install
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                h1.ID,
		InstallUUID:           sw1_1,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	// create a pending script execution request
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         h1.ID,
		ScriptContents: "echo 'a'",
	})
	require.NoError(t, err)
	script1_1 := hsr.ExecutionID

	// set a result for the script
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID: h1.ID, ExecutionID: script1_1, Output: "a", ExitCode: 0,
	}, nil)
	require.NoError(t, err)

	// create a pending uninstall request
	sw1_2 := uuid.NewString()
	err = ds.InsertSoftwareUninstallRequest(ctx, sw1_2, h1.ID, sw1, false)
	require.NoError(t, err)

	// set a result for the software uninstall
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      h1.ID,
		ExecutionID: sw1_2,
		ExitCode:    1,
	}, nil)
	require.NoError(t, err)

	// create a pending vpp app install
	vpp1_1 := uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, h1.ID, vppApp1.VPPAppID, vpp1_1, "event-id-1", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// set the result for the vpp app
	cmdRes := &mdm.CommandResults{
		CommandUUID: vpp1_1,
		Status:      "Error",
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err = nanoDB.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)
	err = activitySvc.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      h1.ID,
		AppStoreID:  vppApp1.VPPAppTeam.AdamID,
		CommandUUID: vpp1_1,
	})
	require.NoError(t, err)

	// the upcoming queue should be empty, each result having emptied the list
	// and each enqueue having triggered the next activity.
	pendingActs, _, err := ds.ListHostUpcomingActivities(ctx, h1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, pendingActs, 0)
}

func testCancelNonActivatedUpcomingActivity(t *testing.T, ds *Datastore) {
	activitySvc := NewTestActivityService(t, ds)
	newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		var apiUser *activity_api.User
		if user != nil {
			apiUser = &activity_api.User{ID: user.ID, Name: user.Name, Email: user.Email}
		}
		return activitySvc.NewActivity(ctx, apiUser, activity)
	}
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	host := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, host, false)
	hostLeftUntouched := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, hostLeftUntouched, false)
	hostIOS := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now(), test.WithPlatform("ios"))
	nanoEnrollAndSetHostMDMData(t, ds, hostIOS, false)

	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)

	// enqueue an activity on hostLeftUntouched, must still be there after the tests
	execIDUntouched := test.CreateHostScriptUpcomingActivity(t, ds, hostLeftUntouched)

	// cancel an activity on a non-existing host
	_, err = ds.CancelHostUpcomingActivity(ctx, 999, "non-existing")
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// cancel a non-existing activity on an existing host
	_, err = ds.CancelHostUpcomingActivity(ctx, host.ID, "non-existing")
	require.ErrorAs(t, err, &nfe)

	pluckExecIDs := func(acts []*fleet.UpcomingActivity) []string {
		var execIDs []string
		for _, act := range acts {
			execIDs = append(execIDs, act.UUID)
		}
		return execIDs
	}

	cases := []struct {
		desc        string
		host        *fleet.Host
		setup       func(t *testing.T) []string
		cancelIndex int
	}{
		{
			desc: "cancel software install",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				exec2 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostScriptResult(t, ds, host, exec1, 0)
				})
				return []string{exec1, exec2}
			},
			cancelIndex: 1,
		},
		{
			desc: "cancel script exec",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				exec2 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				t.Cleanup(func() {
					test.SetHostSoftwareInstallResult(t, ds, host, exec1, 0)
				})
				return []string{exec1, exec2}
			},
			cancelIndex: 1,
		},
		{
			desc: "cancel software uninstall",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				exec2 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostSoftwareInstallResult(t, ds, host, exec1, 0)
				})
				return []string{exec1, exec2}
			},
			cancelIndex: 1,
		},
		{
			desc: "cancel vpp install",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				exec2, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
				t.Cleanup(func() {
					test.SetHostSoftwareUninstallResult(t, ds, host, exec1, 0)
				})
				return []string{exec1, exec2}
			},
			cancelIndex: 1,
		},
		{
			desc: "cancel script with another activity after",
			host: host,
			setup: func(t *testing.T) []string {
				exec1, adamID := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
				exec2 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				exec3 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostVPPAppInstallResult(t, ds, nanoDB, host, exec1, adamID, "Acknowledged", newActivityFn)
					test.SetHostSoftwareInstallResult(t, ds, host, exec3, 0)
				})
				return []string{exec1, exec2, exec3}
			},
			cancelIndex: 1,
		},
		{
			desc: "cancel software uninstall with a couple activities before",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				exec2 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				exec3 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostSoftwareInstallResult(t, ds, host, exec1, 0)
					test.SetHostScriptResult(t, ds, host, exec2, 0)
				})
				return []string{exec1, exec2, exec3}
			},
			cancelIndex: 2,
		},
		{
			desc: "cancel in-house install",
			host: hostIOS,
			setup: func(t *testing.T) []string {
				exec1, adamID := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hostIOS)
				exec2 := test.CreateHostInHouseAppInstallUpcomingActivity(t, ds, hostIOS, u)
				t.Cleanup(func() {
					test.SetHostVPPAppInstallResult(t, ds, nanoDB, host, exec1, adamID, "Acknowledged", newActivityFn)
				})
				return []string{exec1, exec2}
			},
			cancelIndex: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			execIDs := c.setup(t)

			got, _, err := ds.ListHostUpcomingActivities(ctx, c.host.ID, fleet.ListOptions{})
			require.NoError(t, err)
			require.Len(t, got, len(execIDs))
			require.Equal(t, execIDs, pluckExecIDs(got))

			cancelExecID := execIDs[c.cancelIndex]
			expectedExecIDs := append(execIDs[:c.cancelIndex], execIDs[c.cancelIndex+1:]...) // nolint: gocritic
			_, err = ds.CancelHostUpcomingActivity(ctx, c.host.ID, cancelExecID)
			require.NoError(t, err)

			got, _, err = ds.ListHostUpcomingActivities(ctx, c.host.ID, fleet.ListOptions{})
			require.NoError(t, err)
			require.Len(t, got, len(expectedExecIDs))
			require.Equal(t, expectedExecIDs, pluckExecIDs(got))
		})
	}

	// check that hostLeftUntouched was... left untouched
	got, _, err := ds.ListHostUpcomingActivities(ctx, hostLeftUntouched.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, []string{execIDUntouched}, pluckExecIDs(got))
}

func testCancelActivatedUpcomingActivity(t *testing.T, ds *Datastore) {
	activitySvc := NewTestActivityService(t, ds)
	newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		var apiUser *activity_api.User
		if user != nil {
			apiUser = &activity_api.User{ID: user.ID, Name: user.Name, Email: user.Email}
		}
		return activitySvc.NewActivity(ctx, apiUser, activity)
	}
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	host := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, host, false)
	hostLeftUntouched := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, hostLeftUntouched, false)
	hostIOS := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now(), test.WithPlatform("ios"))
	nanoEnrollAndSetHostMDMData(t, ds, hostIOS, false)

	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)

	// enqueue an activity on hostLeftUntouched, must still be there after the tests
	execIDUntouched := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, hostLeftUntouched, u)

	pluckExecIDs := func(acts []*fleet.UpcomingActivity) []string {
		execIDs := []string{}
		for _, act := range acts {
			execIDs = append(execIDs, act.UUID)
		}
		return execIDs
	}

	cases := []struct {
		desc  string
		host  *fleet.Host
		setup func(t *testing.T) []string
	}{
		{
			desc: "cancel script",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				exec2 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostSoftwareInstallResult(t, ds, host, exec2, 0)
				})
				return []string{exec1, exec2}
			},
		},
		{
			desc: "cancel sofware install",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				exec2 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostSoftwareUninstallResult(t, ds, host, exec2, 0)
				})
				return []string{exec1, exec2}
			},
		},
		{
			desc: "cancel sofware uninstall",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				exec2, adamID := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
				t.Cleanup(func() {
					test.SetHostVPPAppInstallResult(t, ds, nanoDB, host, exec2, adamID, "Acknowledged", newActivityFn)
				})
				return []string{exec1, exec2}
			},
		},
		{
			desc: "cancel vpp install",
			host: host,
			setup: func(t *testing.T) []string {
				exec1, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
				exec2 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				t.Cleanup(func() {
					test.SetHostSoftwareInstallResult(t, ds, host, exec2, 0)
				})
				return []string{exec1, exec2}
			},
		},
		{
			desc: "cancel script none after",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				return []string{exec1}
			},
		},
		{
			desc: "cancel sofware install with a couple after",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
				exec2 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				exec3 := test.CreateHostScriptUpcomingActivity(t, ds, host)
				t.Cleanup(func() {
					test.SetHostSoftwareUninstallResult(t, ds, host, exec2, 0)
					test.SetHostScriptResult(t, ds, host, exec3, 0)
				})
				return []string{exec1, exec2, exec3}
			},
		},
		{
			desc: "cancel sofware uninstall none after",
			host: host,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
				return []string{exec1}
			},
		},
		{
			desc: "cancel vpp install same after",
			host: host,
			setup: func(t *testing.T) []string {
				exec1, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
				exec2, adamID := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
				t.Cleanup(func() {
					test.SetHostVPPAppInstallResult(t, ds, nanoDB, host, exec2, adamID, "Acknowledged", newActivityFn)
				})
				return []string{exec1, exec2}
			},
		},
		{
			desc: "cancel in-house install",
			host: hostIOS,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostInHouseAppInstallUpcomingActivity(t, ds, hostIOS, u)
				exec2, adamID := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hostIOS)
				t.Cleanup(func() {
					test.SetHostVPPAppInstallResult(t, ds, nanoDB, hostIOS, exec2, adamID, "Acknowledged", newActivityFn)
				})
				return []string{exec1, exec2}
			},
		},
		{
			desc: "cancel in-house install same after",
			host: hostIOS,
			setup: func(t *testing.T) []string {
				exec1 := test.CreateHostInHouseAppInstallUpcomingActivity(t, ds, hostIOS, u)
				exec2 := test.CreateHostInHouseAppInstallUpcomingActivity(t, ds, hostIOS, u)
				t.Cleanup(func() {
					test.SetHostInHouseAppInstallResult(t, ds, nanoDB, hostIOS, exec2, "Acknowledged", newActivityFn)
				})
				return []string{exec1, exec2}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			execIDs := c.setup(t)

			got, _, err := ds.ListHostUpcomingActivities(ctx, c.host.ID, fleet.ListOptions{})
			require.NoError(t, err)
			require.Len(t, got, len(execIDs))
			require.Equal(t, execIDs, pluckExecIDs(got))

			cancelExecID := execIDs[0]
			expectedExecIDs := execIDs[1:]
			_, err = ds.CancelHostUpcomingActivity(ctx, c.host.ID, cancelExecID)
			require.NoError(t, err)

			got, _, err = ds.ListHostUpcomingActivities(ctx, c.host.ID, fleet.ListOptions{})
			require.NoError(t, err)
			require.Len(t, got, len(expectedExecIDs))
			require.Equal(t, expectedExecIDs, pluckExecIDs(got))

			// the next upcoming activity (and only this one) should show up in those
			// lists of ready-to-process activities.
			var gotExecIDs []string
			scripts, err := ds.ListReadyToExecuteScriptsForHost(ctx, c.host.ID, false)
			require.NoError(t, err)
			require.True(t, len(scripts) <= 1)
			if len(scripts) == 1 {
				gotExecIDs = append(gotExecIDs, scripts[0].ExecutionID)
			}

			sws, err := ds.ListReadyToExecuteSoftwareInstalls(ctx, c.host.ID)
			require.NoError(t, err)
			require.True(t, len(sws) <= 1)
			gotExecIDs = append(gotExecIDs, sws...)

			var nanoExecIDs []string
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				err := sqlx.SelectContext(ctx, q, &nanoExecIDs, `SELECT command_uuid FROM nano_view_queue WHERE id = ? AND active = 1 AND status IS NULL`, c.host.UUID)
				return err
			})
			require.True(t, len(nanoExecIDs) <= 1)
			gotExecIDs = append(gotExecIDs, nanoExecIDs...)

			if len(expectedExecIDs) == 0 {
				require.Len(t, gotExecIDs, 0)
			} else {
				require.Len(t, gotExecIDs, 1)
				require.Equal(t, expectedExecIDs[0], gotExecIDs[0])
			}
		})
	}

	// check that hostLeftUntouched was... left untouched
	got, _, err := ds.ListHostUpcomingActivities(ctx, hostLeftUntouched.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, []string{execIDUntouched}, pluckExecIDs(got))
}

func testBatchCancelAllHostUpcomingActivities(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	host := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, host, false)
	hostIOS := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now(), test.WithPlatform("ios"))
	nanoEnrollAndSetHostMDMData(t, ds, hostIOS, false)
	hostLeftUntouched := test.NewHost(t, ds, "h3.local", "10.10.10.3", "3", "3", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, hostLeftUntouched, false)

	pluckExecIDs := func(acts []*fleet.UpcomingActivity) []string {
		execIDs := []string{}
		for _, act := range acts {
			execIDs = append(execIDs, act.UUID)
		}
		return execIDs
	}

	// edge case: host with no upcoming activities returns empty slice with no error
	canceled, err := ds.BatchCancelAllHostUpcomingActivities(ctx, hostLeftUntouched.ID)
	require.NoError(t, err)
	require.Empty(t, canceled)

	// enqueue an activity on hostLeftUntouched, must still be there after the test
	execIDUntouched := test.CreateHostScriptUpcomingActivity(t, ds, hostLeftUntouched)

	// enqueue mixed activities on the main host: the first becomes activated,
	// the rest stay queued.
	exec1 := test.CreateHostScriptUpcomingActivity(t, ds, host)
	exec2 := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
	exec3 := test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
	exec4, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
	expectedExecIDs := []string{exec1, exec2, exec3, exec4}

	got, _, err := ds.ListHostUpcomingActivities(ctx, host.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, got, len(expectedExecIDs))
	require.Equal(t, expectedExecIDs, pluckExecIDs(got))

	// exec1 should already be activated (single activity at enqueue time)
	meta, err := ds.GetHostUpcomingActivityMeta(ctx, host.ID, exec1)
	require.NoError(t, err)
	require.NotNil(t, meta.ActivatedAt)

	// cancel everything in one shot
	canceled, err = ds.BatchCancelAllHostUpcomingActivities(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, canceled, len(expectedExecIDs))

	// queue should now be empty
	got, _, err = ds.ListHostUpcomingActivities(ctx, host.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, got)

	// exec1 was activated, so its host_script_results row must be marked canceled
	var scriptCanceled bool
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &scriptCanceled,
			`SELECT canceled FROM host_script_results WHERE execution_id = ?`, exec1)
	})
	require.True(t, scriptCanceled)

	// hostLeftUntouched still has its single activity untouched
	got, _, err = ds.ListHostUpcomingActivities(ctx, hostLeftUntouched.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, []string{execIDUntouched}, pluckExecIDs(got))

	// repeat on an iOS host with an in-house app install (activated) followed by
	// a vpp install, to cover the in_house and vpp activated-cancel branches.
	exec5 := test.CreateHostInHouseAppInstallUpcomingActivity(t, ds, hostIOS, u)
	exec6, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hostIOS)

	got, _, err = ds.ListHostUpcomingActivities(ctx, hostIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{exec5, exec6}, pluckExecIDs(got))

	canceledIOS, err := ds.BatchCancelAllHostUpcomingActivities(ctx, hostIOS.ID)
	require.NoError(t, err)
	require.Len(t, canceledIOS, 2)

	got, _, err = ds.ListHostUpcomingActivities(ctx, hostIOS.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, got)

	// exec5 was activated; its host_in_house_software_installs row must be canceled
	var inHouseCanceled bool
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &inHouseCanceled,
			`SELECT canceled FROM host_in_house_software_installs WHERE command_uuid = ?`, exec5)
	})
	require.True(t, inHouseCanceled)
}

func testSetResultAfterCancelUpcomingActivity(t *testing.T, ds *Datastore) {
	activitySvc := NewTestActivityService(t, ds)
	newActivityFn := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		var apiUser *activity_api.User
		if user != nil {
			apiUser = &activity_api.User{ID: user.ID, Name: user.Name, Email: user.Email}
		}
		return activitySvc.NewActivity(ctx, apiUser, activity)
	}
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)
	host := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	nanoEnrollAndSetHostMDMData(t, ds, host, false)
	nanoDB, err := nanomdm_mysql.New(nanomdm_mysql.WithDB(ds.primary.DB))
	require.NoError(t, err)

	// set a script result post-cancel
	exec := test.CreateHostScriptUpcomingActivity(t, ds, host)
	_, err = ds.CancelHostUpcomingActivity(ctx, host.ID, exec)
	require.NoError(t, err)
	test.SetHostScriptResult(t, ds, host, exec, 0)

	// set a software install result post-cancel
	exec = test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host, u)
	_, err = ds.CancelHostUpcomingActivity(ctx, host.ID, exec)
	require.NoError(t, err)
	test.SetHostSoftwareInstallResult(t, ds, host, exec, 0)

	// set a software uninstall result post-cancel
	exec = test.CreateHostSoftwareUninstallUpcomingActivity(t, ds, host, u)
	_, err = ds.CancelHostUpcomingActivity(ctx, host.ID, exec)
	require.NoError(t, err)
	test.SetHostSoftwareUninstallResult(t, ds, host, exec, 0)

	// set a vpp app install result post-cancel
	exec, adamID := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, host)
	_, err = ds.CancelHostUpcomingActivity(ctx, host.ID, exec)
	require.NoError(t, err)
	test.SetHostVPPAppInstallResult(t, ds, nanoDB, host, exec, adamID, "Acknowledged", newActivityFn)
}

func testGetHostUpcomingActivityMeta(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host1 := test.NewHost(t, ds, "h1.local", "10.10.10.1", "1", "1", time.Now())
	host2 := test.NewHost(t, ds, "h2.local", "10.10.10.2", "2", "2", time.Now())
	host1.Platform = "linux"
	host2.Platform = "linux"
	err := ds.UpdateHost(ctx, host1)
	require.NoError(t, err)
	err = ds.UpdateHost(ctx, host2)
	require.NoError(t, err)
	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	// get meta with unknown host
	_, err = ds.GetHostUpcomingActivityMeta(ctx, 999, "non-existing")
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// get meta with unknown exec ID
	_, err = ds.GetHostUpcomingActivityMeta(ctx, host1.ID, "non-existing")
	require.ErrorAs(t, err, &nfe)

	assertActivityMeta := func(want, got *fleet.UpcomingActivityMeta) {
		require.Equal(t, want.ExecutionID, got.ExecutionID)
		// we just assert activated vs non-activated
		require.Equal(t, want.ActivatedAt != nil, got.ActivatedAt != nil)
		require.Equal(t, want.UpcomingActivityType, got.UpcomingActivityType)
		require.Equal(t, want.WellKnownAction, got.WellKnownAction)
	}

	// create an install request that is not any special command
	swExecID := test.CreateHostSoftwareInstallUpcomingActivity(t, ds, host1, u)
	meta, err := ds.GetHostUpcomingActivityMeta(ctx, host1.ID, swExecID)
	require.NoError(t, err)
	assertActivityMeta(&fleet.UpcomingActivityMeta{
		ExecutionID:          swExecID,
		ActivatedAt:          ptr.Time(time.Now()), // will just check nil vs non-nil
		UpcomingActivityType: "software_install",
		WellKnownAction:      fleet.WellKnownActionNone,
	}, meta)

	// create a lock request on host1
	err = ds.LockHostViaScript(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID}, "linux")
	require.NoError(t, err)

	// create a wipe request on host2
	err = ds.WipeHostViaScript(ctx, &fleet.HostScriptRequestPayload{HostID: host2.ID}, "linux")
	require.NoError(t, err)

	// grab the exec ID of the lock
	activities, _, err := ds.ListHostUpcomingActivities(ctx, host1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 2)
	require.Equal(t, swExecID, activities[0].UUID)
	lockExecID := activities[1].UUID

	// grab the exec ID of the wipe
	activities, _, err = ds.ListHostUpcomingActivities(ctx, host2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, activities, 1)
	wipeExecID := activities[0].UUID

	// lock meta is as expected
	meta, err = ds.GetHostUpcomingActivityMeta(ctx, host1.ID, lockExecID)
	require.NoError(t, err)
	assertActivityMeta(&fleet.UpcomingActivityMeta{
		ExecutionID:          lockExecID,
		ActivatedAt:          nil,
		UpcomingActivityType: "script",
		WellKnownAction:      fleet.WellKnownActionLock,
	}, meta)

	// wipe meta is as expected
	meta, err = ds.GetHostUpcomingActivityMeta(ctx, host2.ID, wipeExecID)
	require.NoError(t, err)
	assertActivityMeta(&fleet.UpcomingActivityMeta{
		ExecutionID:          wipeExecID,
		ActivatedAt:          ptr.Time(time.Now()), // will just check nil vs non-nil
		UpcomingActivityType: "script",
		WellKnownAction:      fleet.WellKnownActionWipe,
	}, meta)

	// set a result for the software install
	test.SetHostSoftwareInstallResult(t, ds, host1, swExecID, 0)

	// the lock script is now activated
	meta, err = ds.GetHostUpcomingActivityMeta(ctx, host1.ID, lockExecID)
	require.NoError(t, err)
	assertActivityMeta(&fleet.UpcomingActivityMeta{
		ExecutionID:          lockExecID,
		ActivatedAt:          ptr.Time(time.Now()), // will just check nil vs non-nil
		UpcomingActivityType: "script",
		WellKnownAction:      fleet.WellKnownActionLock,
	}, meta)

	// wipe meta on host2 is unchanged
	meta, err = ds.GetHostUpcomingActivityMeta(ctx, host2.ID, wipeExecID)
	require.NoError(t, err)
	assertActivityMeta(&fleet.UpcomingActivityMeta{
		ExecutionID:          wipeExecID,
		ActivatedAt:          ptr.Time(time.Now()), // will just check nil vs non-nil
		UpcomingActivityType: "script",
		WellKnownAction:      fleet.WellKnownActionWipe,
	}, meta)

	// enqueue a new script activity
	scrExecID := test.CreateHostScriptUpcomingActivity(t, ds, host1)
	// its meta is as expected
	meta, err = ds.GetHostUpcomingActivityMeta(ctx, host1.ID, scrExecID)
	require.NoError(t, err)
	assertActivityMeta(&fleet.UpcomingActivityMeta{
		ExecutionID:          scrExecID,
		ActivatedAt:          nil,
		UpcomingActivityType: "script",
		WellKnownAction:      fleet.WellKnownActionNone,
	}, meta)

	// set a result for the lock action
	test.SetHostScriptResult(t, ds, host1, lockExecID, 0)
	// its meta is now non-existing
	_, err = ds.GetHostUpcomingActivityMeta(ctx, host1.ID, lockExecID)
	require.ErrorAs(t, err, &nfe)

	// set a result for the wipe action
	test.SetHostScriptResult(t, ds, host2, wipeExecID, 0)
	// its meta is now non-existing
	_, err = ds.GetHostUpcomingActivityMeta(ctx, host2.ID, wipeExecID)
	require.ErrorAs(t, err, &nfe)
}

func testUnblockHostsUpcomingActivityQueue(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	// create a few hosts
	hosts := make([]*fleet.Host, 5)
	for i := range hosts {
		host := test.NewHost(t, ds, fmt.Sprintf("h%d.local", i+1), fmt.Sprintf("10.10.10.%d", i+1), fmt.Sprint(i+1), fmt.Sprint(i+1), time.Now())
		hosts[i] = host
	}

	// run without anything in any host queue
	n, err := ds.UnblockHostsUpcomingActivityQueue(ctx, 10, false)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	deleteUpcomingActivityToBlockQueue := func(execID string) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM upcoming_activities WHERE execution_id = ?`, execID)
			return err
		})
	}

	// enqueue some activities on some hosts (the nature of the activity is not relevant)
	host0ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[0].ID, ScriptContents: "A", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host0ScriptB, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[0].ID, ScriptContents: "B", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host1ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[1].ID, ScriptContents: "A", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host2ScriptA, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[2].ID, ScriptContents: "A", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], host0ScriptA.ExecutionID, host0ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], host1ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], host2ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3])
	checkUpcomingActivities(t, ds, hosts[4])

	// nothing to unblock
	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 10, false)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// block queue for host 0
	deleteUpcomingActivityToBlockQueue(host0ScriptA.ExecutionID)

	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 10, false)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	checkUpcomingActivities(t, ds, hosts[0], host0ScriptB.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], host1ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], host2ScriptA.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3])
	checkUpcomingActivities(t, ds, hosts[4])

	// enqueue script C for all hosts
	host0ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[0].ID, ScriptContents: "C", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host1ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[1].ID, ScriptContents: "C", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host2ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[2].ID, ScriptContents: "C", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host3ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[3].ID, ScriptContents: "C", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host4ScriptC, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[4].ID, ScriptContents: "C", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], host0ScriptB.ExecutionID, host0ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], host1ScriptA.ExecutionID, host1ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], host2ScriptA.ExecutionID, host2ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], host3ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[4], host4ScriptC.ExecutionID)

	// block queue for all hosts, but since hosts 3 and 4 are now empty, no need
	// to unblock
	deleteUpcomingActivityToBlockQueue(host0ScriptB.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host1ScriptA.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host2ScriptA.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host3ScriptC.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host4ScriptC.ExecutionID)

	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 10, false)
	require.NoError(t, err)
	require.Equal(t, 3, n)

	checkUpcomingActivities(t, ds, hosts[0], host0ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], host1ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], host2ScriptC.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3])
	checkUpcomingActivities(t, ds, hosts[4])

	// enqueue script D and E for all hosts
	host0ScriptD, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[0].ID, ScriptContents: "D", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host1ScriptD, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[1].ID, ScriptContents: "D", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host2ScriptD, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[2].ID, ScriptContents: "D", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host3ScriptD, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[3].ID, ScriptContents: "D", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host4ScriptD, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[4].ID, ScriptContents: "D", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host0ScriptE, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[0].ID, ScriptContents: "E", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host1ScriptE, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[1].ID, ScriptContents: "E", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host2ScriptE, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[2].ID, ScriptContents: "E", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host3ScriptE, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[3].ID, ScriptContents: "E", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)
	host4ScriptE, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: hosts[4].ID, ScriptContents: "E", UserID: &u.ID, SyncRequest: true})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, hosts[0], host0ScriptC.ExecutionID, host0ScriptD.ExecutionID, host0ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], host1ScriptC.ExecutionID, host1ScriptD.ExecutionID, host1ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], host2ScriptC.ExecutionID, host2ScriptD.ExecutionID, host2ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], host3ScriptD.ExecutionID, host3ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[4], host4ScriptD.ExecutionID, host4ScriptE.ExecutionID)

	// block queue for all hosts
	deleteUpcomingActivityToBlockQueue(host0ScriptC.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host1ScriptC.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host2ScriptC.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host3ScriptD.ExecutionID)
	deleteUpcomingActivityToBlockQueue(host4ScriptD.ExecutionID)

	// process max 3 hosts
	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 3, false)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	// run again, should process the next 2 hosts
	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 3, false)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	// run again, nothing to unblock
	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 3, false)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	checkUpcomingActivities(t, ds, hosts[0], host0ScriptD.ExecutionID, host0ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[1], host1ScriptD.ExecutionID, host1ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[2], host2ScriptD.ExecutionID, host2ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[3], host3ScriptE.ExecutionID)
	checkUpcomingActivities(t, ds, hosts[4], host4ScriptE.ExecutionID)
}

func testReleaseFleetInitiatedUpcomingActivities(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	u := test.NewUser(t, ds, "user1", "user1@example.com", false)
	policy, err := ds.NewGlobalPolicy(ctx, &u.ID, fleet.PolicyPayload{Name: "release-budget-policy", Query: "SELECT 1"})
	require.NoError(t, err)
	require.NotNil(t, policy)

	hosts := make([]*fleet.Host, 3)
	for i := range hosts {
		hosts[i] = test.NewHost(t, ds, fmt.Sprintf("hr%d.local", i+1), fmt.Sprintf("10.20.10.%d", i+1),
			fmt.Sprintf("release-%d", i+1), fmt.Sprintf("release-%d", i+1), time.Now())
	}

	enqueueGated := func(hostID uint, contents string) *fleet.HostScriptResult {
		hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
			HostID:          hostID,
			ScriptContents:  contents,
			PolicyID:        &policy.ID,
			DeferActivation: true,
		})
		require.NoError(t, err)
		return hsr
	}

	readyExecIDs := func(hostID uint) []string {
		ready, err := ds.ListReadyToExecuteScriptsForHost(ctx, hostID, false)
		require.NoError(t, err)
		ids := make([]string, 0, len(ready))
		for _, r := range ready {
			ids = append(ids, r.ExecutionID)
		}
		return ids
	}

	// nothing gated yet
	n, err := ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// gated enqueues are not activated: invisible to the host
	gated0 := enqueueGated(hosts[0].ID, "A")
	gated1 := enqueueGated(hosts[1].ID, "A")
	require.Empty(t, readyExecIDs(hosts[0].ID))
	require.Empty(t, readyExecIDs(hosts[1].ID))

	// the unblock job must not bypass the release budget
	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 10, true)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.Empty(t, readyExecIDs(hosts[0].ID))

	// release honors the budget, oldest host first
	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, []string{gated0.ExecutionID}, readyExecIDs(hosts[0].ID))
	require.Empty(t, readyExecIDs(hosts[1].ID))

	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, []string{gated1.ExecutionID}, readyExecIDs(hosts[1].ID))

	// nothing left to release
	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// a host with an in-flight (activated) activity is not double-released even
	// with gated work behind it: per-host serialization is preserved
	enqueueGated(hosts[0].ID, "B")
	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)
	require.Equal(t, []string{gated0.ExecutionID}, readyExecIDs(hosts[0].ID))

	// a user-initiated script on a host with only gated work activates
	// immediately and jumps ahead of the gated activity (higher priority)
	gated2 := enqueueGated(hosts[2].ID, "G")
	userScript, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         hosts[2].ID,
		ScriptContents: "U",
		UserID:         &u.ID,
		SyncRequest:    true,
	})
	require.NoError(t, err)
	require.Equal(t, []string{userScript.ExecutionID}, readyExecIDs(hosts[2].ID))

	// while the user script is in flight, release leaves the host alone
	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// completing the user script chain-activates the gated activity (a person
	// touching the host releases its queue early, by design)
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      hosts[2].ID,
		ExecutionID: userScript.ExecutionID,
		Output:      "ok",
		ExitCode:    0,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{gated2.ExecutionID}, readyExecIDs(hosts[2].ID))

	// legacy unblock behavior (skipFleetInitiated=false) still rescues hosts
	// with gated-only work, for deployments without the release budget
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM upcoming_activities WHERE execution_id = ?`, gated2.ExecutionID)
		return err
	})
	enqueueGated(hosts[2].ID, "H")
	n, err = ds.UnblockHostsUpcomingActivityQueue(ctx, 10, false)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Len(t, readyExecIDs(hosts[2].ID), 1)

	// the software-install enqueue path honors DeferActivation the same way:
	// invisible to the host until released
	installHost := test.NewHost(t, ds, "hr4.local", "10.20.10.4", "release-4", "release-4", time.Now())
	installerFile, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install foo",
		InstallerFile:   installerFile,
		StorageID:       uuid.NewString(),
		Filename:        "foo.pkg",
		Title:           uuid.NewString(),
		Source:          "apps",
		Version:         "0.0.1",
		UserID:          u.ID,
		UninstallScript: "uninstall foo",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	installExecID, err := ds.InsertSoftwareInstallRequest(ctx, installHost.ID, installerID, fleet.HostSoftwareInstallOptions{
		PolicyID:        &policy.ID,
		DeferActivation: true,
	})
	require.NoError(t, err)

	installs, err := ds.ListReadyToExecuteSoftwareInstalls(ctx, installHost.ID)
	require.NoError(t, err)
	require.Empty(t, installs)

	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	installs, err = ds.ListReadyToExecuteSoftwareInstalls(ctx, installHost.ID)
	require.NoError(t, err)
	require.Equal(t, []string{installExecID}, installs)

	// a "poison" host whose activation deterministically fails must not wedge
	// the release pipeline: with oldest-first selection it would otherwise be
	// re-picked every run and roll back every other host in its chunk
	poisonHost := test.NewHost(t, ds, "hr5.local", "10.20.10.5", "release-5", "release-5", time.Now())
	healthyHost := test.NewHost(t, ds, "hr6.local", "10.20.10.6", "release-6", "release-6", time.Now())
	poisonScript := enqueueGated(poisonHost.ID, "P")
	healthyScript := enqueueGated(healthyHost.ID, "Q")
	// pre-seed a host_script_results row with the poison activity's execution
	// ID so its activation INSERT hits the unique key and fails
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_script_results (host_id, execution_id, script_content_id, output)
			SELECT ua.host_id, ua.execution_id, sua.script_content_id, ''
			FROM upcoming_activities ua
			JOIN script_upcoming_activities sua ON sua.upcoming_activity_id = ua.id
			WHERE ua.execution_id = ?`, poisonScript.ExecutionID)
		return err
	})

	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 10)
	require.Error(t, err, "the poison host's activation error must be reported")
	require.ErrorContains(t, err, fmt.Sprintf("host %d", poisonHost.ID))
	require.Equal(t, 1, n, "the healthy host must be released despite the poison host")
	require.Equal(t, []string{healthyScript.ExecutionID}, readyExecIDs(healthyHost.ID))

	// unwedge the poison host and confirm it releases on the next run
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_script_results WHERE execution_id = ?`, poisonScript.ExecutionID)
		return err
	})
	n, err = ds.ReleaseFleetInitiatedUpcomingActivities(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, []string{poisonScript.ExecutionID}, readyExecIDs(poisonHost.ID))
}

// TestReapableActivatedInstallArgs pins the conversion the reap predicate depends on. A duration
// that reaches the query as 0 makes the cutoff NOW(), so every activated install on the fleet is
// past it. The sub-second case in testReapStuckActivatedMDMInstalls says the same thing end to end,
// but has to race a wall clock to do it.
func TestReapableActivatedInstallArgs(t *testing.T) {
	for _, tc := range []struct {
		olderThan  time.Duration
		wantMicros int64
	}{
		{time.Microsecond, 1},
		{time.Millisecond, 1_000},
		{500 * time.Millisecond, 500_000},
		{999 * time.Millisecond, 999_000},
		{24 * time.Hour, 86_400_000_000},
	} {
		t.Run(tc.olderThan.String(), func(t *testing.T) {
			args := reapableActivatedInstallArgs(tc.olderThan)
			require.Len(t, args, 3)
			require.Equal(t, tc.wantMicros, args[0], "reap age must not truncate")
			require.Equal(t, tc.wantMicros, args[1], "answer age must not truncate")
			require.Equal(t, mdmApplePushDeliveryGraceDays, args[2])
		})
	}
}

func testReapStuckActivatedMDMInstalls(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	u := test.NewUser(t, ds, "reaper-user", "reaper-user@example.com", false)

	const reapAfter = 24 * time.Hour
	agedActivation := time.Now().Add(-48 * time.Hour)

	var hostSeq int
	newMDMHost := func(opts ...test.NewHostOption) *fleet.Host {
		hostSeq++
		h := test.NewHost(t, ds, fmt.Sprintf("reap%d.local", hostSeq), fmt.Sprintf("10.20.30.%d", hostSeq),
			fmt.Sprintf("reap-key-%d", hostSeq), fmt.Sprintf("reap-uuid-%d", hostSeq), time.Now(), opts...)
		nanoEnrollAndSetHostMDMData(t, ds, h, false)
		return h
	}

	// advance covers the case where an insert onto a non-empty queue did not activate itself
	advance := func(host *fleet.Host, fromCompletedExecID string) {
		_, err := ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host.ID, fromCompletedExecID)
		require.NoError(t, err)
	}

	// ageActivations backdates the activation so the rows are older than the reap timeout
	ageActivations := func(execIDs ...string) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			stmt, args, err := sqlx.In(
				`UPDATE upcoming_activities SET activated_at = ? WHERE execution_id IN (?)`, agedActivation, execIDs)
			if err != nil {
				return err
			}
			_, err = q.ExecContext(ctx, stmt, args...)
			return err
		})
	}

	// answeredAt records a command result, which is what a device reply leaves behind. updated_at
	// carries the answer time: GetUnverifiedVPPInstallsForHost selects it as ack_at and the verify
	// handler times its own budget against it, so the reaper ages the answered branch from it too.
	answeredAt := func(host *fleet.Host, execID, status string, at time.Time) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				INSERT INTO nano_command_results (id, command_uuid, status, result, updated_at)
				VALUES (?, ?, ?, '<?xml version="1.0" encoding="UTF-8"?>', ?)`, host.UUID, execID, status, at)
			return err
		})
	}
	// deliver is the reported shape: acknowledged back when the install activated, unverified since
	deliver := func(host *fleet.Host, execID string) {
		answeredAt(host, execID, "Acknowledged", agedActivation)
	}

	ageNanoQueue := func(execID string, at time.Time) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE nano_enrollment_queue SET created_at = ? WHERE command_uuid = ?`, at, execID)
			return err
		})
	}

	deactivateNanoQueue := func(execID string) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE nano_enrollment_queue SET active = 0 WHERE command_uuid = ?`, execID)
			return err
		})
	}

	queuedExecIDs := func(host *fleet.Host) []string {
		acts, _, err := ds.ListHostUpcomingActivities(ctx, host.ID, fleet.ListOptions{})
		require.NoError(t, err)
		ids := make([]string, 0, len(acts))
		for _, a := range acts {
			ids = append(ids, a.UUID)
		}
		return ids
	}

	type verifyState struct {
		VerificationAt       *time.Time `db:"verification_at"`
		VerificationFailedAt *time.Time `db:"verification_failed_at"`
	}
	vppVerifyState := func(execID string) verifyState {
		var vs verifyState
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &vs,
				`SELECT verification_at, verification_failed_at FROM host_vpp_software_installs WHERE command_uuid = ?`, execID)
		})
		return vs
	}
	nanoQueueActive := func(execID string) bool {
		var active bool
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &active,
				`SELECT active FROM nano_enrollment_queue WHERE command_uuid = ?`, execID)
		})
		return active
	}
	hasVerifyLock := func(host *fleet.Host) bool {
		var n int
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n,
				`SELECT COUNT(*) FROM host_mdm_commands WHERE host_id = ? AND command_type = ?`,
				host.ID, fleet.VerifySoftwareInstallVPPPrefix)
		})
		return n > 0
	}

	// nothing to reap on a fleet with no activity at all
	reaped, err := ds.ReapStuckActivatedMDMInstalls(ctx, reapAfter, 10)
	require.NoError(t, err)
	require.Empty(t, reaped)

	// hAcked: delivered, never verified, aged. The reported case. A script is queued behind it to
	// prove the whole queue is released, not just the install.
	hAcked := newMDMHost()
	ackedExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hAcked)
	advance(hAcked, "")
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hAcked.ID, ScriptContents: "echo reaped",
	})
	require.NoError(t, err)
	ackedScriptExec := hsr.ExecutionID
	deliver(hAcked, ackedExec)
	ageActivations(ackedExec)
	// An automatic update, so the emitted activity has to say so. Written as 1 and not TRUE:
	// raw SQL TRUE stores a JSON boolean, while a Go bool through the driver stores the number
	// the reaper's `= 1` test matches, so only 1 reproduces what production writes.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE upcoming_activities SET payload = JSON_SET(payload, '$.from_auto_update', 1) WHERE execution_id = ?`,
			ackedExec)
		return err
	})
	// the acknowledgement that started verification also took the verify lock
	require.NoError(t, ds.AddHostMDMCommands(ctx, []fleet.HostMDMCommand{
		{HostID: hAcked.ID, CommandType: fleet.VerifySoftwareInstallVPPPrefix},
	}))
	require.Equal(t, []string{ackedExec, ackedScriptExec}, queuedExecIDs(hAcked))

	// hOffline: aged, but the command has not been delivered and its queue row is still live and
	// inside the push window, so the device may yet install it. This is the regression guard: a
	// bare age test would fail every install to a host that is merely switched off.
	hOffline := newMDMHost()
	offlineExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hOffline)
	advance(hOffline, "")
	ageActivations(offlineExec)

	// hNotNow: the device answered, but with NotNow, so nanomdm keeps the command queued and will
	// re-serve it. The install has not run, so it must be treated as undelivered and spared.
	hNotNow := newMDMHost()
	notNowExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hNotNow)
	advance(hNotNow, "")
	answeredAt(hNotNow, notNowExec, "NotNow", agedActivation)
	ageActivations(notNowExec)

	// hBackdated: its queue row carries the activity's old created_at, because both enqueue paths
	// copy it to preserve ordering. That says nothing about whether the device can still receive
	// the command, and it is the shape of every install behind a head the reaper has just freed.
	hBackdated := newMDMHost()
	backdatedExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hBackdated)
	advance(hBackdated, "")
	ageActivations(backdatedExec) // 48h, past the reap floor
	ageNanoQueue(backdatedExec, time.Now().Add(-30*24*time.Hour))

	// hLateAck: away longer than the delivery grace, then came back and acknowledged. The delivery
	// branches must not apply to an install that has been answered, or returning after a long
	// absence would fail the install the device is at that moment running.
	hLateAck := newMDMHost()
	lateAckExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hLateAck)
	advance(hLateAck, "")
	answeredAt(hLateAck, lateAckExec, "Acknowledged", time.Now())
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE upcoming_activities SET activated_at = ? WHERE execution_id = ?`,
			time.Now().Add(-9*24*time.Hour), lateAckExec)
		return err
	})

	// hUndeliverable: activated longer ago than the delivery grace, still unanswered, so Fleet has
	// stopped pushing it and it is never going to arrive
	hExpired := newMDMHost()
	expiredExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hExpired)
	advance(hExpired, "")
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE upcoming_activities SET activated_at = ? WHERE execution_id = ?`,
			time.Now().Add(-8*24*time.Hour), expiredExec)
		return err
	})

	// hPulled: undelivered, and its queue row was deactivated out from under it
	hPulled := newMDMHost()
	pulledExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hPulled)
	advance(hPulled, "")
	ageActivations(pulledExec)
	deactivateNanoQueue(pulledExec)

	// hFresh: answered but activated just now, so still inside the timeout
	hFresh := newMDMHost()
	freshExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hFresh)
	advance(hFresh, "")
	answeredAt(hFresh, freshExec, "Acknowledged", time.Now())

	// hJustAcked: past the timeout by activation age, but the device only just came back and
	// acknowledged, so verification is in flight and entitled to its own budget. Reaping on
	// activation age alone would fail an app that is at that moment installing.
	hJustAcked := newMDMHost()
	justAckedExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hJustAcked)
	advance(hJustAcked, "")
	answeredAt(hJustAcked, justAckedExec, "Acknowledged", time.Now())
	ageActivations(justAckedExec)

	// hBatch: a full activation batch. A script goes first so that all the installs queue up
	// behind it and then activate together when it completes.
	hBatch := newMDMHost()
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hBatch.ID, ScriptContents: "echo batch",
	})
	require.NoError(t, err)
	batchExecs := make([]string, 0, 6)
	for range 6 {
		execID, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hBatch)
		batchExecs = append(batchExecs, execID)
	}
	advance(hBatch, hsr.ExecutionID)
	// maxMDMCommandActivations caps a batch at 5, so the sixth is still waiting
	activatedBatch := batchExecs[:5]
	for _, execID := range activatedBatch {
		deliver(hBatch, execID)
	}
	ageActivations(activatedBatch...)

	// hVerified: already verified, but its row was left activated. Nothing to fail, only to
	// advance past, and the verified outcome must survive.
	hVerified := newMDMHost()
	verifiedExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hVerified)
	advance(hVerified, "")
	deliver(hVerified, verifiedExec)
	ageActivations(verifiedExec)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE host_vpp_software_installs SET verification_at = NOW(6) WHERE command_uuid = ?`, verifiedExec)
		return err
	})

	// hInHouse: an in-house app install blocks the queue by the same rule, and on iOS, so this
	// also covers the platform independence of the whole mechanism
	hInHouse := newMDMHost(test.WithPlatform("ios"))
	inHouseExec := test.CreateHostInHouseAppInstallUpcomingActivity(t, ds, hInHouse, u)
	advance(hInHouse, "")
	deliver(hInHouse, inHouseExec)
	ageActivations(inHouseExec)

	// hMixed: a batch holding both reapable and not-yet-reapable installs. Only the reapable ones
	// are failed, and the queue stays blocked by the one whose command can still be delivered.
	hMixed := newMDMHost()
	hsr, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: hMixed.ID, ScriptContents: "echo mixed",
	})
	require.NoError(t, err)
	mixedExecs := make([]string, 0, 3)
	for range 3 {
		execID, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hMixed)
		mixedExecs = append(mixedExecs, execID)
	}
	advance(hMixed, hsr.ExecutionID)
	deliver(hMixed, mixedExecs[0])
	deliver(hMixed, mixedExecs[1])
	ageActivations(mixedExecs...)

	reaped, err = ds.ReapStuckActivatedMDMInstalls(ctx, reapAfter, 10)
	require.NoError(t, err)

	byCommandUUID := make(map[string]fleet.ReapedMDMInstall, len(reaped))
	for _, r := range reaped {
		byCommandUUID[r.CommandUUID] = r
	}
	expectedReaped := append([]string{ackedExec, expiredExec, pulledExec, inHouseExec}, activatedBatch...)
	expectedReaped = append(expectedReaped, mixedExecs[0], mixedExecs[1])
	require.Len(t, reaped, len(expectedReaped))
	for _, execID := range expectedReaped {
		require.Contains(t, byCommandUUID, execID)
	}

	// the reported case: the install is failed and the script behind it runs
	require.Equal(t, []string{ackedScriptExec}, queuedExecIDs(hAcked))
	require.NotNil(t, vppVerifyState(ackedExec).VerificationFailedAt)
	require.False(t, nanoQueueActive(ackedExec), "a reaped command must not stay pushable")
	require.False(t, hasVerifyLock(hAcked), "the verify lock must not outlive the install it was taken for")

	ackedEntry := byCommandUUID[ackedExec]
	require.Equal(t, hAcked.ID, ackedEntry.HostID)
	require.Equal(t, hAcked.UUID, ackedEntry.HostUUID)
	require.NotNil(t, ackedEntry.AppStoreActivity)
	require.Equal(t, string(fleet.SoftwareInstallFailed), ackedEntry.AppStoreActivity.Status)
	require.True(t, ackedEntry.AppStoreActivity.FromAutoUpdate,
		"the activity must carry over that the install came from an automatic update")
	require.Nil(t, ackedEntry.InHouseActivity)

	// a person-requested install reads as such, so the flag is not simply always set
	require.NotNil(t, byCommandUUID[expiredExec].AppStoreActivity)
	require.False(t, byCommandUUID[expiredExec].AppStoreActivity.FromAutoUpdate)

	// an install that can still be delivered is left alone, queue and all
	require.Equal(t, []string{offlineExec}, queuedExecIDs(hOffline))
	require.Nil(t, vppVerifyState(offlineExec).VerificationFailedAt)
	require.True(t, nanoQueueActive(offlineExec))
	require.NotContains(t, byCommandUUID, offlineExec)

	// a NotNow reply is not an answer: nanomdm re-serves the command, so the install is still
	// pending and must be spared exactly like an undelivered one
	require.Equal(t, []string{notNowExec}, queuedExecIDs(hNotNow))
	require.Nil(t, vppVerifyState(notNowExec).VerificationFailedAt)
	require.True(t, nanoQueueActive(notNowExec))
	require.NotContains(t, byCommandUUID, notNowExec)

	// a backdated queue row is not evidence the command is undeliverable. Reaping on it would fail
	// every install sitting behind a head the reaper had only just freed.
	require.Equal(t, []string{backdatedExec}, queuedExecIDs(hBackdated))
	require.Nil(t, vppVerifyState(backdatedExec).VerificationFailedAt)
	require.True(t, nanoQueueActive(backdatedExec))
	require.NotContains(t, byCommandUUID, backdatedExec)

	// an install that can no longer be delivered is reaped, however it got there
	require.Empty(t, queuedExecIDs(hExpired))
	require.NotNil(t, vppVerifyState(expiredExec).VerificationFailedAt)
	require.Empty(t, queuedExecIDs(hPulled))
	require.NotNil(t, vppVerifyState(pulledExec).VerificationFailedAt)

	// still inside the timeout
	require.Equal(t, []string{freshExec}, queuedExecIDs(hFresh))
	require.Nil(t, vppVerifyState(freshExec).VerificationFailedAt)
	require.NotContains(t, byCommandUUID, freshExec)

	// a just-acknowledged install keeps its verification window even though it was activated long
	// before the device came back to answer
	require.Equal(t, []string{justAckedExec}, queuedExecIDs(hJustAcked))
	require.Nil(t, vppVerifyState(justAckedExec).VerificationFailedAt)
	require.NotContains(t, byCommandUUID, justAckedExec)

	// and it keeps it even when the absence ran past the delivery grace, since the answer settles
	// delivery and the grace no longer has anything to say
	require.Equal(t, []string{lateAckExec}, queuedExecIDs(hLateAck))
	require.Nil(t, vppVerifyState(lateAckExec).VerificationFailedAt)
	require.NotContains(t, byCommandUUID, lateAckExec)

	// the whole batch goes in one pass, and the install waiting behind it activates
	require.Equal(t, []string{batchExecs[5]}, queuedExecIDs(hBatch))
	for _, execID := range activatedBatch {
		require.NotNil(t, vppVerifyState(execID).VerificationFailedAt, "batch member %s", execID)
	}

	// a verified install is advanced past, not re-failed
	require.Empty(t, queuedExecIDs(hVerified))
	verified := vppVerifyState(verifiedExec)
	require.NotNil(t, verified.VerificationAt)
	require.Nil(t, verified.VerificationFailedAt, "a verified install must not be overwritten as failed")
	require.NotContains(t, byCommandUUID, verifiedExec)

	// in-house apps reap the same way, and carry the other activity type
	require.Empty(t, queuedExecIDs(hInHouse))
	inHouseEntry := byCommandUUID[inHouseExec]
	require.NotNil(t, inHouseEntry.InHouseActivity)
	require.Nil(t, inHouseEntry.AppStoreActivity)

	// only the reapable half of a mixed batch is failed, and the queue stays blocked on purpose
	require.Equal(t, []string{mixedExecs[2]}, queuedExecIDs(hMixed))
	require.NotNil(t, vppVerifyState(mixedExecs[0]).VerificationFailedAt)
	require.NotNil(t, vppVerifyState(mixedExecs[1]).VerificationFailedAt)
	require.Nil(t, vppVerifyState(mixedExecs[2]).VerificationFailedAt)
	require.NotContains(t, byCommandUUID, mixedExecs[2])

	// running again reaps nothing: everything reapable is already failed, and what is left is
	// left for a reason
	reaped, err = ds.ReapStuckActivatedMDMInstalls(ctx, reapAfter, 10)
	require.NoError(t, err)
	require.Empty(t, reaped)

	// A sub-second age clears the callers' non-positive guards, so it must not then truncate to an
	// interval of zero and match everything. 999ms still truncates to 0 whole seconds, so it
	// discriminates, and the install below sits 1ms inside it. That leaves just under a second
	// before the assertion races the clock, which is as wide as a sub-second timeout allows.
	const subSecondTimeout = 999 * time.Millisecond
	hSubSecond := newMDMHost()
	subSecondExec, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, hSubSecond)
	advance(hSubSecond, "")
	deliver(hSubSecond, subSecondExec)
	setActivatedAgo := func(execID string, micros int) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`UPDATE upcoming_activities SET activated_at = NOW(6) - INTERVAL ? MICROSECOND WHERE execution_id = ?`,
				micros, execID)
			return err
		})
	}
	setActivatedAgo(subSecondExec, 1_000)

	reaped, err = ds.ReapStuckActivatedMDMInstalls(ctx, subSecondTimeout, 10)
	require.NoError(t, err)
	reapedUUIDs := make([]string, 0, len(reaped))
	for _, r := range reaped {
		reapedUUIDs = append(reapedUUIDs, r.CommandUUID)
	}
	require.NotContains(t, reapedUUIDs, subSecondExec,
		"an install younger than a sub-second timeout must survive it")
	require.Nil(t, vppVerifyState(subSecondExec).VerificationFailedAt)

	// the same timeout does reap it once it is genuinely older. Only this install is checked: the
	// hosts answered "just now" above are by now also older than a sub-second timeout, and none of
	// them is used again.
	setActivatedAgo(subSecondExec, 5_000_000)
	reaped, err = ds.ReapStuckActivatedMDMInstalls(ctx, subSecondTimeout, 10)
	require.NoError(t, err)
	reapedUUIDs = reapedUUIDs[:0]
	for _, r := range reaped {
		reapedUUIDs = append(reapedUUIDs, r.CommandUUID)
	}
	require.Contains(t, reapedUUIDs, subSecondExec)
	require.NotNil(t, vppVerifyState(subSecondExec).VerificationFailedAt)

	// maxHosts bounds hosts, not rows: two stuck hosts, one run each
	hLimitA, hLimitB := newMDMHost(), newMDMHost()
	limitExecs := make(map[uint]string, 2)
	for _, h := range []*fleet.Host{hLimitA, hLimitB} {
		execID, _ := test.CreateHostVPPAppInstallUpcomingActivity(t, ds, h)
		advance(h, "")
		deliver(h, execID)
		ageActivations(execID)
		limitExecs[h.ID] = execID
	}
	reaped, err = ds.ReapStuckActivatedMDMInstalls(ctx, reapAfter, 1)
	require.NoError(t, err)
	require.Len(t, reaped, 1)
	require.Equal(t, limitExecs[reaped[0].HostID], reaped[0].CommandUUID)

	reaped, err = ds.ReapStuckActivatedMDMInstalls(ctx, reapAfter, 1)
	require.NoError(t, err)
	require.Len(t, reaped, 1)
	require.Empty(t, queuedExecIDs(hLimitA))
	require.Empty(t, queuedExecIDs(hLimitB))
}

func testActivateScriptPackageInstallWithCorruptPayload(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host1", "192.168.1.1", "1", "1", time.Now())

	titleStmt := `INSERT INTO software_titles (name, source, extension_for) VALUES (?, ?, '')`
	res, err := ds.writer(ctx).ExecContext(ctx, titleStmt, "Test Script", "sh_packages")
	require.NoError(t, err)
	titleID, _ := res.LastInsertId()

	u := test.NewUser(t, ds, "user1", "user1@example.com", false)

	scriptContentStmt := `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`
	res, err = ds.writer(ctx).ExecContext(ctx, scriptContentStmt, "abc123", "#!/bin/bash\necho 'test'")
	require.NoError(t, err)
	scriptContentID, _ := res.LastInsertId()

	installerStmt := `
		INSERT INTO software_installers (
			team_id, global_or_team_id, title_id, storage_id, filename,
			extension, version, platform, install_script_content_id,
			pre_install_query, post_install_script_content_id, uninstall_script_content_id,
			self_service, user_id, user_name, user_email, package_ids,
			fleet_maintained_app_id, url, upgrade_code, patch_query
		)
		VALUES (NULL, 0, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)
	`
	res, err = ds.writer(ctx).ExecContext(ctx, installerStmt,
		titleID, "storage-123", "test-script.sh", "sh", "", "linux", scriptContentID,
		"", scriptContentID, 0, u.ID, u.Name, u.Email, "", "", "", "")
	require.NoError(t, err)
	installerID, _ := res.LastInsertId()

	execID := uuid.NewString()
	uaStmt := `INSERT INTO upcoming_activities (host_id, priority, activity_type, execution_id, payload) VALUES (?, 1, 'software_install', ?, JSON_OBJECT())`
	res, err = ds.writer(ctx).ExecContext(ctx, uaStmt, host.ID, execID)
	require.NoError(t, err)
	activityID, _ := res.LastInsertId()

	siuaStmt := `INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_installer_id, policy_id, software_title_id) VALUES (?, ?, NULL, NULL)`
	_, err = ds.writer(ctx).ExecContext(ctx, siuaStmt, activityID, installerID)
	require.NoError(t, err)

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := ds.activateNextUpcomingActivity(ctx, tx, host.ID, "")
		return err
	})
	require.NoError(t, err)

	var result struct {
		SoftwareTitleID   *uint  `db:"software_title_id"`
		SoftwareTitleName string `db:"software_title_name"`
		InstallerFilename string `db:"installer_filename"`
		Version           string `db:"version"`
	}

	err = ds.writer(ctx).GetContext(ctx, &result,
		"SELECT software_title_id, software_title_name, installer_filename, version FROM host_software_installs WHERE execution_id = ?",
		execID)
	require.NoError(t, err)

	require.NotNil(t, result.SoftwareTitleID)
	require.Equal(t, uint(titleID), *result.SoftwareTitleID) //nolint:gosec // dismiss G115
	require.Equal(t, "Test Script", result.SoftwareTitleName)
	require.Equal(t, "test-script.sh", result.InstallerFilename)
	require.Equal(t, "", result.Version)
}

func testActivateRegularPackageInstall(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host2", "192.168.1.2", "2", "2", time.Now())
	u := test.NewUser(t, ds, "user2", "user2@example.com", false)

	installer, err := fleet.NewTempFileReader(strings.NewReader("fake pkg"), t.TempDir)
	require.NoError(t, err)

	installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "install regular",
		InstallerFile:    installer,
		StorageID:        uuid.NewString(),
		Filename:         "regular.pkg",
		Title:            "Regular Package",
		Source:           "pkg_packages",
		Version:          "1.0.0",
		UserID:           u.ID,
		Extension:        "pkg",
		Platform:         "darwin",
		BundleIdentifier: "com.regular.pkg",
		UninstallScript:  "uninstall regular",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	execID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	var result struct {
		SoftwareTitleID   *uint  `db:"software_title_id"`
		SoftwareTitleName string `db:"software_title_name"`
		InstallerFilename string `db:"installer_filename"`
		Version           string `db:"version"`
	}

	err = ds.writer(ctx).GetContext(ctx, &result,
		"SELECT software_title_id, software_title_name, installer_filename, version FROM host_software_installs WHERE execution_id = ?",
		execID)
	require.NoError(t, err)

	require.NotNil(t, result.SoftwareTitleID)
	require.Equal(t, titleID, *result.SoftwareTitleID)
	require.Equal(t, "Regular Package", result.SoftwareTitleName)
	require.Equal(t, "regular.pkg", result.InstallerFilename)
	require.Equal(t, "1.0.0", result.Version)
}

func testActivateDeletedInstallerShowsPlaceholder(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := test.NewHost(t, ds, "host3", "192.168.1.3", "3", "3", time.Now())
	u := test.NewUser(t, ds, "user3", "user3@example.com", false)

	installer, err := fleet.NewTempFileReader(strings.NewReader("temp"), t.TempDir)
	require.NoError(t, err)

	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install temp",
		InstallerFile:   installer,
		StorageID:       uuid.NewString(),
		Filename:        "temp.pkg",
		Title:           "Temp Package",
		Source:          "pkg_packages",
		Version:         "1.0.0",
		UserID:          u.ID,
		Extension:       "pkg",
		Platform:        "darwin",
		UninstallScript: "uninstall temp",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	execID := uuid.NewString()
	uaStmt := `INSERT INTO upcoming_activities (host_id, priority, activity_type, execution_id, payload) VALUES (?, 1, 'software_install', ?, JSON_OBJECT())`
	res, err := ds.writer(ctx).ExecContext(ctx, uaStmt, host.ID, execID)
	require.NoError(t, err)
	activityID, _ := res.LastInsertId()

	siuaStmt := `INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_installer_id, policy_id, software_title_id) VALUES (?, ?, NULL, NULL)`
	_, err = ds.writer(ctx).ExecContext(ctx, siuaStmt, activityID, installerID)
	require.NoError(t, err)

	deleteStmt := `DELETE FROM software_installers WHERE id = ?`
	_, err = ds.writer(ctx).ExecContext(ctx, deleteStmt, installerID)
	require.NoError(t, err)

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := ds.activateNextUpcomingActivity(ctx, tx, host.ID, "")
		return err
	})
	require.NoError(t, err)

	var result struct {
		SoftwareTitleID   *uint  `db:"software_title_id"`
		SoftwareTitleName string `db:"software_title_name"`
		InstallerFilename string `db:"installer_filename"`
		Version           string `db:"version"`
	}

	err = ds.writer(ctx).GetContext(ctx, &result,
		"SELECT software_title_id, software_title_name, installer_filename, version FROM host_software_installs WHERE execution_id = ?",
		execID)
	require.NoError(t, err)

	require.Nil(t, result.SoftwareTitleID)
	require.Equal(t, "[deleted title]", result.SoftwareTitleName)
	require.Equal(t, "[deleted installer]", result.InstallerFilename)
	require.Equal(t, "unknown", result.Version)
}

func testActivateScriptPackageUninstallWithCorruptPayload(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	titleStmt := `INSERT INTO software_titles (name, source, extension_for) VALUES ('Test Uninstall Script', 'apps', '')`
	res, err := ds.writer(ctx).ExecContext(ctx, titleStmt)
	require.NoError(t, err)
	titleID, _ := res.LastInsertId()

	u := test.NewUser(t, ds, "uninstall-user", "uninstall@example.com", false)

	scriptStmt := `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('echo uninstalling')), 'echo uninstalling')`
	res, err = ds.writer(ctx).ExecContext(ctx, scriptStmt)
	require.NoError(t, err)
	scriptContentID, _ := res.LastInsertId()

	installerStmt := `
		INSERT INTO software_installers (
			team_id, global_or_team_id, title_id, storage_id, filename,
			extension, version, platform, install_script_content_id,
			pre_install_query, post_install_script_content_id, uninstall_script_content_id,
			self_service, user_id, user_name, user_email, package_ids,
			fleet_maintained_app_id, url, upgrade_code, patch_query
		)
		VALUES (NULL, 0, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)
	`
	res, err = ds.writer(ctx).ExecContext(ctx, installerStmt,
		titleID, "storage-id-uninstall", "test-uninstall.sh", "sh", "", "linux", scriptContentID,
		"", scriptContentID, 0, u.ID, u.Name, u.Email, "", "", "", "")
	require.NoError(t, err)
	installerID, _ := res.LastInsertId()

	host := test.NewHost(t, ds, "test-host", "", "test-key", "test-uuid", time.Now())

	execID := "uninstall-exec-123"
	uaStmt := `INSERT INTO upcoming_activities (host_id, activity_type, execution_id, user_id, payload, priority) VALUES (?, 'software_uninstall', ?, NULL, JSON_OBJECT(), 0)`
	res, err = ds.writer(ctx).ExecContext(ctx, uaStmt, host.ID, execID)
	require.NoError(t, err)
	activityID, _ := res.LastInsertId()

	siuaStmt := `INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_installer_id, software_title_id) VALUES (?, ?, NULL)`
	_, err = ds.writer(ctx).ExecContext(ctx, siuaStmt, activityID, installerID)
	require.NoError(t, err)

	err = ds.activateNextSoftwareUninstallActivity(ctx, ds.writer(ctx), host.ID, []string{execID})
	require.NoError(t, err)

	var result struct {
		SoftwareTitleID   *uint  `db:"software_title_id"`
		SoftwareTitleName string `db:"software_title_name"`
		Uninstall         bool   `db:"uninstall"`
	}
	err = sqlx.GetContext(ctx, ds.reader(ctx), &result,
		"SELECT software_title_id, software_title_name, uninstall FROM host_software_installs WHERE execution_id = ?",
		execID)
	require.NoError(t, err)

	require.True(t, result.Uninstall)
	require.NotNil(t, result.SoftwareTitleID)
	require.Equal(t, uint(titleID), *result.SoftwareTitleID) //nolint:gosec // dismiss G115
	require.Equal(t, "Test Uninstall Script", result.SoftwareTitleName)
}

func testListPolicyAutomationActivities(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	activitySvc := NewTestActivityService(t, ds)

	// adminFilter sees all hosts regardless of team.
	adminFilter := fleet.TeamFilter{
		User:            &fleet.User{GlobalRole: new("admin")},
		IncludeObserver: true,
	}

	// Create a policy to hang activities on.
	policy, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "test-policy", Query: "SELECT 1"})
	require.NoError(t, err)
	require.NotNil(t, policy)

	// Create a second policy; its activities must NOT appear in results for the first.
	otherPolicy, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "other-policy", Query: "SELECT 2"})
	require.NoError(t, err)
	require.NotNil(t, otherPolicy)

	// Create two hosts so we can test per-host rows and the host-name filter.
	h1 := test.NewHost(t, ds, "host-alpha", "1.1.1.1", "key1", "uuid1", time.Now())
	h2 := test.NewHost(t, ds, "host-beta", "2.2.2.2", "key2", "uuid2", time.Now())

	makeDetails := func(policyID uint) map[string]any {
		return map[string]any{"policy_id": policyID}
	}

	// Seed one activity of each type for policy 1 linked to h1,
	// plus one success activity linked to both hosts (tests multi-host expansion).
	errorTypes := []string{
		"failed_automation_webhook",
		"failed_automation_ticket",
		"failed_automation_calendar_event",
		"failed_automation_conditional_access",
	}
	successTypes := []string{
		"ran_automation_webhook",
		"ran_automation_ticket",
		"ran_automation_calendar_event",
		"ran_automation_conditional_access",
	}

	for _, typ := range errorTypes {
		require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
			name:    typ,
			details: makeDetails(policy.ID),
			hostIDs: []uint{h1.ID},
		}))
	}
	for _, typ := range successTypes {
		require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
			name:    typ,
			details: makeDetails(policy.ID),
			hostIDs: []uint{h1.ID},
		}))
	}

	// One activity linked to both hosts — produces two rows.
	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "ran_automation_webhook",
		details: makeDetails(policy.ID),
		hostIDs: []uint{h1.ID, h2.ID},
	}))

	// Activity for the other policy — must not appear in results for policy 1.
	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "failed_automation_webhook",
		details: makeDetails(otherPolicy.ID),
		hostIDs: []uint{h1.ID},
	}))

	listOpts := func(extra ...fleet.ListOptions) fleet.ListOptions {
		opts := fleet.ListOptions{OrderKey: "id", IncludeMetadata: true}
		if len(extra) > 0 {
			if extra[0].PerPage != 0 {
				opts.PerPage = extra[0].PerPage
			}
			if extra[0].Page != 0 {
				opts.Page = extra[0].Page
			}
			if extra[0].MatchQuery != "" {
				opts.MatchQuery = extra[0].MatchQuery
			}
		}
		return opts
	}

	t.Run("returns all types by default", func(t *testing.T) {
		// 8 single-host activities + 2 rows from the dual-host one = 10 rows.
		activities, meta, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Len(t, activities, 10)
		for _, a := range activities {
			require.NotZero(t, a.HostID)
			// Policy automation activities are always Fleet-initiated; actor fields
			// are not selected and must be absent (nil) so they're omitted from JSON.
			require.Nil(t, a.ActorID)
			require.Nil(t, a.ActorFullName)
			require.Nil(t, a.ActorEmail)
		}
	})

	t.Run("status=error returns only failed types", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "error")
		require.NoError(t, err)
		require.Len(t, activities, 4)
		for _, a := range activities {
			require.Contains(t, a.Type, "failed_")
		}
	})

	t.Run("status=success returns only positive types", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "success")
		require.NoError(t, err)
		// 4 single-host success activities + 2 rows from dual-host = 6
		require.Len(t, activities, 6)
		for _, a := range activities {
			require.NotContains(t, a.Type, "failed_")
			require.Contains(t, successTypes, a.Type)
		}
	})

	t.Run("profile resends appear, manual resends do not", func(t *testing.T) {
		// A policy-triggered resend carries details.policy_id and is linked to the host, so it
		// shows in the feed as a success. A manual resend records the same activity type with a
		// null policy_id and must not appear under any policy.
		resendPolicy, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "resend-policy", Query: "SELECT 3"})
		require.NoError(t, err)

		require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
			name: fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
			details: map[string]any{
				"host_id":      h1.ID,
				"profile_uuid": "a-profile-uuid",
				"profile_name": "a profile",
				"policy_id":    resendPolicy.ID,
				"policy_name":  "resend-policy",
			},
			hostIDs: []uint{h1.ID},
		}))
		require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
			name: fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
			details: map[string]any{
				"host_id":      h1.ID,
				"profile_uuid": "a-profile-uuid",
				"profile_name": "a profile",
				"policy_id":    nil,
				"policy_name":  nil,
			},
			hostIDs: []uint{h1.ID},
		}))

		activities, _, err := ds.ListPolicyAutomationActivities(ctx, resendPolicy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)
		require.Len(t, activities, 1, "the manual resend must not be attributed to a policy")
		require.Equal(t, fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(), activities[0].Type)
		require.Equal(t, h1.ID, activities[0].HostID)
		require.Equal(t, "success", activities[0].Status)

		// It is a success, so it is absent from the error-filtered feed.
		activities, _, err = ds.ListPolicyAutomationActivities(ctx, resendPolicy.ID, adminFilter, listOpts(), "error")
		require.NoError(t, err)
		require.Empty(t, activities)

		activities, _, err = ds.ListPolicyAutomationActivities(ctx, resendPolicy.ID, adminFilter, listOpts(), "success")
		require.NoError(t, err)
		require.Len(t, activities, 1)
	})

	t.Run("pagination", func(t *testing.T) {
		activities, meta, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(fleet.ListOptions{PerPage: 3}), "")
		require.NoError(t, err)
		require.NotNil(t, meta)
		require.Len(t, activities, 3)
		require.True(t, meta.HasNextResults)
		require.False(t, meta.HasPreviousResults)

		page2, meta2, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(fleet.ListOptions{PerPage: 3, Page: 1}), "")
		require.NoError(t, err)
		require.NotNil(t, meta2)
		require.Len(t, page2, 3)
		require.True(t, meta2.HasPreviousResults)
	})

	t.Run("host name query filters rows", func(t *testing.T) {
		// "host-alpha" matches h1 only: 4 error + 4 success + 1 dual-host row = 9.
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(fleet.ListOptions{MatchQuery: "host-alpha"}), "")
		require.NoError(t, err)
		require.Len(t, activities, 9)
		for _, a := range activities {
			require.Equal(t, h1.ID, a.HostID)
			require.Equal(t, "host-alpha", a.HostDisplayName)
		}
		// "host-b" matches h2 only, which appears in just the dual-host activity.
		activities, _, err = ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(fleet.ListOptions{MatchQuery: "host-b"}), "")
		require.NoError(t, err)
		require.Len(t, activities, 1)
		for _, a := range activities {
			require.Equal(t, h2.ID, a.HostID)
		}
	})

	t.Run("other policy activities excluded", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, otherPolicy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)
		require.Len(t, activities, 1)
		require.Equal(t, h1.ID, activities[0].HostID)
	})

	t.Run("invalid order_key returns error", func(t *testing.T) {
		_, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, fleet.ListOptions{OrderKey: "invalid_column"}, "")
		require.Error(t, err)
	})

	t.Run("include_metadata false returns nil meta", func(t *testing.T) {
		opts := fleet.ListOptions{OrderKey: "id", IncludeMetadata: false}
		_, meta, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, opts, "")
		require.NoError(t, err)
		require.Nil(t, meta)
	})

	t.Run("query with wildcard characters matches literally", func(t *testing.T) {
		// host-alpha has no '_' in its name; a query of "host_alpha" must NOT match
		// it (the underscore is a literal character, not a SQL wildcard).
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(fleet.ListOptions{MatchQuery: "host_alpha"}), "")
		require.NoError(t, err)
		require.Empty(t, activities)
		// An empty result set must be a non-nil slice so it marshals as [] (not null).
		require.NotNil(t, activities)
	})

	t.Run("team filter scopes hosts", func(t *testing.T) {
		// Use a dedicated policy so activities seeded here don't affect count
		// assertions in other subtests.
		teamScopePolicy, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "team-scope-policy", Query: "SELECT 3"})
		require.NoError(t, err)
		require.NotNil(t, teamScopePolicy)

		// Create two teams and assign one host to each.
		teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-A"})
		require.NoError(t, err)
		teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-B"})
		require.NoError(t, err)

		hA := test.NewHost(t, ds, "host-team-a", "10.0.0.1", "keyA", "uuidA", time.Now())
		hB := test.NewHost(t, ds, "host-team-b", "10.0.0.2", "keyB", "uuidB", time.Now())
		// Assign hosts to teams directly to avoid policy-membership side-effects.
		_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET team_id = ? WHERE id = ?`, teamA.ID, hA.ID)
		require.NoError(t, err)
		_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET team_id = ? WHERE id = ?`, teamB.ID, hB.ID)
		require.NoError(t, err)

		// Seed activities for both hosts on the dedicated policy.
		for _, typ := range errorTypes {
			require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
				name:    typ,
				details: makeDetails(teamScopePolicy.ID),
				hostIDs: []uint{hA.ID, hB.ID},
			}))
		}

		// A team-A observer filter sees only hA.
		filterA := fleet.TeamFilter{
			User: &fleet.User{
				Teams: []fleet.UserTeam{{Team: fleet.Team{ID: teamA.ID}, Role: fleet.RoleObserver}},
			},
			IncludeObserver: true,
		}
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, teamScopePolicy.ID, filterA, listOpts(), "")
		require.NoError(t, err)
		require.NotEmpty(t, activities)
		for _, a := range activities {
			require.Equal(t, hA.ID, a.HostID, "team-A filter must not return host from team-B")
		}
	})

	// ── Script-run, software-install and VPP-install branches ─────────────────
	// Disable FK checks so we can insert result rows without satisfying every
	// foreign key in the test setup.
	_, err = ds.writer(ctx).ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0")
	require.NoError(t, err)
	defer func() {
		_, _ = ds.writer(ctx).ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=1")
	}()

	// ── ran_script ────────────────────────────────────────────────────────────
	scriptSuccessExecID := "script-success-exec-1"
	scriptFailureExecID := "script-failure-exec-1"
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_script_results (host_id, execution_id, output, exit_code, policy_id)
         VALUES (?, ?, 'script ok output', 0, ?), (?, ?, 'script fail output', 1, ?)`,
		h1.ID, scriptSuccessExecID, policy.ID,
		h1.ID, scriptFailureExecID, policy.ID)
	require.NoError(t, err)

	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "ran_script",
		details: map[string]any{"script_execution_id": scriptSuccessExecID, "script_name": "my-script.sh"},
		hostIDs: []uint{h1.ID},
	}))
	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "ran_script",
		details: map[string]any{"script_execution_id": scriptFailureExecID, "script_name": "my-script.sh"},
		hostIDs: []uint{h1.ID},
	}))

	// ── installed_software ────────────────────────────────────────────────────
	swSuccessExecID := "sw-success-exec-1"
	swFailureExecID := "sw-failure-exec-1"
	// insert_script_exit_code=0 → execution_status='installed'; exit_code=1 → 'failed_install'
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_software_installs
            (host_id, execution_id, software_installer_id, install_script_exit_code,
             install_script_output, pre_install_query_output, post_install_script_output, policy_id)
         VALUES
            (?, ?, 1, 0, 'install ok', 'pre ok', 'post ok', ?),
            (?, ?, 1, 1, 'install fail', 'pre fail', 'post fail', ?)`,
		h1.ID, swSuccessExecID, policy.ID,
		h1.ID, swFailureExecID, policy.ID)
	require.NoError(t, err)

	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "installed_software",
		details: map[string]any{"install_uuid": swSuccessExecID, "software_title": "My Software", "status": "installed"},
		hostIDs: []uint{h1.ID},
	}))
	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "installed_software",
		details: map[string]any{"install_uuid": swFailureExecID, "software_title": "My Software", "status": "failed_install"},
		hostIDs: []uint{h1.ID},
	}))

	// A historically-successful install whose host_software_installs row was later
	// marked removed (e.g. after the installer package was edited or the software
	// re-installed). The generated status column is NULL for such rows, so the
	// outcome must come from the recorded details.status, not the live column.
	swRemovedExecID := "sw-removed-exec-1"
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_software_installs
            (host_id, execution_id, software_installer_id, install_script_exit_code,
             install_script_output, pre_install_query_output, post_install_script_output, policy_id, removed)
         VALUES
            (?, ?, 1, 0, 'install ok', 'pre ok', 'post ok', ?, 1)`,
		h1.ID, swRemovedExecID, policy.ID)
	require.NoError(t, err)
	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "installed_software",
		details: map[string]any{"install_uuid": swRemovedExecID, "software_title": "My Software", "status": "installed"},
		hostIDs: []uint{h1.ID},
	}))

	// ── installed_app_store_app (VPP) ─────────────────────────────────────────
	// Outcome comes from the recorded details.status (a terminal snapshot), like
	// installed_software — not the live hvsi.verification_* columns. To prove
	// that, the live verification columns are set to the OPPOSITE of each row's
	// details.status: the "success" row is marked verification_failed_at and the
	// "failure" row verification_at. If the query read the live columns, the
	// outcomes would flip and the assertions below would fail.
	vppSuccessCmdUUID := "vpp-success-cmd-1"
	vppFailureCmdUUID := "vpp-failure-cmd-1"
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, policy_id, platform, verification_failed_at)
         VALUES (?, 'A001', ?, ?, 'darwin', NOW())`,
		h1.ID, vppSuccessCmdUUID, policy.ID)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, policy_id, platform, verification_at)
         VALUES (?, 'A002', ?, ?, 'darwin', NOW())`,
		h1.ID, vppFailureCmdUUID, policy.ID)
	require.NoError(t, err)

	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "installed_app_store_app",
		details: map[string]any{"command_uuid": vppSuccessCmdUUID, "software_title": "My VPP App", "status": "installed"},
		hostIDs: []uint{h1.ID},
	}))
	require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
		name:    "installed_app_store_app",
		details: map[string]any{"command_uuid": vppFailureCmdUUID, "software_title": "My VPP App", "status": "failed_install"},
		hostIDs: []uint{h1.ID},
	}))

	t.Run("script_software_vpp appear in all", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)
		types := make(map[string]int)
		for _, a := range activities {
			types[a.Type]++
		}
		require.Positive(t, types["ran_script"])
		require.Positive(t, types["installed_software"])
		require.Positive(t, types["installed_app_store_app"])
	})

	t.Run("status=error includes script_software_vpp failures", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "error")
		require.NoError(t, err)
		types := make(map[string]int)
		for _, a := range activities {
			types[a.Type]++
		}
		// Named automation failures still present.
		require.Equal(t, 4, types["failed_automation_webhook"]+
			types["failed_automation_ticket"]+
			types["failed_automation_calendar_event"]+
			types["failed_automation_conditional_access"])
		// Script/software/VPP failures present.
		require.Positive(t, types["ran_script"])
		require.Positive(t, types["installed_software"])
		require.Positive(t, types["installed_app_store_app"])
	})

	t.Run("status=success includes script_software_vpp successes", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "success")
		require.NoError(t, err)
		types := make(map[string]int)
		for _, a := range activities {
			types[a.Type]++
		}
		// Named automation successes still present.
		require.Positive(t, types["ran_automation_webhook"]+
			types["ran_automation_ticket"]+
			types["ran_automation_calendar_event"]+
			types["ran_automation_conditional_access"])
		// Script/software/VPP successes present.
		require.Positive(t, types["ran_script"])
		require.Positive(t, types["installed_software"])
		require.Positive(t, types["installed_app_store_app"])
	})

	t.Run("task activities are independent of policy_membership", func(t *testing.T) {
		// Modifying a policy's query or targets wipes/prunes policy_membership.
		// Automation history must survive that, so deleting all membership rows
		// for the policy must not drop the script/software/VPP activities.
		_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM policy_membership WHERE policy_id = ?`, policy.ID)
		require.NoError(t, err)

		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)
		types := make(map[string]int)
		for _, a := range activities {
			types[a.Type]++
		}
		require.Positive(t, types["ran_script"])
		require.Positive(t, types["installed_software"])
		require.Positive(t, types["installed_app_store_app"])
	})

	t.Run("removed install row is categorized by recorded status", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)

		// hasRemovedInstall reports whether the removed install activity appears in
		// the given result set (matched by its install_uuid in the details blob).
		hasRemovedInstall := func(as []*fleet.PolicyAutomationActivity) bool {
			for _, a := range as {
				require.NotNil(t, a.Details)
				var m map[string]any
				require.NoError(t, json.Unmarshal(*a.Details, &m))
				if uuid, _ := m["install_uuid"].(string); uuid == swRemovedExecID {
					// The live host_software_installs.status is NULL (removed=1), but
					// the recorded details.status is "installed", so the historical
					// outcome must be reported as success.
					require.Equal(t, "success", a.Status)
					return true
				}
			}
			return false
		}
		require.True(t, hasRemovedInstall(activities), "expected the removed install activity to be returned")

		// The status filter must agree with the reported status: the removed row
		// is a success, so it appears under status=success and not status=error.
		// This guards errorCond/successCond, which the unfiltered query above does
		// not exercise.
		success, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "success")
		require.NoError(t, err)
		require.True(t, hasRemovedInstall(success), "removed install should appear under status=success")

		errored, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "error")
		require.NoError(t, err)
		require.False(t, hasRemovedInstall(errored), "removed install must not appear under status=error")
	})

	t.Run("status and output are populated per activity", func(t *testing.T) {
		activities, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)

		var sawScriptSuccess, sawScriptFailure bool
		var sawSwSuccess, sawSwFailure bool
		var sawVPPSuccess, sawVPPFailure bool
		var sawNamedError, sawNamedSuccess bool

		// detailsValue extracts a string field from an activity's details blob.
		detailsValue := func(a *fleet.PolicyAutomationActivity, key string) string {
			require.NotNil(t, a.Details, "type %s missing details", a.Type)
			var m map[string]any
			require.NoError(t, json.Unmarshal(*a.Details, &m))
			s, _ := m[key].(string)
			return s
		}

		for _, a := range activities {
			// Every activity carries an explicit error/success status.
			require.Contains(t, []string{"error", "success"}, a.Status, "type %s", a.Type)

			switch a.Type {
			case "ran_script":
				// Scripts always carry output; the script name comes through in
				// the details blob. Pre/post-install output is install-only.
				require.NotNil(t, a.Output)
				require.Nil(t, a.PreInstallOutput)
				require.Nil(t, a.PostInstallOutput)
				require.Equal(t, "my-script.sh", detailsValue(a, "script_name"))
				if a.Status == "success" {
					sawScriptSuccess = true
					require.Equal(t, "script ok output", *a.Output)
				} else {
					sawScriptFailure = true
					require.Equal(t, "script fail output", *a.Output)
				}
			case "installed_software":
				// Software installs carry the install-script output plus the
				// pre-install query and post-install script output; the software
				// title comes through in the details blob.
				require.NotNil(t, a.Output)
				require.NotNil(t, a.PreInstallOutput)
				require.NotNil(t, a.PostInstallOutput)
				require.Equal(t, "My Software", detailsValue(a, "software_title"))
				if a.Status == "success" {
					sawSwSuccess = true
					require.Equal(t, "install ok", *a.Output)
					require.Equal(t, "pre ok", *a.PreInstallOutput)
					require.Equal(t, "post ok", *a.PostInstallOutput)
				} else {
					sawSwFailure = true
					require.Equal(t, "install fail", *a.Output)
					require.Equal(t, "pre fail", *a.PreInstallOutput)
					require.Equal(t, "post fail", *a.PostInstallOutput)
				}
			case "installed_app_store_app":
				// VPP apps are installed via MDM command, so there is no output;
				// the software title comes through in the details blob.
				require.Nil(t, a.Output)
				require.Nil(t, a.PreInstallOutput)
				require.Nil(t, a.PostInstallOutput)
				require.Equal(t, "My VPP App", detailsValue(a, "software_title"))
				if a.Status == "success" {
					sawVPPSuccess = true
				} else {
					sawVPPFailure = true
				}
			default:
				// Named automation activities encode outcome in the type and have
				// no output.
				require.Nil(t, a.Output)
				require.Nil(t, a.PreInstallOutput)
				require.Nil(t, a.PostInstallOutput)
				if strings.HasPrefix(a.Type, "failed_") {
					sawNamedError = true
					require.Equal(t, "error", a.Status)
				} else {
					sawNamedSuccess = true
					require.Equal(t, "success", a.Status)
				}
			}
		}

		require.True(t, sawScriptSuccess, "expected a successful ran_script")
		require.True(t, sawScriptFailure, "expected a failed ran_script")
		require.True(t, sawSwSuccess, "expected a successful installed_software")
		require.True(t, sawSwFailure, "expected a failed installed_software")
		require.True(t, sawVPPSuccess, "expected a successful installed_app_store_app")
		require.True(t, sawVPPFailure, "expected a failed installed_app_store_app")
		require.True(t, sawNamedError, "expected a failed named automation")
		require.True(t, sawNamedSuccess, "expected a successful named automation")
	})

	t.Run("installed_software with an unrecorded status is treated as a success", func(t *testing.T) {
		// Older installed_software activities can lack a recorded details.status
		// (the field was added after the activity type, and back then the activity
		// was only emitted on a successful install). 'failed_install' is the sole
		// failure value, so a missing status is a success — and the reported status
		// must agree with the filters: it appears under "All" and status=success,
		// never under status=error.
		execID := "sw-no-status-exec-1"
		_, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_software_installs
                (host_id, execution_id, software_installer_id, install_script_exit_code,
                 install_script_output, policy_id)
             VALUES (?, ?, 1, 0, 'historical output', ?)`,
			h1.ID, execID, policy.ID)
		require.NoError(t, err)

		// details intentionally omits "status".
		require.NoError(t, activitySvc.NewActivity(ctx, nil, dummyActivity{
			name:    "installed_software",
			details: map[string]any{"install_uuid": execID, "software_title": "My Software"},
			hostIDs: []uint{h1.ID},
		}))

		find := func(as []*fleet.PolicyAutomationActivity) *fleet.PolicyAutomationActivity {
			for _, a := range as {
				if a.Type != "installed_software" || a.Details == nil {
					continue
				}
				var m map[string]any
				require.NoError(t, json.Unmarshal(*a.Details, &m))
				if m["install_uuid"] == execID {
					return a
				}
			}
			return nil
		}

		all, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "")
		require.NoError(t, err)
		got := find(all)
		require.NotNil(t, got, "unrecorded-status install must appear under All")
		require.Equal(t, "success", got.Status, "a non-failed_install status is reported as a success")

		success, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "success")
		require.NoError(t, err)
		require.NotNil(t, find(success), "a success shown under All must also appear under status=success")

		errored, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter, listOpts(), "error")
		require.NoError(t, err)
		require.Nil(t, find(errored), "a success must not appear under status=error")
	})

	t.Run("status filters partition the feed for every activity type", func(t *testing.T) {
		// A row is uniquely identified by (activity id, host id) — one activity
		// linked to N hosts expands to N rows.
		key := func(a *fleet.PolicyAutomationActivity) string {
			return fmt.Sprintf("%d-%d", a.ID, a.HostID)
		}
		fetch := func(status string) map[string]*fleet.PolicyAutomationActivity {
			acts, _, err := ds.ListPolicyAutomationActivities(ctx, policy.ID, adminFilter,
				listOpts(fleet.ListOptions{PerPage: 1000}), status)
			require.NoError(t, err)
			m := make(map[string]*fleet.PolicyAutomationActivity, len(acts))
			for _, a := range acts {
				m[key(a)] = a
			}
			return m
		}

		all := fetch("")
		errored := fetch("error")
		success := fetch("success")

		// error and success are disjoint and together reconstruct the full feed.
		for k := range errored {
			_, inSuccess := success[k]
			require.False(t, inSuccess, "row %s appears under both status=error and status=success", k)
		}
		require.Equal(t, len(all), len(errored)+len(success),
			"status=error and status=success must partition the unfiltered feed")

		// Every row shown under All lands in exactly the filter matching its
		// reported status — no type is dropped by either filter.
		for k, a := range all {
			_, inErr := errored[k]
			_, inSucc := success[k]
			require.True(t, inErr || inSucc,
				"row %s (type %s, status %q) shown under All is missing from both filters",
				k, a.Type, a.Status)
			if a.Status == "error" {
				require.True(t, inErr, "row %s (type %s) reports error but is absent from status=error", k, a.Type)
			} else {
				require.True(t, inSucc, "row %s (type %s) reports success but is absent from status=success", k, a.Type)
			}
		}

		// Each task type is represented by both a success and a failure so the
		// partition above is exercised for every branch, not just the named ones.
		for _, typ := range []string{"ran_script", "installed_software", "installed_app_store_app"} {
			var sawErr, sawSucc bool
			for _, a := range all {
				if a.Type != typ {
					continue
				}
				if a.Status == "error" {
					sawErr = true
				} else {
					sawSucc = true
				}
			}
			require.True(t, sawErr, "expected at least one failed %s", typ)
			require.True(t, sawSucc, "expected at least one successful %s", typ)
		}
		// Named automations: a failed_* type is an error, a ran_automation_* is a success.
		var sawNamedErr, sawNamedSucc bool
		for _, a := range all {
			switch {
			case strings.HasPrefix(a.Type, "failed_"):
				sawNamedErr = true
				require.Equal(t, "error", a.Status, "type %s", a.Type)
			case strings.HasPrefix(a.Type, "ran_automation_"):
				sawNamedSucc = true
				require.Equal(t, "success", a.Status, "type %s", a.Type)
			}
		}
		require.True(t, sawNamedErr, "expected a failed named automation")
		require.True(t, sawNamedSucc, "expected a successful named automation")
	})
}
