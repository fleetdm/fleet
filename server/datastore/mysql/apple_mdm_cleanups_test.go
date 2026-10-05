package mysql

import (
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAppleMDMCleanups(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"InactivePurge", testNanoCleanupInactivePurge},
		{"DisabledAndBudgets", testNanoCleanupDisabledAndBudgets},
		{"PinnedWindowDoesNotStarve", testNanoCleanupPinnedWindowDoesNotStarve},
		{"QueueFirstAtomicity", testNanoCleanupQueueFirstAtomicity},
		{"MopKeepsReferencedCommand", testNanoCleanupMopKeepsReferencedCommand},
		{"ShortRetentionTier", testNanoCleanupShortRetentionTier},
		{"StandardRetentionTier", testNanoCleanupStandardRetentionTier},
		{"ShortTierOffFallsBackToStandard", testNanoCleanupShortTierOffFallsBackToStandard},
		{"RetentionCursorResumesAndWraps", testNanoCleanupRetentionCursorResumesAndWraps},
		{"RowBudgetStopsLaterSweeps", testNanoCleanupRowBudgetStopsLaterSweeps},
		{"ScanCapDoesNotStopLaterSweeps", testNanoCleanupScanCapDoesNotStopLaterSweeps},
		{"RetentionKeepsFannedCommand", testNanoCleanupRetentionKeepsFannedCommand},
		{"OrphanMop", testNanoCleanupOrphanMop},
		{"OrphanMopCursorAndBudget", testNanoCleanupOrphanMopCursorAndBudget},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// nanoCleanupFixture seeds one host with device and user enrollments and
// offers helpers to enqueue, answer, deactivate and inspect commands.
type nanoCleanupFixture struct {
	t         *testing.T
	ds        *Datastore
	hostID    uint
	deviceID  string
	userID    string
	commander *apple_mdm.MDMAppleCommander
}

func newNanoCleanupFixture(t *testing.T, ds *Datastore) *nanoCleanupFixture {
	ctx := t.Context()
	host := test.NewHost(t, ds, "nano-cleanup.local", "1.1.1.1", "nano-cleanup-osquery", "nano-cleanup-node", time.Now())
	nanoEnroll(t, ds, host, true)
	userEnrollment, err := ds.GetNanoMDMUserEnrollment(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, userEnrollment)
	commander, _ := createMDMAppleCommanderAndStorage(t, ds)
	return &nanoCleanupFixture{t: t, ds: ds, hostID: host.ID, deviceID: host.UUID, userID: userEnrollment.ID, commander: commander}
}

func (f *nanoCleanupFixture) enqueue(reqType string, enrollmentIDs ...string) string {
	return f.enqueueUUID(reqType, uuid.NewString(), enrollmentIDs...)
}

func (f *nanoCleanupFixture) enqueueUUID(reqType, cmdUUID string, enrollmentIDs ...string) string {
	require.NoError(f.t, f.commander.EnqueueCommand(f.t.Context(), enrollmentIDs, createRawAppleCmd(reqType, cmdUUID)))
	return cmdUUID
}

// completed enqueues a command on the device channel and records a terminal
// result ago in the past, the shape the retention sweeps look for.
func (f *nanoCleanupFixture) completed(reqType, cmdUUID, status string, ago time.Duration) string {
	f.enqueueUUID(reqType, cmdUUID, f.deviceID)
	f.report(f.deviceID, cmdUUID, status)
	f.ageResult(f.deviceID, cmdUUID, ago)
	return cmdUUID
}

func (f *nanoCleanupFixture) ageResult(enrollmentID, cmdUUID string, ago time.Duration) {
	_, err := f.ds.writer(f.t.Context()).ExecContext(f.t.Context(),
		`UPDATE nano_command_results SET updated_at = NOW(6) - INTERVAL ? SECOND WHERE id = ? AND command_uuid = ?`,
		int(ago.Seconds()), enrollmentID, cmdUUID)
	require.NoError(f.t, err)
}

// orphan enqueues a command, removes its queue rows the way nano's own paths
// do, and backdates the command row to createdAgo.
func (f *nanoCleanupFixture) orphan(reqType string, createdAgo time.Duration) string {
	c := f.enqueue(reqType, f.deviceID)
	f.exec(`DELETE FROM nano_enrollment_queue WHERE command_uuid = ?`, c)
	f.exec(`UPDATE nano_commands SET created_at = NOW(6) - INTERVAL ? SECOND WHERE command_uuid = ?`, int(createdAgo.Seconds()), c)
	return c
}

func (f *nanoCleanupFixture) exec(query string, args ...any) {
	_, err := f.ds.writer(f.t.Context()).ExecContext(f.t.Context(), query, args...)
	require.NoError(f.t, err)
}

func (f *nanoCleanupFixture) report(enrollmentID, cmdUUID, status string) {
	// the table requires a plist-looking result body
	_, err := f.ds.writer(f.t.Context()).ExecContext(f.t.Context(),
		`INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, '<?xml version="1.0"?><plist/>')`,
		enrollmentID, cmdUUID, status)
	require.NoError(f.t, err)
}

// deactivate marks the queue row inactive as of deactivatedAgo, leaving
// created_at at createdAgo so the two ages can differ.
func (f *nanoCleanupFixture) deactivate(enrollmentID, cmdUUID string, createdAgo, deactivatedAgo time.Duration) {
	_, err := f.ds.writer(f.t.Context()).ExecContext(f.t.Context(),
		`UPDATE nano_enrollment_queue
		 SET active = 0, created_at = NOW(6) - INTERVAL ? SECOND, updated_at = NOW(6) - INTERVAL ? SECOND
		 WHERE id = ? AND command_uuid = ?`,
		int(createdAgo.Seconds()), int(deactivatedAgo.Seconds()), enrollmentID, cmdUUID)
	require.NoError(f.t, err)
}

func (f *nanoCleanupFixture) age(enrollmentID, cmdUUID string, ago time.Duration) {
	_, err := f.ds.writer(f.t.Context()).ExecContext(f.t.Context(),
		`UPDATE nano_enrollment_queue SET created_at = NOW(6) - INTERVAL ? SECOND, updated_at = NOW(6) - INTERVAL ? SECOND
		 WHERE id = ? AND command_uuid = ?`,
		int(ago.Seconds()), int(ago.Seconds()), enrollmentID, cmdUUID)
	require.NoError(f.t, err)
}

func (f *nanoCleanupFixture) count(table, where string, args ...any) int {
	var n int
	require.NoError(f.t, f.ds.writer(f.t.Context()).GetContext(f.t.Context(), &n, `SELECT COUNT(*) FROM `+table+` WHERE `+where, args...))
	return n
}

func (f *nanoCleanupFixture) queueRows(cmdUUID string) int {
	return f.count("nano_enrollment_queue", "command_uuid = ?", cmdUUID)
}
func (f *nanoCleanupFixture) resultRows(cmdUUID string) int {
	return f.count("nano_command_results", "command_uuid = ?", cmdUUID)
}
func (f *nanoCleanupFixture) commandRows(cmdUUID string) int {
	return f.count("nano_commands", "command_uuid = ?", cmdUUID)
}

const (
	nanoCleanupDay      = 24 * time.Hour
	nanoCleanupDefaults = 1000
)

func nanoCleanupOpts(short time.Duration, rows, cmds int) fleet.MDMAppleCommandCleanupOptions {
	return nanoCleanupTiers(short, 30*nanoCleanupDay, rows, cmds)
}

func nanoCleanupTiers(short, standard time.Duration, rows, cmds int) fleet.MDMAppleCommandCleanupOptions {
	return fleet.MDMAppleCommandCleanupOptions{ShortRetention: short, StandardRetention: standard, MaxRowDeletions: rows, MaxCmdDeletions: cmds}
}

func testNanoCleanupInactivePurge(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)

	// deactivated two days ago, answered: pair and command go
	oldDead := f.enqueue("DeviceInformation", f.deviceID)
	f.report(f.deviceID, oldDead, "Acknowledged")
	f.deactivate(f.deviceID, oldDead, 2*nanoCleanupDay, 2*nanoCleanupDay)
	// deactivated two days ago, never answered: the queue row alone is a pair
	oldDeadNoResult := f.enqueue("DeviceInformation", f.deviceID)
	f.deactivate(f.deviceID, oldDeadNoResult, 2*nanoCleanupDay, 2*nanoCleanupDay)
	// an old command deactivated five minutes ago: updated_at decides, not created_at
	freshDead := f.enqueue("DeviceInformation", f.deviceID)
	f.deactivate(f.deviceID, freshDead, 3*nanoCleanupDay, 5*time.Minute)
	// denylisted type: its inactive row is read back for lock status
	lockDead := f.enqueue("DeviceLock", f.deviceID)
	f.deactivate(f.deviceID, lockDead, 2*nanoCleanupDay, 2*nanoCleanupDay)
	// still active, however old
	live := f.enqueue("DeviceInformation", f.deviceID)
	f.age(f.deviceID, live, 2*nanoCleanupDay)
	// inactive but still a host's current profile command: guarded
	guarded := f.enqueue("InstallProfile", f.deviceID)
	f.deactivate(f.deviceID, guarded, 2*nanoCleanupDay, 2*nanoCleanupDay)
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO host_mdm_apple_profiles (host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
		VALUES (?, 'prof-1', 'com.example.one', ?, UNHEX(MD5('a')), 'install', 'verified')`, f.deviceID, guarded)
	require.NoError(t, err)
	// fanned out to both channels; only the device pair is dead, so the
	// command survives for the user pair
	fanned := f.enqueue("DeviceInformation", f.deviceID, f.userID)
	f.deactivate(f.deviceID, fanned, 2*nanoCleanupDay, 2*nanoCleanupDay)
	// a promoted recovery-lock command is still the host's current one
	recovery := f.enqueue(fleet.SetRecoveryLockCmdName, f.deviceID)
	f.deactivate(f.deviceID, recovery, 2*nanoCleanupDay, 2*nanoCleanupDay)
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, status, operation_type, set_command_uuid)
		VALUES (?, 'enc', 'verified', 'install', ?)`, f.deviceID, recovery)
	require.NoError(t, err)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 3, CommandsDeleted: 2}, stats)

	for _, gone := range []string{oldDead, oldDeadNoResult} {
		require.Zero(t, f.queueRows(gone), gone)
		require.Zero(t, f.resultRows(gone), gone)
		require.Zero(t, f.commandRows(gone), gone)
	}
	for _, kept := range []string{freshDead, lockDead, live, guarded, recovery} {
		require.Equal(t, 1, f.queueRows(kept), kept)
		require.Equal(t, 1, f.commandRows(kept), kept)
	}
	require.Equal(t, 1, f.queueRows(fanned), "user-channel pair survives")
	require.Zero(t, f.count("nano_enrollment_queue", "id = ? AND command_uuid = ?", f.deviceID, fanned), "device pair deleted")
	require.Equal(t, 1, f.commandRows(fanned), "command still referenced by the user pair")

	// a second run finds nothing left to do
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
}

