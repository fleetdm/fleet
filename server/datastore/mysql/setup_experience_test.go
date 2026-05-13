package mysql

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"EnqueueSetupExperienceItems", testEnqueueSetupExperienceItems},
		{"EnqueueSetupExperienceLinuxScriptPackages", testEnqueueSetupExperienceLinuxScriptPackages},
		{"GetSetupExperienceTitles", testGetSetupExperienceTitles},
		{"SetSetupExperienceTitles", testSetSetupExperienceTitles},
		{"ListSetupExperienceStatusResults", testSetupExperienceStatusResults},
		{"SetupExperienceScriptCRUD", testSetupExperienceScriptCRUD},
		{"TestHostInSetupExperience", testHostInSetupExperience},
		{"TestGetSetupExperienceScriptByID", testGetSetupExperienceScriptByID},
		{"TestUpdateSetupExperienceScriptWhileEnqueued", testUpdateSetupExperienceScriptWhileEnqueued},
		{"TestEnqueueSetupExperienceItemsWindows", testEnqueueSetupExperienceItemsWindows},
		{"EnqueueSetupExperienceItemsWithDisplayName", testEnqueueSetupExperienceItemsWithDisplayName},
		{"UpdateStatusGuardsTerminalStates", testUpdateStatusGuardsTerminalStates},
		{"SetSetupExperienceTitlesOnlyMarksActiveInstaller", testSetSetupExperienceTitlesOnlyMarksActiveInstaller},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// TODO(JVE): this test could probably be simplified and most of the ad-hoc SQL removed.
// testEnqueueSetupExperienceLinuxScriptPackages tests that Linux script packages (.sh)
// are properly enqueued for setup experience. This is a regression test for bug #34654.
func testEnqueueSetupExperienceLinuxScriptPackages(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Create a team
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// Create a .sh script package installer for Linux
	tfrSh, err := fleet.NewTempFileReader(strings.NewReader("#!/bin/bash\necho hello"), t.TempDir)
	require.NoError(t, err)
	installerIDSh, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "#!/bin/bash\necho installing",
		InstallerFile:   tfrSh,
		StorageID:       "storage-sh-1",
		Filename:        "install.sh",
		Title:           "Script Package",
		Version:         "1.0",
		Source:          "sh_packages",
		UserID:          user1.ID,
		TeamID:          &team1.ID,
		Platform:        "linux",
		Extension:       "sh",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Create a .deb package installer for Linux (debian-specific)
	tfrDeb, err := fleet.NewTempFileReader(strings.NewReader("deb package"), t.TempDir)
	require.NoError(t, err)
	installerIDDeb, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "dpkg -i test.deb",
		InstallerFile:   tfrDeb,
		StorageID:       "storage-deb-1",
		Filename:        "test.deb",
		Title:           "Deb Package",
		Version:         "1.0",
		Source:          "deb_packages",
		UserID:          user1.ID,
		TeamID:          &team1.ID,
		Platform:        "linux",
		Extension:       "deb",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Create a .tar.gz package installer for Linux (distribution-agnostic like .sh)
	tfrTarGz, err := fleet.NewTempFileReader(strings.NewReader("tarball"), t.TempDir)
	require.NoError(t, err)
	installerIDTarGz, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "tar -xzf test.tar.gz",
		InstallerFile:   tfrTarGz,
		StorageID:       "storage-tar-1",
		Filename:        "test.tar.gz",
		Title:           "TarGz Package",
		Version:         "1.0",
		Source:          "tgz_packages",
		UserID:          user1.ID,
		TeamID:          &team1.ID,
		Platform:        "linux",
		Extension:       "tar.gz",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Mark all installers for setup experience
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?, ?)",
			installerIDSh, installerIDDeb, installerIDTarGz)
		return err
	})

	// Test 1: Script package ONLY on Debian host - should enqueue and return true
	t.Run("sh_only_debian", func(t *testing.T) {
		hostDebianShOnly := "debian-sh-only-" + uuid.NewString()

		// Mark only .sh for setup experience, disable others temporarily
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 0 WHERE id IN (?, ?)", installerIDDeb, installerIDTarGz)
			require.NoError(t, err)
			_, err = q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id = ?", installerIDSh)
			return err
		})

		anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "ubuntu", "debian", hostDebianShOnly, team1.ID)
		require.NoError(t, err)
		require.True(t, anythingEnqueued, "BUG #34654: .sh package alone should trigger setup experience")

		// Verify the .sh package was enqueued
		var rows []setupExperienceInsertTestRows
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows,
				"SELECT host_uuid, name, status, software_installer_id FROM setup_experience_status_results WHERE host_uuid = ?",
				hostDebianShOnly)
		})
		require.Len(t, rows, 1, "BUG #34654: .sh package should be enqueued")
		require.Equal(t, "Script Package", rows[0].Name)
		require.Equal(t, nullableUint(installerIDSh), rows[0].SoftwareInstallerID)

		// Re-enable all for next tests
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?, ?)",
				installerIDSh, installerIDDeb, installerIDTarGz)
			return err
		})
	})

	// Test 2: Script package on RHEL host - should enqueue (sh is distribution-agnostic)
	t.Run("sh_only_rhel", func(t *testing.T) {
		hostRhelShOnly := "rhel-sh-only-" + uuid.NewString()

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 0 WHERE id IN (?, ?)", installerIDDeb, installerIDTarGz)
			require.NoError(t, err)
			_, err = q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id = ?", installerIDSh)
			return err
		})

		anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "fedora", "rhel", hostRhelShOnly, team1.ID)
		require.NoError(t, err)
		require.True(t, anythingEnqueued, "BUG #34654: .sh package should work on RHEL too")

		var rows []setupExperienceInsertTestRows
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows,
				"SELECT host_uuid, name, status, software_installer_id FROM setup_experience_status_results WHERE host_uuid = ?",
				hostRhelShOnly)
		})
		require.Len(t, rows, 1)
		require.Equal(t, "Script Package", rows[0].Name)

		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?, ?)",
				installerIDSh, installerIDDeb, installerIDTarGz)
			return err
		})
	})

	// Test 3: Mixed .sh and .deb on Debian host - both should enqueue
	t.Run("mixed_sh_deb_debian", func(t *testing.T) {
		hostDebianMixed := "debian-mixed-" + uuid.NewString()

		anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "ubuntu", "debian", hostDebianMixed, team1.ID)
		require.NoError(t, err)
		require.True(t, anythingEnqueued)

		var rows []setupExperienceInsertTestRows
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows,
				"SELECT host_uuid, name, status, software_installer_id FROM setup_experience_status_results WHERE host_uuid = ? ORDER BY name",
				hostDebianMixed)
		})
		require.Len(t, rows, 3, "All three packages should be enqueued on debian")

		// Verify all expected packages are there
		names := []string{rows[0].Name, rows[1].Name, rows[2].Name}
		require.Contains(t, names, "Deb Package")
		require.Contains(t, names, "Script Package", "BUG #34654: .sh should be enqueued even when mixed with other packages")
		require.Contains(t, names, "TarGz Package")
	})

	// Test 4: Mixed .sh and .deb on RHEL host - only .sh and .tar.gz should enqueue (not .deb)
	t.Run("mixed_sh_deb_rhel", func(t *testing.T) {
		hostRhelMixed := "rhel-mixed-" + uuid.NewString()

		anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "fedora", "rhel", hostRhelMixed, team1.ID)
		require.NoError(t, err)
		require.True(t, anythingEnqueued)

		var rows []setupExperienceInsertTestRows
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows,
				"SELECT host_uuid, name, status, software_installer_id FROM setup_experience_status_results WHERE host_uuid = ? ORDER BY name",
				hostRhelMixed)
		})
		require.Len(t, rows, 2, "Only .sh and .tar.gz should be enqueued on RHEL (not .deb)")

		names := []string{rows[0].Name, rows[1].Name}
		require.Contains(t, names, "Script Package", "BUG #34654: .sh should be enqueued on RHEL")
		require.Contains(t, names, "TarGz Package")
		require.NotContains(t, names, "Deb Package", ".deb should not be enqueued on RHEL")
	})
}

