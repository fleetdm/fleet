package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallers(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SoftwareInstallRequests", testSoftwareInstallRequests},
		{"ListPendingSoftwareInstalls", testListPendingSoftwareInstalls},
		{"GetSoftwareInstallResults", testGetSoftwareInstallResult},
		{"CleanupUnusedSoftwareInstallers", testCleanupUnusedSoftwareInstallers},
		{"BatchSetSoftwareInstallers", testBatchSetSoftwareInstallers},
		{"BatchSetSoftwareInstallersMultipleCustomPackages", testBatchSetSoftwareInstallersMultipleCustomPackages},
		{"BatchSetSoftwareInstallersWithUpgradeCodes", testBatchSetSoftwareInstallersWithUpgradeCodes},
		{"GetSoftwareInstallersPendingDeletion", testGetSoftwareInstallersPendingDeletion},
		{"GetSoftwareInstallerMetadataByTeamAndTitleID", testGetSoftwareInstallerMetadataByTeamAndTitleID},
		{"GetSoftwarePackagesByTeamAndTitleID", testGetSoftwarePackagesByTeamAndTitleID},
		{"HasSelfServiceSoftwareInstallers", testHasSelfServiceSoftwareInstallers},
		{"DeleteSoftwareInstallers", testDeleteSoftwareInstallers},
		{"DeleteSoftwareInstallerRepointsPolicies", testDeleteSoftwareInstallerRepointsPolicies},
		{"testDeletePendingSoftwareInstallsForPolicy", testDeletePendingSoftwareInstallsForPolicy},
		{"GetHostLastInstallData", testGetHostLastInstallData},
		{"GetOrGenerateSoftwareInstallerTitleID", testGetOrGenerateSoftwareInstallerTitleID},
		{"BatchSetSoftwareInstallersScopedViaLabels", testBatchSetSoftwareInstallersScopedViaLabels},
		{"MatchOrCreateSoftwareInstallerWithAutomaticPolicies", testMatchOrCreateSoftwareInstallerWithAutomaticPolicies},
		{"GetDetailsForUninstallFromExecutionID", testGetDetailsForUninstallFromExecutionID},
		{"GetTeamsWithInstallerByHash", testGetTeamsWithInstallerByHash},
		{"MatchOrCreateSoftwareInstallerDuplicateHash", testMatchOrCreateSoftwareInstallerDuplicateHash},
		{"BatchSetSoftwareInstallersSetupExperienceSideEffects", testBatchSetSoftwareInstallersSetupExperienceSideEffects},
		{"EditDeleteSoftwareInstallersActivateNextActivity", testEditDeleteSoftwareInstallersActivateNextActivity},
		{"BatchSetSoftwareInstallersActivateNextActivity", testBatchSetSoftwareInstallersActivateNextActivity},
		{"SoftwareInstallerReplicaLag", testSoftwareInstallerReplicaLag},
		{"SoftwareTitleDisplayName", testSoftwareTitleDisplayName},
		{"AddSoftwareTitleToMatchingSoftware", testAddSoftwareTitleToMatchingSoftware},
		{"FleetMaintainedAppInstallerUpdates", testFleetMaintainedAppInstallerUpdates},
		{"ListFleetMaintainedAppActiveInstallers", testListFleetMaintainedAppActiveInstallers},
		{"InsertFleetMaintainedAppVersion", testInsertFleetMaintainedAppVersion},
		{"InsertFleetMaintainedAppVersionProtectsLiveActive", testInsertFleetMaintainedAppVersionProtectsLiveActive},
		{"InsertFleetMaintainedAppVersionClonesLiveActive", testInsertFleetMaintainedAppVersionClonesLiveActive},
		{"GetSoftwareInstallerMetadataByStorageID", testGetSoftwareInstallerMetadataByStorageID},
		{"SoftwareTitlePins", testSoftwareTitlePins},
		{"SetFleetMaintainedAppActiveInstallerPin", testSetFleetMaintainedAppActiveInstallerPin},
		{"RepointCustomPackagePolicyToNewInstaller", testRepointPolicyToNewInstaller},
		{"CustomToFMAInstallerReplacement", testCustomToFMAInstallerReplacement},
		{"GetInstallerByTeamAndURL", testGetInstallerByTeamAndURL},
		{"BatchSetFMACancelsPendingOnActiveRow", testBatchSetFMACancelsPendingOnActiveRow},
		{"SoftwareInstallerTitleIDValidation", testSoftwareInstallerTitleIDValidation},
		{"MatchOrCreateSoftwareInstallerDuplicateConflicts", testMatchOrCreateSoftwareInstallerDuplicateConflicts},
		{"SetHostSoftwareInstallResultResolvesOrphanedActivity", testSetHostSoftwareInstallResultResolvesOrphanedActivity},
		{"GetSoftwareTitlesForInstallAll", testGetSoftwareTitlesForInstallAll},
		{"SummaryUpcomingPerHostNoDropout", testSummaryUpcomingPerHostNoDropout},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListPendingSoftwareInstalls(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "host3key", "host3uuid", time.Now())
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	err := ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{
		{
			Name:  "RUBBER",
			Value: "DUCKY",
		},
		{
			Name:  "BIG",
			Value: "BIRD",
		},
		{
			Name:  "COOKIE",
			Value: "MONSTER",
		},
	})
	require.NoError(t, err)

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello $FLEET_SECRET_RUBBER",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world $FLEET_SECRET_BIG",
		UninstallScript:   "goodbye $FLEET_SECRET_COOKIE",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "world",
		PreInstallQuery:   "SELECT 2",
		PostInstallScript: "hello",
		InstallerFile:     tfr2,
		StorageID:         "storage2",
		Filename:          "file2",
		Title:             "file2",
		Version:           "2.0",
		Source:            "apps",
		UserID:            user1.ID,
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr3, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID3, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     tfr3,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "file3",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// ensure that nothing gets automatically activated, we want to control
	// specific activation for this test
	ds.testActivateSpecificNextActivities = []string{"-"}

	hostInstall1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	time.Sleep(time.Millisecond)
	hostInstall2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID2, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	time.Sleep(time.Millisecond)
	hostInstall3, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	time.Sleep(time.Millisecond)
	hostInstall4, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID2, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	pendingHost1, err := ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(pendingHost1))
	require.Equal(t, hostInstall1, pendingHost1[0])
	require.Equal(t, hostInstall2, pendingHost1[1])

	pendingHost2, err := ds.ListPendingSoftwareInstalls(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(pendingHost2))
	require.Equal(t, hostInstall3, pendingHost2[0])
	require.Equal(t, hostInstall4, pendingHost2[1])

	// activate and set a result for hostInstall4 (installerID2)
	ds.testActivateSpecificNextActivities = []string{hostInstall4}
	_, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host2.ID, "")
	require.NoError(t, err)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           hostInstall4,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	// create a new pending install request on host2 for installerID2
	hostInstall5, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID2, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	ds.testActivateSpecificNextActivities = []string{hostInstall5}
	_, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host2.ID, "")
	require.NoError(t, err)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host2.ID,
		InstallUUID:               hostInstall5,
		PreInstallConditionOutput: ptr.String(""), // pre-install query did not return results, so install failed
	}, nil)
	require.NoError(t, err)

	installDetailsList1, err := ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, 2, len(installDetailsList1))

	installDetailsList2, err := ds.ListPendingSoftwareInstalls(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(installDetailsList2))

	require.Contains(t, installDetailsList1, hostInstall1)
	require.Contains(t, installDetailsList1, hostInstall2)
	require.Contains(t, installDetailsList2, hostInstall3)

	exec1, err := ds.GetSoftwareInstallDetails(ctx, hostInstall1)
	require.NoError(t, err)

	require.Equal(t, host1.ID, exec1.HostID)
	require.Equal(t, hostInstall1, exec1.ExecutionID)
	require.Equal(t, "hello DUCKY", exec1.InstallScript)
	require.Equal(t, "world BIRD", exec1.PostInstallScript)
	require.Equal(t, installerID1, exec1.InstallerID)
	require.Equal(t, "SELECT 1", exec1.PreInstallCondition)
	require.False(t, exec1.SelfService)
	assert.Equal(t, "goodbye MONSTER", exec1.UninstallScript)
	// Check that regular install has MaxRetries = 0
	require.EqualValues(t, 0, exec1.MaxRetries, "Regular install should have MaxRetries = 0")

	// add a self-service request for installerID3 on host1
	hostInstall6, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID3, fleet.HostSoftwareInstallOptions{SelfService: true})
	require.NoError(t, err)

	ds.testActivateSpecificNextActivities = []string{hostInstall6}
	_, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host1.ID, "")
	require.NoError(t, err)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host1.ID,
		InstallUUID:               hostInstall6,
		PreInstallConditionOutput: ptr.String("output"),
	}, nil)
	require.NoError(t, err)

	exec2, err := ds.GetSoftwareInstallDetails(ctx, hostInstall6)
	require.NoError(t, err)

	require.Equal(t, host1.ID, exec2.HostID)
	require.Equal(t, hostInstall6, exec2.ExecutionID)
	require.Equal(t, "banana", exec2.InstallScript)
	require.Equal(t, "apple", exec2.PostInstallScript)
	require.Equal(t, installerID3, exec2.InstallerID)
	require.Equal(t, "SELECT 3", exec2.PreInstallCondition)
	require.True(t, exec2.SelfService)

	// Create install request, don't fulfil it, delete and restore host.
	// Should not appear in list of pending installs for that host.
	_, err = ds.InsertSoftwareInstallRequest(ctx, host3.ID, installerID1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// Set LastEnrolledAt before deleting the host (simulating a DEP enrolled host)
	host3.LastEnrolledAt = time.Now()

	err = ds.DeleteHost(ctx, host3.ID)
	require.NoError(t, err)

	err = ds.RestoreMDMApplePendingDEPHost(ctx, host3)
	require.NoError(t, err)

	hostInstalls4, err := ds.ListPendingSoftwareInstalls(ctx, host3.ID)
	require.NoError(t, err)
	require.Empty(t, hostInstalls4)

	// Test MaxRetries for setup experience install
	// Create a software install request that's part of setup experience
	setupExperienceInstallID, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// Insert a setup experience status result to simulate this install is part of setup experience
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO setup_experience_status_results
		(host_uuid, name, status, software_installer_id, host_software_installs_execution_id)
		VALUES (?, ?, ?, ?, ?)`,
		host1.UUID, "test_software", fleet.SetupExperienceStatusPending, installerID1, setupExperienceInstallID)
	require.NoError(t, err)

	// Get the install details and check MaxRetries = setupExperienceSoftwareInstallsRetries
	setupExperienceInstallDetails, err := ds.GetSoftwareInstallDetails(ctx, setupExperienceInstallID)
	require.NoError(t, err)
	require.Equal(t, host1.ID, setupExperienceInstallDetails.HostID)
	require.Equal(t, setupExperienceInstallID, setupExperienceInstallDetails.ExecutionID)
	require.Equal(t, setupExperienceSoftwareInstallsRetries, setupExperienceInstallDetails.MaxRetries, "Setup experience install should have MaxRetries = %d", setupExperienceSoftwareInstallsRetries)
}

// testSetHostSoftwareInstallResultResolvesOrphanedActivity covers #44084: an
// install whose software_installer_id was nulled by FK SET NULL must still be
// resolvable by SetHostSoftwareInstallResult so the install loop stops.
func testSetHostSoftwareInstallResultResolvesOrphanedActivity(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host := test.NewHost(t, ds, "host-orphan", "1", "host-orphan-key", "host-orphan-uuid", time.Now())
	user := test.NewUser(t, ds, "Alice", "alice-orphan@example.com", true)

	tfr, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "echo install",
		InstallerFile:   tfr,
		StorageID:       "storage-orphan",
		Filename:        "orphan.pkg",
		Title:           "orphan",
		Version:         "1.0",
		Source:          "apps",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	executionID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	_, err = ds.GetSoftwareInstallDetails(ctx, executionID)
	require.NoError(t, err)

	// Simulate the FK SET NULL outcome directly so the test isn't coupled to
	// DeleteSoftwareInstaller's own cleanup side-effects.
	_, err = ds.writer(ctx).ExecContext(ctx, `
		UPDATE host_software_installs
		SET software_installer_id = NULL
		WHERE execution_id = ?`, executionID)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `
		UPDATE software_install_upcoming_activities siua
		JOIN upcoming_activities ua ON ua.id = siua.upcoming_activity_id
		SET siua.software_installer_id = NULL
		WHERE ua.execution_id = ?`, executionID)
	require.NoError(t, err)

	_, err = ds.GetSoftwareInstallDetails(ctx, executionID)
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           executionID,
		InstallScriptExitCode: ptr.Int(fleet.ExitCodeInstallerNotFound),
		InstallScriptOutput:   ptr.String("Installer no longer exists on the server. Abandoning install after retry window."),
	}, nil)
	require.NoError(t, err)

	var exitCode sql.NullInt64
	err = sqlx.GetContext(ctx, ds.reader(ctx), &exitCode, `
		SELECT install_script_exit_code
		FROM host_software_installs
		WHERE execution_id = ?`, executionID)
	require.NoError(t, err)
	require.True(t, exitCode.Valid)
	require.EqualValues(t, fleet.ExitCodeInstallerNotFound, exitCode.Int64)

	var remaining int
	err = sqlx.GetContext(ctx, ds.reader(ctx), &remaining, `
		SELECT COUNT(*)
		FROM upcoming_activities
		WHERE host_id = ? AND execution_id = ?`, host.ID, executionID)
	require.NoError(t, err)
	require.Zero(t, remaining, "stale upcoming_activities row should be deleted so orbit's loop stops")
}

func testSoftwareInstallRequests(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	createBuiltinLabels(t, ds)
	labelsByName, err := ds.LabelIDsByName(ctx, []string{fleet.BuiltinLabelNameAllHosts}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, labelsByName, 1)

	cases := map[string]*uint{
		"no team": nil,
		"team":    &team.ID,
	}

	for tc, teamID := range cases {
		t.Run(tc, func(t *testing.T) {
			// non-existent installer
			si, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, 1, false)
			var nfe fleet.NotFoundError
			require.ErrorAs(t, err, &nfe)
			require.Nil(t, si)

			installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:           "foo",
				Source:          "bar",
				InstallScript:   "echo",
				TeamID:          teamID,
				Filename:        "foo.pkg",
				UserID:          user1.ID,
				ValidatedLabels: &fleet.LabelIdentsWithScope{},
			})
			require.NoError(t, err)
			installerMeta, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
			require.NoError(t, err)

			require.NotNil(t, installerMeta.TitleID)
			require.Equal(t, titleID, *installerMeta.TitleID)

			si, err = ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, *installerMeta.TitleID, false)
			require.NoError(t, err)
			require.NotNil(t, si)
			require.Equal(t, "foo.pkg", si.Name)

			inHouseID, inHouseTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:           "inhouse",
				Source:          "ios_apps",
				TeamID:          teamID,
				Filename:        "inhouse.ipa",
				Extension:       "ipa",
				Platform:        "ios",
				UserID:          user1.ID,
				ValidatedLabels: &fleet.LabelIdentsWithScope{},
			})
			require.NoError(t, err)
			require.NotZero(t, inHouseID)
			require.NotZero(t, inHouseTitleID)

			// non-existent host
			_, err = ds.InsertSoftwareInstallRequest(ctx, 12, si.InstallerID, fleet.HostSoftwareInstallOptions{})
			require.ErrorAs(t, err, &nfe)

			// Host with software install pending
			tag := "-pending_install"
			hostPendingInstall, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tag + tc),
				NodeKey:       ptr.String("node-key-macos" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			_, err = ds.InsertSoftwareInstallRequest(ctx, hostPendingInstall.ID, si.InstallerID, fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)

			// Host with in-house app install pending
			tag = "-in-house-pending_install"
			hostInHousePendingInstall, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "ios-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-ios" + tag + tc),
				NodeKey:       ptr.String("node-key-ios" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "ios",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			nanoEnroll(t, ds, hostInHousePendingInstall, false)
			err = ds.InsertHostInHouseAppInstall(ctx, hostInHousePendingInstall.ID, inHouseID, inHouseTitleID, uuid.NewString(), fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)

			// Host with software install failed
			tag = "-failed_install"
			hostFailedInstall, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tag + tc),
				NodeKey:       ptr.String("node-key-macos" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			execID, err := ds.InsertSoftwareInstallRequest(ctx, hostFailedInstall.ID, si.InstallerID, fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)
			_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
				HostID:                hostFailedInstall.ID,
				InstallUUID:           execID,
				InstallScriptExitCode: ptr.Int(1),
			}, nil)
			require.NoError(t, err)

			// Host with in-house app failed install
			tag = "-in-house-failed_install"
			hostInHouseFailedInstall, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "ios-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-ios" + tag + tc),
				NodeKey:       ptr.String("node-key-ios" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "ios",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			nanoEnroll(t, ds, hostInHouseFailedInstall, false)
			cmdUUID := uuid.NewString()
			err = ds.InsertHostInHouseAppInstall(ctx, hostInHouseFailedInstall.ID, inHouseID, inHouseTitleID, cmdUUID, fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)

			// record a failed verification for that in-house app install
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `
					INSERT INTO nano_command_results (id, command_uuid, status, result)
					VALUES (?, ?, 'Error', '<?xml version="1.0"?><plist></plist>')`,
					hostInHouseFailedInstall.UUID, cmdUUID)
				if err != nil {
					return err
				}
				_, err = q.ExecContext(ctx, `
					UPDATE host_in_house_software_installs
					SET verification_command_uuid = ?, verification_failed_at = NOW(6)
					WHERE command_uuid = ? AND host_id = ?`,
					uuid.NewString(), cmdUUID, hostInHouseFailedInstall.ID,
				)
				return err
			})
			_, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), hostInHouseFailedInstall.ID, cmdUUID)
			require.NoError(t, err)

			// Host with software install successful
			tag = "-installed"
			hostInstalled, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tag + tc),
				NodeKey:       ptr.String("node-key-macos" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			execID, err = ds.InsertSoftwareInstallRequest(ctx, hostInstalled.ID, si.InstallerID, fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)
			_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
				HostID:                hostInstalled.ID,
				InstallUUID:           execID,
				InstallScriptExitCode: ptr.Int(0),
			}, nil)
			require.NoError(t, err)

			// host with in-house successful install
			tag = "-in-house-installed"
			hostInHouseInstalled, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "ios-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-ios" + tag + tc),
				NodeKey:       ptr.String("node-key-ios" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "ios",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			nanoEnroll(t, ds, hostInHouseInstalled, false)
			cmdUUID = uuid.NewString()
			err = ds.InsertHostInHouseAppInstall(ctx, hostInHouseInstalled.ID, inHouseID, inHouseTitleID, cmdUUID, fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)

			// record a successful verification for that in-house app install
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `
					INSERT INTO nano_command_results (id, command_uuid, status, result)
					VALUES (?, ?, 'Acknowledged', '<?xml version="1.0"?><plist></plist>')`,
					hostInHouseInstalled.UUID, cmdUUID)
				if err != nil {
					return err
				}
				_, err = q.ExecContext(ctx, `
					UPDATE host_in_house_software_installs
					SET verification_command_uuid = ?, verification_at = NOW(6)
					WHERE command_uuid = ? AND host_id = ?`,
					uuid.NewString(), cmdUUID, hostInHouseInstalled.ID,
				)
				return err
			})
			_, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), hostInHouseInstalled.ID, cmdUUID)
			require.NoError(t, err)

			// Host with pending uninstall
			tag = "-pending_uninstall"
			hostPendingUninstall, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tag + tc),
				NodeKey:       ptr.String("node-key-macos" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			err = ds.InsertSoftwareUninstallRequest(ctx, "uuid"+tag+tc, hostPendingUninstall.ID, si.InstallerID, false)
			require.NoError(t, err)

			// Host with failed uninstall
			tag = "-failed_uninstall"
			hostFailedUninstall, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tag + tc),
				NodeKey:       ptr.String("node-key-macos" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			execID = "uuid" + tag + tc
			err = ds.InsertSoftwareUninstallRequest(ctx, execID, hostFailedUninstall.ID, si.InstallerID, false)
			require.NoError(t, err)
			_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostFailedUninstall.ID,
				ExecutionID: execID,
				ExitCode:    1,
			}, nil)
			require.NoError(t, err)

			// Host with successful uninstall
			tag = "-uninstalled"
			hostUninstalled, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tag + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tag + tc),
				NodeKey:       ptr.String("node-key-macos" + tag + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			execID = "uuid" + tag + tc
			err = ds.InsertSoftwareUninstallRequest(ctx, execID, hostUninstalled.ID, si.InstallerID, false)
			require.NoError(t, err)
			_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
				HostID:      hostUninstalled.ID,
				ExecutionID: execID,
				ExitCode:    0,
			}, nil)
			require.NoError(t, err)

			// Uninstall request with unknown host
			err = ds.InsertSoftwareUninstallRequest(ctx, "uuid"+tag+tc, 99999, si.InstallerID, false)
			assert.ErrorContains(t, err, "Host")

			allHostIDs := []uint{
				hostPendingInstall.ID,
				hostFailedInstall.ID,
				hostInstalled.ID,
				hostPendingUninstall.ID,
				hostFailedUninstall.ID,
				hostUninstalled.ID,
				hostInHousePendingInstall.ID,
				hostInHouseFailedInstall.ID,
				hostInHouseInstalled.ID,
			}
			for _, hid := range allHostIDs {
				err = ds.AddLabelsToHost(ctx, hid, []uint{labelsByName[fleet.BuiltinLabelNameAllHosts]})
				require.NoError(t, err)
			}

			userTeamFilter := fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String("admin")},
			}

			// for this test, teamID is nil for no-team, but the ListHosts filter
			// returns "all teams" if TeamFilter = nil, it needs to use TeamFilter =
			// 0 for "no team" only.
			teamFilter := teamID
			if teamFilter == nil {
				teamFilter = ptr.Uint(0)
			}

			// get the names of hosts, useful for debugging
			getHostNames := func(hosts []*fleet.Host) []string {
				hostNames := make([]string, 0, len(hosts))
				for _, h := range hosts {
					hostNames = append(hostNames, h.Hostname)
				}
				return hostNames
			}
			pluckHostIDs := func(hosts []*fleet.Host) []uint {
				hostIDs := make([]uint, 0, len(hosts))
				for _, h := range hosts {
					hostIDs = append(hostIDs, h.ID)
				}
				return hostIDs
			}

			cases := []struct {
				desc        string
				opts        fleet.HostListOptions
				wantHostIDs []uint
			}{
				{
					desc: "list hosts with software install pending requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareInstallPending),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostPendingInstall.ID},
				},
				{
					desc: "list hosts with in-house app pending install",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: &inHouseTitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareInstallPending),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostInHousePendingInstall.ID},
				},
				{
					desc: "list hosts with all pending requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwarePending),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostPendingInstall.ID, hostPendingUninstall.ID},
				},
				{
					desc: "list hosts with in-house app all pending requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: &inHouseTitleID,
						SoftwareStatusFilter:  new(fleet.SoftwarePending),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostInHousePendingInstall.ID},
				},
				{
					desc: "list hosts with software install failed requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareInstallFailed),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostFailedInstall.ID},
				},
				{
					desc: "list hosts with in-house install failed requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: &inHouseTitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareInstallFailed),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostInHouseFailedInstall.ID},
				},
				{
					desc: "list hosts with all failed requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareFailed),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostFailedInstall.ID, hostFailedUninstall.ID},
				},
				{
					desc: "list hosts with in-house all failed requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: &inHouseTitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareFailed),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostInHouseFailedInstall.ID},
				},
				{
					desc: "list hosts with software installed",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareInstalled),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostInstalled.ID},
				},
				{
					desc: "list hosts with in-house app installed",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: &inHouseTitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareInstalled),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostInHouseInstalled.ID},
				},
				{
					desc: "list hosts with pending software uninstall requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareUninstallPending),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostPendingUninstall.ID},
				},
				{
					desc: "list hosts with failed software uninstall requests",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						SoftwareStatusFilter:  new(fleet.SoftwareUninstallFailed),
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{hostFailedUninstall.ID},
				},
				{
					desc: "list all hosts with the software title",
					opts: fleet.HostListOptions{
						ListOptions:           fleet.ListOptions{PerPage: 100},
						SoftwareTitleIDFilter: installerMeta.TitleID,
						TeamFilter:            teamFilter,
					},
					wantHostIDs: []uint{},
				},
			}
			for _, c := range cases {
				t.Run(c.desc, func(t *testing.T) {
					hosts, err := ds.ListHosts(ctx, userTeamFilter, c.opts)
					require.NoError(t, err)
					require.Len(t, hosts, len(c.wantHostIDs), getHostNames(hosts))
					require.ElementsMatch(t, c.wantHostIDs, pluckHostIDs(hosts))

					if c.opts.SoftwareStatusFilter == nil && c.opts.SoftwareTitleIDFilter != nil {
						// for list hosts by label, if no status is provided, the title ID filter is ignored/no-op,
						// so all host IDs are returned
						c.wantHostIDs = allHostIDs
					}
					hosts, err = ds.ListHostsInLabel(ctx, userTeamFilter, labelsByName[fleet.BuiltinLabelNameAllHosts], c.opts)
					require.NoError(t, err)
					require.Len(t, hosts, len(c.wantHostIDs), getHostNames(hosts))
					require.ElementsMatch(t, c.wantHostIDs, pluckHostIDs(hosts))
				})
			}

			summary, err := ds.GetSummaryHostSoftwareInstalls(ctx, installerMeta.InstallerID)
			require.NoError(t, err)
			require.Equal(t, fleet.SoftwareInstallerStatusSummary{
				Installed:        1,
				PendingInstall:   1,
				FailedInstall:    1,
				PendingUninstall: 1,
				FailedUninstall:  1,
			}, *summary)

			vppSummary, err := ds.GetSummaryHostInHouseAppInstalls(ctx, teamID, inHouseID)
			require.NoError(t, err)
			require.Equal(t, fleet.VPPAppStatusSummary{
				Installed: 1,
				Pending:   1,
				Failed:    1,
			}, *vppSummary)
		})
	}
}

func testGetSoftwareInstallResult(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	teamID := team.ID

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	for _, tc := range []struct {
		name                    string
		expectedStatus          fleet.SoftwareInstallerStatus
		postInstallScriptEC     *int
		preInstallQueryOutput   *string
		installScriptEC         *int
		postInstallScriptOutput *string
		installScriptOutput     *string
	}{
		{
			name:                    "pending install",
			expectedStatus:          fleet.SoftwareInstallPending,
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install post install script",
			expectedStatus:          fleet.SoftwareInstallFailed,
			postInstallScriptEC:     ptr.Int(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install install script",
			expectedStatus:          fleet.SoftwareInstallFailed,
			installScriptEC:         ptr.Int(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install pre install query",
			expectedStatus:          fleet.SoftwareInstallFailed,
			preInstallQueryOutput:   ptr.String(""),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// create a host and software installer
			swFilename := "file_" + tc.name + ".pkg"
			swStorageID := "hash_" + tc.name
			installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:           "foo" + tc.name,
				Source:          "bar" + tc.name,
				InstallScript:   "echo " + tc.name,
				Version:         "1.11",
				TeamID:          &teamID,
				Filename:        swFilename,
				StorageID:       swStorageID,
				UserID:          user1.ID,
				ValidatedLabels: &fleet.LabelIdentsWithScope{},
			})
			require.NoError(t, err)
			host, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test-" + tc.name,
				ComputerName:  "macos-test-" + tc.name,
				OsqueryHostID: ptr.String("osquery-macos-" + tc.name),
				NodeKey:       ptr.String("node-key-macos-" + tc.name),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        &teamID,
			})
			require.NoError(t, err)

			beforeInstallRequest := time.Now()
			installUUID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, fleet.HostSoftwareInstallOptions{})
			require.NoError(t, err)

			res, err := ds.GetSoftwareInstallResults(ctx, installUUID)
			require.NoError(t, err)
			require.NotNil(t, res.UpdatedAt)
			require.Less(t, beforeInstallRequest, res.CreatedAt)
			createdAt := res.CreatedAt
			require.Less(t, beforeInstallRequest, *res.UpdatedAt)

			beforeInstallResult := time.Now()
			_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
				HostID:                    host.ID,
				InstallUUID:               installUUID,
				PreInstallConditionOutput: tc.preInstallQueryOutput,
				InstallScriptExitCode:     tc.installScriptEC,
				InstallScriptOutput:       tc.installScriptOutput,
				PostInstallScriptExitCode: tc.postInstallScriptEC,
				PostInstallScriptOutput:   tc.postInstallScriptOutput,
			}, nil)
			require.NoError(t, err)

			// edit installer to ensure host software install is unaffected
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err = q.ExecContext(ctx, `
					UPDATE software_installers SET filename = 'something different', version = '1.23' WHERE id = ?`,
					installerID)
				require.NoError(t, err)
				return nil
			})

			res, err = ds.GetSoftwareInstallResults(ctx, installUUID)
			require.NoError(t, err)
			require.Equal(t, swFilename, res.SoftwarePackage)
			// hash comes from the installer, which still exists here
			require.NotNil(t, res.HashSHA256)
			require.Equal(t, swStorageID, *res.HashSHA256)

			// delete installer to confirm that we can still access the install record (unless pending)
			err = ds.DeleteSoftwareInstaller(ctx, installerID)
			require.NoError(t, err)

			if tc.expectedStatus == fleet.SoftwareInstallPending { // expect pending to be deleted
				_, err = ds.GetSoftwareInstallResults(ctx, installUUID)
				require.Error(t, err, notFound("HostSoftwareInstallerResult"))
				return
			}

			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				// ensure version is not changed, though we don't expose it yet
				var version string
				err := sqlx.GetContext(ctx, q, &version, `SELECT "version" FROM host_software_installs WHERE execution_id = ?`, installUUID)
				require.NoError(t, err)
				require.Equal(t, "1.11", version)

				return nil
			})

			res, err = ds.GetSoftwareInstallResults(ctx, installUUID)
			require.NoError(t, err)

			require.Equal(t, installUUID, res.InstallUUID)
			require.Equal(t, tc.expectedStatus, res.Status)
			require.Equal(t, swFilename, res.SoftwarePackage)
			// installer was deleted, so its hash is no longer available
			require.Nil(t, res.HashSHA256)
			require.Equal(t, host.ID, res.HostID)
			require.Equal(t, tc.preInstallQueryOutput, res.PreInstallQueryOutput)
			require.Equal(t, tc.postInstallScriptOutput, res.PostInstallScriptOutput)
			require.Equal(t, tc.installScriptOutput, res.Output)
			require.NotNil(t, res.CreatedAt)
			require.Equal(t, createdAt, res.CreatedAt)
			require.NotNil(t, res.UpdatedAt)
			require.Less(t, beforeInstallResult, *res.UpdatedAt)
		})
	}
}

func testCleanupUnusedSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	dir := t.TempDir()
	store, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	assertExisting := func(want []string) {
		dirEnts, err := os.ReadDir(filepath.Join(dir, "software-installers"))
		require.NoError(t, err)
		got := make([]string, 0, len(dirEnts))
		for _, de := range dirEnts {
			if de.Type().IsRegular() {
				got = append(got, de.Name())
			}
		}
		require.ElementsMatch(t, want, got)
	}

	// cleanup an empty store
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now())
	require.NoError(t, err)
	assertExisting(nil)

	// put an installer and save it in the DB
	ins0 := "installer0"
	ins0File := bytes.NewReader([]byte("installer0"))
	err = store.Put(ctx, ins0, ins0File)
	require.NoError(t, err)
	_, _ = ins0File.Seek(0, 0)
	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)
	assertExisting([]string{ins0})

	swi, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer0",
		Title:           "ins0",
		Source:          "apps",
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	assertExisting([]string{ins0})
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now())
	require.NoError(t, err)
	assertExisting([]string{ins0})

	// remove it from the DB, will now cleanup
	err = ds.DeleteSoftwareInstaller(ctx, swi)
	require.NoError(t, err)

	// would clean up, but not created before 1m ago
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	assertExisting([]string{ins0})

	// do actual cleanup
	err = ds.CleanupUnusedSoftwareInstallers(ctx, store, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assertExisting(nil)
}

func testBatchSetSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// create a couple hosts
	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host1.ID, host2.ID}))
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// TODO(roberto): perform better assertions, we should have everything
	// to check that the actual values of everything match.
	assertSoftware := func(wantTitles []fleet.SoftwareTitle) {
		tmFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
		titles, _, _, err := ds.ListSoftwareTitles(
			ctx,
			fleet.SoftwareTitleListOptions{TeamID: &team.ID},
			tmFilter,
		)
		require.NoError(t, err)
		require.Len(t, titles, len(wantTitles))

		for _, title := range titles {
			meta, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, title.ID, false)
			require.NoError(t, err)
			require.NotNil(t, meta.TitleID)
		}
	}

	// batch set with everything empty
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, nil)
	require.NoError(t, err)
	softwareInstallers, err := ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, softwareInstallers)
	assertSoftware(nil)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, softwareInstallers)
	assertSoftware(nil)

	// add a single installer
	ins0 := "installer0"
	ins0File := bytes.NewReader([]byte("installer0"))
	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)
	displayName := "Display name 1"
	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:        "install",
		InstallerFile:        tfr0,
		StorageID:            ins0,
		Filename:             "installer0",
		Title:                "ins0",
		Source:               "apps",
		Version:              "1",
		PreInstallQuery:      "foo",
		UserID:               user1.ID,
		Platform:             "darwin",
		URL:                  "https://example.com",
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		BundleIdentifier:     "com.example.ins0",
		DisplayName:          displayName,
		FleetMaintainedAppID: ptr.Uint(maintainedApp.ID),
	}})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	require.NotNil(t, softwareInstallers[0].TeamID)
	require.Equal(t, team.ID, *softwareInstallers[0].TeamID)
	require.NotNil(t, softwareInstallers[0].TitleID)
	require.Equal(t, "https://example.com", softwareInstallers[0].URL)
	require.Equal(t, maintainedApp.ID, *softwareInstallers[0].FleetMaintainedAppID)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", ExtensionFor: ""},
	})
	meta, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *softwareInstallers[0].TitleID, false)
	require.NoError(t, err)
	require.Equal(t, displayName, meta.DisplayName)

	// add a new installer + ins0 installer
	// mark ins0 as install_during_setup
	ins1 := "installer1"
	ins1File := bytes.NewReader([]byte("installer1"))
	tfr1, err := fleet.NewTempFileReader(ins1File, t.TempDir)
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     tfr1,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
			UserID:            user1.ID,
			Platform:          "darwin",
			URL:               "https://example2.com",
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 2)
	require.NotNil(t, softwareInstallers[0].TitleID)
	require.NotNil(t, softwareInstallers[0].TeamID)
	require.Equal(t, team.ID, *softwareInstallers[0].TeamID)
	require.Equal(t, "https://example.com", softwareInstallers[0].URL)
	require.NotNil(t, softwareInstallers[1].TitleID)
	require.NotNil(t, softwareInstallers[1].TeamID)
	require.Equal(t, team.ID, *softwareInstallers[1].TeamID)
	require.Equal(t, "https://example2.com", softwareInstallers[1].URL)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", ExtensionFor: ""},
		{Name: ins1, Source: "apps", ExtensionFor: ""},
	})

	// remove ins0 fails due to install_during_setup
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     tfr1,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
			UserID:            user1.ID,
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerInstalledDuringSetup)

	// batch-set both installers again, this time with nil install_during_setup for ins0,
	// will keep it as true.
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: nil,
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     tfr1,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
			UserID:            user1.ID,
			Platform:          "darwin",
			URL:               "https://example2.com",
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	// mark ins0 as NOT install_during_setup
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			DisplayName:        displayName,
			InstallDuringSetup: ptr.Bool(false),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     tfr1,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
			UserID:            user1.ID,
			Platform:          "darwin",
			URL:               "https://example2.com",
			DisplayName:       displayName,
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 2)
	ins0TitleID := softwareInstallers[0].TitleID

	// remove ins0
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     tfr1,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
			UserID:            user1.ID,
			DisplayName:       displayName,
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	require.NotNil(t, softwareInstallers[0].TitleID)
	require.NotNil(t, softwareInstallers[0].TeamID)
	require.Empty(t, softwareInstallers[0].URL)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins1, Source: "apps", ExtensionFor: ""},
	})

	// display name is deleted for ins0
	_, err = ds.getSoftwareTitleDisplayName(ctx, team.ID, *ins0TitleID)
	require.ErrorContains(t, err, "not found")

	instDetails1, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *softwareInstallers[0].TitleID, false)
	require.NoError(t, err)

	// add pending and completed installs for ins1
	_, err = ds.InsertSoftwareInstallRequest(ctx, host1.ID, instDetails1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	execID2, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, instDetails1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           execID2,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	summary, err := ds.GetSummaryHostSoftwareInstalls(ctx, instDetails1.InstallerID)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{Installed: 1, PendingInstall: 1}, *summary)

	// batch-set without changes
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:     "install",
			PostInstallScript: "post-install",
			InstallerFile:     tfr1,
			StorageID:         ins1,
			Filename:          ins1,
			Title:             ins1,
			Source:            "apps",
			Version:           "2",
			PreInstallQuery:   "select 1 from bar;",
			UserID:            user1.ID,
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	// installs stats haven't changed
	summary, err = ds.GetSummaryHostSoftwareInstalls(ctx, instDetails1.InstallerID)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{Installed: 1, PendingInstall: 1}, *summary)

	// remove ins1 and add ins0
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install",
			InstallerFile:   tfr0,
			StorageID:       ins0,
			Filename:        ins0,
			Title:           ins0,
			Source:          "apps",
			Version:         "1",
			PreInstallQuery: "select 0 from foo;",
			UserID:          user1.ID,
			Platform:        "darwin",
			URL:             "https://example.com",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	// stats don't report anything about ins1 anymore
	summary, err = ds.GetSummaryHostSoftwareInstalls(ctx, instDetails1.InstallerID)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{Installed: 0, PendingInstall: 0}, *summary)
	pendingHost1, err := ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Empty(t, pendingHost1)

	// add pending and completed installs for ins0
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	instDetails0, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *softwareInstallers[0].TitleID, false)
	require.NoError(t, err)

	_, err = ds.InsertSoftwareInstallRequest(ctx, host1.ID, instDetails0.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	execID2b, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, instDetails0.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           execID2b,
		InstallScriptExitCode: ptr.Int(1),
	}, nil)
	require.NoError(t, err)

	pendingHost1, err = ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Len(t, pendingHost1, 1)

	summary, err = ds.GetSummaryHostSoftwareInstalls(ctx, instDetails0.InstallerID)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{FailedInstall: 1, PendingInstall: 1}, *summary)

	// Add software installer with same name different bundle id
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:    "install",
		InstallerFile:    tfr0,
		StorageID:        ins0,
		Filename:         "installer0",
		Title:            "ins0",
		Source:           "apps",
		Version:          "1",
		PreInstallQuery:  "foo",
		UserID:           user1.ID,
		Platform:         "darwin",
		URL:              "https://example.com",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		BundleIdentifier: "com.example.different.ins0",
	}})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", ExtensionFor: "", BundleIdentifier: ptr.String("com.example.different.ins0")},
	})

	// Add software installer with the same bundle id but different name
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:    "install",
		InstallerFile:    tfr0,
		StorageID:        ins0,
		Filename:         "installer0",
		Title:            "ins0-different",
		Source:           "apps",
		Version:          "1",
		PreInstallQuery:  "foo",
		UserID:           user1.ID,
		Platform:         "darwin",
		URL:              "https://example.com",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		BundleIdentifier: "com.example.ins0",
	}})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: "ins0-different", Source: "apps", ExtensionFor: "", BundleIdentifier: ptr.String("com.example.ins0")},
	})

	// remove everything
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, softwareInstallers)
	assertSoftware([]fleet.SoftwareTitle{})

	// stats don't report anything about ins0 anymore
	summary, err = ds.GetSummaryHostSoftwareInstalls(ctx, instDetails0.InstallerID)
	require.NoError(t, err)
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{FailedInstall: 0, PendingInstall: 0}, *summary)
	pendingHost1, err = ds.ListPendingSoftwareInstalls(ctx, host1.ID)
	require.NoError(t, err)
	require.Empty(t, pendingHost1)
}

func testBatchSetSoftwareInstallersMultipleCustomPackages(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "h1", "1", "h1key", "h1uuid", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	// pkg builds a custom package payload for the given title. santa wraps it for the
	// shared "Santa" title used by the single-title lifecycle checks below; each santa
	// package differs only by storage id (hash) and version.
	pkg := func(title string, bundle string, storage string, version string) *fleet.UploadSoftwareInstallerPayload {
		tfr, err := fleet.NewTempFileReader(bytes.NewReader([]byte(storage)), t.TempDir)
		require.NoError(t, err)
		return &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			InstallerFile:    tfr,
			StorageID:        storage,
			Filename:         storage,
			Title:            title,
			Source:           "apps",
			Version:          version,
			UserID:           user1.ID,
			Platform:         "darwin",
			URL:              "https://example.com/" + storage,
			BundleIdentifier: bundle,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		}
	}
	santa := func(storage string, version string) *fleet.UploadSoftwareInstallerPayload {
		return pkg("Santa", "com.northpolesec.santa", storage, version)
	}

	// apply two packages of the same title
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		santa("santaA", "2026.2"),
		santa("santaB", "2026.4"),
	})
	require.NoError(t, err)

	all, err := ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.NotNil(t, all[0].TitleID)
	titleID := *all[0].TitleID

	// both packages belong to one title, ordered first-added first (id ascending)
	pkgs, err := ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	require.Less(t, pkgs[0].InstallerID, pkgs[1].InstallerID)
	require.Equal(t, "santaA", pkgs[0].StorageID)
	require.Equal(t, "santaB", pkgs[1].StorageID)
	firstID, secondID := pkgs[0].InstallerID, pkgs[1].InstallerID

	// re-apply with the list reordered: ids and order are unchanged
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		santa("santaB", "2026.4"),
		santa("santaA", "2026.2"),
	})
	require.NoError(t, err)
	pkgs, err = ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	require.Equal(t, firstID, pkgs[0].InstallerID)
	require.Equal(t, secondID, pkgs[1].InstallerID)
	require.Equal(t, "santaA", pkgs[0].StorageID)

	// adding a package appends it after the existing ones
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		santa("santaA", "2026.2"),
		santa("santaB", "2026.4"),
		santa("santaC", "2026.6"),
	})
	require.NoError(t, err)
	pkgs, err = ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 3)
	require.Equal(t, firstID, pkgs[0].InstallerID)
	require.Equal(t, secondID, pkgs[1].InstallerID)
	require.Greater(t, pkgs[2].InstallerID, secondID)
	require.Equal(t, "santaC", pkgs[2].StorageID)

	// a policy and pending install point at santaA (kept) and santaB (about to be dropped)
	keepPolicy, err := ds.NewTeamPolicy(ctx, team.ID, &user1.ID, fleet.PolicyPayload{Name: "keep", Query: "SELECT 1;", SoftwareInstallerID: &firstID})
	require.NoError(t, err)
	dropPolicy, err := ds.NewTeamPolicy(ctx, team.ID, &user1.ID, fleet.PolicyPayload{Name: "drop", Query: "SELECT 1;", SoftwareInstallerID: &secondID})
	require.NoError(t, err)
	_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, firstID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, secondID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// removing a package deletes it (source of truth), surviving siblings keep ids
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		santa("santaA", "2026.2"),
		santa("santaC", "2026.6"),
	})
	require.NoError(t, err)
	pkgs, err = ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	require.Equal(t, firstID, pkgs[0].InstallerID)
	require.Equal(t, "santaA", pkgs[0].StorageID)
	require.Equal(t, "santaC", pkgs[1].StorageID)

	// dropping santaB re-points its policy to the first-added surviving package (santaA)
	// and cancels its pending install; santaA's own policy is untouched
	dropped, err := ds.TeamPolicy(ctx, team.ID, dropPolicy.ID)
	require.NoError(t, err)
	require.NotNil(t, dropped.SoftwareInstallerID)
	require.Equal(t, firstID, *dropped.SoftwareInstallerID)
	kept, err := ds.TeamPolicy(ctx, team.ID, keepPolicy.ID)
	require.NoError(t, err)
	require.NotNil(t, kept.SoftwareInstallerID)
	require.Equal(t, firstID, *kept.SoftwareInstallerID)
	pending, err := ds.ListPendingSoftwareInstalls(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	// a hash duplicate within the batch fails and leaves the title unchanged
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		santa("santaA", "2026.2"),
		santa("santaA", "2026.2-dup"),
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "already added")
	pkgs, err = ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)

	// exceeding the per-title package limit fails
	tooMany := make([]*fleet.UploadSoftwareInstallerPayload, 0, fleet.MaxPackagesPerTitle+1)
	for i := range fleet.MaxPackagesPerTitle + 1 {
		tooMany = append(tooMany, santa(fmt.Sprintf("santa-%d", i), fmt.Sprintf("v%d", i)))
	}
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, tooMany)
	require.Error(t, err)
	require.ErrorContains(t, err, "packages")

	// mixing a Fleet-maintained app with a custom package on one title fails
	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Santa",
		Slug:             "santa",
		Platform:         "darwin",
		UniqueIdentifier: "com.northpolesec.santa",
	})
	require.NoError(t, err)
	fma := santa("santaFMA", "2026.8")
	fma.FleetMaintainedAppID = new(maintainedApp.ID)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		santa("santaA", "2026.2"),
		fma,
	})
	require.Error(t, err)

	// switch the title to a Fleet-maintained app, then back to a custom package:
	// the stale FMA row must be removed so the title holds only the custom package.
	fmaOnly := santa("santaFMA2", "2027.1")
	fmaOnly.FleetMaintainedAppID = new(maintainedApp.ID)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{fmaOnly})
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{santa("santaX", "2027.2")})
	require.NoError(t, err)
	pkgs, err = ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 1)
	require.Equal(t, "santaX", pkgs[0].StorageID)
	var fmaRows int
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &fmaRows,
			`SELECT COUNT(*) FROM software_installers WHERE global_or_team_id = ? AND title_id = ? AND fleet_maintained_app_id IS NOT NULL`,
			team.ID, titleID)
	})
	require.Zero(t, fmaRows)

	// removing the title's last package (title no longer in the batch) nulls out a
	// policy that pointed at it, since there is no sibling to re-point to
	orphanPolicy, err := ds.NewTeamPolicy(ctx, team.ID, &user1.ID, fleet.PolicyPayload{Name: "orphan", Query: "SELECT 1;", SoftwareInstallerID: &pkgs[0].InstallerID})
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{pkg("Bravo", "com.example.bravo", "bravo-x", "1.0")})
	require.NoError(t, err)
	orphaned, err := ds.TeamPolicy(ctx, team.ID, orphanPolicy.ID)
	require.NoError(t, err)
	require.Nil(t, orphaned.SoftwareInstallerID)

	// A single package file (one path entry) can hold packages for different titles on a
	// separate team, including a mix of a single-package title and a multi-package title,
	// with one title's packages interleaved with another's. Each still lands on its own
	// resolved title in file order.
	mixedTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-mixed"})
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &mixedTeam.ID, []*fleet.UploadSoftwareInstallerPayload{
		pkg("Bravo", "com.example.bravo", "bravo-1", "1.0"),
		pkg("Alpha", "com.example.alpha", "alpha-1", "1.0"),
		pkg("Bravo", "com.example.bravo", "bravo-2", "2.0"),
	})
	require.NoError(t, err)

	mixedAll, err := ds.GetSoftwareInstallers(ctx, mixedTeam.ID)
	require.NoError(t, err)
	require.Len(t, mixedAll, 3)

	// each package resolved to a title by its storage id; Bravo's two packages share a
	// title distinct from Alpha's
	titleOf := map[string]uint{}
	for _, si := range mixedAll {
		require.NotNil(t, si.TitleID)
		titleOf[si.HashSHA256] = *si.TitleID
	}
	require.Equal(t, titleOf["bravo-1"], titleOf["bravo-2"])
	require.NotEqual(t, titleOf["alpha-1"], titleOf["bravo-1"])

	// Alpha holds one package
	alphaPkgs, err := ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &mixedTeam.ID, titleOf["alpha-1"])
	require.NoError(t, err)
	require.Len(t, alphaPkgs, 1)

	// Bravo's packages keep file order (bravo-1 first-added) even though Alpha was listed between them.
	bravoPkgs, err := ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &mixedTeam.ID, titleOf["bravo-1"])
	require.NoError(t, err)
	require.Len(t, bravoPkgs, 2)
	require.Equal(t, "bravo-1", bravoPkgs[0].StorageID)
	require.Equal(t, "bravo-2", bravoPkgs[1].StorageID)
	require.Less(t, bravoPkgs[0].InstallerID, bravoPkgs[1].InstallerID)
}

func testBatchSetSoftwareInstallersWithUpgradeCodes(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// helper to get upgrade_code from software_titles table
	getUpgradeCodeForTitle := func(titleID uint) *string {
		var upgradeCode *string
		err := sqlx.GetContext(ctx, ds.reader(ctx), &upgradeCode,
			`SELECT upgrade_code FROM software_titles WHERE id = ?`, titleID)
		require.NoError(t, err)
		return upgradeCode
	}

	// Create a Windows installer with an upgrade code
	ins0 := "windows-installer"
	ins0File := bytes.NewReader([]byte("installer0"))
	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)
	upgradeCode := "{12345678-1234-1234-1234-123456789012}"

	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:   "install.ps1",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer0.msi",
		Title:           "Windows App",
		Source:          "programs",
		Version:         "1.0",
		UserID:          user1.ID,
		Platform:        "windows",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		UpgradeCode:     upgradeCode,
	}})
	require.NoError(t, err)

	softwareInstallers, err := ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	require.NotNil(t, softwareInstallers[0].TitleID)
	titleID := *softwareInstallers[0].TitleID

	// Verify the upgrade_code was stored in software_titles
	storedUpgradeCode := getUpgradeCodeForTitle(titleID)
	require.NotNil(t, storedUpgradeCode)
	require.Equal(t, upgradeCode, *storedUpgradeCode)

	// Update the installer (same upgrade_code, different version) - should match the same title
	ins0File = bytes.NewReader([]byte("installer0-v2"))
	tfr0, err = fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)

	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:   "install.ps1",
		InstallerFile:   tfr0,
		StorageID:       ins0 + "-v2",
		Filename:        "installer0-v2.msi",
		Title:           "Windows App",
		Source:          "programs",
		Version:         "2.0",
		UserID:          user1.ID,
		Platform:        "windows",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		UpgradeCode:     upgradeCode,
	}})
	require.NoError(t, err)

	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	require.NotNil(t, softwareInstallers[0].TitleID)
	// Title ID should be the same since upgrade_code matches
	require.Equal(t, titleID, *softwareInstallers[0].TitleID)

	// Verify upgrade_code is still correct
	storedUpgradeCode = getUpgradeCodeForTitle(titleID)
	require.NotNil(t, storedUpgradeCode)
	require.Equal(t, upgradeCode, *storedUpgradeCode)

	// Add a second Windows installer with no upgrade code
	ins1 := "windows-installer2"
	ins1File := bytes.NewReader([]byte("installer1"))
	tfr1, err := fleet.NewTempFileReader(ins1File, t.TempDir)
	require.NoError(t, err)

	// Reset tfr0 for reuse
	ins0File = bytes.NewReader([]byte("installer0-v2"))
	tfr0, err = fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)

	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install.ps1",
			InstallerFile:   tfr0,
			StorageID:       ins0 + "-v2",
			Filename:        "installer0-v2.msi",
			Title:           "Windows App",
			Source:          "programs",
			Version:         "2.0",
			UserID:          user1.ID,
			Platform:        "windows",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			UpgradeCode:     upgradeCode,
		},
		{
			InstallScript:   "install2.ps1",
			InstallerFile:   tfr1,
			StorageID:       ins1,
			Filename:        "installer1.msi",
			Title:           "Another Windows App",
			Source:          "programs",
			Version:         "1.0",
			UserID:          user1.ID,
			Platform:        "windows",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			UpgradeCode:     "",
		},
	})
	require.NoError(t, err)

	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 2)

	// Find the second installer and verify its upgrade_code
	var secondTitleID uint
	for _, si := range softwareInstallers {
		if *si.TitleID != titleID {
			secondTitleID = *si.TitleID
			break
		}
	}
	require.NotZero(t, secondTitleID)

	storedUpgradeCode2 := getUpgradeCodeForTitle(secondTitleID)
	require.NotNil(t, storedUpgradeCode2)
	require.Empty(t, *storedUpgradeCode2)

	// Verify non-Windows installers don't get upgrade_code set
	ins2 := "mac-installer"
	ins2File := bytes.NewReader([]byte("installer2"))
	tfr2, err := fleet.NewTempFileReader(ins2File, t.TempDir)
	require.NoError(t, err)

	// Reset tfr0 and tfr1 for reuse
	ins0File = bytes.NewReader([]byte("installer0-v2"))
	tfr0, err = fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)
	ins1File = bytes.NewReader([]byte("installer1"))
	tfr1, err = fleet.NewTempFileReader(ins1File, t.TempDir)
	require.NoError(t, err)

	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install.ps1",
			InstallerFile:   tfr0,
			StorageID:       ins0 + "-v2",
			Filename:        "installer0-v2.msi",
			Title:           "Windows App",
			Source:          "programs",
			Version:         "2.0",
			UserID:          user1.ID,
			Platform:        "windows",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			UpgradeCode:     upgradeCode,
		},
		{
			InstallScript:   "install2.ps1",
			InstallerFile:   tfr1,
			StorageID:       ins1,
			Filename:        "installer1.msi",
			Title:           "Another Windows App",
			Source:          "programs",
			Version:         "1.0",
			UserID:          user1.ID,
			Platform:        "windows",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			UpgradeCode:     "",
		},
		{
			InstallScript:    "install3.sh",
			InstallerFile:    tfr2,
			StorageID:        ins2,
			Filename:         "installer2.pkg",
			Title:            "Mac App",
			Source:           "apps",
			Version:          "1.0",
			UserID:           user1.ID,
			Platform:         "darwin",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: "com.example.macapp",
		},
	})
	require.NoError(t, err)

	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 3)

	// Find the mac installer and verify upgrade_code is NULL
	var macTitleID uint
	for _, si := range softwareInstallers {
		if *si.TitleID != titleID && *si.TitleID != secondTitleID {
			macTitleID = *si.TitleID
			break
		}
	}
	require.NotZero(t, macTitleID)

	macUpgradeCode := getUpgradeCodeForTitle(macTitleID)
	require.Nil(t, macUpgradeCode)

	// Clean up
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)

	// Regression test for GitHub issue #48054: a Windows FMA added via gitops must not create a
	// duplicate software title when a host already reported the same software with an upgrade_code.
	// Host reports Aircall (installed manually via winget) with an upgrade_code, creating a title
	// whose unique_identifier is derived from the upgrade_code.
	aircallHost := test.NewHost(t, ds, "aircall-host", "", "aircall-host-key", "aircall-host-uuid", time.Now())
	aircallUpgradeCode := "{9F7A3B21-4C5D-4E6F-8A9B-0C1D2E3F4A5B}"
	_, err = ds.UpdateHostSoftware(ctx, aircallHost.ID, []fleet.Software{
		{Name: "Aircall", Version: "1.0", Source: "programs", UpgradeCode: &aircallUpgradeCode},
	})
	require.NoError(t, err)

	var aircallTitleIDs []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &aircallTitleIDs, `SELECT id FROM software_titles WHERE name = 'Aircall' AND source = 'programs'`))
	require.Len(t, aircallTitleIDs, 1)
	hostTitleID := aircallTitleIDs[0]
	require.Equal(t, aircallUpgradeCode, *getUpgradeCodeForTitle(hostTitleID))

	// Add the Aircall FMA for the team via gitops, with no upgrade_code on the payload.
	aircallFile := bytes.NewReader([]byte("aircall-installer"))
	aircallTFR, err := fleet.NewTempFileReader(aircallFile, t.TempDir)
	require.NoError(t, err)

	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:   "install.ps1",
		InstallerFile:   aircallTFR,
		StorageID:       "aircall-installer",
		Filename:        "aircall.msi",
		Title:           "Aircall",
		Source:          "programs",
		Version:         "1.0",
		UserID:          user1.ID,
		Platform:        "windows",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	}})
	require.NoError(t, err)

	// Still exactly one Aircall title, matched by name, keeping the host-reported upgrade_code.
	var aircallTitleIDsAfter []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &aircallTitleIDsAfter, `SELECT id FROM software_titles WHERE name = 'Aircall' AND source = 'programs'`))
	require.Equal(t, []uint{hostTitleID}, aircallTitleIDsAfter, "adding the Aircall FMA should not create a duplicate software title")
	require.Equal(t, aircallUpgradeCode, *getUpgradeCodeForTitle(hostTitleID), "matched title should keep the host-reported upgrade_code")

	// The Aircall installer should be linked to that same title.
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	require.NotNil(t, softwareInstallers[0].TitleID)
	require.Equal(t, hostTitleID, *softwareInstallers[0].TitleID, "installer should point at the existing title")
}

func testBatchSetSoftwareInstallersSetupExperienceSideEffects(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// create a host
	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host1.ID}))
	host1.TeamID = &team.ID
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	assertSoftware := func(wantTitles []fleet.SoftwareTitle) {
		tmFilter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
		titles, _, _, err := ds.ListSoftwareTitles(
			ctx,
			fleet.SoftwareTitleListOptions{TeamID: &team.ID},
			tmFilter,
		)
		require.NoError(t, err)
		require.Len(t, titles, len(wantTitles))

		for _, title := range titles {
			meta, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, title.ID, false)
			require.NoError(t, err)
			require.NotNil(t, meta.TitleID)
		}
	}

	// add two installers
	ins0 := "installer0"
	ins0File := bytes.NewReader([]byte("installer0"))
	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)

	ins1 := "installer1"
	ins1File := bytes.NewReader([]byte("installer1"))
	tfr1, err := fleet.NewTempFileReader(ins1File, t.TempDir)
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:      "install",
			PostInstallScript:  "post-install",
			InstallerFile:      tfr1,
			StorageID:          ins1,
			Filename:           ins1,
			Title:              ins1,
			Source:             "apps",
			Version:            "2",
			PreInstallQuery:    "select 1 from bar;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example2.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
	})

	require.NoError(t, err)
	softwareInstallers, err := ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 2)
	require.NotNil(t, softwareInstallers[0].TitleID)
	require.NotNil(t, softwareInstallers[0].TeamID)
	require.Equal(t, team.ID, *softwareInstallers[0].TeamID)
	require.Equal(t, "https://example.com", softwareInstallers[0].URL)
	require.NotNil(t, softwareInstallers[1].TitleID)
	require.NotNil(t, softwareInstallers[1].TeamID)
	require.Equal(t, team.ID, *softwareInstallers[1].TeamID)
	require.Equal(t, "https://example2.com", softwareInstallers[1].URL)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", ExtensionFor: ""},
		{Name: ins1, Source: "apps", ExtensionFor: ""},
	})

	// Add setup_experience_status_results for both installers
	_, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", host1.UUID, *host1.TeamID)
	require.NoError(t, err)

	statuses, err := ds.ListSetupExperienceResultsByHostUUID(ctx, host1.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	// Enqueue the actual install requests
	for _, status := range statuses {
		execID, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, *status.SoftwareInstallerID, fleet.HostSoftwareInstallOptions{ForSetupExperience: true})
		require.NoError(t, err)
		status.HostSoftwareInstallsExecutionID = &execID
		status.Status = fleet.SetupExperienceStatusRunning
		err = ds.UpdateSetupExperienceStatusResult(ctx, status)
		require.NoError(t, err)
	}

	// batch-set without changes
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:      "install",
			PostInstallScript:  "post-install",
			InstallerFile:      tfr1,
			StorageID:          ins1,
			Filename:           ins1,
			Title:              ins1,
			Source:             "apps",
			Version:            "2",
			PreInstallQuery:    "select 1 from bar;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example2.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	statuses, err = ds.ListSetupExperienceResultsByHostUUID(ctx, host1.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	for _, status := range statuses {
		require.Equal(t, fleet.SetupExperienceStatusRunning, status.Status)
	}

	// batch-set ins0's install script. ins0's in-flight SE install must not be cancelled.
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install2",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:      "install",
			PostInstallScript:  "post-install",
			InstallerFile:      tfr1,
			StorageID:          ins1,
			Filename:           ins1,
			Title:              ins1,
			Source:             "apps",
			Version:            "2",
			PreInstallQuery:    "select 1 from bar;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example2.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
	})

	require.NoError(t, err)

	statuses, err = ds.ListSetupExperienceResultsByHostUUID(ctx, host1.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	// both SE rows must stay running: ins0 was the edited installer, ins1 was unrelated
	ins1ExecID := ""
	ins0Found := false
	ins1Found := false
	for _, status := range statuses {
		if status.Name == ins0 {
			assert.False(t, ins0Found, "duplicate ins0 found")
			ins0Found = true
			require.Equal(t, fleet.SetupExperienceStatusRunning, status.Status)
		} else {
			assert.False(t, ins1Found, "duplicate ins1 found")
			assert.Equal(t, ins1, status.Name)
			require.Equal(t, fleet.SetupExperienceStatusRunning, status.Status)
			require.NotNil(t, status.HostSoftwareInstallsExecutionID)
			ins1ExecID = *status.HostSoftwareInstallsExecutionID
		}
	}

	// activate and set a result for ins1 as if the install completed
	ds.testActivateSpecificNextActivities = []string{ins1ExecID}
	_, err = ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host1.ID, "")
	require.NoError(t, err)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host1.ID,
		InstallUUID:           ins1ExecID,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)

	require.NoError(t, err)

	// batch-set change ins1's install script to update it. This should do nothing to the setup
	// experience result because the install already completed
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install2",
			InstallerFile:      tfr0,
			StorageID:          ins0,
			Filename:           ins0,
			Title:              ins0,
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:      "install3",
			PostInstallScript:  "post-install",
			InstallerFile:      tfr1,
			StorageID:          ins1,
			Filename:           ins1,
			Title:              ins1,
			Source:             "apps",
			Version:            "2",
			PreInstallQuery:    "select 1 from bar;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example2.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	statuses, err = ds.ListSetupExperienceResultsByHostUUID(ctx, host1.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	// both SE rows still running. ins1 is unchanged because SetHostSoftwareInstallResult doesn't advance the SE row.
	ins0Found = false
	ins1Found = false
	for _, status := range statuses {
		if status.Name == ins0 {
			assert.False(t, ins0Found, "duplicate ins0 found")
			ins0Found = true
			require.Equal(t, fleet.SetupExperienceStatusRunning, status.Status)
		} else {
			assert.False(t, ins1Found, "duplicate ins1 found")
			assert.Equal(t, ins1, status.Name)
			require.Equal(t, fleet.SetupExperienceStatusRunning, status.Status)
		}
	}
}

func testGetSoftwareInstallersPendingDeletion(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	newTFR := func(content string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(bytes.NewReader([]byte(content)), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	// Three installers: a macOS package with a bundle identifier and a display
	// name override, a Windows package matched by name (no bundle identifier),
	// and an FMA-backed package.
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:    "install",
			InstallerFile:    newTFR("installer0"),
			StorageID:        "installer0",
			Filename:         "installer0",
			Title:            "ins0",
			Source:           "apps",
			Version:          "1",
			UserID:           user1.ID,
			Platform:         "darwin",
			URL:              "https://example.com/ins0",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: "com.example.ins0",
			DisplayName:      "Cool App",
		},
		{
			InstallScript:   "install",
			UninstallScript: "uninstall",
			InstallerFile:   newTFR("installer1"),
			StorageID:       "installer1",
			Filename:        "installer1",
			Title:           "ins1",
			Source:          "programs",
			Version:         "2",
			UserID:          user1.ID,
			Platform:        "windows",
			URL:             "https://example.com/ins1",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:        "install",
			InstallerFile:        newTFR("installer2"),
			StorageID:            "installer2",
			Filename:             "installer2",
			Title:                "Maintained1",
			Source:               "apps",
			Version:              "3",
			UserID:               user1.ID,
			Platform:             "darwin",
			URL:                  "https://example.com/maintained1",
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			BundleIdentifier:     "fleet.maintained1",
			FleetMaintainedAppID: new(maintainedApp.ID),
		},
	})
	require.NoError(t, err)

	displayNames := func(pkgs []fleet.DeletedSoftwarePackage) []string {
		names := make([]string, 0, len(pkgs))
		for _, p := range pkgs {
			require.NotNil(t, p.TeamID)
			require.Equal(t, team.ID, *p.TeamID)
			require.NotZero(t, p.TitleID)
			names = append(names, p.DisplayName)
		}
		return names
	}

	// empty incoming: everything is pending deletion; display-name override
	// used for ins0, title-name fallback for the others; FMA row included.
	deleted, err := ds.GetSoftwareInstallersPendingDeletion(ctx, &team.ID, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"Cool App", "ins1", "Maintained1"}, displayNames(deleted))

	// bundle-identifier match excludes ins0.
	deleted, err = ds.GetSoftwareInstallersPendingDeletion(ctx, &team.ID, []fleet.SoftwareTitleIdentifier{
		{UniqueIdentifier: "com.example.ins0", Source: "apps"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"ins1", "Maintained1"}, displayNames(deleted))

	// name match (no bundle identifier) excludes ins1.
	deleted, err = ds.GetSoftwareInstallersPendingDeletion(ctx, &team.ID, []fleet.SoftwareTitleIdentifier{
		{UniqueIdentifier: "ins1", Source: "programs"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Cool App", "Maintained1"}, displayNames(deleted))

	// source must match too: same unique identifier, wrong source.
	deleted, err = ds.GetSoftwareInstallersPendingDeletion(ctx, &team.ID, []fleet.SoftwareTitleIdentifier{
		{UniqueIdentifier: "com.example.ins0", Source: "programs"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Cool App", "ins1", "Maintained1"}, displayNames(deleted))

	// all matched: nothing pending deletion.
	deleted, err = ds.GetSoftwareInstallersPendingDeletion(ctx, &team.ID, []fleet.SoftwareTitleIdentifier{
		{UniqueIdentifier: "com.example.ins0", Source: "apps"},
		{UniqueIdentifier: "ins1", Source: "programs"},
		{UniqueIdentifier: "fleet.maintained1", Source: "apps"},
	})
	require.NoError(t, err)
	require.Empty(t, deleted)

	// other teams are not affected: no-team has no installers.
	deleted, err = ds.GetSoftwareInstallersPendingDeletion(ctx, nil, nil)
	require.NoError(t, err)
	require.Empty(t, deleted)

	// prediction matches reality: batch-set keeping only ins0 deletes exactly
	// what was predicted.
	predicted, err := ds.GetSoftwareInstallersPendingDeletion(ctx, &team.ID, []fleet.SoftwareTitleIdentifier{
		{UniqueIdentifier: "com.example.ins0", Source: "apps"},
	})
	require.NoError(t, err)
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:    "install",
			InstallerFile:    newTFR("installer0"),
			StorageID:        "installer0",
			Filename:         "installer0",
			Title:            "ins0",
			Source:           "apps",
			Version:          "1",
			UserID:           user1.ID,
			Platform:         "darwin",
			URL:              "https://example.com/ins0",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			BundleIdentifier: "com.example.ins0",
		},
	})
	require.NoError(t, err)
	remaining, err := ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	remainingTitleIDs := map[uint]struct{}{*remaining[0].TitleID: {}}
	for _, p := range predicted {
		_, stillThere := remainingTitleIDs[p.TitleID]
		require.False(t, stillThere, "predicted-deleted title %d (%s) survived the batch set", p.TitleID, p.DisplayName)
	}
	require.Len(t, predicted, 2)
}

func testGetSoftwareInstallerMetadataByTeamAndTitleID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:             "foo",
		Source:            "bar",
		InstallScript:     "echo install",
		PostInstallScript: "echo post-install",
		PreInstallQuery:   "SELECT 1",
		TeamID:            &team.ID,
		Filename:          "foo.pkg",
		Platform:          "darwin",
		UserID:            user1.ID,
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	installerMeta, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)
	require.Equal(t, "darwin", installerMeta.Platform)

	metaByTeamAndTitle, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *installerMeta.TitleID, true)
	require.NoError(t, err)
	require.Equal(t, "echo install", metaByTeamAndTitle.InstallScript)
	require.Equal(t, "echo post-install", metaByTeamAndTitle.PostInstallScript)
	require.EqualValues(t, installerID, metaByTeamAndTitle.InstallerID)
	require.Equal(t, "SELECT 1", metaByTeamAndTitle.PreInstallQuery)

	installerID, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "bar",
		Source:          "bar",
		InstallScript:   "echo install",
		TeamID:          &team.ID,
		Filename:        "foo.pkg",
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	installerMeta, err = ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)

	metaByTeamAndTitle, err = ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, *installerMeta.TitleID, true)
	require.NoError(t, err)
	require.Equal(t, "echo install", metaByTeamAndTitle.InstallScript)
	require.Equal(t, "", metaByTeamAndTitle.PostInstallScript)
	require.EqualValues(t, installerID, metaByTeamAndTitle.InstallerID)
	require.Equal(t, "", metaByTeamAndTitle.PreInstallQuery)
}

func testHasSelfServiceSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	test.CreateInsertGlobalVPPToken(t, ds)

	const platform = "linux"
	// No installers
	hasSelfService, err := ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.False(t, hasSelfService)

	// Create a non-self service installer
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "foo",
		Source:          "bar",
		InstallScript:   "echo install",
		TeamID:          &team.ID,
		Filename:        "foo.pkg",
		Platform:        platform,
		SelfService:     false,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.False(t, hasSelfService)

	// Create a self-service installer for team
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "foo2",
		Source:          "bar2",
		InstallScript:   "echo install",
		TeamID:          &team.ID,
		Filename:        "foo2.pkg",
		Platform:        platform,
		SelfService:     true,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a non self-service VPP for global/linux (not truly possible as VPP is Apple but for testing)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: platform}}, Name: "vpp1", BundleIdentifier: "com.app.vpp1"}, nil)
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a self-service VPP for global/linux (not truly possible as VPP is Apple but for testing)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_2", Platform: platform}, SelfService: true}, Name: "vpp2", BundleIdentifier: "com.app.vpp2"}, nil)
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, nil)
	require.NoError(t, err)
	assert.True(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, platform, &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a global self-service installer
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "foo global",
		Source:          "bar",
		InstallScript:   "echo install",
		TeamID:          nil,
		Filename:        "foo global.pkg",
		Platform:        platform,
		SelfService:     true,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "ubuntu", nil)
	require.NoError(t, err)
	assert.True(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "ubuntu", &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)

	// Create a new team for .sh testing
	teamSh, err := ds.NewTeam(ctx, &fleet.Team{Name: "team sh darwin test"})
	require.NoError(t, err)

	// Initially, darwin should not see any self-service installers in this team
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "darwin", &teamSh.ID)
	require.NoError(t, err)
	assert.False(t, hasSelfService, "darwin should not see self-service before .sh is created")

	// Create a self-service .sh installer (stored as platform='linux', extension='sh')
	// This should be visible to darwin hosts due to the .sh exception
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "sh script for darwin",
		Source:          "sh_packages",
		InstallScript:   "#!/bin/bash\necho install",
		TeamID:          &teamSh.ID,
		Filename:        "script.sh",
		Platform:        "linux", // .sh files are stored as linux
		Extension:       "sh",
		SelfService:     true,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Darwin host should now see self-service .sh package
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "darwin", &teamSh.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService, "darwin host should see self-service .sh packages")

	// Linux host should also see it
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "linux", &teamSh.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService, "linux host should see self-service .sh packages")

	// Windows host shouldn't see .sh packages
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "windows", &teamSh.ID)
	require.NoError(t, err)
	assert.False(t, hasSelfService, "windows host should NOT see .sh packages")

	// Create a self-service VPP for team/darwin
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_3", Platform: fleet.MacOSPlatform}, SelfService: true}, Name: "vpp3", BundleIdentifier: "com.app.vpp3"}, &team.ID)
	require.NoError(t, err)
	// Check darwin
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "darwin", nil)
	require.NoError(t, err)
	assert.False(t, hasSelfService)
	hasSelfService, err = ds.HasSelfServiceSoftwareInstallers(ctx, "darwin", &team.ID)
	require.NoError(t, err)
	assert.True(t, hasSelfService)
}

func testDeleteSoftwareInstallers(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	dir := t.TempDir()
	store, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// put an installer and save it in the DB
	ins0 := "installer.pkg"
	ins0File := bytes.NewReader([]byte("installer0"))
	err = store.Put(ctx, ins0, ins0File)
	require.NoError(t, err)
	_, _ = ins0File.Seek(0, 0)
	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	softwareInstallerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer.pkg",
		Title:           "ins0",
		Source:          "apps",
		Platform:        "darwin",
		TeamID:          &team1.ID,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	p1, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:                "p1",
		Query:               "SELECT 1;",
		SoftwareInstallerID: &softwareInstallerID,
	})
	require.NoError(t, err)

	err = ds.DeleteSoftwareInstaller(ctx, softwareInstallerID)
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerWithAssociatedInstallPolicy)

	_, err = ds.DeleteTeamPolicies(ctx, team1.ID, []uint{p1.ID})
	require.NoError(t, err)

	// mark the installer as "installed during setup", which prevents deletion
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET install_during_setup = 1 WHERE id = ?`, softwareInstallerID)
		return err
	})

	err = ds.DeleteSoftwareInstaller(ctx, softwareInstallerID)
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerInstalledDuringSetup)

	// clear "installed during setup", which allows deletion
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET install_during_setup = 0 WHERE id = ?`, softwareInstallerID)
		return err
	})

	err = ds.DeleteSoftwareInstaller(ctx, softwareInstallerID)
	require.NoError(t, err)

	// deleting again returns an error, no such installer
	err = ds.DeleteSoftwareInstaller(ctx, softwareInstallerID)
	var nfe *common_mysql.NotFoundError
	require.ErrorAs(t, err, &nfe)
}

func testDeletePendingSoftwareInstallsForPolicy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	dir := t.TempDir()
	store, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(t, err)
	ins0 := "installer.pkg"
	ins0File := bytes.NewReader([]byte("installer0"))
	err = store.Put(ctx, ins0, ins0File)
	require.NoError(t, err)
	_, _ = ins0File.Seek(0, 0)

	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)

	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer.pkg",
		Title:           "ins0",
		Source:          "apps",
		Platform:        "darwin",
		TeamID:          &team1.ID,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	policy1, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:                "p1",
		Query:               "SELECT 1;",
		SoftwareInstallerID: &installerID1,
	})
	require.NoError(t, err)

	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer.pkg",
		Title:           "ins1",
		Source:          "apps",
		Platform:        "darwin",
		TeamID:          &team1.ID,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	policy2, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:                "p2",
		Query:               "SELECT 2;",
		SoftwareInstallerID: &installerID2,
	})
	require.NoError(t, err)

	const hostSoftwareInstallsCount = "SELECT count(1) FROM host_software_installs WHERE status = ? and execution_id = ?"
	var count int

	// install for correct policy & correct status
	executionID, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID1, fleet.HostSoftwareInstallOptions{PolicyID: &policy1.ID})
	require.NoError(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, hostSoftwareInstallsCount, fleet.SoftwareInstallPending, executionID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = ds.deletePendingSoftwareInstallsForPolicy(ctx, &team1.ID, policy1.ID)
	require.NoError(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, hostSoftwareInstallsCount, fleet.SoftwareInstallPending, executionID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// install for different policy & correct status
	executionID, err = ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID2, fleet.HostSoftwareInstallOptions{PolicyID: &policy2.ID})
	require.NoError(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, hostSoftwareInstallsCount, fleet.SoftwareInstallPending, executionID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = ds.deletePendingSoftwareInstallsForPolicy(ctx, &team1.ID, policy1.ID)
	require.NoError(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, hostSoftwareInstallsCount, fleet.SoftwareInstallPending, executionID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// install for correct policy & incorrect status
	executionID, err = ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID1, fleet.HostSoftwareInstallOptions{PolicyID: &policy1.ID})
	require.NoError(t, err)

	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           executionID,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	err = ds.deletePendingSoftwareInstallsForPolicy(ctx, &team1.ID, policy1.ID)
	require.NoError(t, err)

	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT count(1) FROM host_software_installs WHERE execution_id = ?`, executionID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func testGetHostLastInstallData(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now(), test.WithTeamID(team1.ID))
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now(), test.WithTeamID(team1.ID))

	dir := t.TempDir()
	store, err := filesystem.NewSoftwareInstallerStore(dir)
	require.NoError(t, err)

	// put an installer and save it in the DB
	ins0 := "installer.pkg"
	ins0File := bytes.NewReader([]byte("installer0"))
	err = store.Put(ctx, ins0, ins0File)
	require.NoError(t, err)
	_, _ = ins0File.Seek(0, 0)
	tfr0, err := fleet.NewTempFileReader(ins0File, t.TempDir)
	require.NoError(t, err)

	softwareInstallerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer.pkg",
		Title:           "ins1",
		Source:          "apps",
		Platform:        "darwin",
		TeamID:          &team1.ID,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	softwareInstallerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install2",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer2.pkg",
		Title:           "ins2",
		Source:          "apps",
		Platform:        "darwin",
		TeamID:          &team1.ID,
		UserID:          user1.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// No installations on host1 yet.
	host1LastInstall, err := ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Install installer.pkg on host1.
	installUUID1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, softwareInstallerID1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, installUUID1)

	// Last installation should be pending.
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID1, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// Set result of last installation.
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:      host1.ID,
		InstallUUID: installUUID1,

		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	// Last installation should be "installed".
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID1, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstalled, *host1LastInstall.Status)

	// Install installer2.pkg on host1.
	installUUID2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, softwareInstallerID2, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, installUUID2)

	// Last installation for installer1.pkg should be "installed".
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID1, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstalled, *host1LastInstall.Status)
	// Last installation for installer2.pkg should be "pending".
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID2)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID2, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// Perform another installation of installer1.pkg.
	installUUID3, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, softwareInstallerID1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, installUUID3)

	// Last installation for installer1.pkg should be "pending" again.
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID3, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// Set result of last installer1.pkg installation, but first we need to set a
	// result for installUUID2 so that this last installer1.pkg request is
	// activated.
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:      host1.ID,
		InstallUUID: installUUID2,

		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:      host1.ID,
		InstallUUID: installUUID3,

		InstallScriptExitCode: ptr.Int(1),
	}, nil)
	require.NoError(t, err)

	// Last installation for installer1.pkg should be "failed".
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID3, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *host1LastInstall.Status)

	// No installations on host2.
	host2LastInstall, err := ds.GetHostLastInstallData(ctx, host2.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.Nil(t, host2LastInstall)
	host2LastInstall, err = ds.GetHostLastInstallData(ctx, host2.ID, softwareInstallerID2)
	require.NoError(t, err)
	require.Nil(t, host2LastInstall)
}

