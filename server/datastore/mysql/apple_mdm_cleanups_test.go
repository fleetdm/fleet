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
	return &nanoCleanupFixture{t: t, ds: ds, deviceID: host.UUID, userID: userEnrollment.ID, commander: commander}
}

func (f *nanoCleanupFixture) enqueue(reqType string, enrollmentIDs ...string) string {
	cmdUUID := uuid.NewString()
	require.NoError(f.t, f.commander.EnqueueCommand(f.t.Context(), enrollmentIDs, createRawAppleCmd(reqType, cmdUUID)))
	return cmdUUID
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
	return fleet.MDMAppleCommandCleanupOptions{ShortRetention: short, MaxRowDeletions: rows, MaxCmdDeletions: cmds}
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

	stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults))
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 3, CommandsDeleted: 2}, stats)

	for _, gone := range []string{oldDead, oldDeadNoResult} {
		require.Zero(t, f.queueRows(gone), gone)
		require.Zero(t, f.resultRows(gone), gone)
		require.Zero(t, f.commandRows(gone), gone)
	}
	for _, kept := range []string{freshDead, lockDead, live, guarded} {
		require.Equal(t, 1, f.queueRows(kept), kept)
		require.Equal(t, 1, f.commandRows(kept), kept)
	}
	require.Equal(t, 1, f.queueRows(fanned), "user-channel pair survives")
	require.Zero(t, f.count("nano_enrollment_queue", "id = ? AND command_uuid = ?", f.deviceID, fanned), "device pair deleted")
	require.Equal(t, 1, f.commandRows(fanned), "command still referenced by the user pair")

	// a second run finds nothing left to do
	stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults))
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
	stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(0, nanoCleanupDefaults, nanoCleanupDefaults))
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 5, remaining())

	// a zero row budget deletes nothing
	stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, 0, nanoCleanupDefaults))
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{}, stats)
	require.Equal(t, 5, remaining())

	// a row budget of 3 deletes exactly 3 pairs and reports the backlog
	stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, 3, nanoCleanupDefaults))
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 3, CommandsDeleted: 3, RowBudgetExhausted: true}, stats)
	require.Equal(t, 2, remaining())
	require.Equal(t, 2, f.count("nano_commands", "command_uuid IN (?, ?, ?, ?, ?)", dead[0], dead[1], dead[2], dead[3], dead[4]))

	// a command budget of 1 leaves one orphaned command behind
	stats, err = ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, 1))
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
	_, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults))
	require.ErrorContains(t, err, "boom")
	require.Equal(t, 1, f.queueRows(dead))
	require.Equal(t, 1, f.resultRows(dead))
	require.Equal(t, 1, f.commandRows(dead))

	nanoCleanupAfterQueueDeleteHook = nil
	stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults))
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

	stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults))
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

	deleted, exhausted, err := ds.mopNanoCommands(ctx, []string{renew, orphan}, nanoCleanupDefaults)
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

	stats, err := ds.CleanupNanoCommands(ctx, nanoCleanupOpts(nanoCleanupDay, nanoCleanupDefaults, nanoCleanupDefaults))
	require.NoError(t, err)
	require.Equal(t, fleet.MDMAppleCommandCleanupStats{InactivePairsDeleted: 1, CommandsDeleted: 1}, stats)
	require.Zero(t, f.queueRows(eligible), "reached past the pinned window in one run")
	for _, p := range pinned {
		require.Equal(t, 1, f.queueRows(p), p)
	}
}