func testEnqueueSetupExperienceItemsWindows(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// Create some software installers and add them to setup experience
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
		Title:             "Software1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          "windows",
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     tfr2,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "Software2",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          "windows",
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?)", installerID1, installerID2)
		return err
	})

	host1UUID := "11111111-1111-1111-1111-111111111111"
	host2UUID := "22222222-2222-2222-2222-222222222222"

	// Freshly enrolled host, should get items enqueued
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "windows-test-1",
		OsqueryHostID:  ptr.String("osquery-windows-1"),
		NodeKey:        ptr.String("node-key-windows-1"),
		UUID:           host1UUID,
		Platform:       "windows",
		HardwareSerial: "654321a-1",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-1*time.Hour), host1UUID)
		return err
	})

	// Enroll date > 24 hours ago and is windows. This should not get items enqueued.
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "windows-test-2",
		OsqueryHostID:  ptr.String("osquery-windows-2"),
		NodeKey:        ptr.String("node-key-windows-2"),
		UUID:           host2UUID,
		Platform:       "windows",
		HardwareSerial: "654321b-2",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-25*time.Hour), host2UUID)
		return err
	})

	anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "windows", "windows", host1UUID, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "windows", "windows", host2UUID, team2.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)

	// Re-Autopilot of an existing host: last_enrolled_at is >24h old (the
	// pre-existing record predates this Autopilot cycle), but the host has
	// just MDM-enrolled and is in awaiting_configuration=Pending.
	host3UUID := "33333333-3333-3333-3333-333333333333"
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "windows-test-3-reautopilot",
		OsqueryHostID:  ptr.String("osquery-windows-3"),
		NodeKey:        ptr.String("node-key-windows-3"),
		UUID:           host3UUID,
		Platform:       "windows",
		HardwareSerial: "654321c-3",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-25*time.Hour), host3UUID)
		return err
	})
	// Insert a Windows MDM enrollment with awaiting_configuration=Pending,
	// matching what a fresh Autopilot enrollment on the same host would create
	// before last_enrolled_at gets refreshed.
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "device-host3",
		MDMHardwareID:          "hw-host3",
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-H3",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               host3UUID,
		AwaitingConfiguration:  fleet.WindowsMDMAwaitingConfigurationPending,
	}))

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "windows", "windows", host3UUID, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued,
		"re-Autopilot of an existing host (>24h old) with awaiting_configuration!=None must bypass the age guard")

	// Re-BYOD of an existing host: last_enrolled_at is >24h old AND the host's BYOD enrollment never
	// enters awaiting_configuration (not_in_oobe=1). The freshly-created mdm_windows_enrollments row
	// is the signal that this IS a real re-enrollment we want setup-experience for.
	host4UUID := "44444444-4444-4444-4444-444444444444"
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "windows-test-4-rebyod",
		OsqueryHostID:  ptr.String("osquery-windows-4"),
		NodeKey:        ptr.String("node-key-windows-4"),
		UUID:           host4UUID,
		Platform:       "windows",
		HardwareSerial: "654321d-4",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-25*time.Hour), host4UUID)
		return err
	})
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "device-host4",
		MDMHardwareID:          "hw-host4",
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-H4",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           true,
		HostUUID:               host4UUID,
		AwaitingConfiguration:  fleet.WindowsMDMAwaitingConfigurationNone,
	}))

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "windows", "windows", host4UUID, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued,
		"re-BYOD of an existing host (>24h old) with a fresh mdm_windows_enrollments row must bypass the age guard")

	// Original #35717 protection: a fleetd MSI upgrade on a long-running host does NOT create a new
	// mdm_windows_enrollments row, so the existing one is also old. Both fallback signals must miss,
	// and the age guard must still skip enqueueing.
	host5UUID := "55555555-5555-5555-5555-555555555555"
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "windows-test-5-fleetd-upgrade",
		OsqueryHostID:  ptr.String("osquery-windows-5"),
		NodeKey:        ptr.String("node-key-windows-5"),
		UUID:           host5UUID,
		Platform:       "windows",
		HardwareSerial: "654321e-5",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-25*time.Hour), host5UUID)
		return err
	})
	require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "device-host5",
		MDMHardwareID:          "hw-host5",
		MDMDeviceState:         microsoft_mdm.MDMDeviceStateEnrolled,
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-H5",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           true,
		HostUUID:               host5UUID,
		AwaitingConfiguration:  fleet.WindowsMDMAwaitingConfigurationNone,
	}))
	// Backdate the MDM enrollment row beyond the freshEnrollmentWindow (5m) to simulate a long-running
	// host whose orbit just started supporting setup-experience.
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE mdm_windows_enrollments SET created_at = ? WHERE host_uuid = ?", time.Now().Add(-72*time.Hour), host5UUID)
		return err
	})

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "windows", "windows", host5UUID, team1.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued,
		"long-running host with a stale mdm_windows_enrollments row (e.g. fleetd MSI upgrade per #35717) must still skip enqueueing")
}

func testEnqueueSetupExperienceItems(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	// Create some teams
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team3, err := ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	// Create some software installers and add them to setup experience
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
		Title:             "Software1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID2, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "banana",
		PreInstallQuery:   "SELECT 3",
		PostInstallScript: "apple",
		InstallerFile:     tfr2,
		StorageID:         "storage3",
		Filename:          "file3",
		Title:             "Software2",
		Version:           "3.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?)", installerID1, installerID2)
		return err
	})

	// Create some VPP apps and add them to setup experience
	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	vpp1, err := ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b2"}
	vpp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team2.ID)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id IN (?, ?)", vpp1.AdamID, vpp2.AdamID)
		return err
	})

	// Create some scripts and add them to setup experience
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script1", ScriptContents: "SCRIPT 1", TeamID: &team1.ID})
	require.NoError(t, err)
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script2", ScriptContents: "SCRIPT 2", TeamID: &team2.ID})
	require.NoError(t, err)

	script1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)

	script2, err := ds.GetSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)

	hostTeam1 := "123"
	hostTeam2 := "456"
	hostTeam2Missing := "555"
	hostTeam3 := "789"
	hostTeam1Old := "000"
	hostTeam1New := "007"

	// No enroll date. This should be treated as a new host and have items enqueued.
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-1",
		OsqueryHostID:  ptr.String("osquery-macos-1"),
		NodeKey:        ptr.String("node-key-macos-1"),
		UUID:           hostTeam1,
		Platform:       "darwin",
		HardwareSerial: "654321a",
	})
	require.NoError(t, err)

	// Enroll date < 24 hours ago. This should be treated as a new host and have items enqueued.
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-2",
		OsqueryHostID:  ptr.String("osquery-macos-2"),
		NodeKey:        ptr.String("node-key-macos-2"),
		UUID:           hostTeam2,
		Platform:       "darwin",
		HardwareSerial: "654321a-2",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-1*time.Hour), hostTeam2)
		return err
	})

	// Deliberately not adding a record for the hostTeam2Missing, to verify that
	// we still enqueue items for it if it doesn't exist in the database.

	// Enroll date > 24 hours ago but is macOS. This should get items enqueued.
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-4",
		OsqueryHostID:  ptr.String("osquery-macos-4"),
		NodeKey:        ptr.String("node-key-macos-4"),
		UUID:           hostTeam1Old,
		Platform:       "darwin",
		HardwareSerial: "654321a-4",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Now().Add(-25*time.Hour), hostTeam1Old)
		return err
	})

	// Enroll date of the Fleet "zero time". This should have items enqueued.
	_, err = ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-test-4",
		OsqueryHostID:  ptr.String("osquery-macos-5"),
		NodeKey:        ptr.String("node-key-macos-5"),
		UUID:           hostTeam1New,
		Platform:       "darwin",
		HardwareSerial: "654321a-4",
	})
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE hosts SET last_enrolled_at = ? WHERE uuid = ?", time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), hostTeam1New)
		return err
	})

	anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam1, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	awaitingConfig, err := ds.GetHostAwaitingConfiguration(ctx, hostTeam1)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam1New, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam1New)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam2, team2.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam2)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam2Missing, team2.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam2Missing)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam3, team3.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)
	// Nothing is configured for setup experience in team 3, so we do not set
	// host_mdm_apple_awaiting_configuration.
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam3)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, awaitingConfig)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam1Old, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)
	// This host enrolled > 24 hours ago, but it's darwin, so we should enqueue items for it.
	awaitingConfig, err = ds.GetHostAwaitingConfiguration(ctx, hostTeam1Old)
	require.NoError(t, err)
	require.True(t, awaitingConfig)

	seRows := []setupExperienceInsertTestRows{}

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &seRows, "SELECT host_uuid, name, status, software_installer_id, setup_experience_script_id, vpp_app_team_id FROM setup_experience_status_results")
	})

	// five hosts with three items enqueued each.
	require.Len(t, seRows, 15)

	for _, tc := range []setupExperienceInsertTestRows{
		{
			HostUUID:            hostTeam1,
			Name:                "Software1",
			Status:              "pending",
			SoftwareInstallerID: nullableUint(installerID1),
		},
		{
			HostUUID:            hostTeam2,
			Name:                "Software2",
			Status:              "pending",
			SoftwareInstallerID: nullableUint(installerID2),
		},
		{
			HostUUID:     hostTeam1,
			Name:         app1.Name,
			Status:       "pending",
			VPPAppTeamID: nullableUint(1),
		},
		{
			HostUUID:     hostTeam2,
			Name:         app2.Name,
			Status:       "pending",
			VPPAppTeamID: nullableUint(2),
		},
		{
			HostUUID: hostTeam1,
			Name:     "script1",
			Status:   "pending",
			ScriptID: nullableUint(script1.ID),
		},
		{
			HostUUID: hostTeam2,
			Name:     "script2",
			Status:   "pending",
			ScriptID: nullableUint(script2.ID),
		},
	} {
		var found bool
		for _, row := range seRows {
			if row == tc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find entry in setup_experience_status_results table: %#v", tc)
		}
	}

	require.Condition(t, func() (success bool) {
		for _, row := range seRows {
			if row.HostUUID == hostTeam3 {
				return false
			}
		}

		return true
	})

	// Remove team2's setup experience items
	err = ds.DeleteSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)

	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team2.ID, []uint{})
	require.NoError(t, err)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam1, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	// team2 now has nothing enqueued
	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam2, team2.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)
	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam2Missing, team2.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam3, team3.ID)
	require.NoError(t, err)
	require.False(t, anythingEnqueued)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &seRows, "SELECT host_uuid, name, status, software_installer_id, setup_experience_script_id, vpp_app_team_id FROM setup_experience_status_results")
	})

	// Only the team 1 and team 3 hosts should have items enqueued now.
	// Two hosts with three items each.
	require.Len(t, seRows, 9)

	for _, tc := range []setupExperienceInsertTestRows{
		{
			HostUUID:            hostTeam1,
			Name:                "Software1",
			Status:              "pending",
			SoftwareInstallerID: nullableUint(installerID1),
		},
		{
			HostUUID:     hostTeam1,
			Name:         app1.Name,
			Status:       "pending",
			VPPAppTeamID: nullableUint(1),
		},
		{
			HostUUID: hostTeam1,
			Name:     "script1",
			Status:   "pending",
			ScriptID: nullableUint(script1.ID),
		},
	} {
		var found bool
		for _, row := range seRows {
			if row == tc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find entry in setup_experience_status_results table: %#v", tc)
		}
	}

	for _, row := range seRows {
		if row.HostUUID == hostTeam2 {
			team := 2
			t.Errorf("team %d shouldn't have any any entries", team)
		}
	}
}