func testGetOrGenerateSoftwareInstallerTitleID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "", "host3key", "host3uuid", time.Now())

	software1 := []fleet.Software{
		{Name: "Existing Title", Version: "0.0.1", Source: "apps", BundleIdentifier: "existing.title"},
	}
	software2 := []fleet.Software{
		{Name: "Existing Title", Version: "v0.0.2", Source: "apps", BundleIdentifier: "existing.title"},
		{Name: "Existing Title", Version: "0.0.3", Source: "apps", BundleIdentifier: "existing.title"},
		{Name: "Existing Title Without Bundle", Version: "0.0.3", Source: "apps"},
		{Name: "FMA Old Name", Version: "1.0", Source: "apps", BundleIdentifier: "com.fma"},
	}
	software3 := []fleet.Software{
		{Name: "Win Title 1", Version: "11.0", Source: "programs", UpgradeCode: ptr.String("")},
		{Name: "Win Title 2", Version: "11.0", Source: "programs", UpgradeCode: ptr.String("CODEEXISTS")},
		{Name: "Win Title 3", Version: "11.0", Source: "programs", UpgradeCode: ptr.String("")},
		{Name: "Win Title 4", Version: "11.0", Source: "programs", UpgradeCode: ptr.String("12345")},
		{Name: "Win Title 5", Version: "11.0", Source: "programs", UpgradeCode: ptr.String("ABCDEF")},
		{Name: "Win Title 6", Version: "11.0", Source: "programs", UpgradeCode: ptr.String("GHIJKL")},
	}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host3.ID, software3)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	tests := []struct {
		name                string
		payload             *fleet.UploadSoftwareInstallerPayload
		expectedName        string
		expectedSource      string
		expectedUpgradeCode *string
	}{
		{
			name: "title that already exists, no bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "Existing Title",
				Source: "apps",
			},
			expectedSource: "apps",
		},
		{
			name: "title that already exists, mismatched bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "Existing Title",
				Source:           "apps",
				BundleIdentifier: "com.existing.bundle",
			},
			expectedSource: "apps",
		},
		{
			name: "title that already exists but doesn't have a bundle identifier",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "Existing Title Without Bundle",
				Source: "apps",
			},
			expectedSource: "apps",
		},
		{
			name: "title that already exists, no bundle identifier in DB, bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "Existing Title Without Bundle",
				Source:           "apps",
				BundleIdentifier: "com.new.bundleid",
			},
			expectedSource: "apps",
		},
		{
			name: "title that doesn't exist, no bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "New Title",
				Source: "some_source",
			},
			expectedSource: "some_source",
		},
		{
			name: "title that doesn't exist, with bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "New Title With Bundle",
				Source:           "some_source",
				BundleIdentifier: "com.new.bundle",
			},
			expectedSource: "some_source",
		},
		{
			name: "title that already exists with bundle identifier",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "Existing Title",
				Source:           "apps",
				BundleIdentifier: "existing.title",
			},
			expectedSource: "apps",
		},
		{
			name: "title that already exists with bundle identifier, different source",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "Existing Title",
				Source:           "ios_apps",
				BundleIdentifier: "existing.title",
			},
			expectedSource: "ios_apps",
		},
		{
			name: "don't rename macos FMA titles",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:                "FMA New Name",
				Source:               "apps",
				BundleIdentifier:     "com.fma",
				FleetMaintainedAppID: ptr.Uint(2),
			},
			expectedName:   "FMA Old Name",
			expectedSource: "apps",
		},
		{
			name: "installer: no upgrade code, existing title: same name, no upgrade code",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "Win Title 1",
				Source: "programs",
			},
			expectedName:        "Win Title 1",
			expectedSource:      "programs",
			expectedUpgradeCode: ptr.String(""),
		},
		{
			name: "installer: no upgrade code, existing title: same name, has upgrade code",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "Win Title 2",
				Source: "programs",
			},
			expectedName:        "Win Title 2",
			expectedSource:      "programs",
			expectedUpgradeCode: ptr.String("CODEEXISTS"),
		},
		{
			name: "installer: has upgrade code, existing title: same name, no upgrade code",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:       "Win Title 3",
				Source:      "programs",
				UpgradeCode: "NEWCODE",
			},
			expectedName:        "Win Title 3",
			expectedSource:      "programs",
			expectedUpgradeCode: ptr.String("NEWCODE"),
		},
		{
			name: "installer: has upgrade code, existing title: same name, different upgrade code",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:       "Win Title 4",
				Source:      "programs",
				UpgradeCode: "DIFFERENTCODE",
			},
			expectedName:        "Win Title 4",
			expectedSource:      "programs",
			expectedUpgradeCode: ptr.String("DIFFERENTCODE"), // should make a new title
		},
		{
			name: "installer: has upgrade code, existing title: same name, same upgrade code",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:       "Win Title 5",
				Source:      "programs",
				UpgradeCode: "ABCDEF",
			},
			expectedName:        "Win Title 5",
			expectedSource:      "programs",
			expectedUpgradeCode: ptr.String("ABCDEF"),
		},
		{
			name: "installer: has upgrade code and FMA, existing title: different name, same upgrade code",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:                "New Name",
				Source:               "programs",
				UpgradeCode:          "GHIJKL",
				FleetMaintainedAppID: ptr.Uint(1), // FMAs should overwrite name for upgrade code
			},
			expectedName:        "New Name",
			expectedSource:      "programs",
			expectedUpgradeCode: ptr.String("GHIJKL"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ds.getOrGenerateSoftwareInstallerTitleID(ctx, ds.writer(ctx), tt.payload)
			require.NoError(t, err)
			require.NotEmpty(t, id)

			var actual struct {
				Name        string  `db:"name"`
				Source      string  `db:"source"`
				UpgradeCode *string `db:"upgrade_code"`
			}
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				err := sqlx.GetContext(ctx, q, &actual, `SELECT name, source, upgrade_code FROM software_titles WHERE id = ?`, id)
				require.NoError(t, err)
				return nil
			})
			if tt.expectedName != "" {
				require.Equal(t, tt.expectedName, actual.Name)
			}
			require.Equal(t, tt.expectedSource, actual.Source)
			require.Equal(t, tt.expectedUpgradeCode, actual.UpgradeCode)
		})
	}
}