func testNanoCleanupDisabledAndBudgets(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)

	// small scans so the budgets below span several of them
	orig := nanoCleanupScanBatchSize
	nanoCleanupScanBatchSize = 2
	t.Cleanup(func() { nanoCleanupScanBatchSize = orig })

	dead := make([]string, 0, 5)
	for range 5 {
		c := f.enqueue("DeviceInformation", f.deviceID)
		f.report(f.deviceID, c, "Acknowledged")
		f.deactivate(f.deviceID, c, 2*nanoCleanupDay, 2*nanoCleanupDay)
		dead = append(dead, c)
	}
	remaining := func() int { return f.count("nano_enrollment_queue", "active = 0") }

	// short retention 0 disables the purge
	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(0, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 5, remaining())

	// a zero row budget deletes nothing
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, 0, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 5, remaining())

	// a row budget of 3 deletes exactly 3 pairs and reports the backlog
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, 3, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 3, CommandsDeleted: 3, RowBudgetExhausted: true}, stats)
	require.Equal(t, 2, remaining())
	require.Equal(t, 2, f.count("nano_commands", "command_uuid IN (?, ?, ?, ?, ?)", dead[0], dead[1], dead[2], dead[3], dead[4]))

	// a command budget of 1 leaves one orphaned command behind
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, 1), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 2, CommandsDeleted: 1, CmdBudgetExhausted: true}, stats)
	require.Zero(t, remaining())
	require.Equal(t, 1, f.count("nano_commands", "command_uuid IN (?, ?, ?, ?, ?)", dead[0], dead[1], dead[2], dead[3], dead[4]))
}