// testEnqueueSetupExperienceItemsWithDisplayName verifies that when a custom
// display name is set for a software title, the enqueue function uses it to
// determine the alphabetical install order (instead of the default
// software_titles.name). This ordering also orders the steps in the
// setup experience UI. The UI uses the display name if it is set, and
// the name if not.
func testEnqueueSetupExperienceItemsWithDisplayName(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_display_name_test"})
	require.NoError(t, err)

	user := test.NewUser(t, ds, "DisplayNameUser", "displaynameuser@example.com", true)

	// Create two software installers with titles that sort in a known order:
	//   "AAA_Software" < "ZZZ_Software"  (alphabetically)
	// We will then assign custom display names that invert this order:
	//   "AAA_Software" → "Zulu Custom"
	//   "ZZZ_Software" → "Alpha Custom"
	// After enqueue, the rows ordered by id (insert order) should reflect
	// the display-name alphabetical order:
	//   id=N   → ZZZ_Software (display name "Alpha Custom", sorts first)
	//   id=N+1 → AAA_Software (display name "Zulu Custom", sorts second)
	// But the `name` column still stores the original st.name.
	// Note that the setup experience UI will also follow this ordering;
	// it will display "Alpha Custom" and then "Zulu Custom".

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello1"), t.TempDir)
	require.NoError(t, err)
	installerID1, titleID1, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install1",
		UninstallScript: "uninstall1",
		InstallerFile:   tfr1,
		StorageID:       "storage_dn_1",
		Filename:        "file_dn_1",
		Title:           "AAA_Software",
		Version:         "1.0",
		Source:          "apps",
		UserID:          user.ID,
		TeamID:          &team.ID,
		Platform:        string(fleet.MacOSPlatform),
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr2, err := fleet.NewTempFileReader(strings.NewReader("hello2"), t.TempDir)
	require.NoError(t, err)
	installerID2, titleID2, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install2",
		UninstallScript: "uninstall2",
		InstallerFile:   tfr2,
		StorageID:       "storage_dn_2",
		Filename:        "file_dn_2",
		Title:           "ZZZ_Software",
		Version:         "2.0",
		Source:          "apps",
		UserID:          user.ID,
		TeamID:          &team.ID,
		Platform:        string(fleet.MacOSPlatform),
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// Mark both installers for setup experience
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?)", installerID1, installerID2)
		return err
	})

	// Set custom display names that invert the alphabetical order
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if err := updateSoftwareTitleDisplayName(ctx, q, &team.ID, titleID1, "Zulu Custom"); err != nil {
			return err
		}
		return updateSoftwareTitleDisplayName(ctx, q, &team.ID, titleID2, "Alpha Custom")
	})

	// Create two VPP apps with titles that sort in a known order, then invert with display names.
	vppApp1 := &fleet.VPPApp{
		Name:             "AAA_VPP_App",
		BundleIdentifier: "com.aaa.vpp",
		VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "dn_adam_1", Platform: fleet.MacOSPlatform}},
	}
	vpp1, err := ds.InsertVPPAppWithTeam(ctx, vppApp1, &team.ID)
	require.NoError(t, err)

	vppApp2 := &fleet.VPPApp{
		Name:             "ZZZ_VPP_App",
		BundleIdentifier: "com.zzz.vpp",
		VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "dn_adam_2", Platform: fleet.MacOSPlatform}},
	}
	vpp2, err := ds.InsertVPPAppWithTeam(ctx, vppApp2, &team.ID)
	require.NoError(t, err)

	// Mark both VPP apps for setup experience
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id IN (?, ?)", vpp1.AdamID, vpp2.AdamID)
		return err
	})

	// Set custom display names for VPP apps (invert order)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		if err := updateSoftwareTitleDisplayName(ctx, q, &team.ID, vppApp1.TitleID, "Zulu VPP Custom"); err != nil {
			return err
		}
		return updateSoftwareTitleDisplayName(ctx, q, &team.ID, vppApp2.TitleID, "Alpha VPP Custom")
	})

	// Create a host assigned to the team and enqueue setup experience.
	// The host must be on the team so that ListSetupExperienceResultsByHostUUID
	// can look up the team's display names.
	hostUUID := "host-display-name-test-" + uuid.NewString()
	host1, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-dn-test",
		OsqueryHostID:  ptr.String("osquery-dn-test"),
		NodeKey:        ptr.String("node-key-dn-test"),
		UUID:           hostUUID,
		Platform:       "darwin",
		HardwareSerial: "dn-serial-1",
	})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host1.ID}))
	require.NoError(t, err)

	anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostUUID, team.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	// --- Verify all rows are globally ordered by display name ---
	// enqueueSetupExperienceItems inserts software (installers and VPP apps)
	// together in a single query ordered by COALESCE(display_name, st.name),
	// so the auto-incremented id reflects the global display-name order.
	// ListSetupExperienceResultsByHostUUID returns rows ordered by sesr.id,
	// preserving that insert order. Scripts are inserted last.
	//
	// Expected order (all software globally sorted by display name):
	//   0. ZZZ_Software  (installer, display name "Alpha Custom")
	//   1. ZZZ_VPP_App   (VPP app,   display name "Alpha VPP Custom")
	//   2. AAA_Software  (installer, display name "Zulu Custom")
	//   3. AAA_VPP_App   (VPP app,   display name "Zulu VPP Custom")
	allResults, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID, team.ID)
	require.NoError(t, err)
	require.Len(t, allResults, 4, "expected 4 results total (2 installers + 2 VPP apps)")

	assert.Equal(t, "ZZZ_Software", allResults[0].Name, "row 0: ZZZ_Software (display name 'Alpha Custom')")
	assert.Equal(t, "Alpha Custom", allResults[0].DisplayName, "row 0: display name should be 'Alpha Custom'")
	assert.NotNil(t, allResults[0].SoftwareInstallerID, "row 0: should be a software installer")

	assert.Equal(t, "ZZZ_VPP_App", allResults[1].Name, "row 1: ZZZ_VPP_App (display name 'Alpha VPP Custom')")
	assert.Equal(t, "Alpha VPP Custom", allResults[1].DisplayName, "row 1: display name should be 'Alpha VPP Custom'")
	assert.NotNil(t, allResults[1].VPPAppTeamID, "row 1: should be a VPP app")
	assert.Less(t, allResults[0].ID, allResults[1].ID)

	assert.Equal(t, "AAA_Software", allResults[2].Name, "row 2: AAA_Software (display name 'Zulu Custom')")
	assert.Equal(t, "Zulu Custom", allResults[2].DisplayName, "row 2: display name should be 'Zulu Custom'")
	assert.NotNil(t, allResults[2].SoftwareInstallerID, "row 2: should be a software installer")
	assert.Less(t, allResults[1].ID, allResults[2].ID)

	assert.Equal(t, "AAA_VPP_App", allResults[3].Name, "row 3: AAA_VPP_App (display name 'Zulu VPP Custom')")
	assert.Equal(t, "Zulu VPP Custom", allResults[3].DisplayName, "row 3: display name should be 'Zulu VPP Custom'")
	assert.NotNil(t, allResults[3].VPPAppTeamID, "row 3: should be a VPP app")

	// --- Verify fallback: no display name → order uses st.name ---
	// Add a third installer and a third VPP app, both without custom display
	// names, then re-enqueue for a new host and verify the globally
	// interleaved order. Items without a display name fall back to st.name.
	tfr3, err := fleet.NewTempFileReader(strings.NewReader("hello3"), t.TempDir)
	require.NoError(t, err)
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install3",
		UninstallScript: "uninstall3",
		InstallerFile:   tfr3,
		StorageID:       "storage_dn_3",
		Filename:        "file_dn_3",
		Title:           "MMM_NoDisplayName",
		Version:         "3.0",
		Source:          "apps",
		UserID:          user.ID,
		TeamID:          &team.ID,
		Platform:        string(fleet.MacOSPlatform),
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id NOT IN (?, ?)", installerID1, installerID2)
		return err
	})

	vppApp3 := &fleet.VPPApp{
		Name:             "MMM_VPP_NoDisplayName",
		BundleIdentifier: "com.mmm.vpp",
		VPPAppTeam:       fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "dn_adam_3", Platform: fleet.MacOSPlatform}},
	}
	vpp3, err := ds.InsertVPPAppWithTeam(ctx, vppApp3, &team.ID)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id = ?", vpp3.AdamID)
		return err
	})

	// Re-enqueue for a new host (also on the team) to pick up all installers and VPP apps.
	hostUUID2 := "host-display-name-fallback-" + uuid.NewString()
	host2, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:       "macos-dn-test-2",
		OsqueryHostID:  ptr.String("osquery-dn-test-2"),
		NodeKey:        ptr.String("node-key-dn-test-2"),
		UUID:           hostUUID2,
		Platform:       "darwin",
		HardwareSerial: "dn-serial-2",
	})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host2.ID}))
	require.NoError(t, err)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostUUID2, team.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	// Verify the globally interleaved order across installers and VPP apps.
	// The combined INSERT in enqueueSetupExperienceItems orders by
	// COALESCE(display_name, st.name), and ListSetupExperienceResultsByHostUUID
	// returns rows ordered by sesr.id (i.e. insert order).
	//
	// Expected global order (sorted by COALESCE(display_name, st.name)):
	//   0. ZZZ_Software          (installer, display name "Alpha Custom")
	//   1. ZZZ_VPP_App           (VPP app,   display name "Alpha VPP Custom")
	//   2. MMM_NoDisplayName     (installer, no display name → falls back to st.name)
	//   3. MMM_VPP_NoDisplayName (VPP app,   no display name → falls back to st.name)
	//   4. AAA_Software          (installer, display name "Zulu Custom")
	//   5. AAA_VPP_App           (VPP app,   display name "Zulu VPP Custom")
	fallbackResults, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID2, team.ID)
	require.NoError(t, err)
	require.Len(t, fallbackResults, 6, "expected 6 results total (3 installers + 3 VPP apps)")

	assert.Equal(t, "ZZZ_Software", fallbackResults[0].Name, "row 0: ZZZ_Software (display name 'Alpha Custom')")
	assert.Equal(t, "Alpha Custom", fallbackResults[0].DisplayName)
	assert.NotNil(t, fallbackResults[0].SoftwareInstallerID)

	assert.Equal(t, "ZZZ_VPP_App", fallbackResults[1].Name, "row 1: ZZZ_VPP_App (display name 'Alpha VPP Custom')")
	assert.Equal(t, "Alpha VPP Custom", fallbackResults[1].DisplayName)
	assert.NotNil(t, fallbackResults[1].VPPAppTeamID)
	assert.Less(t, fallbackResults[0].ID, fallbackResults[1].ID)

	assert.Equal(t, "MMM_NoDisplayName", fallbackResults[2].Name, "row 2: MMM_NoDisplayName (no display name, falls back to st.name)")
	assert.Empty(t, fallbackResults[2].DisplayName)
	assert.NotNil(t, fallbackResults[2].SoftwareInstallerID)
	assert.Less(t, fallbackResults[1].ID, fallbackResults[2].ID)

	assert.Equal(t, "MMM_VPP_NoDisplayName", fallbackResults[3].Name, "row 3: MMM_VPP_NoDisplayName (no display name, falls back to st.name)")
	assert.Empty(t, fallbackResults[3].DisplayName)
	assert.NotNil(t, fallbackResults[3].VPPAppTeamID)

	assert.Equal(t, "AAA_Software", fallbackResults[4].Name, "row 4: AAA_Software (display name 'Zulu Custom')")
	assert.Equal(t, "Zulu Custom", fallbackResults[4].DisplayName)
	assert.NotNil(t, fallbackResults[4].SoftwareInstallerID)

	assert.Equal(t, "AAA_VPP_App", fallbackResults[5].Name, "row 5: AAA_VPP_App (display name 'Zulu VPP Custom')")
	assert.Equal(t, "Zulu VPP Custom", fallbackResults[5].DisplayName)
	assert.NotNil(t, fallbackResults[5].VPPAppTeamID)
	assert.Less(t, fallbackResults[4].ID, fallbackResults[5].ID)
}

