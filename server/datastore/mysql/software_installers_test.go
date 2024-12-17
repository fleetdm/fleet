package mysql

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
		{"GetSoftwareInstallerMetadataByTeamAndTitleID", testGetSoftwareInstallerMetadataByTeamAndTitleID},
		{"HasSelfServiceSoftwareInstallers", testHasSelfServiceSoftwareInstallers},
		{"DeleteSoftwareInstallers", testDeleteSoftwareInstallers},
		{"testDeletePendingSoftwareInstallsForPolicy", testDeletePendingSoftwareInstallsForPolicy},
		{"GetHostLastInstallData", testGetHostLastInstallData},
		{"GetOrGenerateSoftwareInstallerTitleID", testGetOrGenerateSoftwareInstallerTitleID},
		{"BatchSetSoftwareInstallersScopedViaLabels", testBatchSetSoftwareInstallersScopedViaLabels},
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

	host1 := test.NewHost(t, ds, "host1", "1", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "2", "host2key", "host2uuid", time.Now())
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
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

	hostInstall1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID1, false, nil)
	require.NoError(t, err)

	hostInstall2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID2, false, nil)
	require.NoError(t, err)

	hostInstall3, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID1, false, nil)
	require.NoError(t, err)

	hostInstall4, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID2, false, nil)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           hostInstall4,
		InstallScriptExitCode: ptr.Int(0),
	})
	require.NoError(t, err)

	hostInstall5, err := ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID2, false, nil)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host2.ID,
		InstallUUID:               hostInstall5,
		PreInstallConditionOutput: ptr.String(""), // pre-install query did not return results, so install failed
	})
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
	require.Equal(t, "hello", exec1.InstallScript)
	require.Equal(t, "world", exec1.PostInstallScript)
	require.Equal(t, installerID1, exec1.InstallerID)
	require.Equal(t, "SELECT 1", exec1.PreInstallCondition)
	require.False(t, exec1.SelfService)
	assert.Equal(t, "goodbye", exec1.UninstallScript)

	hostInstall6, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID3, true, nil)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                    host1.ID,
		InstallUUID:               hostInstall6,
		PreInstallConditionOutput: ptr.String("output"),
	})
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
}