func testBatchSetSoftwareInstallersScopedViaLabels(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a host to have a pending install request
	host := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())

	// create a couple teams and a user
	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "2"})
	require.NoError(t, err)
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// create some installer payloads to be used by test cases
	installers := make([]*fleet.UploadSoftwareInstallerPayload, 3)
	for i := range installers {
		file := bytes.NewReader([]byte("installer" + fmt.Sprint(i)))
		tfr, err := fleet.NewTempFileReader(file, t.TempDir)
		require.NoError(t, err)
		installers[i] = &fleet.UploadSoftwareInstallerPayload{
			InstallScript:   "install",
			InstallerFile:   tfr,
			StorageID:       "installer" + fmt.Sprint(i),
			Filename:        "installer" + fmt.Sprint(i),
			Title:           "ins" + fmt.Sprint(i),
			Source:          "apps",
			Version:         "1",
			PreInstallQuery: "foo",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com",
		}
	}

	// create some labels to be used by test cases
	labels := make([]*fleet.Label, 4)
	for i := range labels {
		lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "label" + fmt.Sprint(i)})
		require.NoError(t, err)
		labels[i] = lbl
	}

	type testPayload struct {
		Installer           *fleet.UploadSoftwareInstallerPayload
		Labels              []*fleet.Label
		Exclude             bool
		ShouldCancelPending *bool // nil if the installer is new (could not have pending), otherwise true/false if it was edited
	}

	// test scenarios - note that subtests must NOT be used as the sequence of
	// tests matters - they cannot be run in isolation.
	cases := []struct {
		desc    string
		team    *fleet.Team
		payload []testPayload
	}{
		{
			desc:    "empty payload",
			payload: nil,
		},
		{
			desc: "no team, installer0, no label",
			payload: []testPayload{
				{Installer: installers[0]},
			},
		},
		{
			desc: "team 1, installer0, include label0",
			team: tm1,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[0]}},
			},
		},
		{
			desc: "no team, installer0 no change, add installer1 with exclude label1",
			payload: []testPayload{
				{Installer: installers[0], ShouldCancelPending: ptr.Bool(false)},
				{Installer: installers[1], Labels: []*fleet.Label{labels[1]}, Exclude: true},
			},
		},
		{
			desc: "no team, installer0 no change, installer1 change to include label1",
			payload: []testPayload{
				{Installer: installers[0], ShouldCancelPending: ptr.Bool(false)},
				{Installer: installers[1], Labels: []*fleet.Label{labels[1]}, Exclude: false, ShouldCancelPending: ptr.Bool(true)},
			},
		},
		{
			desc: "team 1, installer0, include label0 and add label1",
			team: tm1,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[0], labels[1]}, ShouldCancelPending: ptr.Bool(true)},
			},
		},
		{
			desc: "team 1, installer0, remove label0 and keep label1",
			team: tm1,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[1]}, ShouldCancelPending: ptr.Bool(true)},
			},
		},
		{
			desc: "team 1, installer0, switch to label0 and label2",
			team: tm1,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[0], labels[2]}, ShouldCancelPending: ptr.Bool(true)},
			},
		},
		{
			desc: "team 2, 3 installers, mix of labels",
			team: tm2,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[0]}, Exclude: false},
				{Installer: installers[1], Labels: []*fleet.Label{labels[0], labels[1], labels[2]}, Exclude: true},
				{Installer: installers[2], Labels: []*fleet.Label{labels[1], labels[2]}, Exclude: false},
			},
		},
		{
			desc: "team 1, installer0 no change and add installer2",
			team: tm1,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[0], labels[2]}, ShouldCancelPending: ptr.Bool(false)},
				{Installer: installers[2]},
			},
		},
		{
			desc: "team 1, installer0 switch to labels 1 and 3, installer2 no change",
			team: tm1,
			payload: []testPayload{
				{Installer: installers[0], Labels: []*fleet.Label{labels[1], labels[3]}, ShouldCancelPending: ptr.Bool(true)},
				{Installer: installers[2], ShouldCancelPending: ptr.Bool(false)},
			},
		},
		{
			desc: "team 2, remove installer0, labels of install1 and no change installer2",
			team: tm2,
			payload: []testPayload{
				{Installer: installers[1], ShouldCancelPending: ptr.Bool(true)},
				{Installer: installers[2], Labels: []*fleet.Label{labels[1], labels[2]}, Exclude: false, ShouldCancelPending: ptr.Bool(false)},
			},
		},
		{
			desc:    "no team, remove all",
			payload: []testPayload{},
		},
	}
	for _, c := range cases {
		t.Log("Running test case ", c.desc)

		var teamID *uint
		var globalOrTeamID uint
		if c.team != nil {
			teamID = &c.team.ID
			globalOrTeamID = c.team.ID
		}

		// cleanup any existing install requests for the host
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM host_software_installs WHERE host_id = ?`, host.ID)
			return err
		})

		installerIDs := make([]uint, len(c.payload))
		if len(c.payload) > 0 {
			// create pending install requests for each updated installer, to see if
			// it cancels it or not as expected.
			err := ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(teamID, []uint{host.ID}))
			require.NoError(t, err)
			for i, payload := range c.payload {
				if payload.ShouldCancelPending != nil {
					// the installer must exist
					var swID uint
					ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
						err := sqlx.GetContext(ctx, q, &swID, `SELECT id FROM software_installers WHERE global_or_team_id = ?
						AND title_id IN (SELECT id FROM software_titles WHERE name = ? AND source = ? AND extension_for = '')`,
							globalOrTeamID, payload.Installer.Title, payload.Installer.Source)
						return err
					})
					_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, swID, fleet.HostSoftwareInstallOptions{})
					require.NoError(t, err)
					installerIDs[i] = swID
				}
			}
		}

		// create the payload by copying the test one, so that the original installers
		// structs are not modified
		payload := make([]*fleet.UploadSoftwareInstallerPayload, len(c.payload))
		for i, p := range c.payload {
			installer := *p.Installer
			installer.ValidatedLabels = &fleet.LabelIdentsWithScope{LabelScope: fleet.LabelScopeIncludeAny}
			if p.Exclude {
				installer.ValidatedLabels.LabelScope = fleet.LabelScopeExcludeAny
			}
			byName := make(map[string]fleet.LabelIdent, len(p.Labels))
			for _, lbl := range p.Labels {
				byName[lbl.Name] = fleet.LabelIdent{LabelName: lbl.Name, LabelID: lbl.ID}
			}
			installer.ValidatedLabels.ByName = byName
			payload[i] = &installer
		}

		err := ds.BatchSetSoftwareInstallers(ctx, teamID, payload)
		require.NoError(t, err)
		installers, err := ds.GetSoftwareInstallers(ctx, globalOrTeamID)
		require.NoError(t, err)
		require.Len(t, installers, len(c.payload))

		// get the metadata for each installer to assert the batch did set the
		// expected ones.
		installersByFilename := make(map[string]*fleet.SoftwareInstaller, len(installers))
		for _, ins := range installers {
			meta, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, *ins.TitleID, false)
			require.NoError(t, err)
			installersByFilename[meta.Name] = meta
		}

		// validate that the inserted software is as expected
		for i, payload := range c.payload {
			meta, ok := installersByFilename[payload.Installer.Filename]
			require.True(t, ok, "installer %s was not created", payload.Installer.Filename)
			require.Equal(t, meta.SoftwareTitle, payload.Installer.Title)

			wantLabelIDs := make([]uint, len(payload.Labels))
			for j, lbl := range payload.Labels {
				wantLabelIDs[j] = lbl.ID
			}
			if payload.Exclude {
				require.Empty(t, meta.LabelsIncludeAny)
				gotLabelIDs := make([]uint, len(meta.LabelsExcludeAny))
				for i, lbl := range meta.LabelsExcludeAny {
					gotLabelIDs[i] = lbl.LabelID
				}
				require.ElementsMatch(t, wantLabelIDs, gotLabelIDs)
			} else {
				require.Empty(t, meta.LabelsExcludeAny)
				gotLabelIDs := make([]uint, len(meta.LabelsIncludeAny))
				for j, lbl := range meta.LabelsIncludeAny {
					gotLabelIDs[j] = lbl.LabelID
				}
				require.ElementsMatch(t, wantLabelIDs, gotLabelIDs)
			}

			// check if it deleted pending installs or not
			if payload.ShouldCancelPending != nil {
				lastInstall, err := ds.GetHostLastInstallData(ctx, host.ID, installerIDs[i])
				require.NoError(t, err)
				if *payload.ShouldCancelPending {
					require.Nil(t, lastInstall, "should have cancelled pending installs")
				} else {
					require.NotNil(t, lastInstall, "should not have cancelled pending installs")
				}
			}
		}
	}
}

func testMatchOrCreateSoftwareInstallerWithAutomaticPolicies(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)

	// Test pkg without automatic install doesn't create policy.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		BundleIdentifier: "com.manual.foobar",
		Extension:        "pkg",
		StorageID:        "storage0",
		Filename:         "foobar0",
		Title:            "Manual foobar",
		Version:          "1.0",
		Source:           "apps",
		UserID:           user1.ID,
		TeamID:           &team1.ID,
		AutomaticInstall: false,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team1Policies, _, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Empty(t, team1Policies)

	// Test pkg.
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		BundleIdentifier: "com.foo.bar",
		Extension:        "pkg",
		StorageID:        "storage1",
		Filename:         "foobar1",
		Title:            "Foobar",
		Version:          "1.0",
		Source:           "apps",
		UserID:           user1.ID,
		TeamID:           &team1.ID,
		AutomaticInstall: true,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team1Policies, _, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, team1Policies, 1)
	require.Equal(t, "[Install software] Foobar (pkg)", team1Policies[0].Name)
	require.Equal(t, "SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo.bar';", team1Policies[0].Query)
	require.Equal(t, "Policy triggers automatic install of Foobar on each host that's missing this software.", team1Policies[0].Description)
	require.Equal(t, "darwin", team1Policies[0].Platform)
	require.NotNil(t, team1Policies[0].SoftwareInstallerID)
	require.Equal(t, installerID1, *team1Policies[0].SoftwareInstallerID)
	require.NotNil(t, team1Policies[0].TeamID)
	require.Equal(t, team1.ID, *team1Policies[0].TeamID)

	// Test Mac FMA
	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{ID: 1})
	require.NoError(t, err)
	installerFMA, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:         tfr1,
		BundleIdentifier:      "com.foo.fma",
		Platform:              "darwin",
		Extension:             "dmg",
		FleetMaintainedAppID:  ptr.Uint(fma.ID),
		StorageID:             "storage1",
		Filename:              "foobar1",
		Title:                 "FooFMA",
		Version:               "1.0",
		Source:                "apps",
		UserID:                user1.ID,
		TeamID:                &team1.ID,
		AutomaticInstall:      true,
		AutomaticInstallQuery: "SELECT 1 FROM osquery_info",
		ValidatedLabels:       &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team1Policies, _, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, team1Policies, 2)
	require.Equal(t, "[Install software] FooFMA", team1Policies[1].Name)
	require.Equal(t, "SELECT 1 FROM osquery_info", team1Policies[1].Query)
	require.Equal(t, "Policy triggers automatic install of FooFMA on each host that's missing this software.", team1Policies[1].Description)
	require.Equal(t, "darwin", team1Policies[1].Platform)
	require.NotNil(t, team1Policies[1].SoftwareInstallerID)
	require.Equal(t, installerFMA, *team1Policies[1].SoftwareInstallerID)
	require.NotNil(t, team1Policies[1].TeamID)
	require.Equal(t, team1.ID, *team1Policies[1].TeamID)

	// Test msi.
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		Extension:        "msi",
		StorageID:        "storage2",
		Filename:         "zoobar1",
		Title:            "Zoobar",
		Version:          "1.0",
		Source:           "programs",
		UserID:           user1.ID,
		TeamID:           nil,
		AutomaticInstall: true,
		PackageIDs:       []string{"id1"},
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// check upgrade code handling
	msiPackagesWithNoUpgradeCode, err := ds.GetMSIInstallersWithoutUpgradeCode(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint]string{installerID2: "storage2"}, msiPackagesWithNoUpgradeCode)
	require.NoError(t, ds.UpdateInstallerUpgradeCode(ctx, installerID2, "upgradecode"))
	msiPackagesWithNoUpgradeCode, err = ds.GetMSIInstallersWithoutUpgradeCode(ctx)
	require.NoError(t, err)
	require.Empty(t, msiPackagesWithNoUpgradeCode)
	msiThatShouldHaveUpgradeCode, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID2)
	require.NoError(t, err)
	require.Equal(t, "upgradecode", msiThatShouldHaveUpgradeCode.UpgradeCode)

	noTeamPolicies, _, err := ds.ListTeamPolicies(ctx, fleet.PolicyNoTeamID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, noTeamPolicies, 1)
	require.Equal(t, "[Install software] Zoobar (msi)", noTeamPolicies[0].Name)
	require.Equal(t, "SELECT 1 FROM programs WHERE identifying_number = 'id1';", noTeamPolicies[0].Query)
	require.Equal(t, "Policy triggers automatic install of Zoobar on each host that's missing this software.", noTeamPolicies[0].Description)
	require.Equal(t, "windows", noTeamPolicies[0].Platform)
	require.NotNil(t, noTeamPolicies[0].SoftwareInstallerID)
	require.Equal(t, installerID2, *noTeamPolicies[0].SoftwareInstallerID)
	require.NotNil(t, noTeamPolicies[0].TeamID)
	require.Equal(t, fleet.PolicyNoTeamID, *noTeamPolicies[0].TeamID)

	// Test deb.
	installerID3, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		Extension:        "deb",
		StorageID:        "storage3",
		Filename:         "barfoo1",
		Title:            "Barfoo",
		Version:          "1.0",
		Source:           "deb_packages",
		UserID:           user1.ID,
		TeamID:           &team2.ID,
		AutomaticInstall: true,
		PackageIDs:       []string{"id1"},
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team2Policies, _, err := ds.ListTeamPolicies(ctx, team2.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, team2Policies, 1)
	require.Equal(t, "[Install software] Barfoo (deb)", team2Policies[0].Name)
	require.Equal(t, `SELECT 1 WHERE EXISTS (
	SELECT 1 WHERE (SELECT COUNT(*) FROM deb_packages) = 0
) OR EXISTS (
	SELECT 1 FROM deb_packages WHERE name = 'Barfoo' AND status = 'install ok installed'
);`, team2Policies[0].Query)
	require.Equal(t, `Policy triggers automatic install of Barfoo on each host that's missing this software.
Software won't be installed on Linux hosts with RPM-based distributions because this policy's query is written to always pass on these hosts.`, team2Policies[0].Description)
	require.Equal(t, "linux", team2Policies[0].Platform)
	require.NotNil(t, team2Policies[0].SoftwareInstallerID)
	require.Equal(t, installerID3, *team2Policies[0].SoftwareInstallerID)
	require.NotNil(t, team2Policies[0].TeamID)
	require.Equal(t, team2.ID, *team2Policies[0].TeamID)

	// Test rpm.
	installerID4, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		Extension:        "rpm",
		StorageID:        "storage4",
		Filename:         "barzoo1",
		Title:            "Barzoo",
		Version:          "1.0",
		Source:           "rpm_packages",
		UserID:           user1.ID,
		TeamID:           &team2.ID,
		AutomaticInstall: true,
		PackageIDs:       []string{"id1"},
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team2Policies, _, err = ds.ListTeamPolicies(ctx, team2.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, team2Policies, 2)
	require.Equal(t, "[Install software] Barzoo (rpm)", team2Policies[1].Name)
	require.Equal(t, `SELECT 1 WHERE EXISTS (
	SELECT 1 WHERE (SELECT COUNT(*) FROM rpm_packages) = 0
) OR EXISTS (
	SELECT 1 FROM rpm_packages WHERE name = 'Barzoo'
);`, team2Policies[1].Query)
	require.Equal(t, `Policy triggers automatic install of Barzoo on each host that's missing this software.
Software won't be installed on Linux hosts with Debian-based distributions because this policy's query is written to always pass on these hosts.`, team2Policies[1].Description)
	require.Equal(t, "linux", team2Policies[1].Platform)
	require.NotNil(t, team2Policies[0].SoftwareInstallerID)
	require.Equal(t, installerID4, *team2Policies[1].SoftwareInstallerID)
	require.NotNil(t, team2Policies[1].TeamID)
	require.Equal(t, team2.ID, *team2Policies[1].TeamID)

	_, err = ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "[Install software] OtherFoobar (pkg)",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	// Test pkg and policy with name already exists.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		BundleIdentifier: "com.foo2.bar2",
		Extension:        "pkg",
		StorageID:        "storage5",
		Filename:         "foobar5",
		Title:            "OtherFoobar",
		Version:          "2.0",
		Source:           "apps",
		UserID:           user1.ID,
		TeamID:           &team1.ID,
		AutomaticInstall: true,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team1Policies, _, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, team1Policies, 4)
	require.Equal(t, "[Install software] OtherFoobar (pkg) 2", team1Policies[3].Name)

	team3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 3"})
	require.NoError(t, err)

	_, err = ds.NewTeamPolicy(ctx, team3.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "[Install software] Something2 (msi)",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	_, err = ds.NewTeamPolicy(ctx, team3.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "[Install software] Something2 (msi) 2",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	// This name is on another team, so it shouldn't count.
	_, err = ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "[Install software] Something2 (msi) 3",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	// Test msi and policy with name already exists.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		Extension:        "msi",
		StorageID:        "storage6",
		Filename:         "foobar6",
		Title:            "Something2",
		PackageIDs:       []string{"id2"},
		Version:          "2.0",
		Source:           "programs",
		UserID:           user1.ID,
		TeamID:           &team3.ID,
		AutomaticInstall: true,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	team3Policies, _, err := ds.ListTeamPolicies(ctx, team3.ID, fleet.ListOptions{}, fleet.ListOptions{}, "", "")
	require.NoError(t, err)
	require.Len(t, team3Policies, 3)
	require.Equal(t, "[Install software] Something2 (msi) 3", team3Policies[2].Name)
}

func testGetDetailsForUninstallFromExecutionID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)

	// create a couple software titles
	installer1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		BundleIdentifier: "foobar0",
		Extension:        "pkg",
		StorageID:        "storage0",
		Filename:         "foobar0",
		Title:            "foobar",
		Version:          "1.0",
		Source:           "apps",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	installer2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		BundleIdentifier: "foobar1",
		Extension:        "pkg",
		StorageID:        "storage1",
		Filename:         "foobar1",
		Title:            "barfoo",
		Version:          "1.0",
		Source:           "apps",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// get software title for unknown exec id
	title, selfService, err := ds.GetDetailsForUninstallFromExecutionID(ctx, "unknown")
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Empty(t, title)
	require.False(t, selfService)

	// create a couple pending software install request, the first will be
	// immediately present in host_software_installs too (activated)
	req1, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installer1, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	req2, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installer2, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	_, _, err = ds.GetDetailsForUninstallFromExecutionID(ctx, req1)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// record a result for req1, will be deleted from upcoming_activities
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           req1,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	_, _, err = ds.GetDetailsForUninstallFromExecutionID(ctx, req1)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// create an uninstall request for installer1
	req3 := uuid.NewString()
	err = ds.InsertSoftwareUninstallRequest(ctx, req3, host.ID, installer1, true)
	require.NoError(t, err)

	title, selfService, err = ds.GetDetailsForUninstallFromExecutionID(ctx, req3)
	require.NoError(t, err)
	require.Equal(t, "foobar", title)
	require.True(t, selfService)

	// record a result for req2, will activate req3 so it is now in host_software_installs too
	_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           req2,
		InstallScriptExitCode: ptr.Int(0),
	}, nil)
	require.NoError(t, err)

	title, selfService, err = ds.GetDetailsForUninstallFromExecutionID(ctx, req3)
	require.NoError(t, err)
	require.Equal(t, "foobar", title)
	require.True(t, selfService)
}

func testGetTeamsWithInstallerByHash(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)

	hash1, hash2, hash3 := "hash1", "hash2", "hash3"

	// Add some software installers to No team
	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallerFile:    tfr1,
			BundleIdentifier: "bid1",
			Extension:        "pkg",
			StorageID:        hash1,
			Filename:         "installer1.pkg",
			Title:            "installer1",
			Version:          "1.0",
			Source:           "apps",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			Platform:         "darwin",
			URL:              "https://example.com/1",
		}, {
			InstallerFile:    tfr1,
			BundleIdentifier: "bid2",
			Extension:        "pkg",
			StorageID:        hash2,
			Filename:         "installer2.pkg",
			Title:            "installer2",
			Version:          "2.0",
			Source:           "apps",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			Platform:         "darwin",
			URL:              "https://example.com/2",
		},
	})
	require.NoError(t, err)

	// Add some installers to Team 1
	err = ds.BatchSetSoftwareInstallers(ctx, &team1.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallerFile:    tfr1,
			BundleIdentifier: "bid1",
			Extension:        "pkg",
			StorageID:        hash1,
			Filename:         "installer1.pkg",
			Title:            "installer1",
			Version:          "1.0",
			Source:           "apps",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			TeamID:           &team1.ID,
			Platform:         "darwin",
			URL:              "https://example.com/1",
		},
		{
			InstallerFile:    tfr1,
			BundleIdentifier: "bid3",
			Extension:        "pkg",
			StorageID:        hash3,
			Filename:         "installer3.pkg",
			Title:            "installer3",
			Version:          "3.0",
			Source:           "apps",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			TeamID:           &team1.ID,
			Platform:         "darwin",
			URL:              "https://example.com/4",
		},
	})
	require.NoError(t, err)

	// add an in-house app to the team
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team1.ID,
		UserID:           user.ID,
		Title:            "inhouse",
		Filename:         "inhouse.ipa",
		BundleIdentifier: "com.inhouse",
		StorageID:        "inhouse",
		Extension:        "ipa",
		Version:          "1.2.3",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// get installer IDs from added installers
	var installer1NoTeam, installer1Team1, installer2NoTeam uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, q, &installer1NoTeam, "SELECT id FROM software_installers WHERE filename = ? AND global_or_team_id = ?", "installer1.pkg", 0)
		require.NoError(t, err)
		require.NotEmpty(t, installer1NoTeam)

		err = sqlx.GetContext(ctx, q, &installer1Team1, "SELECT id FROM software_installers WHERE filename = ? AND global_or_team_id = ?", "installer1.pkg", team1.ID)
		require.NoError(t, err)
		require.NotEmpty(t, installer1Team1)

		err = sqlx.GetContext(ctx, q, &installer2NoTeam, "SELECT id FROM software_installers WHERE filename = ? AND global_or_team_id = ?", "installer2.pkg", 0)
		require.NoError(t, err)
		require.NotEmpty(t, installer2NoTeam)
		return nil
	})

	// fetching by non-existent hash returns empty map
	installers, err := ds.GetTeamsWithInstallerByHash(ctx, "not_found", "foobar")
	require.NoError(t, err)
	require.Empty(t, installers)

	// there should be 2 installers, one for No team and one for Team 1
	installers, err = ds.GetTeamsWithInstallerByHash(ctx, hash1, "https://example.com/1")
	require.NoError(t, err)
	require.Len(t, installers, 2)

	require.Len(t, installers[0], 1)
	require.Equal(t, installer1NoTeam, installers[0][0].InstallerID)
	require.Nil(t, installers[0][0].TeamID)

	require.Len(t, installers[1], 1)
	require.Equal(t, installer1Team1, installers[1][0].InstallerID)
	require.NotNil(t, installers[1][0].TeamID)
	require.Equal(t, team1.ID, *installers[1][0].TeamID)

	for _, is := range installers {
		i := is[0]
		require.Equal(t, "installer1", i.Title)
		require.Equal(t, "pkg", i.Extension)
		require.Equal(t, "1.0", i.Version)
		require.Equal(t, "darwin", i.Platform)
		require.Equal(t, hash1, i.StorageID)
	}

	installers, err = ds.GetTeamsWithInstallerByHash(ctx, hash2, "https://example.com/2")
	require.NoError(t, err)
	require.Len(t, installers, 1)
	require.Len(t, installers[0], 1)
	require.Equal(t, installers[0][0].InstallerID, installer2NoTeam)

	// in-house hash with invalid url
	installers, err = ds.GetTeamsWithInstallerByHash(ctx, "inhouse", "https://no-such-match")
	require.NoError(t, err)
	require.Len(t, installers, 0)

	// in-house hash without url match
	installers, err = ds.GetTeamsWithInstallerByHash(ctx, "inhouse", "")
	require.NoError(t, err)
	require.Len(t, installers, 1)
	require.Len(t, installers[team1.ID], 2) // ios and ipados
	require.Equal(t, "inhouse.ipa", installers[team1.ID][0].Filename)
	require.Equal(t, "inhouse.ipa", installers[team1.ID][1].Filename)
	var foundPlatforms []string
	for _, inst := range installers[team1.ID] {
		foundPlatforms = append(foundPlatforms, inst.Platform)
		require.Equal(t, "inhouse", inst.StorageID)
		require.Nil(t, inst.HTTPETag) // in-house apps don't have ETags
	}
	require.ElementsMatch(t, []string{"ios", "ipados"}, foundPlatforms)

	// Simulate the scenario from issue #42260: an FMA version update creates
	// a second row with the same storage_id but different version and is_active = 0.
	// FMA rows dedupe by version, so the same bytes can back more than one version.
	// GetTeamsWithInstallerByHash must only return the active row.
	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "installer1",
		Slug:             "installer1/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.installer1.fma",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, storage_id, filename, extension, version, platform, title_id,
				 install_script_content_id, uninstall_script_content_id, is_active, url, package_ids, patch_query,
				 fleet_maintained_app_id)
			SELECT team_id, global_or_team_id, storage_id, filename, extension, 'old_version', platform, title_id,
				install_script_content_id, uninstall_script_content_id, 0, url, package_ids, patch_query, ?
			FROM software_installers WHERE id = ?
		`, fma.ID, installer1NoTeam)
		return err
	})

	// Should still return only the active installer per team, not the inactive duplicate
	installers, err = ds.GetTeamsWithInstallerByHash(ctx, hash1, "https://example.com/1")
	require.NoError(t, err)
	require.Len(t, installers, 2) // No team + Team 1, each with 1 active installer

	require.Len(t, installers[0], 1)
	require.Equal(t, installer1NoTeam, installers[0][0].InstallerID)

	require.Len(t, installers[1], 1)
	require.Equal(t, installer1Team1, installers[1][0].InstallerID)
}