type setupExperienceInsertTestRows struct {
	HostUUID            string        `db:"host_uuid"`
	Name                string        `db:"name"`
	Status              string        `db:"status"`
	SoftwareInstallerID sql.NullInt64 `db:"software_installer_id"`
	ScriptID            sql.NullInt64 `db:"setup_experience_script_id"`
	VPPAppTeamID        sql.NullInt64 `db:"vpp_app_team_id"`
}

func nullableUint(val uint) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(val), Valid: true} // nolint: gosec
}

func testGetSetupExperienceTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

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
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
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
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr4, err := fleet.NewTempFileReader(strings.NewReader("hello2"), t.TempDir)
	require.NoError(t, err)
	installerID4, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "pear",
		PreInstallQuery:   "SELECT 4",
		PostInstallScript: "apple",
		InstallerFile:     tfr4,
		StorageID:         "storage3",
		Filename:          "file4",
		Title:             "file4",
		Version:           "4.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.IOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	tfr5, err := fleet.NewTempFileReader(strings.NewReader("hello3"), t.TempDir)
	require.NoError(t, err)
	installerID5, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "orange",
		PreInstallQuery:   "SELECT 5",
		PostInstallScript: "grape",
		InstallerFile:     tfr5,
		StorageID:         "storage4",
		Filename:          "file5",
		Title:             "file5",
		Version:           "5.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          "linux",
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	titles, count, meta, err := ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, 1, count)
	assert.NotNil(t, meta)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE software_installers SET install_during_setup = 1 WHERE id IN (?, ?, ?, ?)", installerID1, installerID3, installerID4, installerID5)
		return err
	})

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, installerID1, titles[0].ID)
	assert.Equal(t, 1, count)
	assert.NotNil(t, meta)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	assert.Equal(t, installerID3, titles[0].ID)
	assert.Equal(t, 1, count)
	assert.NotNil(t, meta)

	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.IOSPlatform}}, BundleIdentifier: "b2"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	app3 := &fleet.VPPApp{Name: "vpp_app_3", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b3"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	require.NoError(t, err)

	vpp1, err := ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	vpp2, err := ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	vpp3, err := ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	require.NoError(t, err)

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE vpp_apps_teams SET install_during_setup = 1 WHERE adam_id IN (?, ?, ?)", vpp1.AdamID, vpp2.AdamID, vpp3.AdamID)
		return err
	})

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, vpp1.AdamID, titles[1].AppStoreApp.AppStoreID)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, vpp3.AdamID, titles[1].AppStoreApp.AppStoreID)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)

	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{
		TeamID:         &team1.ID,
		Name:           "the script.sh",
		ScriptContents: "hello",
	})
	require.NoError(t, err)

	sec, err := ds.GetSetupExperienceCount(ctx, "darwin", &team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(1), sec.Installers)
	require.Equal(t, uint(1), sec.VPP)
	require.Equal(t, uint(1), sec.Scripts)

	sec, err = ds.GetSetupExperienceCount(ctx, "linux", &team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(1), sec.Installers)
	require.Equal(t, uint(0), sec.VPP)
	require.Equal(t, uint(0), sec.Scripts)

	sec, err = ds.GetSetupExperienceCount(ctx, "darwin", &team2.ID)
	require.NoError(t, err)
	require.Equal(t, uint(1), sec.Installers)
	require.Equal(t, uint(1), sec.VPP)
	require.Equal(t, uint(0), sec.Scripts)

	sec, err = ds.GetSetupExperienceCount(ctx, "darwin", nil)
	require.NoError(t, err)
	require.Equal(t, uint(0), sec.Installers)
	require.Equal(t, uint(0), sec.VPP)
	require.Equal(t, uint(0), sec.Scripts)

	// add an ipa installer and check that it isn't listed for setup experience
	payload := fleet.UploadSoftwareInstallerPayload{
		TeamID:           &team1.ID,
		UserID:           user1.ID,
		Title:            "ipa_test",
		Filename:         "ipa_test.ipa",
		BundleIdentifier: "com.ipa_test",
		StorageID:        "testingtesting123",
		Platform:         "ios",
		Extension:        "ipa",
		Version:          "1.2.3",
		ValidatedLabels:  &fleet.LabelIdentsWithScope{},
	}
	_, _, err = ds.MatchOrCreateSoftwareInstaller(ctx, &payload)
	require.NoError(t, err)

	// definitely not listed for darwin
	titles, _, _, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	require.Equal(t, "file1", titles[0].Name)
	require.Equal(t, "vpp_app_1", titles[1].Name)

	// but also not listed for ios
	titles, _, _, err = ds.ListSetupExperienceSoftwareTitles(ctx, "ios", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 1)
	require.Equal(t, "vpp_app_2", titles[0].Name)
}