func testSoftwareInstallRequests(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

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

			// non-existent host
			_, err = ds.InsertSoftwareInstallRequest(ctx, 12, si.InstallerID, false, nil)
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
			_, err = ds.InsertSoftwareInstallRequest(ctx, hostPendingInstall.ID, si.InstallerID, false, nil)
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
			_, err = ds.InsertSoftwareInstallRequest(ctx, hostFailedInstall.ID, si.InstallerID, false, nil)
			require.NoError(t, err)
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err = q.ExecContext(ctx, `
					UPDATE host_software_installs SET install_script_exit_code = 1 WHERE host_id = ? AND software_installer_id = ?`,
					hostFailedInstall.ID, si.InstallerID)
				require.NoError(t, err)
				return nil
			})

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
			_, err = ds.InsertSoftwareInstallRequest(ctx, hostInstalled.ID, si.InstallerID, false, nil)
			require.NoError(t, err)
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err = q.ExecContext(ctx, `
					UPDATE host_software_installs SET install_script_exit_code = 0 WHERE host_id = ? AND software_installer_id = ?`,
					hostInstalled.ID, si.InstallerID)
				require.NoError(t, err)
				return nil
			})

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
			err = ds.InsertSoftwareUninstallRequest(ctx, "uuid"+tag+tc, hostPendingUninstall.ID, si.InstallerID)
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
			err = ds.InsertSoftwareUninstallRequest(ctx, "uuid"+tag+tc, hostFailedUninstall.ID, si.InstallerID)
			require.NoError(t, err)
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err = q.ExecContext(ctx, `
					UPDATE host_software_installs SET uninstall_script_exit_code = 1 WHERE host_id = ? AND software_installer_id = ?`,
					hostFailedUninstall.ID, si.InstallerID)
				require.NoError(t, err)
				return nil
			})

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
			err = ds.InsertSoftwareUninstallRequest(ctx, "uuid"+tag+tc, hostUninstalled.ID, si.InstallerID)
			require.NoError(t, err)
			ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err = q.ExecContext(ctx, `
					UPDATE host_software_installs SET uninstall_script_exit_code = 0 WHERE host_id = ? AND software_installer_id = ?`,
					hostUninstalled.ID, si.InstallerID)
				require.NoError(t, err)
				return nil
			})

			// Uninstall request with unknown host
			err = ds.InsertSoftwareUninstallRequest(ctx, "uuid"+tag+tc, 99999, si.InstallerID)
			assert.ErrorContains(t, err, "Host")

			userTeamFilter := fleet.TeamFilter{
				User: &fleet.User{GlobalRole: ptr.String("admin")},
			}

			// list hosts with software install pending requests
			expectStatus := fleet.SoftwareInstallPending
			hosts, err := ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			require.Equal(t, hostPendingInstall.ID, hosts[0].ID)

			// list hosts with all pending requests
			expectStatus = fleet.SoftwarePending
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 2)
			assert.ElementsMatch(t, []uint{hostPendingInstall.ID, hostPendingUninstall.ID}, []uint{hosts[0].ID, hosts[1].ID})

			// list hosts with software install failed requests
			expectStatus = fleet.SoftwareInstallFailed
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			assert.ElementsMatch(t, []uint{hostFailedInstall.ID}, []uint{hosts[0].ID})

			// list hosts with all failed requests
			expectStatus = fleet.SoftwareFailed
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 2)
			assert.ElementsMatch(t, []uint{hostFailedInstall.ID, hostFailedUninstall.ID}, []uint{hosts[0].ID, hosts[1].ID})

			// list hosts with software installed
			expectStatus = fleet.SoftwareInstalled
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			assert.ElementsMatch(t, []uint{hostInstalled.ID}, []uint{hosts[0].ID})

			// list hosts with pending software uninstall requests
			expectStatus = fleet.SoftwareUninstallPending
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			assert.ElementsMatch(t, []uint{hostPendingUninstall.ID}, []uint{hosts[0].ID})

			// list hosts with failed software uninstall requests
			expectStatus = fleet.SoftwareUninstallFailed
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				SoftwareStatusFilter:  &expectStatus,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
			assert.ElementsMatch(t, []uint{hostFailedUninstall.ID}, []uint{hosts[0].ID})

			// list all hosts with the software title that shows up in host_software (after fleetd software query is run)
			hosts, err = ds.ListHosts(ctx, userTeamFilter, fleet.HostListOptions{
				ListOptions:           fleet.ListOptions{PerPage: 100},
				SoftwareTitleIDFilter: installerMeta.TitleID,
				TeamFilter:            teamID,
			})
			require.NoError(t, err)
			assert.Empty(t, hosts)

			// get software title includes status
			summary, err := ds.GetSummaryHostSoftwareInstalls(ctx, installerMeta.InstallerID)
			require.NoError(t, err)
			require.Equal(t, fleet.SoftwareInstallerStatusSummary{
				Installed:        1,
				PendingInstall:   1,
				FailedInstall:    1,
				PendingUninstall: 1,
				FailedUninstall:  1,
			}, *summary)
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
			installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:           "foo" + tc.name,
				Source:          "bar" + tc.name,
				InstallScript:   "echo " + tc.name,
				Version:         "1.11",
				TeamID:          &teamID,
				Filename:        swFilename,
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
			installUUID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, false, nil)
			require.NoError(t, err)

			res, err := ds.GetSoftwareInstallResults(ctx, installUUID)
			require.NoError(t, err)
			require.NotNil(t, res.UpdatedAt)
			require.Less(t, beforeInstallRequest, res.CreatedAt)
			createdAt := res.CreatedAt
			require.Less(t, beforeInstallRequest, *res.UpdatedAt)

			beforeInstallResult := time.Now()
			err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
				HostID:                    host.ID,
				InstallUUID:               installUUID,
				PreInstallConditionOutput: tc.preInstallQueryOutput,
				InstallScriptExitCode:     tc.installScriptEC,
				InstallScriptOutput:       tc.installScriptOutput,
				PostInstallScriptExitCode: tc.postInstallScriptEC,
				PostInstallScriptOutput:   tc.postInstallScriptOutput,
			})
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

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
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
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{{
		InstallScript:   "install",
		InstallerFile:   tfr0,
		StorageID:       ins0,
		Filename:        "installer0",
		Title:           "ins0",
		Source:          "apps",
		Version:         "1",
		PreInstallQuery: "foo",
		UserID:          user1.ID,
		Platform:        "darwin",
		URL:             "https://example.com",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	}})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, softwareInstallers, 1)
	require.NotNil(t, softwareInstallers[0].TeamID)
	require.Equal(t, team.ID, *softwareInstallers[0].TeamID)
	require.NotNil(t, softwareInstallers[0].TitleID)
	require.Equal(t, "https://example.com", softwareInstallers[0].URL)
	assertSoftware([]fleet.SoftwareTitle{
		{Name: ins0, Source: "apps", Browser: ""},
	})

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
		{Name: ins0, Source: "apps", Browser: ""},
		{Name: ins1, Source: "apps", Browser: ""},
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
			ValidatedLabels:   &fleet.LabelIdentsWithScope{},
		},
	})
	require.NoError(t, err)

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
		{Name: ins1, Source: "apps", Browser: ""},
	})

	// remove everything
	err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{})
	require.NoError(t, err)
	softwareInstallers, err = ds.GetSoftwareInstallers(ctx, team.ID)
	require.NoError(t, err)
	require.Empty(t, softwareInstallers)
	assertSoftware([]fleet.SoftwareTitle{})
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
	require.ErrorIs(t, err, errDeleteInstallerWithAssociatedPolicy)

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
	var nfe *notFoundError
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
	executionID, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID1, false, &policy1.ID)
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
	executionID, err = ds.InsertSoftwareInstallRequest(ctx, host1.ID, installerID2, false, &policy2.ID)
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
	executionID, err = ds.InsertSoftwareInstallRequest(ctx, host2.ID, installerID1, false, &policy1.ID)
	require.NoError(t, err)

	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host2.ID,
		InstallUUID:           executionID,
		InstallScriptExitCode: ptr.Int(0),
	})
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
	installUUID1, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, softwareInstallerID1, false, nil)
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
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:      host1.ID,
		InstallUUID: installUUID1,

		InstallScriptExitCode: ptr.Int(0),
	})
	require.NoError(t, err)

	// Last installation should be "installed".
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID1, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstalled, *host1LastInstall.Status)

	// Install installer2.pkg on host1.
	installUUID2, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, softwareInstallerID2, false, nil)
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
	installUUID3, err := ds.InsertSoftwareInstallRequest(ctx, host1.ID, softwareInstallerID1, false, nil)
	require.NoError(t, err)
	require.NotEmpty(t, installUUID3)

	// Last installation for installer1.pkg should be "pending" again.
	host1LastInstall, err = ds.GetHostLastInstallData(ctx, host1.ID, softwareInstallerID1)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, installUUID3, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// Set result of last installer1.pkg installation.
	err = ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:      host1.ID,
		InstallUUID: installUUID3,

		InstallScriptExitCode: ptr.Int(1),
	})
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

	software1 := []fleet.Software{
		{Name: "Existing Title", Version: "0.0.1", Source: "apps", BundleIdentifier: "existing.title"},
	}
	software2 := []fleet.Software{
		{Name: "Existing Title", Version: "v0.0.2", Source: "apps", BundleIdentifier: "existing.title"},
		{Name: "Existing Title", Version: "0.0.3", Source: "apps", BundleIdentifier: "existing.title"},
		{Name: "Existing Title Without Bundle", Version: "0.0.3", Source: "apps"},
	}

	_, err := ds.UpdateHostSoftware(ctx, host1.ID, software1)
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, host2.ID, software2)
	require.NoError(t, err)
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	tests := []struct {
		name    string
		payload *fleet.UploadSoftwareInstallerPayload
	}{
		{
			name: "title that already exists, no bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "Existing Title",
				Source: "apps",
			},
		},
		{
			name: "title that already exists, mismatched bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "Existing Title",
				Source:           "apps",
				BundleIdentifier: "com.existing.bundle",
			},
		},
		{
			name: "title that already exists but doesn't have a bundle identifier",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "Existing Title Without Bundle",
				Source: "apps",
			},
		},
		{
			name: "title that already exists, no bundle identifier in DB, bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "Existing Title Without Bundle",
				Source:           "apps",
				BundleIdentifier: "com.new.bundleid",
			},
		},
		{
			name: "title that doesn't exist, no bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:  "New Title",
				Source: "some_source",
			},
		},
		{
			name: "title that doesn't exist, with bundle identifier in payload",
			payload: &fleet.UploadSoftwareInstallerPayload{
				Title:            "New Title With Bundle",
				Source:           "some_source",
				BundleIdentifier: "com.new.bundle",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ds.getOrGenerateSoftwareInstallerTitleID(ctx, tt.payload)
			require.NoError(t, err)
			require.NotEmpty(t, id)
		})
	}
}