func testEditDeleteSoftwareInstallersActivateNextActivity(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// create a few installers
	newInstallerFile := func(ident string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(ident), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	err := ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer1"),
			StorageID:       "installer1",
			Filename:        "installer1",
			Title:           "installer1",
			Source:          "apps",
			Version:         "1",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/1",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer2"),
			StorageID:       "installer2",
			Filename:        "installer2",
			Title:           "installer2",
			Source:          "apps",
			Version:         "2",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/2",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)
	installers, err := ds.GetSoftwareInstallers(ctx, 0)
	require.NoError(t, err)
	require.Len(t, installers, 2)
	sort.Slice(installers, func(i, j int) bool {
		return installers[i].URL < installers[j].URL
	})
	ins1, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *installers[0].TitleID, false)
	require.NoError(t, err)
	ins2, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *installers[1].TitleID, false)
	require.NoError(t, err)

	// create a few hosts
	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "host3key", "host3uuid", time.Now())

	// enqueue software installs on each host
	host1Ins1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, ins1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host1Ins2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, ins2.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	// add a script exec as last activity for host1
	host1Script, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: host1.ID, ScriptContents: "echo", UserID: &user.ID, SyncRequest: true,
	})
	require.NoError(t, err)
	host2Ins1, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, ins1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host2Ins2, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, ins2.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	// add a script exec as first activity for host3
	host3Script, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: host3.ID, ScriptContents: "echo", UserID: &user.ID, SyncRequest: true,
	})
	require.NoError(t, err)
	host3Ins2, err := ds.InsertSoftwareInstallRequest(ctx, host3.ID, ins2.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, host1, host1Ins1, host1Ins2, host1Script.ExecutionID)
	checkUpcomingActivities(t, ds, host2, host2Ins1, host2Ins2)
	checkUpcomingActivities(t, ds, host3, host3Script.ExecutionID, host3Ins2)

	// simulate an update to installer 1 metadata
	err = ds.ProcessInstallerUpdateSideEffects(ctx, ins1.InstallerID, true, false)
	require.NoError(t, err)

	// installer 1 activities were deleted, next activity was activated
	checkUpcomingActivities(t, ds, host1, host1Ins2, host1Script.ExecutionID)
	checkUpcomingActivities(t, ds, host2, host2Ins2)
	checkUpcomingActivities(t, ds, host3, host3Script.ExecutionID, host3Ins2)

	// delete installer 2
	err = ds.DeleteSoftwareInstaller(ctx, ins2.InstallerID)
	require.NoError(t, err)

	// installer 2 activities were deleted, next activity was activated for host1 and host2
	checkUpcomingActivities(t, ds, host1, host1Script.ExecutionID)
	checkUpcomingActivities(t, ds, host2)
	checkUpcomingActivities(t, ds, host3, host3Script.ExecutionID)
}