func testNanoCleanupQueueFirstAtomicity(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)

	dead := f.enqueue("DeviceInformation", f.deviceID)
	f.report(f.deviceID, dead, "Acknowledged")
	f.deactivate(f.deviceID, dead, 2*nanoCleanupDay, 2*nanoCleanupDay)

	// failing after the queue delete must roll the queue row back too:
	// a result-less queue row would be re-served to the device
	nanoCleanupAfterQueueDeleteHook = func() error { return errors.New("boom") }
	t.Cleanup(func() { nanoCleanupAfterQueueDeleteHook = nil })
	_, _, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.ErrorContains(t, err, "boom")
	require.Equal(t, 1, f.queueRows(dead))
	require.Equal(t, 1, f.resultRows(dead))
	require.Equal(t, 1, f.commandRows(dead))

	nanoCleanupAfterQueueDeleteHook = nil
	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(dead))
	require.Zero(t, f.resultRows(dead))
	require.Zero(t, f.commandRows(dead))
}

func testNanoCleanupMopKeepsReferencedCommand(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)

	// a certificate renewal points at the command through a real FK, so the
	// reference guard pins its dead pair: nothing is deleted, no error
	renew := f.enqueue("InstallProfile", f.deviceID)
	f.deactivate(f.deviceID, renew, 2*nanoCleanupDay, 2*nanoCleanupDay)
	_, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO nano_cert_auth_associations (id, sha256, renew_command_uuid) VALUES (?, ?, ?)`,
		f.deviceID, "abc123", renew)
	require.NoError(t, err)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 1, f.queueRows(renew))
	require.Equal(t, 1, f.commandRows(renew))

	// the mop itself must also refuse a referenced command when its pair is
	// gone by another path, and still delete an unreferenced sibling
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM nano_enrollment_queue WHERE command_uuid = ?`, renew)
	require.NoError(t, err)
	orphan := f.enqueue("DeviceInformation", f.deviceID)
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM nano_enrollment_queue WHERE command_uuid = ?`, orphan)
	require.NoError(t, err)

	deleted, exhausted, err := ds.mopNanoCommands(ctx, []string{renew, orphan}, nanoCleanupDefaults, nanoCommandMopFilter)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	require.False(t, exhausted)
	require.Equal(t, 1, f.commandRows(renew), "FK-referenced command survives the mop")
	require.Zero(t, f.commandRows(orphan))
}

func testNanoCleanupPinnedWindowDoesNotStarve(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)

	// windows of two, so the three pinned rows fill more than one scan
	orig := nanoCleanupScanBatchSize
	nanoCleanupScanBatchSize = 2
	t.Cleanup(func() { nanoCleanupScanBatchSize = orig })

	// three dead profile commands still current for the host sort first
	// (older created_at) and are pinned by the reference guard
	var pinned []string
	for i := range 3 {
		c := f.enqueue("InstallProfile", f.deviceID)
		f.deactivate(f.deviceID, c, 3*nanoCleanupDay+time.Duration(i)*time.Minute, 2*nanoCleanupDay)
		_, err := ds.writer(ctx).ExecContext(ctx, `
			INSERT INTO host_mdm_apple_profiles (host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
			VALUES (?, ?, ?, ?, UNHEX(MD5(?)), 'install', 'verified')`, f.deviceID, "prof-"+c[:8], "com.example."+c[:8], c, c)
		require.NoError(t, err)
		pinned = append(pinned, c)
	}
	// the eligible row sorts behind them
	eligible := f.enqueue("DeviceInformation", f.deviceID)
	f.deactivate(f.deviceID, eligible, 2*nanoCleanupDay, 2*nanoCleanupDay)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(eligible), "reached past the pinned window in one run")
	for _, p := range pinned {
		require.Equal(t, 1, f.queueRows(p), p)
	}
}

func testNanoCleanupShortRetentionTier(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// every short class, completed two days ago, in each terminal status
	gone := []string{
		f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day),
		f.completed("InstalledApplicationList", fleet.RefetchAppsCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusError, 2*day),
		f.completed("CertificateList", fleet.RefetchCertsCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusCommandFormatError, 2*day),
		f.completed("Settings", fleet.DeviceNameCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day),
		f.completed("DeclarativeManagement", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day),
	}
	kept := []string{
		// too young
		f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, time.Hour),
		// NotNow is not an answer
		f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusNotNow, 2*day),
		// no prefix: a manually run inventory command belongs to the standard window
		f.completed("DeviceInformation", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day),
		// a Settings command that isn't a rename is not in any class
		f.completed("Settings", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day),
	}
	// a VPP verification whose install is still unverified is pinned
	verify := fleet.VerifySoftwareInstallVPPPrefix + uuid.NewString()
	f.completed("InstalledApplicationList", verify, fleet.MDMAppleStatusAcknowledged, 2*day)
	f.exec(`INSERT INTO vpp_apps (adam_id, platform, name, latest_version) VALUES ('adam-1', 'darwin', 'App', '1.0')`)
	f.exec(`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, verification_command_uuid) VALUES (?, 'adam-1', 'darwin', ?, ?)`,
		f.hostID, uuid.NewString(), verify)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{ShortPairsDeleted: 5, CommandsDeleted: 5}, stats)
	for _, c := range gone {
		require.Zero(t, f.queueRows(c), c)
		require.Zero(t, f.resultRows(c), c)
		require.Zero(t, f.commandRows(c), c)
	}
	for _, c := range kept {
		require.Equal(t, 1, f.queueRows(c), c)
		require.Equal(t, 1, f.resultRows(c), c)
	}
	require.Equal(t, 1, f.queueRows(verify), "unverified install pins its verification command")

	// once verified, the same pair goes
	f.exec(`UPDATE host_vpp_software_installs SET verification_at = NOW() WHERE verification_command_uuid = ?`, verify)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{ShortPairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(verify))
}

func testNanoCleanupStandardRetentionTier(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// manually run inventory commands age out at 30 days in every terminal status
	manual := f.completed("DeviceInformation", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day)
	manualErr := f.completed("InstalledApplicationList", uuid.NewString(), fleet.MDMAppleStatusError, 31*day)
	manualFormatErr := f.completed("CertificateList", uuid.NewString(), fleet.MDMAppleStatusCommandFormatError, 31*day)
	manualYoung := f.completed("DeviceInformation", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 29*day)
	// a type on no list is never swept
	never := f.completed("ShutDownDevice", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 400*day)
	// the host's current profile command is pinned until superseded
	profile := f.completed("InstallProfile", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day)
	f.exec(`INSERT INTO host_mdm_apple_profiles (host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
		VALUES (?, 'prof-1', 'com.example.one', ?, UNHEX(MD5('a')), 'install', 'verified')`, f.deviceID, profile)
	// recovery lock and managed account commands are pinned while pending
	recovery := f.completed(fleet.SetRecoveryLockCmdName, uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day)
	f.exec(`INSERT INTO host_recovery_key_passwords (host_uuid, encrypted_password, status, operation_type, pending_set_command_uuid)
		VALUES (?, 'enc', 'pending', 'install', ?)`, f.deviceID, recovery)
	admin := f.completed(fleet.SetAutoAdminPasswordCmdName, uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day)
	f.exec(`INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, command_uuid, status, pending_command_uuid)
		VALUES (?, 'enc', 'other', 'verified', ?)`, f.deviceID, admin)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{StandardPairsDeleted: 3, CommandsDeleted: 3}, stats)
	for _, c := range []string{manual, manualErr, manualFormatErr} {
		require.Zero(t, f.queueRows(c), c)
	}
	for _, c := range []string{manualYoung, never, profile, recovery, admin} {
		require.Equal(t, 1, f.queueRows(c), c)
	}

	// superseded profile and rotated admin password sweep; a promoted recovery
	// lock command is still the host's current one and stays
	f.exec(`UPDATE host_mdm_apple_profiles SET command_uuid = ? WHERE command_uuid = ?`, uuid.NewString(), profile)
	f.exec(`UPDATE host_recovery_key_passwords SET pending_set_command_uuid = NULL, set_command_uuid = ? WHERE pending_set_command_uuid = ?`, recovery, recovery)
	f.exec(`UPDATE host_managed_local_account_passwords SET pending_command_uuid = NULL WHERE pending_command_uuid = ?`, admin)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{StandardPairsDeleted: 2, CommandsDeleted: 2}, stats)
	for _, c := range []string{profile, admin} {
		require.Zero(t, f.queueRows(c), c)
	}
	require.Equal(t, 1, f.queueRows(recovery), "current recovery lock command is pinned")
	require.Equal(t, 1, f.queueRows(never))

	// once a newer command holds the recovery lock, the old one sweeps
	f.exec(`UPDATE host_recovery_key_passwords SET set_command_uuid = ? WHERE set_command_uuid = ?`, uuid.NewString(), recovery)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{StandardPairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(recovery))

	// standard retention 0 disables the sweep entirely
	old := f.completed("DeviceInformation", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 400*day)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupTiers(day, 0, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 1, f.queueRows(old))
}

func testNanoCleanupShortTierOffFallsBackToStandard(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// with the short tier off, its classes age out with the standard window
	// instead of being kept forever, including the types not on the standard list
	old := []string{
		f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day),
		f.completed("Settings", fleet.DeviceNameCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day),
		f.completed("DeclarativeManagement", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 31*day),
	}
	young := f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupTiers(0, 30*day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{StandardPairsDeleted: 3, CommandsDeleted: 3}, stats)
	for _, c := range old {
		require.Zero(t, f.queueRows(c), c)
	}
	require.Equal(t, 1, f.queueRows(young), "still inside the standard window")
}

func testNanoCleanupRetentionCursorResumesAndWraps(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// one scan of two rows per run, so the pinned rows in front take a whole run
	origBatch, origScans := nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun
	nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun = 2, 1
	t.Cleanup(func() { nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun = origBatch, origScans })

	// three pinned current-profile commands sort first (older), one eligible behind
	for i := range 3 {
		c := f.completed("InstallProfile", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 40*day-time.Duration(i)*time.Hour)
		f.exec(`INSERT INTO host_mdm_apple_profiles (host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
			VALUES (?, ?, ?, ?, UNHEX(MD5(?)), 'install', 'verified')`, f.deviceID, "prof-"+c[:8], "com.example."+c[:8], c, c)
	}
	eligible := f.completed("DeviceInformation", uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 35*day)
	opts := nanoCleanupTiers(0, 30*day, nanoCleanupDefaults, nanoCleanupDefaults)
	const key = "standard:" + fleet.MDMAppleStatusAcknowledged

	// run 1 scans the first two pinned rows and stops on the scan cap: nothing
	// deleted, but the backlog is reported so the cron warns
	state, stats, err := ds.CleanupNanoCommands(ctx, opts, nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{RowBudgetExhausted: true}, stats)
	require.Contains(t, state.Retention, key, "a full page leaves a cursor to resume from")
	require.Equal(t, 1, f.queueRows(eligible))

	// run 2 resumes past them and reaches the eligible row; its page was full
	// too, so more may remain
	state, stats, err = ds.CleanupNanoCommands(ctx, opts, state)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{StandardPairsDeleted: 1, CommandsDeleted: 1, RowBudgetExhausted: true}, stats)
	require.Zero(t, f.queueRows(eligible))
	require.Contains(t, state.Retention, key)

	// run 3 finds the end of the range and laps: the cursor is cleared
	state, stats, err = ds.CleanupNanoCommands(ctx, opts, state)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.NotContains(t, state.Retention, key, "reaching the end resets the scan to the oldest rows")

	// a restarted run with no state does not restart from the wrong place either
	state, _, err = ds.CleanupNanoCommands(ctx, opts, &fleet.MDMAppleCommandCleanupState{})
	require.NoError(t, err)
	require.NotNil(t, state.Retention)
}

func testNanoCleanupRowBudgetStopsLaterSweeps(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// two dead pairs for the inactive purge and one eligible completed refetch
	for range 2 {
		c := f.enqueue("DeviceInformation", f.deviceID)
		f.deactivate(f.deviceID, c, 2*day, 2*day)
	}
	refetch := f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day)

	// a budget of 2 is spent by the purge; the retention sweep does not run
	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, 2, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 2, CommandsDeleted: 2, RowBudgetExhausted: true}, stats)
	require.Equal(t, 1, f.queueRows(refetch), "later sweeps wait for the next run")

	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, 2, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{ShortPairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(refetch))
}

func testNanoCleanupRetentionKeepsFannedCommand(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// one command to both channels: the device answered long ago, the user
	// channel never did. Only the answered pair goes and the command stays
	// for the pending one; once that answers and ages, the command follows.
	fanned := f.enqueueUUID("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), f.deviceID, f.userID)
	f.report(f.deviceID, fanned, fleet.MDMAppleStatusAcknowledged)
	f.ageResult(f.deviceID, fanned, 2*day)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{ShortPairsDeleted: 1}, stats)
	require.Zero(t, f.count("nano_enrollment_queue", "id = ? AND command_uuid = ?", f.deviceID, fanned))
	require.Equal(t, 1, f.count("nano_enrollment_queue", "id = ? AND command_uuid = ?", f.userID, fanned))
	require.Equal(t, 1, f.commandRows(fanned), "still referenced by the pending user pair")

	f.report(f.userID, fanned, fleet.MDMAppleStatusAcknowledged)
	f.ageResult(f.userID, fanned, 2*day)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{ShortPairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(fanned))
	require.Zero(t, f.commandRows(fanned))
}

func testNanoCleanupScanCapDoesNotStopLaterSweeps(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// the purge gets one scan of two rows, both pinned: it stops on its scan
	// cap without spending any budget
	origBatch, origScans := nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun
	nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun = 2, 1
	t.Cleanup(func() { nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun = origBatch, origScans })
	for range 2 {
		c := f.enqueue("InstallProfile", f.deviceID)
		f.deactivate(f.deviceID, c, 2*day, 2*day)
		f.exec(`INSERT INTO host_mdm_apple_profiles (host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
			VALUES (?, ?, ?, ?, UNHEX(MD5(?)), 'install', 'verified')`, f.deviceID, "prof-"+c[:8], "com.example."+c[:8], c, c)
	}
	// a completed refetch the short tier should still reach this run
	refetch := f.completed("DeviceInformation", fleet.RefetchDeviceCommandUUIDPrefix+uuid.NewString(), fleet.MDMAppleStatusAcknowledged, 2*day)

	_, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{ShortPairsDeleted: 1, CommandsDeleted: 1, RowBudgetExhausted: true}, stats)
	require.Zero(t, f.queueRows(refetch), "the retention sweep ran despite the purge stopping on its scan cap")
}

func testNanoCleanupOrphanMop(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	// old and unreferenced: goes
	gone := f.orphan("DeviceInformation", 2*day)
	// too young to be sure its enqueue finished
	young := f.orphan("DeviceInformation", time.Hour)
	// still queued for the device
	live := f.enqueue("DeviceInformation", f.deviceID)
	f.exec(`UPDATE nano_commands SET created_at = NOW(6) - INTERVAL ? SECOND WHERE command_uuid = ?`, int((2 * day).Seconds()), live)
	// soft references the pair guard never sees: a device rename and a lock
	rename := f.orphan("Settings", 2*day)
	f.exec(`INSERT INTO host_mdm_apple_device_names (host_uuid, status, command_uuid, expected_device_name) VALUES (?, 'pending', ?, 'mac')`, f.deviceID, rename)
	lock := f.orphan("DeviceLock", 2*day)
	f.exec(`INSERT INTO host_mdm_actions (host_id, lock_ref) VALUES (?, ?)`, f.hostID, lock)
	// the soft references the pair guard knows about pin the walk too
	profile := f.orphan("InstallProfile", 2*day)
	f.exec(`INSERT INTO host_mdm_apple_profiles (host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
		VALUES (?, 'prof-1', 'com.example.one', ?, UNHEX(MD5('a')), 'install', 'verified')`, f.deviceID, profile)
	install := f.orphan("InstallApplication", 2*day)
	f.exec(`INSERT INTO vpp_apps (adam_id, platform, name, latest_version) VALUES ('adam-1', 'darwin', 'App', '1.0')`)
	f.exec(`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'adam-1', 'darwin', ?)`, f.hostID, install)
	// real FKs: pinned, and no FK error either
	renew := f.orphan("InstallProfile", 2*day)
	f.exec(`INSERT INTO nano_cert_auth_associations (id, sha256, renew_command_uuid) VALUES (?, 'abc123', ?)`, f.deviceID, renew)
	bootstrap := f.orphan("InstallEnterpriseApplication", 2*day)
	f.exec(`INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid, command_uuid) VALUES (?, ?)`, f.deviceID, bootstrap)

	state, stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{OrphanCommandsDeleted: 1}, stats)
	require.Zero(t, f.commandRows(gone))
	for _, c := range []string{young, live, rename, lock, profile, install, renew, bootstrap} {
		require.Equal(t, 1, f.commandRows(c), c)
	}
	require.Equal(t, fleet.MDMAppleCommandOrphanCursor{}, state.Orphan, "a short page laps the walk")

	// a zero command budget skips the walk entirely
	stale := f.orphan("DeviceInformation", 2*day)
	_, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, 0), nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 1, f.commandRows(stale))
}

func testNanoCleanupOrphanMopCursorAndBudget(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	f := newNanoCleanupFixture(t, ds)
	day := nanoCleanupDay

	origBatch, origScans := nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun
	nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun = 2, 1
	t.Cleanup(func() { nanoCleanupScanBatchSize, nanoCleanupMaxScansPerRun = origBatch, origScans })

	// three referenced commands sort first (oldest), three orphans behind them
	for i := range 3 {
		c := f.orphan("DeviceLock", 40*day-time.Duration(i)*time.Hour)
		f.exec(`INSERT INTO host_mdm_actions (host_id, lock_ref) VALUES (?, ?)`, uint(1000+i), c)
	}
	var orphans []string
	for i := range 3 {
		orphans = append(orphans, f.orphan("DeviceInformation", 30*day-time.Duration(i)*time.Hour))
	}
	countOrphans := func() int {
		return f.count("nano_commands", "command_uuid IN (?, ?, ?)", orphans[0], orphans[1], orphans[2])
	}
	opts := nanoCleanupOpts(day, nanoCleanupDefaults, nanoCleanupDefaults)

	// run 1: one page of two referenced rows, nothing deleted, cursor stored
	state, stats, err := ds.CleanupNanoCommands(ctx, opts, nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{CmdBudgetExhausted: true}, stats)
	require.NotEqual(t, fleet.MDMAppleCommandOrphanCursor{}, state.Orphan)
	require.Equal(t, 3, countOrphans())

	// run 2 resumes: third referenced row plus the first orphan
	state, stats, err = ds.CleanupNanoCommands(ctx, opts, state)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{OrphanCommandsDeleted: 1, CmdBudgetExhausted: true}, stats)
	require.Equal(t, 2, countOrphans())

	// run 3 with a budget of 1: the page holds two orphans, only one may go,
	// and the cursor stays before the page so the other is retried
	before := state.Orphan
	state, stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(day, nanoCleanupDefaults, 1), state)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{OrphanCommandsDeleted: 1, CmdBudgetExhausted: true}, stats)
	require.Equal(t, before, state.Orphan, "budget hit keeps the cursor before the page")
	require.Equal(t, 1, countOrphans())

	// run 4 finishes the range and laps
	state, stats, err = ds.CleanupNanoCommands(ctx, opts, state)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{OrphanCommandsDeleted: 1}, stats)
	require.Zero(t, countOrphans())
	require.Equal(t, fleet.MDMAppleCommandOrphanCursor{}, state.Orphan)
}