func testBatchSetSoftwareInstallersScopedViaLabels(t *testing.T, ds *Datastore) {
	// ctx := context.Background()
	//
	// // create a couple teams and a user
	// tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "1"})
	// require.NoError(t, err)
	// tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "2"})
	// require.NoError(t, err)
	// user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	//
	// // create some installer payloads to be used by test cases
	// installers := make([]*fleet.UploadSoftwareInstallerPayload, 3)
	// for i := range installers {
	// 	file := bytes.NewReader([]byte("installer" + fmt.Sprint(i)))
	// 	tfr, err := fleet.NewTempFileReader(file, t.TempDir)
	// 	require.NoError(t, err)
	// 	installers[i] = &fleet.UploadSoftwareInstallerPayload{
	// 		InstallScript:   "install",
	// 		InstallerFile:   tfr,
	// 		StorageID:       "installer" + fmt.Sprint(i),
	// 		Filename:        "installer" + fmt.Sprint(i),
	// 		Title:           "ins" + fmt.Sprint(i),
	// 		Source:          "apps",
	// 		Version:         "1",
	// 		PreInstallQuery: "foo",
	// 		UserID:          user.ID,
	// 		Platform:        "darwin",
	// 		URL:             "https://example.com",
	// 	}
	// }
	//
	// // create some labels to be used by test cases
	// labels := make([]*fleet.Label, 4)
	// for i := range labels {
	// 	lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: "label" + fmt.Sprint(i)})
	// 	require.NoError(t, err)
	// 	labels[i] = lbl
	// }
}