func testBatchSetSoftwareInstallersActivateNextActivity(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// create a few installers
	newInstallerFile := func(ident string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(ident), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	err := ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer1"),
			StorageID:       "installer1",
			Filename:        "installer1",
			Title:           "installer1",
			Source:          "apps",
			Version:         "1",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/1",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer2"),
			StorageID:       "installer2",
			Filename:        "installer2",
			Title:           "installer2",
			Source:          "apps",
			Version:         "2",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/2",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer3"),
			StorageID:       "installer3",
			Filename:        "installer3",
			Title:           "installer3",
			Source:          "apps",
			Version:         "3",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/3",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)
	installers, err := ds.GetSoftwareInstallers(ctx, 0)
	require.NoError(t, err)
	require.Len(t, installers, 3)
	sort.Slice(installers, func(i, j int) bool {
		return installers[i].URL < installers[j].URL
	})
	ins1, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *installers[0].TitleID, false)
	require.NoError(t, err)
	ins2, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *installers[1].TitleID, false)
	require.NoError(t, err)
	ins3, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, nil, *installers[2].TitleID, false)
	require.NoError(t, err)

	// create a few hosts
	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	host3 := test.NewHost(t, ds, "host3", "3", "host3key", "host3uuid", time.Now())

	// enqueue software installs on each host
	host1Ins1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, ins1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host1Ins2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, ins2.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host1Ins3, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, ins3.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host2Ins2, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, ins2.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host2Ins1, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, ins1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host2Ins3, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, ins3.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host3Ins3, err := ds.InsertSoftwareInstallRequest(ctx, host3.ID, ins3.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host3Ins2, err := ds.InsertSoftwareInstallRequest(ctx, host3.ID, ins2.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	host3Ins1, err := ds.InsertSoftwareInstallRequest(ctx, host3.ID, ins1.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, host1, host1Ins1, host1Ins2, host1Ins3)
	checkUpcomingActivities(t, ds, host2, host2Ins2, host2Ins1, host2Ins3)
	checkUpcomingActivities(t, ds, host3, host3Ins3, host3Ins2, host3Ins1)

	// no change
	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer1"),
			StorageID:       "installer1",
			Filename:        "installer1",
			Title:           "installer1",
			Source:          "apps",
			Version:         "1",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/1",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer2"),
			StorageID:       "installer2",
			Filename:        "installer2",
			Title:           "installer2",
			Source:          "apps",
			Version:         "2",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/2",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer3"),
			StorageID:       "installer3",
			Filename:        "installer3",
			Title:           "installer3",
			Source:          "apps",
			Version:         "3",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/3",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, host1, host1Ins1, host1Ins2, host1Ins3)
	checkUpcomingActivities(t, ds, host2, host2Ins2, host2Ins1, host2Ins3)
	checkUpcomingActivities(t, ds, host3, host3Ins3, host3Ins2, host3Ins1)

	// remove installer 1, update installer 2
	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer2"),
			PreInstallQuery: "SELECT 1", // <- metadata updated
			StorageID:       "installer2",
			Filename:        "installer2",
			Title:           "installer2",
			Source:          "apps",
			Version:         "2",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/2",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
		{
			InstallScript:   "install",
			InstallerFile:   newInstallerFile("installer3"),
			StorageID:       "installer3",
			Filename:        "installer3",
			Title:           "installer3",
			Source:          "apps",
			Version:         "3",
			UserID:          user.ID,
			Platform:        "darwin",
			URL:             "https://example.com/3",
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	// installer 1 and 2 activities were deleted, next activity was activated
	checkUpcomingActivities(t, ds, host1, host1Ins3)
	checkUpcomingActivities(t, ds, host2, host2Ins3)
	checkUpcomingActivities(t, ds, host3, host3Ins3)

	// add a pending script on host 1 and 2
	host1Script, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: host1.ID, ScriptContents: "echo", UserID: &user.ID, SyncRequest: true,
	})
	require.NoError(t, err)
	host2Script, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID: host2.ID, ScriptContents: "echo", UserID: &user.ID, SyncRequest: true,
	})
	require.NoError(t, err)

	// clear everything
	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)

	checkUpcomingActivities(t, ds, host1, host1Script.ExecutionID)
	checkUpcomingActivities(t, ds, host2, host2Script.ExecutionID)
	checkUpcomingActivities(t, ds, host3)
}

func testSoftwareInstallerReplicaLag(t *testing.T, _ *Datastore) {
	opts := &testing_utils.DatastoreTestOptions{DummyReplica: true}
	ds := CreateMySQLDSWithOptions(t, opts)
	defer ds.Close()

	ctx := context.Background()
	test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team 1"})
	require.NoError(t, err)
	opts.RunReplication()

	// upload software installer
	installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:            "foo",
		Source:           "apps",
		Version:          "1.0",
		InstallScript:    "echo",
		StorageID:        "storage",
		Filename:         "installer.pkg",
		BundleIdentifier: "com.foo.installer",
		UserID:           user.ID,
		TeamID:           &team.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	require.NotZero(t, installerID)
	require.NotZero(t, titleID)
	// opts.RunReplication() // - replication should not be needed after fix
	ctx = ctxdb.RequirePrimary(ctx, true)

	// then validate it GetSoftwareInstallerMetadataByTeamAndTitleID()
	gotInstaller, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &team.ID, titleID, false)
	require.NoError(t, err)
	require.NotNil(t, gotInstaller)
}

func testSoftwareTitleDisplayName(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host0 := test.NewHost(t, ds, "host0", "", "host0key", "host0uuid", time.Now())

	installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		Extension:        "msi",
		StorageID:        "storageid",
		Filename:         "originalname.msi",
		Title:            "OriginalName1",
		PackageIDs:       []string{"id2"},
		Version:          "2.0",
		Source:           "programs",
		AutomaticInstall: true,
		UserID:           user1.ID,

		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Display name is empty by default
	titles, _, _, err := ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}},
	)
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Empty(t, titles[0].DisplayName)

	title, err := ds.SoftwareTitleByID(ctx, titleID, ptr.Uint(0), fleet.TeamFilter{})
	require.NoError(t, err)
	assert.Empty(t, title.DisplayName)

	err = ds.SaveInstallerUpdates(ctx, &fleet.UpdateSoftwareInstallerPayload{
		DisplayName:       ptr.String("update1"),
		TitleID:           titleID,
		InstallerFile:     &fleet.TempFileReader{},
		InstallScript:     new(string),
		PreInstallQuery:   new(string),
		PostInstallScript: new(string),
		SelfService:       ptr.Bool(false),
		UninstallScript:   new(string),
	})
	require.NoError(t, err)

	// Display name entry should be in join table
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		type result struct {
			DisplayName     string `db:"display_name"`
			SoftwareTitleID uint   `db:"software_title_id"`
			TeamID          uint   `db:"team_id"`
		}
		var r []result

		err := sqlx.SelectContext(ctx, q, &r, "SELECT display_name, software_title_id, team_id FROM software_title_display_names")
		require.NoError(t, err)

		assert.Len(t, r, 1)
		assert.Equal(t, r[0], result{"update1", titleID, 0})
		return nil
	})

	// List contains display name
	titles, _, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}},
	)
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, "update1", titles[0].DisplayName)

	// Entity contains display name
	title, err = ds.SoftwareTitleByID(ctx, titleID, ptr.Uint(0), fleet.TeamFilter{})
	require.NoError(t, err)
	assert.Equal(t, "update1", title.DisplayName)

	// Update host's software so we get a software version
	software0 := []fleet.Software{
		{Name: "OriginalName1", Version: "0.0.1", Source: "programs", TitleID: ptr.Uint(titleID)},
	}
	_, err = ds.UpdateHostSoftware(ctx, host0.ID, software0)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))

	softwareList, _, err := ds.ListSoftware(ctx, fleet.SoftwareListOptions{})
	require.NoError(t, err)
	assert.Len(t, softwareList, 1)
	assert.Equal(t, titleID, *softwareList[0].TitleID)
	assert.Equal(t, "update1", softwareList[0].DisplayName)

	software, err := ds.SoftwareByID(ctx, softwareList[0].ID, ptr.Uint(0), false, &fleet.TeamFilter{User: &fleet.User{
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})
	require.NoError(t, err)
	assert.Equal(t, titleID, *software.TitleID)
	assert.Equal(t, "update1", software.DisplayName)

	// Update the display name again, should see the change
	err = ds.SaveInstallerUpdates(ctx, &fleet.UpdateSoftwareInstallerPayload{
		DisplayName:       ptr.String("update2"),
		TitleID:           titleID,
		InstallerFile:     &fleet.TempFileReader{},
		InstallScript:     new(string),
		PreInstallQuery:   new(string),
		PostInstallScript: new(string),
		SelfService:       ptr.Bool(false),
		UninstallScript:   new(string),
	})
	require.NoError(t, err)

	// List contains display name
	titles, _, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}},
	)
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, "update2", titles[0].DisplayName)

	// Entity contains display name
	title, err = ds.SoftwareTitleByID(ctx, titleID, ptr.Uint(0), fleet.TeamFilter{})
	require.NoError(t, err)
	assert.Equal(t, "update2", title.DisplayName)

	softwareList, _, err = ds.ListSoftware(ctx, fleet.SoftwareListOptions{})
	require.NoError(t, err)
	assert.Len(t, softwareList, 1)
	assert.Equal(t, titleID, *softwareList[0].TitleID)
	assert.Equal(t, "update2", softwareList[0].DisplayName)

	software, err = ds.SoftwareByID(ctx, softwareList[0].ID, ptr.Uint(0), false, &fleet.TeamFilter{User: &fleet.User{
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})
	require.NoError(t, err)
	assert.Equal(t, titleID, *software.TitleID)
	assert.Equal(t, "update2", software.DisplayName)

	// Update display name to be empty
	err = ds.SaveInstallerUpdates(ctx, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:           titleID,
		InstallerFile:     &fleet.TempFileReader{},
		InstallScript:     new(string),
		PreInstallQuery:   new(string),
		PostInstallScript: new(string),
		SelfService:       ptr.Bool(false),
		UninstallScript:   new(string),
		DisplayName:       ptr.String(""),
	})
	require.NoError(t, err)

	// List contains display name
	titles, _, _, err = ds.ListSoftwareTitles(
		ctx,
		fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(0)},
		fleet.TeamFilter{User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}},
	)
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Empty(t, titles[0].DisplayName)

	// Entity contains display name
	title, err = ds.SoftwareTitleByID(ctx, titleID, ptr.Uint(0), fleet.TeamFilter{})
	require.NoError(t, err)
	assert.Empty(t, title.DisplayName)

	softwareList, _, err = ds.ListSoftware(ctx, fleet.SoftwareListOptions{})
	require.NoError(t, err)
	assert.Len(t, softwareList, 1)
	assert.Equal(t, titleID, *softwareList[0].TitleID)
	assert.Empty(t, softwareList[0].DisplayName)

	software, err = ds.SoftwareByID(ctx, softwareList[0].ID, ptr.Uint(0), false, &fleet.TeamFilter{User: &fleet.User{
		GlobalRole: ptr.String(fleet.RoleAdmin),
	}})
	require.NoError(t, err)
	assert.Equal(t, titleID, *software.TitleID)
	assert.Empty(t, software.DisplayName)

	// Delete software installer, display name should be deleted
	_, err = ds.DeleteTeamPolicies(ctx, 0, []uint{1})
	require.NoError(t, err)
	require.NoError(t, ds.DeleteSoftwareInstaller(ctx, installerID))
	_, err = ds.getSoftwareTitleDisplayName(ctx, 0, titleID)
	require.ErrorContains(t, err, "not found")

	// Add installer, vpp, in-house app with custom names
	_, titleID, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:    tfr1,
		Extension:        "msi",
		StorageID:        "storageid",
		Filename:         "originalname.msi",
		Title:            "OriginalName1",
		PackageIDs:       []string{"id2"},
		Version:          "2.0",
		Source:           "programs",
		AutomaticInstall: true,
		UserID:           user1.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	err = ds.SaveInstallerUpdates(ctx, &fleet.UpdateSoftwareInstallerPayload{
		DisplayName:       ptr.String("update2"),
		TitleID:           titleID,
		InstallerFile:     &fleet.TempFileReader{},
		InstallScript:     new(string),
		PreInstallQuery:   new(string),
		PostInstallScript: new(string),
		SelfService:       ptr.Bool(false),
		UninstallScript:   new(string),
	})
	require.NoError(t, err)

	payload := fleet.UploadSoftwareInstallerPayload{
		UserID:           user1.ID,
		Title:            "foo",
		BundleIdentifier: "com.foo",
		Filename:         "foo.ipa",
		StorageID:        "testingtesting123",
		Platform:         "ios",
		Extension:        "ipa",
		Version:          "1.2.3",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}
	ipaInstallerID, ipaTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.NoError(t, err)

	err = ds.SaveInHouseAppUpdates(ctx, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     ipaTitleID,
		InstallerID: ipaInstallerID,
		DisplayName: ptr.String("ipa_foo"),
	})
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, ds)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: "darwin"}, DisplayName: ptr.String("VPP1")},
		Name:             "vpp1",
		BundleIdentifier: "com.app.vpp1",
	}, nil)
	require.NoError(t, err)

	// Batch insert installers should delete previous display names
	// and ignore in-house and vpp names
	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallScript:      "install",
			InstallerFile:      &fleet.TempFileReader{},
			StorageID:          "storageid",
			Filename:           "originalname.msi",
			Title:              "OriginalName1",
			DisplayName:        "batch_name1",
			Source:             "apps",
			Version:            "1",
			PreInstallQuery:    "select 0 from foo;",
			UserID:             user1.ID,
			Platform:           "darwin",
			URL:                "https://example.com",
			InstallDuringSetup: ptr.Bool(true),
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

	getAllDisplayNames := func() []string {
		var names []string
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			err := sqlx.SelectContext(ctx, q, &names, `SELECT display_name FROM software_title_display_names`)
			require.NoError(t, err)
			return nil
		})
		return names
	}

	names := getAllDisplayNames()
	require.Len(t, names, 3)
	require.NotContains(t, names, "update2")
	require.Contains(t, names, "batch_name1")
	require.Contains(t, names, "VPP1")
	require.Contains(t, names, "ipa_foo")

	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	names = getAllDisplayNames()
	require.Len(t, names, 2)
	require.Contains(t, names, "VPP1")
	require.Contains(t, names, "ipa_foo")
}