func testSetSetupExperienceTitles(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, ds)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

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
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID1
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
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID2
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
		TeamID:            &team2.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID3
	require.NoError(t, err)

	tfr4, err := fleet.NewTempFileReader(strings.NewReader("hello2"), t.TempDir)
	require.NoError(t, err)
	installerID4, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "pear",
		PreInstallQuery:   "SELECT 4",
		PostInstallScript: "apple",
		InstallerFile:     tfr4,
		StorageID:         "storage3",
		Filename:          "file4",
		Title:             "file4",
		Version:           "4.0",
		Source:            "apps",
		SelfService:       true,
		UserID:            user1.ID,
		TeamID:            &team2.ID,
		Platform:          string(fleet.IOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID4
	require.NoError(t, err)

	titles, count, meta, err := ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 2)
	assert.Equal(t, 2, count)
	assert.NotNil(t, meta)

	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	app2 := &fleet.VPPApp{Name: "vpp_app_2", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "2", Platform: fleet.IOSPlatform}}, BundleIdentifier: "b2"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app2, &team1.ID)
	require.NoError(t, err)

	app3 := &fleet.VPPApp{Name: "vpp_app_3", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "3", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b3"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app3, &team2.ID)
	require.NoError(t, err)

	// iOS version of app1, has the same adam ID
	app4 := &fleet.VPPApp{Name: "vpp_app_1: iOS", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.IOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app4, &team1.ID)
	require.NoError(t, err)

	titleSoftware := make(map[string]uint)
	titleVPP := make(map[string]uint)

	softwareTitles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team1.ID}, fleet.TeamFilter{TeamID: &team1.ID})
	require.NoError(t, err)

	for _, title := range softwareTitles {
		if title.AppStoreApp != nil {
			titleVPP[title.AppStoreApp.AppStoreID+":"+title.AppStoreApp.Platform] = title.ID
		} else if title.SoftwarePackage != nil {
			titleSoftware[title.SoftwarePackage.Name] = title.ID
		}
	}

	softwareTitles, _, _, err = ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team2.ID}, fleet.TeamFilter{TeamID: &team2.ID})
	require.NoError(t, err)

	for _, title := range softwareTitles {
		if title.AppStoreApp != nil {
			titleVPP[title.AppStoreApp.AppStoreID] = title.ID
		} else if title.SoftwarePackage != nil {
			titleSoftware[title.SoftwarePackage.Name] = title.ID
		}
	}

	// Single installer
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, []uint{titleSoftware["file1"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 3)
	assert.Equal(t, 3, count)
	assert.Equal(t, "file1", titles[0].SoftwarePackage.Name)
	assert.Equal(t, "file2", titles[1].SoftwarePackage.Name)
	assert.Equal(t, "1", titles[2].AppStoreApp.AppStoreID)
	assert.NotNil(t, meta)

	assert.True(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[2].AppStoreApp.InstallDuringSetup)

	// Single vpp app replaces installer
	// This VPP app has darwin and ios versions, which shouldn't keep users from adding the darwin one.
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, []uint{titleVPP["1:darwin"]})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 3)
	require.Equal(t, 3, count)
	assert.Equal(t, "file1", titles[0].SoftwarePackage.Name)
	assert.Equal(t, "file2", titles[1].SoftwarePackage.Name)
	assert.Equal(t, "1", titles[2].AppStoreApp.AppStoreID)
	assert.NotNil(t, meta)

	assert.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].SoftwarePackage.InstallDuringSetup)
	assert.True(t, *titles[2].AppStoreApp.InstallDuringSetup)

	// Team 2 unaffected
	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team2.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 2)
	require.Equal(t, 2, count)
	assert.Equal(t, "file3", titles[0].SoftwarePackage.Name)
	assert.Equal(t, "3", titles[1].AppStoreApp.AppStoreID)
	require.NotNil(t, meta)

	assert.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].AppStoreApp.InstallDuringSetup)

	// VPP app can be added for iOS
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "ios", team1.ID, []uint{titleVPP["2:ios"]})
	require.NoError(t, err)
	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "ios", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, titles, 2)
	require.Equal(t, 2, count)
	require.NotNil(t, meta)
	installDuringSetupApps := 0
	for _, title := range titles {
		// iOS should only have vpp apps
		require.NotNil(t, title.AppStoreApp)
		if title.ID == titleVPP["2:ios"] {
			require.True(t, *title.AppStoreApp.InstallDuringSetup)
			installDuringSetupApps++
		} else {
			require.False(t, *title.AppStoreApp.InstallDuringSetup)
		}
	}
	require.Equal(t, 1, installDuringSetupApps)

	// iOS software. iOS only supports VPP apps so should not check installers
	// even if one somehow exists
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "ios", team2.ID, []uint{titleSoftware["file4"]})
	require.ErrorContains(t, err, "not available")

	// ios vpp app is invalid for darwin platform
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, []uint{titleVPP["2:ios"]})
	require.ErrorContains(t, err, "invalid platform for requested AppStoreApp")

	// wrong team
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, []uint{titleVPP["3"]})
	require.ErrorContains(t, err, "not available")

	// good other team assignment
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team2.ID, []uint{titleVPP["3"]})
	require.NoError(t, err)

	// non-existent title ID
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, []uint{999})
	require.ErrorContains(t, err, "not available")

	// Failures and other team assignments didn't affected the number of apps on team 1
	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 3)
	assert.Equal(t, 3, count)
	assert.NotNil(t, meta)

	// Empty slice removes all tiles
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, []uint{})
	require.NoError(t, err)

	titles, count, meta, err = ds.ListSetupExperienceSoftwareTitles(ctx, "darwin", team1.ID, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, titles, 3)
	assert.Equal(t, 3, count)
	assert.NotNil(t, meta)

	assert.False(t, *titles[0].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[1].SoftwarePackage.InstallDuringSetup)
	assert.False(t, *titles[2].AppStoreApp.InstallDuringSetup)
}

func testSetupExperienceStatusResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hostUUID := uuid.NewString()

	// Create a software installer
	// We need a new user first
	user, err := ds.NewUser(ctx, &fleet.User{Name: "Foo", Email: "foo@example.com", GlobalRole: ptr.String("admin"), Password: []byte("12characterslong!")})
	require.NoError(t, err)
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Filename:        "test.pkg",
		Title:           "Test Software",
		Version:         "1.0.0",
		Source:          "apps",
		Platform:        "darwin",
		Extension:       "pkg",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	installer, err := ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	require.NoError(t, err)

	// VPP setup: create a token so that we can insert a VPP app
	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Donkey Kong", "Jungle")
	require.NoError(t, err)
	tok1, err := ds.InsertVPPToken(ctx, dataToken)
	assert.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tok1.ID, []uint{})
	assert.NoError(t, err)
	vppApp, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{BundleIdentifier: "com.test.test", Name: "test.app", LatestVersion: "1.0.0"}, nil)
	require.NoError(t, err)
	var vppAppsTeamsID uint
	err = sqlx.GetContext(context.Background(), ds.reader(ctx),
		&vppAppsTeamsID, `SELECT id FROM vpp_apps_teams WHERE adam_id = ?`,
		vppApp.AdamID,
	)
	require.NoError(t, err)

	// TODO: use DS methods once those are written
	var scriptID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO setup_experience_scripts (name) VALUES (?)`,
			"test_script")
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		scriptID = uint(id) // nolint: gosec
		return nil
	})

	insertSetupExperienceStatusResult := func(sesr *fleet.SetupExperienceStatusResult) {
		stmt := `INSERT INTO setup_experience_status_results (id, host_uuid, name, status, software_installer_id, host_software_installs_execution_id, vpp_app_team_id, nano_command_uuid, setup_experience_script_id, script_execution_id, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx, stmt,
				sesr.ID, sesr.HostUUID, sesr.Name, sesr.Status, sesr.SoftwareInstallerID, sesr.HostSoftwareInstallsExecutionID, sesr.VPPAppTeamID, sesr.NanoCommandUUID, sesr.SetupExperienceScriptID, sesr.ScriptExecutionID, sesr.Error)
			require.NoError(t, err)
			id, err := res.LastInsertId()
			require.NoError(t, err)
			sesr.ID = uint(id) // nolint: gosec
			return nil
		})
	}

	expRes := []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:            hostUUID,
			Name:                "Test Software",
			Status:              fleet.SetupExperienceStatusPending,
			SoftwareInstallerID: ptr.Uint(installerID),
			SoftwareTitleID:     installer.TitleID,
			Source:              ptr.String("apps"),
		},
		{
			HostUUID:        hostUUID,
			Name:            "vpp",
			Status:          fleet.SetupExperienceStatusPending,
			VPPAppTeamID:    ptr.Uint(vppAppsTeamsID),
			SoftwareTitleID: ptr.Uint(vppApp.TitleID),
			Source:          ptr.String("apps"),
		},
		{
			HostUUID:                hostUUID,
			Name:                    "script",
			Status:                  fleet.SetupExperienceStatusPending,
			SetupExperienceScriptID: ptr.Uint(scriptID),
			Source:                  nil, // Scripts don't have a source (no software title)
		},
	}

	for _, r := range expRes {
		insertSetupExperienceStatusResult(r)
	}

	res, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID, 0)
	require.NoError(t, err)
	require.Len(t, res, 3)
	for i, s := range expRes {
		require.Equal(t, s, res[i])
	}
}

func testSetupExperienceScriptCRUD(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// create a script for team1
	wantScript1 := &fleet.Script{
		Name:           "script",
		TeamID:         &team1.ID,
		ScriptContents: "echo foo",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// get the script for team1
	gotScript1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, gotScript1)
	require.Equal(t, wantScript1.Name, gotScript1.Name)
	require.Equal(t, wantScript1.TeamID, gotScript1.TeamID)
	require.NotZero(t, gotScript1.ScriptContentID)

	b, err := ds.GetAnyScriptContents(ctx, gotScript1.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScript1.ScriptContents, string(b))

	// create a script for team2
	wantScript2 := &fleet.Script{
		Name:           "script",
		TeamID:         &team2.ID,
		ScriptContents: "echo bar",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScript2)
	require.NoError(t, err)

	// get the script for team2
	gotScript2, err := ds.GetSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)
	require.NotNil(t, gotScript2)
	require.Equal(t, wantScript2.Name, gotScript2.Name)
	require.Equal(t, wantScript2.TeamID, gotScript2.TeamID)
	require.NotZero(t, gotScript2.ScriptContentID)
	require.NotEqual(t, gotScript1.ScriptContentID, gotScript2.ScriptContentID)

	b, err = ds.GetAnyScriptContents(ctx, gotScript2.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScript2.ScriptContents, string(b))

	// create a script with no team id
	wantScriptNoTeam := &fleet.Script{
		Name:           "script",
		ScriptContents: "echo bar",
	}

	err = ds.SetSetupExperienceScript(ctx, wantScriptNoTeam)
	require.NoError(t, err)

	// get the script nil team id is equivalent to team id 0
	gotScriptNoTeam, err := ds.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, gotScriptNoTeam)
	require.Equal(t, wantScriptNoTeam.Name, gotScriptNoTeam.Name)
	require.Nil(t, gotScriptNoTeam.TeamID)
	require.NotZero(t, gotScriptNoTeam.ScriptContentID)
	require.Equal(t, gotScript2.ScriptContentID, gotScriptNoTeam.ScriptContentID) // should be the same as team2

	b, err = ds.GetAnyScriptContents(ctx, gotScriptNoTeam.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, wantScriptNoTeam.ScriptContents, string(b))

	// try to create another with name "script" and no team id. Should succeed
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script", ScriptContents: "echo baz"})
	require.NoError(t, err)

	// try to create another script with no team id and a different name. Should succeed
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "script2", ScriptContents: "echo baz"})
	require.NoError(t, err)

	// try to add a script for a team that doesn't exist
	var fkErr fleet.ForeignKeyError
	err = ds.SetSetupExperienceScript(ctx, &fleet.Script{TeamID: ptr.Uint(42), Name: "script", ScriptContents: "echo baz"})
	require.Error(t, err)
	require.ErrorAs(t, err, &fkErr)

	// delete the script for team1
	err = ds.DeleteSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)

	// get the script for team1
	_, err = ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// try to delete script for team1 again
	err = ds.DeleteSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err) // TODO: confirm if we want to return not found on deletes

	// try to delete script for team that doesn't exist
	err = ds.DeleteSetupExperienceScript(ctx, ptr.Uint(42))
	require.NoError(t, err) // TODO: confirm if we want to return not found on deletes

	// add same script for team1 again(even though there will be no update since it doesn't exist)
	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// get the script for team1
	oldScript1 := gotScript1
	newScript1, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, newScript1)
	require.Equal(t, wantScript1.Name, newScript1.Name)
	require.Equal(t, wantScript1.TeamID, newScript1.TeamID)
	require.NotZero(t, newScript1.ScriptContentID)
	// script contents are deleted by CleanupUnusedScriptContents not by DeleteSetupExperienceScript
	// so the content id should be the same as the old
	require.Equal(t, oldScript1.ScriptContentID, newScript1.ScriptContentID)

	// add same script for team1 again
	err = ds.SetSetupExperienceScript(ctx, wantScript1)
	require.NoError(t, err)

	// Verify that the script contents remained the same
	newScript1, err = ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, newScript1)
	require.Equal(t, wantScript1.Name, newScript1.Name)
	require.Equal(t, wantScript1.TeamID, newScript1.TeamID)
	require.NotZero(t, newScript1.ScriptContentID)
	// script contents are deleted by CleanupUnusedScriptContents not by DeleteSetupExperienceScript
	// so the content id should be the same as the old
	require.Equal(t, oldScript1.ScriptContentID, newScript1.ScriptContentID)
}