func testGetSoftwarePackagesByTeamAndTitleID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Pkg Lister", "pkglister@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "-lbl", Query: "SELECT 1"})
	require.NoError(t, err)

	mk := func(storage string, filename string, labels *fleet.LabelIdentsWithScope) *fleet.UploadSoftwareInstallerPayload {
		return &fleet.UploadSoftwareInstallerPayload{
			StorageID:        storage,
			Filename:         filename,
			Title:            "Multi App",
			BundleIdentifier: "com.example.multi",
			Extension:        "pkg",
			Source:           "apps",
			Platform:         "darwin",
			Version:          "1.0",
			InstallScript:    "install " + storage,
			UserID:           user.ID,
			ValidatedLabels:  labels,
			TeamID:           &team.ID,
		}
	}

	// Two custom packages of the same version but different content on one title; only
	// the first is scoped to a label.
	withLabel := &fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeIncludeAny,
		ByName:     map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}},
	}
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, mk("multi-1", "multi-1.pkg", withLabel))
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mk("multi-2", "multi-2.pkg", &fleet.LabelIdentsWithScope{}))
	require.NoError(t, err)

	pkgs, err := ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	// returned first-added first, each with its own label scope
	require.Equal(t, "multi-1.pkg", pkgs[0].Name)
	require.Equal(t, "multi-1", pkgs[0].StorageID)
	require.Equal(t, "install multi-1", pkgs[0].InstallScript)
	require.Len(t, pkgs[0].LabelsIncludeAny, 1)
	require.Equal(t, lbl.ID, pkgs[0].LabelsIncludeAny[0].LabelID)
	require.Equal(t, "multi-2.pkg", pkgs[1].Name)
	require.Empty(t, pkgs[1].LabelsIncludeAny)

	// a title with no packages returns none
	none, err := ds.GetSoftwarePackagesByTeamAndTitleID(ctx, &team.ID, titleID+1000)
	require.NoError(t, err)
	require.Empty(t, none)
}

func testMatchOrCreateSoftwareInstallerDuplicateHash(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team A"})
	require.NoError(t, err)
	teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "Team B"})
	require.NoError(t, err)

	const sameHash = "dup-hash-001"

	mkPayload := func(teamID *uint, filename, title string) *fleet.UploadSoftwareInstallerPayload {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("same-bytes"), t.TempDir)
		require.NoError(t, err)
		return &fleet.UploadSoftwareInstallerPayload{
			InstallerFile:   tfr,
			Extension:       "sh",
			StorageID:       sameHash,
			Filename:        filename,
			Title:           title,
			Version:         "1.0",
			Source:          "apps",
			Platform:        "darwin",
			UserID:          user.ID,
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			TeamID:          teamID,
		}
	}

	// Create on Team A → success
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPayload(&teamA.ID, "a.sh", "title-a"))
	require.NoError(t, err)

	// Duplicate on Team A with different name/title but same hash → reject
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPayload(&teamA.ID, "b.sh", "title-b"))
	var iae *fleet.InvalidArgumentError
	require.ErrorAs(t, err, &iae)

	// Same hash on different team → allowed
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPayload(&teamB.ID, "c.sh", "title-c"))
	require.NoError(t, err)

	// Global scope first time → allowed
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPayload(nil, "global1.sh", "title-g1"))
	require.NoError(t, err)

	// Global scope second time (duplicate hash) → reject
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPayload(nil, "global2.sh", "title-g2"))
	var iae2 *fleet.InvalidArgumentError
	require.ErrorAs(t, err, &iae2)

	// Test that binary packages (.pkg) with duplicate hash ARE allowed
	mkPkgPayload := func(teamID *uint, filename, title string) *fleet.UploadSoftwareInstallerPayload {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("same-binary-bytes"), t.TempDir)
		require.NoError(t, err)
		return &fleet.UploadSoftwareInstallerPayload{
			InstallerFile:   tfr,
			Extension:       "pkg",
			StorageID:       "same-pkg-hash",
			Filename:        filename,
			Title:           title,
			Version:         "1.0",
			Source:          "apps",
			Platform:        "darwin",
			UserID:          user.ID,
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			TeamID:          teamID,
		}
	}

	// Binary packages with same hash on same team → allowed
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPkgPayload(&teamA.ID, "pkg1.pkg", "title-pkg1"))
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPkgPayload(&teamA.ID, "pkg2.pkg", "title-pkg2"))
	require.NoError(t, err, "binary packages with same hash should be allowed on same team")

	// Same title and hash on the same team → rejected by the within-title hash check
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, mkPayload(&teamA.ID, "a.sh", "title-a"))
	require.ErrorContains(t, err, "same SHA-256 hash")
}

func testAddSoftwareTitleToMatchingSoftware(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	software1 := []fleet.Software{
		{Name: "Win Title", Version: "1.0", Source: "programs", UpgradeCode: ptr.String("CODE_1")},
	}

	// create a vpp app
	test.CreateInsertGlobalVPPToken(t, ds)
	app, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_1", Platform: "ios"}, DisplayName: ptr.String("VPP1")},
		Name:             "iOS Title",
		BundleIdentifier: "com.foo",
	}, nil)
	require.NoError(t, err)

	host2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "ios-test",
		OsqueryHostID: ptr.String("osquery-ios"),
		NodeKey:       ptr.String("node-key-ios"),
		UUID:          uuid.NewString(),
		Platform:      "ios",
	})
	require.NoError(t, err)
	software2 := []fleet.Software{
		{Name: "iOS Title", Version: "1.0", Source: "ios_apps", BundleIdentifier: "com.foo"},
	}

	_, err = ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// creates a second software title with the same name
	payload := &fleet.UploadSoftwareInstallerPayload{
		Title:           "Win Title",
		Source:          "programs",
		UpgradeCode:     "CODE_2",
		Filename:        "something.msi",
		Version:         "1.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	}

	_, newTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	require.NoError(t, err)
	require.NotEmpty(t, newTitleID)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var gotTitleID uint
		err := sqlx.GetContext(ctx, q, &gotTitleID, `SELECT title_id FROM software WHERE name = ?`, "Win Title")
		require.NoError(t, err)
		require.NotEqual(t, newTitleID, gotTitleID) // title with different upgrade code is new
		return nil
	})

	// check that host has the ios app installed and title is correct.
	found, err := hostInstalledSoftware(ds, ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, app.TitleID, found[0].ID)

	// add macOS installer with the same bundle identifier
	payloadMacOS := &fleet.UploadSoftwareInstallerPayload{
		Title:            "A Mac Title",
		Source:           "apps",
		Platform:         "darwin",
		Filename:         "something.pkg",
		Version:          "1.0",
		BundleIdentifier: "com.foo",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}

	_, titleIDMacOS, err := ds.MatchOrCreateSoftwareInstaller(ctx, payloadMacOS)
	require.NoError(t, err)
	require.NotEqual(t, app.TitleID, titleIDMacOS)

	// check that the installed ios app did not change title ID to the new installer
	found, err = hostInstalledSoftware(ds, ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, app.TitleID, found[0].ID)
}

func testFleetMaintainedAppInstallerUpdates(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
	require.NoError(t, err)

	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "testpkg",
		Source:               "apps",
		Platform:             "darwin",
		PreInstallQuery:      "SELECT 1",
		InstallScript:        "echo install",
		PostInstallScript:    "echo post install",
		UninstallScript:      "echo uninstall",
		InstallerFile:        tfr,
		StorageID:            "storageid1",
		Filename:             "test.pkg",
		Version:              "1.0",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: ptr.Uint(maintainedApp.ID),
		InstallDuringSetup:   ptr.Bool(false),
		SelfService:          false,
	})
	require.NoError(t, err)

	tmFilter := fleet.TeamFilter{User: test.UserAdmin}
	titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(0), Platform: "darwin", AvailableForInstall: true}, tmFilter)
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	require.False(t, *titles[0].SoftwarePackage.SelfService)

	installer, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)
	require.NotNil(t, installer)

	installScript := installer.InstallScriptContentID
	postInstallScript := installer.PostInstallScriptContentID
	uninstallScript := installer.UninstallScriptContentID

	require.NotZero(t, installScript)
	require.NotZero(t, postInstallScript)
	require.NotZero(t, uninstallScript)
	require.Equal(t, "SELECT 1", installer.PreInstallQuery)

	// batch add the installer with different scripts, setup experience, self service
	err = ds.BatchSetSoftwareInstallers(ctx, nil, []*fleet.UploadSoftwareInstallerPayload{
		{
			Title:                "testpkg",
			Source:               "apps",
			PreInstallQuery:      "SELECT 1 DIFFERENT",
			InstallScript:        "echo install 2",
			PostInstallScript:    "echo post install 2",
			UninstallScript:      "echo uninstall 2",
			InstallerFile:        tfr,
			StorageID:            "storageid1",
			Filename:             "test.pkg",
			Version:              "1.0",
			UserID:               user.ID,
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			FleetMaintainedAppID: ptr.Uint(maintainedApp.ID),
			InstallDuringSetup:   ptr.Bool(true),
			SelfService:          true,
		},
	})
	require.NoError(t, err)

	titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(0), Platform: "darwin", AvailableForInstall: true}, tmFilter)
	require.NoError(t, err)
	require.Len(t, titles, 1)
	require.True(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	require.True(t, *titles[0].SoftwarePackage.SelfService)

	installer, err = ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)
	require.NotNil(t, installer)

	// all fields that should have changed did change
	require.NotEqual(t, installScript, installer.InstallScriptContentID)
	require.NotEqual(t, postInstallScript, installer.PostInstallScriptContentID)
	require.NotEqual(t, uninstallScript, installer.UninstallScriptContentID)
	require.Equal(t, "SELECT 1 DIFFERENT", installer.PreInstallQuery)
}

func testListFleetMaintainedAppActiveInstallers(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	newFile := func(s string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(s), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	// Active FMA installer on team1.
	fmaTeam1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "FooFMA",
		Source:               "apps",
		Platform:             "darwin",
		InstallScript:        "echo install",
		UninstallScript:      "echo uninstall",
		InstallerFile:        newFile("t1"),
		StorageID:            "storage-t1",
		Filename:             "foo.pkg",
		Version:              "1.0",
		UserID:               user.ID,
		TeamID:               &team1.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintainedApp.ID),
	})
	require.NoError(t, err)

	// Active FMA installer on no-team.
	fmaNoTeam, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "FooFMA",
		Source:               "apps",
		Platform:             "darwin",
		InstallScript:        "echo install",
		UninstallScript:      "echo uninstall",
		InstallerFile:        newFile("nt"),
		StorageID:            "storage-nt",
		Filename:             "foo.pkg",
		Version:              "2.0",
		UserID:               user.ID,
		TeamID:               nil,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintainedApp.ID),
	})
	require.NoError(t, err)

	// Non-FMA installer on team2 — must be excluded.
	customTeam2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:           "Custom",
		Source:          "apps",
		Platform:        "darwin",
		InstallScript:   "echo install",
		UninstallScript: "echo uninstall",
		InstallerFile:   newFile("c2"),
		StorageID:       "storage-c2",
		Filename:        "custom.pkg",
		Version:         "9.0",
		UserID:          user.ID,
		TeamID:          &team2.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// An inactive (older) cached version of team1's FMA — must be excluded.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, storage_id, filename, extension, version, platform, title_id,
				 fleet_maintained_app_id, install_script_content_id, uninstall_script_content_id, is_active, package_ids, patch_query)
			SELECT team_id, global_or_team_id, 'storage-t1-old', filename, extension, '0.9', platform, title_id,
				fleet_maintained_app_id, install_script_content_id, uninstall_script_content_id, 0, package_ids, patch_query
			FROM software_installers WHERE id = ?
		`, fmaTeam1)
		return err
	})

	got, err := ds.ListFleetMaintainedAppActiveInstallers(ctx)
	require.NoError(t, err)

	byInstallerID := make(map[uint]fleet.FMAAutoUpdateCandidate, len(got))
	for _, c := range got {
		byInstallerID[c.InstallerID] = c
	}

	// Only the two active FMA rows are returned; the custom installer and the
	// inactive version are excluded.
	require.Len(t, got, 2)
	require.NotContains(t, byInstallerID, customTeam2)

	t1 := byInstallerID[fmaTeam1]
	require.NotNil(t, t1.TeamID)
	require.Equal(t, team1.ID, *t1.TeamID)
	require.Equal(t, "1.0", t1.Version)
	require.Equal(t, "maintained1", t1.Slug)
	require.Equal(t, maintainedApp.ID, t1.FleetMaintainedAppID)

	nt := byInstallerID[fmaNoTeam]
	require.Nil(t, nt.TeamID) // no-team scope maps to nil
	require.Equal(t, "2.0", nt.Version)
	require.Equal(t, "maintained1", nt.Slug)
}

func testInsertFleetMaintainedAppVersion(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-fma-insert"})
	require.NoError(t, err)

	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Maintained1", Slug: "maintained1", Platform: "darwin", UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "lbl-fma", Query: "SELECT 1"})
	require.NoError(t, err)

	newFile := func(s string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(s), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	// Active v1 installer with per-team config the cron must carry forward.
	activeID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title: "FooFMA", Source: "apps", Platform: "darwin",
		InstallScript: "echo install v1", UninstallScript: "echo uninstall v1",
		PreInstallQuery: "SELECT pre", PostInstallScript: "echo post",
		SelfService:   true,
		InstallerFile: newFile("v1"), StorageID: "sha-v1", Filename: "foo-1.0.pkg", Extension: "pkg",
		PackageIDs: []string{"OLD-PKG"},
		Version:    "1.0", UserID: user.ID, TeamID: &team.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{
			LabelScope: fleet.LabelScopeIncludeAny,
			ByName:     map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}},
		},
		FleetMaintainedAppID: new(maintainedApp.ID),
	})
	require.NoError(t, err)

	// Seed a caret pin that must survive the insert untouched, and mark v1 for the
	// setup experience so we can assert the flag is carried forward.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (?, ?, ?)`,
			team.ID, titleID, "^1"); err != nil {
			return err
		}
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET install_during_setup = 1 WHERE id = ?`, activeID)
		return err
	})

	// Cache v2 (inactive), cloning v1's config; package_ids must come from the
	// payload (version-specific MSI ProductCode), not be cloned from v1.
	v2ID, err := ds.InsertFleetMaintainedAppVersion(ctx, activeID, &fleet.UploadSoftwareInstallerPayload{
		Version: "2.0", Filename: "foo-2.0.pkg", Extension: "pkg", StorageID: "sha-v2",
		PackageIDs: []string{"NEW-PKG"},
		URL:        "https://example.test/foo-2.0.pkg", InstallScript: "echo install v2", UninstallScript: "echo uninstall v2",
	})
	require.NoError(t, err)
	require.NotEqual(t, activeID, v2ID)

	type row struct {
		Active             bool   `db:"is_active"`
		SelfService        bool   `db:"self_service"`
		InstallDuringSetup bool   `db:"install_during_setup"`
		Pre                string `db:"pre_install_query"`
		Version            string `db:"version"`
		Storage            string `db:"storage_id"`
		PackageIDs         string `db:"package_ids"`
		InstallID          *uint  `db:"install_script_content_id"`
		PostID             *uint  `db:"post_install_script_content_id"`
	}
	getRow := func(id uint) row {
		var r row
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &r,
				`SELECT is_active, self_service, install_during_setup, pre_install_query, version, storage_id, package_ids, install_script_content_id, post_install_script_content_id
				 FROM software_installers WHERE id = ?`, id)
		})
		return r
	}
	r1, r2 := getRow(activeID), getRow(v2ID)

	// v1 stays active; v2 inactive.
	require.True(t, r1.Active)
	require.False(t, r2.Active)
	// Config carried forward.
	require.True(t, r2.SelfService)
	require.True(t, r2.InstallDuringSetup, "install_during_setup must be carried forward")
	require.Equal(t, "NEW-PKG", r2.PackageIDs, "package_ids must be bound from payload, not cloned from v1")
	require.Equal(t, "SELECT pre", r2.Pre)
	require.Equal(t, "2.0", r2.Version)
	require.Equal(t, "sha-v2", r2.Storage)
	// New install script for the new version; post-install carried forward.
	require.NotNil(t, r2.InstallID)
	require.NotNil(t, r1.InstallID)
	require.NotEqual(t, *r1.InstallID, *r2.InstallID)
	require.NotNil(t, r2.PostID)
	require.NotNil(t, r1.PostID)
	require.Equal(t, *r1.PostID, *r2.PostID)

	// Label cloned onto v2.
	var labelCount int
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &labelCount,
			`SELECT COUNT(*) FROM software_installer_labels WHERE software_installer_id = ? AND label_id = ?`, v2ID, lbl.ID)
	})
	require.Equal(t, 1, labelCount)

	// Pin untouched.
	pin, err := ds.GetPinnedVersion(ctx, &team.ID, titleID)
	require.NoError(t, err)
	require.NotNil(t, pin)
	require.Equal(t, "^1", *pin)

	// Idempotent: re-caching v2 returns the same id, no new row.
	again, err := ds.InsertFleetMaintainedAppVersion(ctx, activeID, &fleet.UploadSoftwareInstallerPayload{
		Version: "2.0", Filename: "foo-2.0.pkg", Extension: "pkg", StorageID: "sha-v2",
		URL: "https://example.test/foo-2.0.pkg", InstallScript: "echo install v2", UninstallScript: "echo uninstall v2",
	})
	require.NoError(t, err)
	require.Equal(t, v2ID, again)

	// Force v2 to be the oldest non-active version so eviction is deterministic.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET uploaded_at = '2000-01-01 00:00:00' WHERE id = ?`, v2ID)
		return err
	})

	// Eviction: caching v3 brings the count to 3; the oldest non-active (v2) is
	// evicted while the active installer (v1) is always protected.
	v3ID, err := ds.InsertFleetMaintainedAppVersion(ctx, activeID, &fleet.UploadSoftwareInstallerPayload{
		Version: "3.0", Filename: "foo-3.0.pkg", Extension: "pkg", StorageID: "sha-v3",
		URL: "https://example.test/foo-3.0.pkg", InstallScript: "echo install v3", UninstallScript: "echo uninstall v3",
	})
	require.NoError(t, err)

	var remaining []uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &remaining,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id = ? ORDER BY id`, team.ID, titleID)
	})
	require.ElementsMatch(t, []uint{activeID, v3ID}, remaining, "v2 evicted, active protected")
}

// testInsertFleetMaintainedAppVersionProtectsLiveActive verifies that eviction
// keeps the row that is actually is_active=1 at eviction time (e.g. an admin
// rollback during the cron's download window), not the caller's stale view.
func testInsertFleetMaintainedAppVersionProtectsLiveActive(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-fma-live-active"})
	require.NoError(t, err)
	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Maintained2", Slug: "maintained2", Platform: "darwin", UniqueIdentifier: "fleet.maintained2",
	})
	require.NoError(t, err)
	newFile := func(s string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(s), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	v1, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title: "FooFMA", Source: "apps", Platform: "darwin", InstallScript: "echo i", UninstallScript: "echo u",
		InstallerFile: newFile("v1"), StorageID: "live-v1", Filename: "foo-1.0.pkg", Extension: "pkg",
		Version: "1.0", UserID: user.ID, TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintainedApp.ID),
	})
	require.NoError(t, err)

	// Cache v2, then promote it to active (simulating a concurrent admin rollback).
	v2, err := ds.InsertFleetMaintainedAppVersion(ctx, v1, &fleet.UploadSoftwareInstallerPayload{
		Version: "2.0", Filename: "foo-2.0.pkg", Extension: "pkg", StorageID: "live-v2",
		URL: "https://example.test/2", InstallScript: "echo i2", UninstallScript: "echo u2",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET is_active = (id = ?) WHERE global_or_team_id = ? AND fleet_maintained_app_id = ?`,
			v2, team.ID, maintainedApp.ID)
		return err
	})

	// Insert v3 passing the STALE active id (v1). Eviction must protect the live
	// active (v2) and the new row (v3), evicting v1 — not evict v2.
	v3, err := ds.InsertFleetMaintainedAppVersion(ctx, v1, &fleet.UploadSoftwareInstallerPayload{
		Version: "3.0", Filename: "foo-3.0.pkg", Extension: "pkg", StorageID: "live-v3",
		URL: "https://example.test/3", InstallScript: "echo i3", UninstallScript: "echo u3",
	})
	require.NoError(t, err)

	var remaining []uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &remaining,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND title_id = ? ORDER BY id`, team.ID, titleID)
	})
	require.ElementsMatch(t, []uint{v2, v3}, remaining, "live active (v2) protected, stale v1 evicted")
}

// testInsertFleetMaintainedAppVersionClonesLiveActive verifies the new version's
// per-team config is cloned from the row that is actually is_active=1 at insert
// time, not from the caller's (possibly stale) activeInstallerID — e.g. when an
// admin promotes a different cached row and edits it during the download window.
func testInsertFleetMaintainedAppVersionClonesLiveActive(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Carol", "carol@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-fma-clone-live"})
	require.NoError(t, err)
	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Maintained3", Slug: "maintained3", Platform: "darwin", UniqueIdentifier: "fleet.maintained3",
	})
	require.NoError(t, err)
	newFile := func(s string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(s), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	// v1 active with self-service OFF.
	v1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title: "FooFMA", Source: "apps", Platform: "darwin", InstallScript: "echo i", UninstallScript: "echo u",
		SelfService: false, InstallerFile: newFile("v1"), StorageID: "clone-v1", Filename: "foo-1.0.pkg", Extension: "pkg",
		Version: "1.0", UserID: user.ID, TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintainedApp.ID),
	})
	require.NoError(t, err)

	// Cache v2, promote it, and turn ON self-service + setup-experience on it
	// (simulating an admin rollback + edit during the cron's download window).
	v2, err := ds.InsertFleetMaintainedAppVersion(ctx, v1, &fleet.UploadSoftwareInstallerPayload{
		Version: "2.0", Filename: "foo-2.0.pkg", Extension: "pkg", StorageID: "clone-v2",
		URL: "https://example.test/2", InstallScript: "echo i2", UninstallScript: "echo u2",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE software_installers SET is_active = (id = ?), self_service = (id = ?), install_during_setup = (id = ?)
			 WHERE global_or_team_id = ? AND fleet_maintained_app_id = ?`,
			v2, v2, v2, team.ID, maintainedApp.ID)
		return err
	})

	// Insert v3 with the STALE caller id (v1). Config must clone from live active (v2).
	v3, err := ds.InsertFleetMaintainedAppVersion(ctx, v1, &fleet.UploadSoftwareInstallerPayload{
		Version: "3.0", Filename: "foo-3.0.pkg", Extension: "pkg", StorageID: "clone-v3",
		URL: "https://example.test/3", InstallScript: "echo i3", UninstallScript: "echo u3",
	})
	require.NoError(t, err)

	var r struct {
		SelfService        bool `db:"self_service"`
		InstallDuringSetup bool `db:"install_during_setup"`
	}
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &r,
			`SELECT self_service, install_during_setup FROM software_installers WHERE id = ?`, v3)
	})
	require.True(t, r.SelfService, "self_service cloned from live active (v2), not stale v1")
	require.True(t, r.InstallDuringSetup, "install_during_setup cloned from live active (v2), not stale v1")
}

// testGetSoftwareInstallerMetadataByStorageID verifies metadata recovery works
// for any row with the content hash — including an inactive one (e.g. after a
// rollback) — so the cron's byte-dedup path isn't locked out when no team has the
// version active.
func testGetSoftwareInstallerMetadataByStorageID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Dave", "dave@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team-fma-meta-hash"})
	require.NoError(t, err)
	maintainedApp, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Maintained4", Slug: "maintained4", Platform: "windows", UniqueIdentifier: "fleet.maintained4",
	})
	require.NoError(t, err)
	newFile := func(s string) *fleet.TempFileReader {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(s), t.TempDir)
		require.NoError(t, err)
		return tfr
	}

	id, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title: "FooFMA", Source: "programs", Platform: "windows", InstallScript: "echo i", UninstallScript: "echo u",
		PackageIDs: []string{"PROD-CODE"}, UpgradeCode: "UP-CODE",
		InstallerFile: newFile("v1"), StorageID: "hash-meta-1", Filename: "foo.msi", Extension: "msi",
		Version: "1.0", UserID: user.ID, TeamID: &team.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(maintainedApp.ID),
	})
	require.NoError(t, err)

	// Make it inactive — the is_active=1-filtered lookups would miss it.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE software_installers SET is_active = 0 WHERE id = ?`, id)
		return err
	})

	pids, ucode, err := ds.GetSoftwareInstallerMetadataByStorageID(ctx, "hash-meta-1")
	require.NoError(t, err)
	require.Equal(t, []string{"PROD-CODE"}, pids, "recovers package IDs from an inactive row")
	require.Equal(t, "UP-CODE", ucode)

	// Unknown hash → empty, no error.
	pids, ucode, err = ds.GetSoftwareInstallerMetadataByStorageID(ctx, "no-such-hash")
	require.NoError(t, err)
	require.Empty(t, pids)
	require.Empty(t, ucode)
}

func testRepointPolicyToNewInstaller(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	t.Run("custom_package", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team" + t.Name()})
		require.NoError(t, err)

		tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
		require.NoError(t, err)

		installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			Title:              "testpkg",
			Source:             "apps",
			Platform:           "darwin",
			PreInstallQuery:    "SELECT 1",
			InstallScript:      "echo install",
			PostInstallScript:  "echo post install",
			UninstallScript:    "echo uninstall",
			InstallerFile:      tfr,
			StorageID:          "storageid1",
			Filename:           "test.pkg",
			Version:            "1.0",
			UserID:             user.ID,
			ValidatedLabels:    &fleet.LabelIdentsWithScope{},
			InstallDuringSetup: ptr.Bool(false),
			SelfService:        false,
			TeamID:             ptr.Uint(team.ID),
		})
		require.NoError(t, err)

		policy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
			Name:                "p1",
			Query:               "SELECT 1;",
			SoftwareInstallerID: &installerID,
		})
		require.NoError(t, err)

		tmFilter := fleet.TeamFilter{User: test.UserAdmin, TeamID: ptr.Uint(team.ID)}
		titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(team.ID), Platform: "darwin", AvailableForInstall: true}, tmFilter)
		require.NoError(t, err)
		require.Len(t, titles, 1)
		require.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
		require.False(t, *titles[0].SoftwarePackage.SelfService)

		installer, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
		require.NoError(t, err)
		require.NotNil(t, installer)

		installScript := installer.InstallScriptContentID
		postInstallScript := installer.PostInstallScriptContentID
		uninstallScript := installer.UninstallScriptContentID

		require.NotZero(t, installScript)
		require.NotZero(t, postInstallScript)
		require.NotZero(t, uninstallScript)
		require.Equal(t, "SELECT 1", installer.PreInstallQuery)

		// batch add (gitops), this should succeed because we now update the pointer in the policy for the new version
		err = ds.BatchSetSoftwareInstallers(ctx, ptr.Uint(team.ID), []*fleet.UploadSoftwareInstallerPayload{
			{
				Title:              "testpkg",
				Source:             "apps",
				Platform:           "darwin",
				PreInstallQuery:    "SELECT 1 DIFFERENT",
				InstallScript:      "echo install 2",
				PostInstallScript:  "echo post install 2",
				UninstallScript:    "echo uninstall 2",
				InstallerFile:      tfr,
				StorageID:          "storageid1",
				Filename:           "test.pkg",
				Version:            "2.0", // Note the new version, this means we evict version 1.0 because it's a custom package
				UserID:             user.ID,
				ValidatedLabels:    &fleet.LabelIdentsWithScope{},
				InstallDuringSetup: ptr.Bool(true),
				SelfService:        true,
				TeamID:             ptr.Uint(team.ID),
			},
		})
		require.NoError(t, err)
		titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(team.ID), Platform: "darwin", AvailableForInstall: true}, tmFilter)
		require.NoError(t, err)
		require.Len(t, titles, 1)
		require.Len(t, titles[0].SoftwarePackage.AutomaticInstallPolicies, 1)
		require.Equal(t, policy.ID, titles[0].SoftwarePackage.AutomaticInstallPolicies[0].ID)

		metadata, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, ptr.Uint(team.ID), titles[0].ID, false)
		require.NoError(t, err)

		policyAfterUpdate, err := ds.TeamPolicy(ctx, team.ID, policy.ID)
		require.NoError(t, err)
		require.Equal(t, metadata.InstallerID, *policyAfterUpdate.SoftwareInstallerID)
	})

	t.Run("fma", func(t *testing.T) {
		team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team" + t.Name()})
		require.NoError(t, err)

		fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{ID: 1})
		require.NoError(t, err)

		tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
		require.NoError(t, err)

		installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			FleetMaintainedAppID: ptr.Uint(fma.ID),
			Title:                "testpkg_fma",
			Source:               "apps",
			Platform:             "darwin",
			PreInstallQuery:      "SELECT 1",
			InstallScript:        "echo install",
			PostInstallScript:    "echo post install",
			UninstallScript:      "echo uninstall",
			InstallerFile:        tfr,
			StorageID:            "storageid1",
			Filename:             "test_fma.pkg",
			Version:              "1.0",
			UserID:               user.ID,
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			InstallDuringSetup:   ptr.Bool(false),
			SelfService:          false,
			TeamID:               ptr.Uint(team.ID),
		})
		require.NoError(t, err)

		policy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
			Name:                "p2",
			Query:               "SELECT 1;",
			SoftwareInstallerID: &installerID,
		})
		require.NoError(t, err)

		tmFilter := fleet.TeamFilter{User: test.UserAdmin, TeamID: ptr.Uint(team.ID)}
		titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(team.ID), Platform: "darwin", AvailableForInstall: true}, tmFilter)
		require.NoError(t, err)
		require.Len(t, titles, 1)
		require.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
		require.False(t, *titles[0].SoftwarePackage.SelfService)

		installer, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
		require.NoError(t, err)
		require.NotNil(t, installer)

		installScript := installer.InstallScriptContentID
		postInstallScript := installer.PostInstallScriptContentID
		uninstallScript := installer.UninstallScriptContentID

		require.NotZero(t, installScript)
		require.NotZero(t, postInstallScript)
		require.NotZero(t, uninstallScript)
		require.Equal(t, "SELECT 1", installer.PreInstallQuery)

		for i := 2; i <= 3; i++ {
			// Simulate multiple gitops runs that each increment the FMA version.
			// This will lead to v1.0 getting evicted.
			err = ds.BatchSetSoftwareInstallers(ctx, ptr.Uint(team.ID), []*fleet.UploadSoftwareInstallerPayload{
				{
					FleetMaintainedAppID: ptr.Uint(fma.ID),
					Title:                "testpkg_fma",
					Source:               "apps",
					Platform:             "darwin",
					PreInstallQuery:      "SELECT 1 DIFFERENT",
					InstallScript:        "echo install 2",
					PostInstallScript:    "echo post install 2",
					UninstallScript:      "echo uninstall 2",
					InstallerFile:        tfr,
					StorageID:            "storageid2",
					Filename:             "test_fma.pkg",
					Version:              fmt.Sprintf("%d.0", i),
					UserID:               user.ID,
					ValidatedLabels:      &fleet.LabelIdentsWithScope{},
					InstallDuringSetup:   ptr.Bool(true),
					SelfService:          true,
					TeamID:               ptr.Uint(team.ID),
				},
			})
			require.NoError(t, err)
		}

		titles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: ptr.Uint(team.ID), Platform: "darwin", AvailableForInstall: true}, tmFilter)
		require.NoError(t, err)
		require.Len(t, titles, 1)
		require.Len(t, titles[0].SoftwarePackage.AutomaticInstallPolicies, 1)
		require.Equal(t, policy.ID, titles[0].SoftwarePackage.AutomaticInstallPolicies[0].ID)

		metadata, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, ptr.Uint(team.ID), titles[0].ID, false)
		require.NoError(t, err)

		policyAfterUpdate, err := ds.TeamPolicy(ctx, team.ID, policy.ID)
		require.NoError(t, err)
		require.Equal(t, metadata.InstallerID, *policyAfterUpdate.SoftwareInstallerID)
	})
}

func testCustomToFMAInstallerReplacement(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_custom_to_fma"})
	require.NoError(t, err)

	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "pkg1",
		Slug:             "pkg1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.pkg1",
	})
	require.NoError(t, err)

	tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
	require.NoError(t, err)

	// Seed a custom installer for title "pkg1".
	customInstallerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:              "pkg1",
		Source:             "apps",
		Platform:           "darwin",
		PreInstallQuery:    "SELECT 1",
		InstallScript:      "echo install",
		PostInstallScript:  "echo post install",
		UninstallScript:    "echo uninstall",
		InstallerFile:      tfr,
		StorageID:          "storageid_custom",
		Filename:           "pkg1.pkg",
		Version:            "1.0",
		UserID:             user.ID,
		ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		InstallDuringSetup: new(true),
		SelfService:        false,
		TeamID:             new(team.ID),
	})
	require.NoError(t, err)

	// Attach a policy to the custom installer so we also exercise the
	// policy re-point on deletion (policies.software_installer_id FK has no
	// ON DELETE CASCADE).
	policy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
		Name:                "p_custom_to_fma",
		Query:               "SELECT 1;",
		SoftwareInstallerID: &customInstallerID,
	})
	require.NoError(t, err)

	// Seed a display name for the title so we can verify it's updated in
	// place (rather than deleted + re-inserted) when the batch sets a new
	// display name.
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO software_title_display_names (team_id, software_title_id, display_name)
			VALUES (?, ?, ?)
		`, team.ID, titleID, "Initial Name")
		return err
	})
	var initialDisplayNameID uint
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &initialDisplayNameID, `
			SELECT id FROM software_title_display_names
			WHERE team_id = ? AND software_title_id = ?
		`, team.ID, titleID)
	})

	// GitOps run: same title, now an FMA payload with a display name.
	err = ds.BatchSetSoftwareInstallers(ctx, new(team.ID), []*fleet.UploadSoftwareInstallerPayload{
		{
			FleetMaintainedAppID: new(fma.ID),
			Title:                "pkg1",
			Source:               "apps",
			Platform:             "darwin",
			PreInstallQuery:      "SELECT 1 DIFFERENT",
			InstallScript:        "echo install 2",
			PostInstallScript:    "echo post install 2",
			UninstallScript:      "echo uninstall 2",
			InstallerFile:        tfr,
			StorageID:            "storageid_fma",
			Filename:             "pkg1_fma.pkg",
			Version:              "2.0",
			UserID:               user.ID,
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			InstallDuringSetup:   new(true),
			SelfService:          true,
			DisplayName:          "Cool Package",
			TeamID:               new(team.ID),
		},
	})
	require.NoError(t, err)

	tmFilter := fleet.TeamFilter{User: test.UserAdmin, TeamID: new(team.ID)}
	titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: new(team.ID), Platform: "darwin", AvailableForInstall: true}, tmFilter)
	require.NoError(t, err)
	require.Len(t, titles, 1)

	// The old custom row (fleet_maintained_app_id IS NULL) must be gone.
	var customRows int
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &customRows, `
			SELECT COUNT(*) FROM software_installers
			WHERE global_or_team_id = ? AND title_id = ? AND fleet_maintained_app_id IS NULL
		`, team.ID, titles[0].ID)
	})
	require.Zero(t, customRows, "stale custom installer row was not cleaned up after FMA replacement")

	// Exactly one active installer remains for the title, and it's the FMA.
	var activeCount int
	var activeFMAID *uint
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &activeCount, `
			SELECT COUNT(*) FROM software_installers
			WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1
		`, team.ID, titles[0].ID)
	})
	require.Equal(t, 1, activeCount)
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &activeFMAID, `
			SELECT fleet_maintained_app_id FROM software_installers
			WHERE global_or_team_id = ? AND title_id = ? AND is_active = 1
		`, team.ID, titles[0].ID)
	})
	require.NotNil(t, activeFMAID)
	require.Equal(t, fma.ID, *activeFMAID)

	// The policy that pointed at the deleted custom installer must have
	// been re-pointed to the new FMA row, not dropped/nulled.
	metadata, err := ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, new(team.ID), titles[0].ID, false)
	require.NoError(t, err)
	policyAfter, err := ds.TeamPolicy(ctx, team.ID, policy.ID)
	require.NoError(t, err)
	require.NotNil(t, policyAfter.SoftwareInstallerID)
	require.Equal(t, metadata.InstallerID, *policyAfter.SoftwareInstallerID)
	require.NotEqual(t, customInstallerID, *policyAfter.SoftwareInstallerID)

	// The display name from the batch payload must survive the side-effect
	// cleanup. The display_name row should be UPDATEd in place (same id),
	// not DELETEd and re-INSERTed.
	displayName, err := ds.getSoftwareTitleDisplayName(ctx, team.ID, titles[0].ID)
	require.NoError(t, err)
	require.Equal(t, "Cool Package", displayName)
	var afterDisplayNameID uint
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &afterDisplayNameID, `
			SELECT id FROM software_title_display_names
			WHERE team_id = ? AND software_title_id = ?
		`, team.ID, titles[0].ID)
	})
	require.Equal(t, initialDisplayNameID, afterDisplayNameID, "display_name row should be upserted in place, not deleted and re-inserted")

	// Same-version case: the custom installer and the incoming FMA share a
	// version string. Converting a custom package to an FMA replaces the row and
	// re-points its FKs (same as the different-version case above), leaving the
	// FMA as the single active row for the title.
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_custom_to_fma_same_version"})
	require.NoError(t, err)

	fma2, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "pkg2",
		Slug:             "pkg2",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.pkg2",
	})
	require.NoError(t, err)

	customInstallerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:              "pkg2",
		Source:             "apps",
		Platform:           "darwin",
		PreInstallQuery:    "SELECT 1",
		InstallScript:      "echo install",
		PostInstallScript:  "echo post install",
		UninstallScript:    "echo uninstall",
		InstallerFile:      tfr,
		StorageID:          "storageid_custom2",
		Filename:           "pkg2.pkg",
		Version:            "1.0",
		UserID:             user.ID,
		ValidatedLabels:    &fleet.LabelIdentsWithScope{},
		InstallDuringSetup: new(true),
		TeamID:             new(team2.ID),
	})
	require.NoError(t, err)

	err = ds.BatchSetSoftwareInstallers(ctx, new(team2.ID), []*fleet.UploadSoftwareInstallerPayload{
		{
			FleetMaintainedAppID: new(fma2.ID),
			Title:                "pkg2",
			Source:               "apps",
			Platform:             "darwin",
			PreInstallQuery:      "SELECT 1",
			InstallScript:        "echo install",
			PostInstallScript:    "echo post install",
			UninstallScript:      "echo uninstall",
			InstallerFile:        tfr,
			StorageID:            "storageid_fma2",
			Filename:             "pkg2_fma.pkg",
			Version:              "1.0", // same version as the custom installer
			UserID:               user.ID,
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			InstallDuringSetup:   new(true),
			TeamID:               new(team2.ID),
		},
	})
	require.NoError(t, err)

	tmFilter2 := fleet.TeamFilter{User: test.UserAdmin, TeamID: new(team2.ID)}
	titles2, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: new(team2.ID), Platform: "darwin", AvailableForInstall: true}, tmFilter2)
	require.NoError(t, err)
	require.Len(t, titles2, 1, "exactly one installer row should remain after same-version custom\u2192FMA conversion")

	var installerRows []struct {
		ID       uint  `db:"id"`
		FMAID    *uint `db:"fleet_maintained_app_id"`
		IsActive bool  `db:"is_active"`
	}
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, tx, &installerRows, `
			SELECT id, fleet_maintained_app_id, is_active FROM software_installers
			WHERE global_or_team_id = ? AND title_id = ?
		`, team2.ID, titles2[0].ID)
	})
	require.Len(t, installerRows, 1)
	require.NotEqual(t, customInstallerID2, installerRows[0].ID, "custom row should be replaced by the FMA row")
	require.NotNil(t, installerRows[0].FMAID, "row should have been converted to FMA")
	require.Equal(t, fma2.ID, *installerRows[0].FMAID)
	require.True(t, installerRows[0].IsActive)
}

func testGetInstallerByTeamAndURL(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	tfr, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)

	etag := `"abc123"`

	err = ds.BatchSetSoftwareInstallers(ctx, &team1.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallerFile:    tfr,
			BundleIdentifier: "com.example.app",
			Extension:        "pkg",
			StorageID:        "hash1",
			Filename:         "app.pkg",
			Title:            "App",
			Version:          "1.0",
			Source:           "apps",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			TeamID:           &team1.ID,
			Platform:         "darwin",
			URL:              "https://example.com/app/latest",
			HTTPETag:         &etag,
		},
	})
	require.NoError(t, err)

	// Correct team and URL returns the installer with ETag
	existing, err := ds.GetInstallerByTeamAndURL(ctx, &team1.ID, "https://example.com/app/latest")
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.Equal(t, "hash1", existing.StorageID)
	assert.Equal(t, "app.pkg", existing.Filename)
	require.NotNil(t, existing.HTTPETag)
	assert.Equal(t, etag, *existing.HTTPETag)

	// Wrong team returns nil
	existing, err = ds.GetInstallerByTeamAndURL(ctx, &team2.ID, "https://example.com/app/latest")
	require.NoError(t, err)
	assert.Nil(t, existing)

	// Wrong URL returns nil
	existing, err = ds.GetInstallerByTeamAndURL(ctx, &team1.ID, "https://example.com/other")
	require.NoError(t, err)
	assert.Nil(t, existing)

	// nil team (cross-team fallback) returns the installer from any team
	existing, err = ds.GetInstallerByTeamAndURL(ctx, nil, "https://example.com/app/latest")
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.Equal(t, "hash1", existing.StorageID)
	assert.Equal(t, "app.pkg", existing.Filename)

	// nil team with non-existent URL returns nil
	existing, err = ds.GetInstallerByTeamAndURL(ctx, nil, "https://example.com/nonexistent")
	require.NoError(t, err)
	assert.Nil(t, existing)

	// URL with query params (GlobalProtect pattern)
	err = ds.BatchSetSoftwareInstallers(ctx, &team1.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallerFile:    tfr,
			BundleIdentifier: "com.example.gp",
			Extension:        "msi",
			StorageID:        "hash2",
			Filename:         "gp.msi",
			Title:            "GlobalProtect",
			Version:          "1.0",
			Source:           "programs",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			TeamID:           &team1.ID,
			Platform:         "windows",
			URL:              "https://example.com/gp?version=64&platform=windows",
			HTTPETag:         &etag,
		},
	})
	require.NoError(t, err)

	existing, err = ds.GetInstallerByTeamAndURL(ctx, &team1.ID, "https://example.com/gp?version=64&platform=windows")
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.Equal(t, "hash2", existing.StorageID)

	// Simulate an FMA rollback: two rows for the same (team, URL), where the
	// inactive row has a higher id than the active row. The lookup must return
	// the active row even though ORDER BY id DESC would otherwise pick the
	// inactive one.
	rollbackURL := "https://example.com/rollback"
	err = ds.BatchSetSoftwareInstallers(ctx, &team2.ID, []*fleet.UploadSoftwareInstallerPayload{
		{
			InstallerFile:    tfr,
			BundleIdentifier: "com.example.rb",
			Extension:        "pkg",
			StorageID:        "active_hash",
			Filename:         "rb.pkg",
			Title:            "Rollback",
			Version:          "1.0",
			Source:           "apps",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			TeamID:           &team2.ID,
			Platform:         "darwin",
			URL:              rollbackURL,
			HTTPETag:         &etag,
		},
	})
	require.NoError(t, err)

	inactiveETag := `"inactive-etag"`
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, storage_id, filename, extension, version, platform, title_id,
				 install_script_content_id, uninstall_script_content_id, is_active, url, package_ids, patch_query, http_etag)
			SELECT team_id, global_or_team_id, 'inactive_hash', filename, extension, 'old_version', platform, title_id,
				install_script_content_id, uninstall_script_content_id, 0, url, package_ids, patch_query, ?
			FROM software_installers WHERE team_id = ? AND url = ?
		`, inactiveETag, team2.ID, rollbackURL)
		return err
	})

	existing, err = ds.GetInstallerByTeamAndURL(ctx, &team2.ID, rollbackURL)
	require.NoError(t, err)
	require.NotNil(t, existing)
	assert.Equal(t, "active_hash", existing.StorageID, "must return the active installer, not the inactive duplicate")
	require.NotNil(t, existing.HTTPETag)
	assert.Equal(t, etag, *existing.HTTPETag)
}

func testBatchSetFMACancelsPendingOnActiveRow(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_fma_ordering"})
	require.NoError(t, err)
	host := test.NewHost(t, ds, "host_fma_ordering", "1", "hostkey_fma_ordering", "hostuuid_fma_ordering", time.Now())

	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "pkg_ord",
		Slug:             "pkg_ord",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.pkg_ord",
	})
	require.NoError(t, err)

	basePayload := func(version, storageID, installScript string) *fleet.UploadSoftwareInstallerPayload {
		tfr, err := fleet.NewTempFileReader(strings.NewReader(storageID), t.TempDir)
		require.NoError(t, err)
		return &fleet.UploadSoftwareInstallerPayload{
			FleetMaintainedAppID: &fma.ID,
			Title:                "pkg_ord",
			Source:               "apps",
			Platform:             "darwin",
			PreInstallQuery:      "SELECT 1",
			InstallScript:        installScript,
			PostInstallScript:    "echo post",
			UninstallScript:      "echo uninstall",
			InstallerFile:        tfr,
			StorageID:            storageID,
			Filename:             "pkg_ord.pkg",
			Version:              version,
			UserID:               user.ID,
			ValidatedLabels:      &fleet.LabelIdentsWithScope{},
			TeamID:               &team.ID,
		}
	}

	// v1.0 first (active), then v2.0 (active, v1 demoted to inactive cache).
	require.NoError(t, ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{basePayload("1.0", "storage_v1", "echo v1")}))
	require.NoError(t, ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{basePayload("2.0", "storage_v2", "echo v2")}))

	var v2ID uint
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &v2ID, `
			SELECT id FROM software_installers
			WHERE global_or_team_id = ? AND version = ? AND is_active = 1
		`, team.ID, "2.0")
	})

	_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, v2ID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	var pending int
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &pending, `
			SELECT COUNT(*) FROM software_install_upcoming_activities
			WHERE software_installer_id = ?
		`, v2ID)
	})
	require.Equal(t, 1, pending)

	require.NoError(t, ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{basePayload("2.0", "storage_v2_updated", "echo v2 updated")}))

	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &pending, `
			SELECT COUNT(*) FROM software_install_upcoming_activities
			WHERE software_installer_id = ?
		`, v2ID)
	})
	require.Zero(t, pending, "re-submitting the active FMA version must cancel its pending installs")
}

func testSoftwareInstallerTitleIDValidation(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	payload := func(title, bundleID, filename, storageID string) *fleet.UploadSoftwareInstallerPayload {
		return &fleet.UploadSoftwareInstallerPayload{
			StorageID:        storageID,
			Filename:         filename,
			Title:            title,
			BundleIdentifier: bundleID,
			Extension:        "pkg",
			Source:           "apps",
			Platform:         "darwin",
			Version:          "1.0",
			UserID:           user.ID,
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
			TeamID:           &team.ID,
		}
	}

	_, targetTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, payload("Target", "com.example.target", "target.pkg", "target-v1"))
	require.NoError(t, err)
	_, otherTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, payload("Other", "com.example.other", "other.pkg", "other-v1"))
	require.NoError(t, err)

	matching := payload("Target", "com.example.target", "target-v2.pkg", "target-v2")
	matching.TitleID = &targetTitleID
	_, gotTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, matching)
	require.NoError(t, err)
	require.Equal(t, targetTitleID, gotTitleID)

	rowCounts := func() (titles, installers int) {
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			if err := sqlx.GetContext(ctx, q, &titles, `SELECT COUNT(*) FROM software_titles`); err != nil {
				return err
			}
			return sqlx.GetContext(ctx, q, &installers, `SELECT COUNT(*) FROM software_installers`)
		})
		return titles, installers
	}

	assertRejectedWithoutWrites := func(p *fleet.UploadSoftwareInstallerPayload) {
		t.Helper()
		beforeTitles, beforeInstallers := rowCounts()
		_, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, p)
		require.ErrorContains(t, err, fmt.Sprintf(fleet.SoftwarePackageTitleMismatchMessage, p.Filename))
		afterTitles, afterInstallers := rowCounts()
		require.Equal(t, beforeTitles, afterTitles)
		require.Equal(t, beforeInstallers, afterInstallers)
	}

	mismatching := payload("Target", "com.example.target", "target-mismatch.pkg", "target-mismatch")
	mismatching.TitleID = &otherTitleID
	assertRejectedWithoutWrites(mismatching)

	nonexistentTitleID := uint(999999)
	nonexistent := payload("Target", "com.example.target", "target-nonexistent.pkg", "target-nonexistent")
	nonexistent.TitleID = &nonexistentTitleID
	assertRejectedWithoutWrites(nonexistent)

	noResolvedTitle := payload("New", "com.example.new", "new.pkg", "new")
	noResolvedTitle.TitleID = &targetTitleID
	assertRejectedWithoutWrites(noResolvedTitle)

	withoutTitleID := payload("Unspecified", "com.example.unspecified", "unspecified.pkg", "unspecified")
	_, newTitleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, withoutTitleID)
	require.NoError(t, err)
	require.NotZero(t, newTitleID)
}