func testUpdateSetupExperienceScriptWhileEnqueued(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// create scripts for team1 and team2
	initialScript1 := &fleet.Script{
		Name:           "script",
		TeamID:         &team1.ID,
		ScriptContents: "echo foo",
	}

	initialScript2 := &fleet.Script{
		Name:           "script",
		TeamID:         &team2.ID,
		ScriptContents: "echo bar",
	}

	// and an "updated" script for team1
	updatedScript1 := &fleet.Script{
		Name:           "script",
		TeamID:         &team1.ID,
		ScriptContents: "echo updated foo",
	}

	err = ds.SetSetupExperienceScript(ctx, initialScript1)
	require.NoError(t, err)
	team1OriginalScript, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, team1OriginalScript)

	err = ds.SetSetupExperienceScript(ctx, initialScript2)
	require.NoError(t, err)
	team2OriginalScript, err := ds.GetSetupExperienceScript(ctx, &team2.ID)
	require.NoError(t, err)
	require.NotNil(t, team2OriginalScript)

	hostTeam1UUID := "123"
	hostTeam2UUID := "456"

	anythingEnqueued, err := ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam1UUID, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam2UUID, team2.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	host1OriginalItems, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam1UUID, team1.ID)
	require.NoError(t, err)
	require.Len(t, host1OriginalItems, 1)
	require.Equal(t, fleet.SetupExperienceStatusPending, host1OriginalItems[0].Status)
	require.NotNil(t, host1OriginalItems[0].SetupExperienceScriptID)
	require.Equal(t, team1OriginalScript.ID, *host1OriginalItems[0].SetupExperienceScriptID)

	host2OriginalItems, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam2UUID, team2.ID)
	require.NoError(t, err)
	require.Len(t, host2OriginalItems, 1)
	require.Equal(t, fleet.SetupExperienceStatusPending, host2OriginalItems[0].Status)
	require.NotNil(t, host2OriginalItems[0].SetupExperienceScriptID)
	require.Equal(t, team2OriginalScript.ID, *host2OriginalItems[0].SetupExperienceScriptID)

	// "Update" the script for team1 with its original contents which should cause no change to the enqueued execution
	err = ds.SetSetupExperienceScript(ctx, initialScript1)
	require.NoError(t, err)

	team1UpdatedScript, err := ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, team1UpdatedScript)
	require.Equal(t, team1OriginalScript.ScriptContentID, team1UpdatedScript.ScriptContentID)
	require.Equal(t, team1OriginalScript.ID, team1UpdatedScript.ID)

	host1NewItems, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam1UUID, team1.ID)
	require.NoError(t, err)
	require.Len(t, host1NewItems, 1)
	require.Equal(t, team1OriginalScript.ID, *host1NewItems[0].SetupExperienceScriptID)

	// Should not have perturbed Host 2's enqueued execution either
	host2NewItems, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam2UUID, team2.ID)
	require.NoError(t, err)
	require.Len(t, host2NewItems, 1)
	require.Equal(t, team2OriginalScript.ID, *host2NewItems[0].SetupExperienceScriptID)

	// update script for team1 which should delete the enqueued execution
	err = ds.SetSetupExperienceScript(ctx, updatedScript1)
	require.NoError(t, err)

	team1UpdatedScript, err = ds.GetSetupExperienceScript(ctx, &team1.ID)
	require.NoError(t, err)
	require.NotNil(t, team1UpdatedScript)
	require.NotEqual(t, team1OriginalScript.ScriptContentID, team1UpdatedScript.ScriptContentID)
	require.NotEqual(t, team1OriginalScript.ID, team1UpdatedScript.ID)

	host1NewItems, err = ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam1UUID, team1.ID)
	require.NoError(t, err)
	require.Len(t, host1NewItems, 0)

	// Should not have affected host 2's enqueued execution
	host2NewItems, err = ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam2UUID, team2.ID)
	require.NoError(t, err)
	require.Len(t, host2NewItems, 1)
	require.Equal(t, team2OriginalScript.ID, *host2NewItems[0].SetupExperienceScriptID)

	// re-enqueue items for host 1, should enqueue the updated script
	anythingEnqueued, err = ds.EnqueueSetupExperienceItems(ctx, "darwin", "darwin", hostTeam1UUID, team1.ID)
	require.NoError(t, err)
	require.True(t, anythingEnqueued)

	host1NewItems, err = ds.ListSetupExperienceResultsByHostUUID(ctx, hostTeam1UUID, team1.ID)
	require.NoError(t, err)
	require.Len(t, host1NewItems, 1)
	require.Equal(t, team1UpdatedScript.ID, *host1NewItems[0].SetupExperienceScriptID)
}

func testHostInSetupExperience(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	err := ds.SetHostAwaitingConfiguration(ctx, "abc", true)
	require.NoError(t, err)

	inSetupExperience, err := ds.GetHostAwaitingConfiguration(ctx, "abc")
	require.NoError(t, err)
	require.True(t, inSetupExperience)

	err = ds.SetHostAwaitingConfiguration(ctx, "abc", false)
	require.NoError(t, err)

	inSetupExperience, err = ds.GetHostAwaitingConfiguration(ctx, "abc")
	require.NoError(t, err)
	require.False(t, inSetupExperience)

	// host without a record in the table returns not found
	inSetupExperience, err = ds.GetHostAwaitingConfiguration(ctx, "404")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.False(t, inSetupExperience)
}