func testMatchOrCreateSoftwareInstallerDuplicateConflicts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	const conflictMsg = "already has an Apple App Store (VPP) on"

	// macOS installer conflicting with a VPP app on the same bundle id.
	test.CreateInsertGlobalVPPToken(t, ds)
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "adam_vpp_mac", Platform: fleet.MacOSPlatform}},
		Name:             "Mac VPP",
		BundleIdentifier: "com.example.vpp",
	}, &team.ID)
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "mac-vpp-clash-storage",
		Filename:         "vpp-clash.pkg",
		Title:            "Mac VPP Clash",
		BundleIdentifier: "com.example.vpp",
		Extension:        "pkg",
		Source:           "apps",
		Platform:         "darwin",
		Version:          "1.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.ErrorContains(t, err, conflictMsg)

	// macOS installer sharing a bundle id with an in-house app is allowed:
	// in-house apps only target iOS/iPadOS, so they don't conflict with macOS.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "iha-storage",
		Filename:         "iha.ipa",
		Title:            "iOS App",
		BundleIdentifier: "com.example.iha",
		Extension:        "ipa",
		Source:           "ios_apps",
		Platform:         "ios",
		Version:          "1.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "mac-iha-coexist-storage",
		Filename:         "mac-iha-coexist.pkg",
		Title:            "Mac IHA Coexist",
		BundleIdentifier: "com.example.iha",
		Extension:        "pkg",
		Source:           "apps",
		Platform:         "darwin",
		Version:          "1.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.NoError(t, err)

	// macOS: a second version of the same title is allowed (multiple packages per title).
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "mac-base-storage",
		Filename:         "mac-app.pkg",
		Title:            "Mac App",
		BundleIdentifier: "com.example.mac",
		Extension:        "pkg",
		Source:           "apps",
		Platform:         "darwin",
		Version:          "1.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "mac-v2-storage",
		Filename:         "mac-app-v2.pkg",
		Title:            "Mac App",
		BundleIdentifier: "com.example.mac",
		Extension:        "pkg",
		Source:           "apps",
		Platform:         "darwin",
		Version:          "2.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.NoError(t, err)

	// Windows: a second version of the same title is allowed.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-base-storage",
		Filename:        "win-app.msi",
		Title:           "Win App",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "1.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-v2-storage",
		Filename:        "win-app-v2.msi",
		Title:           "Win App",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "2.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// Windows: a second package matching the same upgrade code is allowed.
	const winUpgradeCode = "{ABCDEF12-3456-7890-ABCD-EF1234567890}"
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-uc-base-storage",
		Filename:        "win-uc.msi",
		Title:           "Win UC App",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "1.0",
		UpgradeCode:     winUpgradeCode,
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-uc-v2-storage",
		Filename:        "win-uc-other.msi",
		Title:           "Win UC App Renamed",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "2.0",
		UpgradeCode:     winUpgradeCode,
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// Windows: existing installer has an upgrade code, new upload has the same
	// Title but no upgrade code.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-uc-existing-storage",
		Filename:        "win-uc-existing.msi",
		Title:           "Win UC Same Name",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "1.0",
		UpgradeCode:     "{11111111-1111-1111-1111-111111111111}",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-uc-noupgrade-storage",
		Filename:        "win-uc-custom.msi",
		Title:           "Win UC Same Name",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "2.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// Reverse: existing installer has no upgrade code, new upload has the same
	// Title with an upgrade code.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-plain-base-storage",
		Filename:        "win-plain.msi",
		Title:           "Win Plain App",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "1.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "win-plain-uc-storage",
		Filename:        "win-plain-uc.msi",
		Title:           "Win Plain App",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "2.0",
		UpgradeCode:     "{22222222-2222-2222-2222-222222222222}",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// Linux: a second version of the same title is allowed.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "linux-base-storage",
		Filename:        "linux-app.deb",
		Title:           "Linux App",
		Extension:       "deb",
		Source:          "deb_packages",
		Platform:        "linux",
		Version:         "1.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "linux-v2-storage",
		Filename:        "linux-app-v2.deb",
		Title:           "Linux App",
		Extension:       "deb",
		Source:          "deb_packages",
		Platform:        "linux",
		Version:         "2.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// Linux .deb: a duplicate content hash on the title is rejected.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "linux-base-storage",
		Filename:        "linux-app-dup.deb",
		Title:           "Linux App",
		Extension:       "deb",
		Source:          "deb_packages",
		Platform:        "linux",
		Version:         "3.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.ErrorContains(t, err, "same SHA-256 hash")

	// Linux .rpm: a second build is allowed, a duplicate content hash is rejected.
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "rpm-base-storage",
		Filename:        "linux-app.rpm",
		Title:           "Linux RPM App",
		Extension:       "rpm",
		Source:          "rpm_packages",
		Platform:        "linux",
		Version:         "1.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "rpm-base-storage",
		Filename:        "linux-app-dup.rpm",
		Title:           "Linux RPM App",
		Extension:       "rpm",
		Source:          "rpm_packages",
		Platform:        "linux",
		Version:         "2.0",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.ErrorContains(t, err, "same SHA-256 hash")

	// Same title and version but different content is allowed (e.g. Arm vs Intel builds).
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "arch-storage-arm",
		Filename:         "arch-app-arm.pkg",
		Title:            "Arch App",
		BundleIdentifier: "com.example.arch",
		Extension:        "pkg",
		Source:           "apps",
		Platform:         "darwin",
		Version:          "1.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.NoError(t, err)

	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:        "arch-storage-intel",
		Filename:         "arch-app-intel.pkg",
		Title:            "Arch App",
		BundleIdentifier: "com.example.arch",
		Extension:        "pkg",
		Source:           "apps",
		Platform:         "darwin",
		Version:          "1.0",
		UserID:           user.ID,
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		TeamID:           &team.ID,
	})
	require.NoError(t, err)

	// A title holds at most fleet.MaxPackagesPerTitle packages, so the next one is rejected.
	for i := range fleet.MaxPackagesPerTitle {
		_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			StorageID:       fmt.Sprintf("limit-storage-%d", i),
			Filename:        fmt.Sprintf("limit-%d.msi", i),
			Title:           "Limit App",
			Extension:       "msi",
			Source:          "programs",
			Platform:        "windows",
			Version:         fmt.Sprintf("1.%d", i),
			UserID:          user.ID,
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
			TeamID:          &team.ID,
		})
		require.NoError(t, err)
	}
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		StorageID:       "limit-storage-extra",
		Filename:        "limit-extra.msi",
		Title:           "Limit App",
		Extension:       "msi",
		Source:          "programs",
		Platform:        "windows",
		Version:         "9.9",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
		TeamID:          &team.ID,
	})
	require.ErrorContains(t, err, fmt.Sprintf("already has %d packages", fleet.MaxPackagesPerTitle))
}

func testGetSoftwareTitlesForInstallAll(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	// drive activation explicitly so each install/uninstall lands in a known state
	ds.testActivateSpecificNextActivities = []string{"-"}
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })

	user := test.NewUser(t, ds, "iall author", "iall-author@example.com", true)
	host := test.NewHost(t, ds, "iall-host", "", "iall-key", "iall-uuid", time.Now(), test.WithPlatform("ubuntu"))

	cat, err := ds.NewSoftwareCategory(ctx, 0, "iall-utilities")
	require.NoError(t, err)

	// host is a member of this label; used to verify label scoping is applied
	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "iall-label", Query: "SELECT 1"})
	require.NoError(t, err)
	require.NoError(t, ds.RecordLabelQueryExecutions(ctx, host, map[uint]*bool{lbl.ID: new(true)}, time.Now(), false))

	noLabels := fleet.LabelIdentsWithScope{}

	newInstaller := func(name string, selfService bool, categoryIDs []uint, teamID *uint, labels fleet.LabelIdentsWithScope) (uint, uint) {
		installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			StorageID:       name + "-storage",
			Filename:        name + ".deb",
			Title:           name,
			Extension:       "deb",
			Source:          "deb_packages",
			Platform:        "linux",
			Version:         "1.0",
			InstallScript:   "install",
			UninstallScript: "uninstall",
			SelfService:     selfService,
			UserID:          user.ID,
			CategoryIDs:     categoryIDs,
			ValidatedLabels: &labels,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return installerID, titleID
	}

	completeInstall := func(installerID uint, exitCode int) {
		uid, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, fleet.HostSoftwareInstallOptions{SelfService: true})
		require.NoError(t, err)
		ds.testActivateSpecificNextActivities = []string{uid}
		activated, err := ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host.ID, "")
		require.NoError(t, err)
		require.Equal(t, []string{uid}, activated)
		ds.testActivateSpecificNextActivities = []string{"-"}
		_, err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
			HostID:                    host.ID,
			InstallUUID:               uid,
			PreInstallConditionOutput: new("ok"),
			InstallScriptExitCode:     new(exitCode),
		}, nil)
		require.NoError(t, err)
	}

	completeUninstall := func(installerID uint, exitCode int) {
		uid := uuid.NewString()
		require.NoError(t, ds.InsertSoftwareUninstallRequest(ctx, uid, host.ID, installerID, true))
		ds.testActivateSpecificNextActivities = []string{uid}
		activated, err := ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), host.ID, "")
		require.NoError(t, err)
		require.Equal(t, []string{uid}, activated)
		ds.testActivateSpecificNextActivities = []string{"-"}
		_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
			HostID:      host.ID,
			ExecutionID: uid,
			ExitCode:    exitCode,
		}, nil)
		require.NoError(t, err)
	}

	names := func(titles []*fleet.HostSoftwareWithInstaller) []string {
		out := make([]string, 0, len(titles))
		for _, ti := range titles {
			out = append(out, ti.Name)
		}
		return out
	}

	// available to install: available (also in the category), a successfully-uninstalled
	// title, and a title scoped to a label the host is a member of
	newInstaller("available", true, []uint{cat.ID}, nil, noLabels)
	uninstalledID, _ := newInstaller("uninstalled", true, nil, nil, noLabels)
	newInstaller("label-in", true, nil, nil, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeIncludeAny,
		ByName:     map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}},
	})

	// previously installed/pending titles are skipped; failed_install and failed_uninstall
	// are included so install_all re-queues them (matches per-row Retry).
	installedID, _ := newInstaller("installed", true, nil, nil, noLabels)
	installedUpdateID, _ := newInstaller("installed-update", true, nil, nil, noLabels)
	failedID, _ := newInstaller("failed", true, nil, nil, noLabels)
	failedUninstallID, _ := newInstaller("failed-uninstall", true, nil, nil, noLabels)
	pendingID, _ := newInstaller("pending", true, nil, nil, noLabels)
	newInstaller("inventory", true, nil, nil, noLabels)
	newInstaller("not-self-service", false, nil, nil, noLabels)
	// out of scope: host is a member of the exclude-any label
	newInstaller("label-out", true, nil, nil, fleet.LabelIdentsWithScope{
		LabelScope: fleet.LabelScopeExcludeAny,
		ByName:     map[string]fleet.LabelIdent{lbl.Name: {LabelID: lbl.ID, LabelName: lbl.Name}},
	})

	completeInstall(installedID, 0)
	completeInstall(installedUpdateID, 0)
	completeInstall(failedID, 1)
	completeInstall(uninstalledID, 0)
	completeUninstall(uninstalledID, 0)
	completeInstall(failedUninstallID, 0)
	completeUninstall(failedUninstallID, 1)

	// inventory: one title present but never installed by Fleet, and one Fleet-installed
	// title whose inventory version is older than the 1.0 installer (update available)
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{
		{Name: "inventory", Version: "0.5", Source: "deb_packages"},
		{Name: "installed-update", Version: "0.5", Source: "deb_packages"},
	})
	require.NoError(t, err)

	// pending is last: it keeps the queue head occupied
	_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, pendingID, fleet.HostSoftwareInstallOptions{SelfService: true})
	require.NoError(t, err)

	// no category: only the available titles, returned in alphabetical order by name.
	// failed_install and failed_uninstall are included so install_all re-queues them.
	got, categoryName, err := ds.GetSoftwareTitlesForInstallAll(ctx, host, nil)
	require.NoError(t, err)
	require.Nil(t, categoryName)
	require.Equal(t, []string{"available", "failed", "failed-uninstall", "label-in", "uninstalled"}, names(got))

	// scoped to a category: only the in-category title, and the name is returned
	got, categoryName, err = ds.GetSoftwareTitlesForInstallAll(ctx, host, &cat.ID)
	require.NoError(t, err)
	require.NotNil(t, categoryName)
	require.Equal(t, cat.Name, *categoryName)
	require.Equal(t, []string{"available"}, names(got))

	// nonexistent category, or a category belonging to another team -> bad request
	_, _, err = ds.GetSoftwareTitlesForInstallAll(ctx, host, new(uint(9_999_999)))
	var bre *fleet.BadRequestError
	require.ErrorAs(t, err, &bre)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "iall-team"})
	require.NoError(t, err)
	teamCat, err := ds.NewSoftwareCategory(ctx, team.ID, "iall-team-cat")
	require.NoError(t, err)
	_, _, err = ds.GetSoftwareTitlesForInstallAll(ctx, host, &teamCat.ID)
	require.ErrorAs(t, err, &bre)

	// team scoping: a team host sees only its team's self-service installer
	teamHost := test.NewHost(t, ds, "iall-team-host", "", "iall-team-key", "iall-team-uuid", time.Now(), test.WithPlatform("ubuntu"))
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{teamHost.ID})))
	teamHost, err = ds.Host(ctx, teamHost.ID)
	require.NoError(t, err)
	newInstaller("team-app", true, nil, &team.ID, noLabels)
	got, _, err = ds.GetSoftwareTitlesForInstallAll(ctx, teamHost, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"team-app"}, names(got))

	// VPP apps and package installers are returned together, sorted by name (not
	// grouped by type). Verify on an MDM-connected darwin host, where both a macOS
	// package and a VPP app are available (the ubuntu host above is not MDM-connected,
	// so VPP apps never apply to it).
	test.CreateInsertGlobalVPPToken(t, ds)
	macTeam, err := ds.NewTeam(ctx, &fleet.Team{Name: "iall-mac-team"})
	require.NoError(t, err)
	macHost := test.NewHost(t, ds, "iall-mac-host", "", "iall-mac-key", "iall-mac-uuid", time.Now(), test.WithPlatform("darwin"))
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&macTeam.ID, []uint{macHost.ID})))
	macHost, err = ds.Host(ctx, macHost.ID)
	require.NoError(t, err)
	nanoEnrollAndSetHostMDMData(t, ds, macHost, false)

	newMacOSInstaller := func(name string) {
		_, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			StorageID:       name + "-storage",
			Filename:        name + ".pkg",
			Title:           name,
			Extension:       "pkg",
			Source:          "apps",
			Platform:        "darwin",
			Version:         "1.0",
			InstallScript:   "install",
			UninstallScript: "uninstall",
			SelfService:     true,
			UserID:          user.ID,
			TeamID:          &macTeam.ID,
			ValidatedLabels: &fleet.LabelIdentsWithScope{},
		})
		require.NoError(t, err)
	}
	newMacOSInstaller("chrome") // package
	newMacOSInstaller("zoom")   // package
	_, err = ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "slack", // VPP app; sorts between the two packages
		BundleIdentifier: "com.example.slack",
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID:    fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform},
			SelfService: true,
		},
	}, &macTeam.ID)
	require.NoError(t, err)
	got, _, err = ds.GetSoftwareTitlesForInstallAll(ctx, macHost, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"chrome", "slack", "zoom"}, names(got))
}

func testSoftwareTitlePins(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Maintained1",
		Slug:             "maintained1",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
	require.NoError(t, err)
	_, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title:                "testpkg",
		Source:               "apps",
		Platform:             "darwin",
		InstallScript:        "echo install",
		UninstallScript:      "echo uninstall",
		InstallerFile:        tfr,
		StorageID:            "storageid1",
		Filename:             "test.pkg",
		Version:              "1.0",
		UserID:               user.ID,
		ValidatedLabels:      &fleet.LabelIdentsWithScope{},
		FleetMaintainedAppID: new(fma.ID),
	})
	require.NoError(t, err)

	noTeam := new(uint(0))
	otherTeam := new(uint(42))

	// No row -> not found; the caller treats this as "Latest".
	_, err = ds.GetPinnedVersion(ctx, noTeam, titleID)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// A literal pin round-trips.
	require.NoError(t, ds.SetPinnedVersion(ctx, noTeam, titleID, "1.0"))
	pin, err := ds.GetPinnedVersion(ctx, noTeam, titleID)
	require.NoError(t, err)
	require.Equal(t, new("1.0"), pin)

	// Upsert overwrites in place (literal -> caret).
	require.NoError(t, ds.SetPinnedVersion(ctx, noTeam, titleID, "^1"))
	pin, err = ds.GetPinnedVersion(ctx, noTeam, titleID)
	require.NoError(t, err)
	require.Equal(t, new("^1"), pin)

	// A different team's pin on the same title is independent.
	require.NoError(t, ds.SetPinnedVersion(ctx, otherTeam, titleID, "2.0"))
	pin, err = ds.GetPinnedVersion(ctx, otherTeam, titleID)
	require.NoError(t, err)
	require.Equal(t, new("2.0"), pin)
	pin, err = ds.GetPinnedVersion(ctx, noTeam, titleID)
	require.NoError(t, err)
	require.Equal(t, new("^1"), pin)

	// Deleting one team's pin leaves the other intact; deleting again is a no-op.
	require.NoError(t, ds.DeletePinnedVersion(ctx, noTeam, titleID))
	_, err = ds.GetPinnedVersion(ctx, noTeam, titleID)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, ds.DeletePinnedVersion(ctx, noTeam, titleID))
	pin, err = ds.GetPinnedVersion(ctx, otherTeam, titleID)
	require.NoError(t, err)
	require.Equal(t, new("2.0"), pin)
}

func testSetFleetMaintainedAppActiveInstallerPin(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name: "Maintained1", Slug: "maintained1", Platform: "darwin", UniqueIdentifier: "fleet.maintained1",
	})
	require.NoError(t, err)

	tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
	require.NoError(t, err)
	v1ID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Title: "testpkg", Source: "apps", Platform: "darwin",
		InstallScript: "echo install", UninstallScript: "echo uninstall",
		InstallerFile: tfr, StorageID: "storageid1", Filename: "test.pkg", Version: "1.0",
		UserID: user.ID, ValidatedLabels: &fleet.LabelIdentsWithScope{}, FleetMaintainedAppID: new(fma.ID),
	})
	require.NoError(t, err)

	// Add a second cached version (inactive) for the same no-team title.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, storage_id, filename, extension, version, platform, title_id,
				 fleet_maintained_app_id, install_script_content_id, uninstall_script_content_id, is_active, package_ids, patch_query)
			SELECT team_id, global_or_team_id, 'storageid2', 'test2.pkg', extension, '2.0', platform, title_id,
				fleet_maintained_app_id, install_script_content_id, uninstall_script_content_id, 0, package_ids, patch_query
			FROM software_installers WHERE id = ?
		`, v1ID)
		return err
	})
	var v2ID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &v2ID, `SELECT id FROM software_installers WHERE title_id=? AND global_or_team_id=0 AND version='2.0'`, titleID)
	})

	// GetFleetMaintainedVersionsByTitleID returns each cached version's own filename.
	fmaVersions, err := ds.GetFleetMaintainedVersionsByTitleID(ctx, nil, titleID, false)
	require.NoError(t, err)
	gotFilenames := map[string]string{}
	for _, fv := range fmaVersions {
		gotFilenames[fv.Version] = fv.Filename
	}
	require.Equal(t, map[string]string{"1.0": "test.pkg", "2.0": "test2.pkg"}, gotFilenames)

	activeID := func() uint {
		var id uint
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT id FROM software_installers WHERE title_id=? AND global_or_team_id=0 AND is_active=1`, titleID)
		})
		return id
	}

	// A non-nil pin is authoritative: it upserts the pin row.
	require.NoError(t, ds.SetFleetMaintainedAppActiveInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{TitleID: titleID, PinnedVersion: new("^1")}, v1ID))
	require.Equal(t, v1ID, activeID())
	pin, err := ds.GetPinnedVersion(ctx, nil, titleID)
	require.NoError(t, err)
	require.Equal(t, new("^1"), pin)

	// A nil pin flips the active installer but leaves the pin row untouched —
	// this is what the auto-update cron relies on to avoid clobbering an admin's pin.
	require.NoError(t, ds.SetFleetMaintainedAppActiveInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{TitleID: titleID, PinnedVersion: nil}, v2ID))
	require.Equal(t, v2ID, activeID())
	pin, err = ds.GetPinnedVersion(ctx, nil, titleID)
	require.NoError(t, err)
	require.Equal(t, new("^1"), pin) // unchanged

	// A non-nil empty pin clears it (Latest).
	require.NoError(t, ds.SetFleetMaintainedAppActiveInstaller(ctx, &fleet.UpdateSoftwareInstallerPayload{TitleID: titleID, PinnedVersion: new("")}, v1ID))
	require.Equal(t, v1ID, activeID())
	_, err = ds.GetPinnedVersion(ctx, nil, titleID)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// A host with two queued installs for the same installer (one lower priority,
// the other later created_at) must still be counted once: the old OR-based
// anti-join let each row dominate the other and dropped the host entirely.
func testSummaryUpcomingPerHostNoDropout(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })
	// Don't auto-activate; we want both rows to sit in upcoming_activities.
	ds.testActivateSpecificNextActivities = []string{"-"}

	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())

	tfr, err := fleet.NewTempFileReader(strings.NewReader("install"), t.TempDir)
	require.NoError(t, err)
	installerID, titleID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install",
		UninstallScript: "uninstall",
		InstallerFile:   tfr,
		StorageID:       "dropout-storage",
		Filename:        "dropout.pkg",
		Title:           "Dropout App",
		Version:         "1.0",
		Source:          "apps",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Insert two upcoming software_install rows for the same host+installer that
	// cross-dominate under the old OR predicate: row B has the lower priority,
	// row A has the later created_at.
	insertUpcoming := func(execID string, priority int, createdOffsetMicros int) {
		res, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO upcoming_activities
	(host_id, priority, fleet_initiated, activity_type, execution_id, payload, created_at)
VALUES
	(?, ?, 1, 'software_install', ?, JSON_OBJECT('self_service', false), NOW(6) + INTERVAL ? MICROSECOND)`,
			host.ID, priority, execID, createdOffsetMicros)
		require.NoError(t, err)
		uaID, err := res.LastInsertId()
		require.NoError(t, err)
		_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO software_install_upcoming_activities
	(upcoming_activity_id, software_installer_id, software_title_id)
VALUES (?, ?, ?)`, uaID, installerID, titleID)
		require.NoError(t, err)
	}
	insertUpcoming("dropout-B", -1, 0)  // lower priority, earlier created_at
	insertUpcoming("dropout-A", 0, 100) // higher priority, later created_at

	summary, err := ds.GetSummaryHostSoftwareInstalls(ctx, installerID)
	require.NoError(t, err)
	// The host must be counted exactly once (not dropped, not double-counted).
	require.Equal(t, fleet.SoftwareInstallerStatusSummary{PendingInstall: 1}, *summary)
}

// testDeleteSoftwareInstallerRepointsPolicies verifies that deleting one package of several re-points
// install-automation policies to the first-added surviving package, while deleting the last package a
// policy references still returns the 409 telling the admin to disable the automation first.
func testDeleteSoftwareInstallerRepointsPolicies(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Delete Repoint", "delete-repoint@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "delete-repoint-team"})
	require.NoError(t, err)

	newPkg := func(storage, filename string) uint {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("hello-"+storage), t.TempDir)
		require.NoError(t, err)
		id, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			InstallerFile:    tfr,
			StorageID:        storage,
			Filename:         filename,
			Title:            "RepointApp",
			Version:          "1.0",
			Source:           "apps",
			BundleIdentifier: "com.example.repoint",
			UserID:           user.ID,
			TeamID:           &team.ID,
			Platform:         "darwin",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		})
		require.NoError(t, err)
		return id
	}
	installerA := newPkg("repoint-a", "pkgA.pkg")
	installerB := newPkg("repoint-b", "pkgB.pkg")
	require.Less(t, installerA, installerB)

	pol, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
		Name:                "repoint policy",
		Query:               "SELECT 1;",
		SoftwareInstallerID: &installerA,
	})
	require.NoError(t, err)

	// Deleting the referenced package while a sibling remains re-points the policy to the survivor.
	require.NoError(t, ds.DeleteSoftwareInstaller(ctx, installerA))
	got, err := ds.Policy(ctx, pol.ID)
	require.NoError(t, err)
	require.NotNil(t, got.SoftwareInstallerID)
	require.Equal(t, installerB, *got.SoftwareInstallerID)

	// Deleting the last package the policy references is refused (disable automation first).
	err = ds.DeleteSoftwareInstaller(ctx, installerB)
	require.Error(t, err)
	require.ErrorIs(t, err, errDeleteInstallerWithAssociatedInstallPolicy)

	// The policy still points at the (undeleted) package.
	got, err = ds.Policy(ctx, pol.ID)
	require.NoError(t, err)
	require.NotNil(t, got.SoftwareInstallerID)
	require.Equal(t, installerB, *got.SoftwareInstallerID)
}