func testUpdateStatusGuardsTerminalStates(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hostUUID := uuid.NewString()

	// --- Set up foreign-key references ---

	// User (required for software installer)
	user, err := ds.NewUser(ctx, &fleet.User{
		Name:       "GuardTest",
		Email:      "guard@example.com",
		GlobalRole: new("admin"),
		Password:   []byte("12characterslong!"),
	})
	require.NoError(t, err)

	// Software installer
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		Filename:        "guard_test.pkg",
		Title:           "Guard Test Software",
		Version:         "1.0.0",
		Source:          "apps",
		Platform:        "darwin",
		Extension:       "pkg",
		UserID:          user.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	// VPP token + app
	dataToken, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "Guard Kong", "GuardJungle")
	require.NoError(t, err)
	tok, err := ds.InsertVPPToken(ctx, dataToken)
	require.NoError(t, err)
	_, err = ds.UpdateVPPTokenTeams(ctx, tok.ID, []uint{})
	require.NoError(t, err)
	vppApp, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		BundleIdentifier: "com.guard.test",
		Name:             "guard_test.app",
		LatestVersion:    "1.0.0",
	}, nil)
	require.NoError(t, err)
	var vppAppsTeamsID uint
	err = sqlx.GetContext(ctx, ds.reader(ctx), &vppAppsTeamsID,
		`SELECT id FROM vpp_apps_teams WHERE adam_id = ?`, vppApp.AdamID)
	require.NoError(t, err)

	// Setup experience script (raw SQL, same pattern as testSetupExperienceStatusResults)
	var scriptID uint
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO setup_experience_scripts (name) VALUES (?)`, "guard_test_script")
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		scriptID = uint(id) //nolint: gosec
		return nil
	})

	// --- Helpers ---

	insertRow := func(sesr *fleet.SetupExperienceStatusResult) {
		stmt := `INSERT INTO setup_experience_status_results
			(id, host_uuid, name, status, software_installer_id,
			 host_software_installs_execution_id, vpp_app_team_id,
			 nano_command_uuid, setup_experience_script_id,
			 script_execution_id, error)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			res, err := q.ExecContext(ctx, stmt,
				sesr.ID, sesr.HostUUID, sesr.Name, sesr.Status,
				sesr.SoftwareInstallerID,
				sesr.HostSoftwareInstallsExecutionID,
				sesr.VPPAppTeamID, sesr.NanoCommandUUID,
				sesr.SetupExperienceScriptID,
				sesr.ScriptExecutionID, sesr.Error)
			require.NoError(t, err)
			id, err := res.LastInsertId()
			require.NoError(t, err)
			sesr.ID = uint(id) //nolint: gosec
			return nil
		})
	}

	readStatus := func(id uint) fleet.SetupExperienceStatusResultStatus {
		var status fleet.SetupExperienceStatusResultStatus
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &status,
				"SELECT status FROM setup_experience_status_results WHERE id = ?", id)
		})
		return status
	}

	// --- Negative tests: terminal states must not be overwritten ---

	terminalStatuses := []fleet.SetupExperienceStatusResultStatus{
		fleet.SetupExperienceStatusCancelled,
		fleet.SetupExperienceStatusFailure,
		fleet.SetupExperienceStatusSuccess,
	}

	for _, termStatus := range terminalStatuses {
		// Software installer row
		execID := uuid.NewString()
		row := &fleet.SetupExperienceStatusResult{
			HostUUID:                        hostUUID,
			Name:                            "sw-" + string(termStatus),
			Status:                          termStatus,
			SoftwareInstallerID:             new(installerID),
			HostSoftwareInstallsExecutionID: new(execID),
		}
		insertRow(row)
		updated, err := ds.MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx, hostUUID, execID, fleet.SetupExperienceStatusFailure)
		require.NoError(t, err)
		require.False(t, updated, "software installer row in %s should not be updated", termStatus)
		require.Equal(t, termStatus, readStatus(row.ID))

		// VPP row
		nanoUUID := uuid.NewString()
		row = &fleet.SetupExperienceStatusResult{
			HostUUID:        hostUUID,
			Name:            "vpp-" + string(termStatus),
			Status:          termStatus,
			VPPAppTeamID:    new(vppAppsTeamsID),
			NanoCommandUUID: new(nanoUUID),
		}
		insertRow(row)
		updated, err = ds.MaybeUpdateSetupExperienceVPPStatus(ctx, hostUUID, nanoUUID, fleet.SetupExperienceStatusFailure)
		require.NoError(t, err)
		require.False(t, updated, "VPP row in %s should not be updated", termStatus)
		require.Equal(t, termStatus, readStatus(row.ID))

		// Script row
		scriptExecID := uuid.NewString()
		row = &fleet.SetupExperienceStatusResult{
			HostUUID:                hostUUID,
			Name:                    "script-" + string(termStatus),
			Status:                  termStatus,
			SetupExperienceScriptID: new(scriptID),
			ScriptExecutionID:       new(scriptExecID),
		}
		insertRow(row)
		updated, err = ds.MaybeUpdateSetupExperienceScriptStatus(ctx, hostUUID, scriptExecID, fleet.SetupExperienceStatusFailure)
		require.NoError(t, err)
		require.False(t, updated, "script row in %s should not be updated", termStatus)
		require.Equal(t, termStatus, readStatus(row.ID))
	}

	// --- Positive control: pending row CAN be updated ---

	pendingExecID := uuid.NewString()
	pendingRow := &fleet.SetupExperienceStatusResult{
		HostUUID:                        hostUUID,
		Name:                            "sw-pending-positive",
		Status:                          fleet.SetupExperienceStatusPending,
		SoftwareInstallerID:             new(installerID),
		HostSoftwareInstallsExecutionID: new(pendingExecID),
	}
	insertRow(pendingRow)
	updated, err := ds.MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx, hostUUID, pendingExecID, fleet.SetupExperienceStatusFailure)
	require.NoError(t, err)
	require.True(t, updated, "pending row should be updated")
	require.Equal(t, fleet.SetupExperienceStatusFailure, readStatus(pendingRow.ID))

	// --- Bug-scenario test: canceled VPP row must not flip to failure ---

	cancelledNanoUUID := uuid.NewString()
	cancelledVPPRow := &fleet.SetupExperienceStatusResult{
		HostUUID:        hostUUID,
		Name:            "vpp-canceled-bug",
		Status:          fleet.SetupExperienceStatusCancelled,
		VPPAppTeamID:    new(vppAppsTeamsID),
		NanoCommandUUID: new(cancelledNanoUUID),
	}
	insertRow(cancelledVPPRow)
	updated, err = ds.MaybeUpdateSetupExperienceVPPStatus(ctx, hostUUID, cancelledNanoUUID, fleet.SetupExperienceStatusFailure)
	require.NoError(t, err)
	require.False(t, updated, "cancelled VPP row must not be overwritten by late failure result")
	require.Equal(t, fleet.SetupExperienceStatusCancelled, readStatus(cancelledVPPRow.ID))
}

func testGetSetupExperienceScriptByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	script := &fleet.Script{
		Name:           "setup_experience_script",
		ScriptContents: "echo hello",
	}

	err := ds.SetSetupExperienceScript(ctx, script)
	require.NoError(t, err)

	scriptByTeamID, err := ds.GetSetupExperienceScript(ctx, nil)
	require.NoError(t, err)

	gotScript, err := ds.GetSetupExperienceScriptByID(ctx, scriptByTeamID.ID)
	require.NoError(t, err)

	require.Equal(t, script.Name, gotScript.Name)
	require.NotZero(t, gotScript.ScriptContentID)

	b, err := ds.GetAnyScriptContents(ctx, gotScript.ScriptContentID)
	require.NoError(t, err)
	require.Equal(t, script.ScriptContents, string(b))
}

func testSetSetupExperienceTitlesOnlyMarksActiveInstaller(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team_setup_exp_active"})
	require.NoError(t, err)

	fma, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "pkg_active",
		Slug:             "pkg_active",
		Platform:         "darwin",
		UniqueIdentifier: "fleet.pkg_active",
	})
	require.NoError(t, err)

	tfr, err := fleet.NewTempFileReader(strings.NewReader("file contents"), t.TempDir)
	require.NoError(t, err)

	// Create two cached FMA versions via successive GitOps runs. v1.0 ends
	// up inactive, v2.0 active.
	for _, version := range []string{"1.0", "2.0"} {
		err = ds.BatchSetSoftwareInstallers(ctx, &team.ID, []*fleet.UploadSoftwareInstallerPayload{
			{
				FleetMaintainedAppID: &fma.ID,
				Title:                "pkg_active",
				Source:               "apps",
				Platform:             "darwin",
				PreInstallQuery:      "SELECT 1",
				InstallScript:        "echo install",
				PostInstallScript:    "echo post install",
				UninstallScript:      "echo uninstall",
				InstallerFile:        tfr,
				StorageID:            "storage_id",
				Filename:             "pkg_active.pkg",
				Version:              version,
				UserID:               user.ID,
				ValidatedLabels:      &fleet.LabelIdentsWithScope{},
				InstallDuringSetup:   new(false),
				SelfService:          false,
				TeamID:               &team.ID,
			},
		})
		require.NoError(t, err)
	}

	// Grab the two installer IDs so we can assert per-row.
	type row struct {
		ID      uint `db:"id"`
		Active  bool `db:"is_active"`
		InSetup bool `db:"install_during_setup"`
		TitleID uint `db:"title_id"`
		Version string
	}
	var rows []row
	tmFilter := fleet.TeamFilter{User: test.UserAdmin, TeamID: &team.ID}
	titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{TeamID: &team.ID, Platform: "darwin", AvailableForInstall: true}, tmFilter)
	require.NoError(t, err)
	require.Len(t, titles, 1)
	titleID := titles[0].ID

	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, tx, &rows, `
			SELECT id, is_active, install_during_setup, title_id, version
			FROM software_installers
			WHERE global_or_team_id = ? AND title_id = ?
			ORDER BY version ASC
		`, team.ID, titleID)
	})
	require.Len(t, rows, 2, "expected 2 cached FMA versions")
	require.False(t, rows[0].Active, "v1.0 should be inactive")
	require.True(t, rows[1].Active, "v2.0 should be active")

	// Sanity: neither row has install_during_setup set yet (BatchSet was
	// called with InstallDuringSetup=false).
	require.False(t, rows[0].InSetup)
	require.False(t, rows[1].InSetup)

	// Add the title to setup experience.
	err = ds.SetSetupExperienceSoftwareTitles(ctx, "darwin", team.ID, []uint{titleID})
	require.NoError(t, err)

	// Re-read: only the active (v2.0) row should have install_during_setup=true.
	rows = nil
	ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, tx, &rows, `
			SELECT id, is_active, install_during_setup, title_id, version
			FROM software_installers
			WHERE global_or_team_id = ? AND title_id = ?
			ORDER BY version ASC
		`, team.ID, titleID)
	})
	require.Len(t, rows, 2)
	require.False(t, rows[0].InSetup, "cached inactive v1.0 must not be marked install_during_setup")
	require.True(t, rows[1].InSetup, "active v2.0 should be marked install_during_setup")
}
