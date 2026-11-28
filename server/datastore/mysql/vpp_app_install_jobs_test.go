package mysql

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestVPPAppInstallJobs(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"CleanupAttempts", testCleanupVPPAppInstallAttempts},
		{"CleanupAttemptsBatchCap", testCleanupVPPAppInstallAttemptsBatchCap},
		{"RefreshBatchResumes", testRefreshVPPAppInstallStateBatchResumes},
		{"DeleteOrphans", testDeleteOrphanVPPAppInstallState},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func insertVPPAppInstallAttempt(t *testing.T, ds *Datastore, n int, finishedAt time.Time) {
	t.Helper()
	ctx := t.Context()
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO log_vpp_app_install_attempts
		(install_id, host_id, adam_id, platform, command_uuid, attempt, outcome, finished_at)
		VALUES (?, 1, 'app-a', 'ios', ?, 0, 'installed', ?)`, n, fmt.Sprint("cmd-", n), finishedAt)
	require.NoError(t, err)
}

func countVPPAppInstallAttempts(t *testing.T, ds *Datastore) int {
	t.Helper()
	ctx := t.Context()
	var n int
	require.NoError(t, sqlx.GetContext(ctx, ds.writer(ctx), &n, `SELECT COUNT(*) FROM log_vpp_app_install_attempts`))
	return n
}

func testCleanupVPPAppInstallAttempts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC()
	cutoff := now.Add(-30 * 24 * time.Hour)

	insertVPPAppInstallAttempt(t, ds, 1, cutoff.Add(-time.Hour))
	insertVPPAppInstallAttempt(t, ds, 2, cutoff.Add(-24*time.Hour))
	insertVPPAppInstallAttempt(t, ds, 3, cutoff.Add(time.Hour))
	insertVPPAppInstallAttempt(t, ds, 4, now)

	deleted, err := ds.CleanupVPPAppInstallAttempts(ctx, cutoff)
	require.NoError(t, err)
	require.EqualValues(t, 2, deleted)

	var uuids []string
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &uuids, `SELECT command_uuid FROM log_vpp_app_install_attempts ORDER BY id`))
	require.Equal(t, []string{"cmd-3", "cmd-4"}, uuids)

	deleted, err = ds.CleanupVPPAppInstallAttempts(ctx, cutoff)
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func testCleanupVPPAppInstallAttemptsBatchCap(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC()
	old := now.Add(-time.Hour)
	for i := range 7 {
		insertVPPAppInstallAttempt(t, ds, i, old.Add(time.Duration(i)*time.Second))
	}
	insertVPPAppInstallAttempt(t, ds, 100, now.Add(time.Hour))

	// 3 batches of 2 leave 1 expired row for the next run.
	deleted, err := cleanupVPPAppInstallAttemptsDB(ctx, ds, now, 2, 3)
	require.NoError(t, err)
	require.EqualValues(t, 6, deleted)
	require.Equal(t, 2, countVPPAppInstallAttempts(t, ds))

	deleted, err = cleanupVPPAppInstallAttemptsDB(ctx, ds, now, 2, 3)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	require.Equal(t, 1, countVPPAppInstallAttempts(t, ds))
}

func testRefreshVPPAppInstallStateBatchResumes(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('app-a', 'ios'), ('app-b', 'android')`)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES
		(1, 'app-a', 'ios', 'cmd-1'), (1, 'app-a', 'ios', 'cmd-2'), (1, 'app-b', 'android', 'cmd-3'),
		(2, 'app-a', 'ios', 'cmd-4'), (3, 'app-a', 'ios', 'cmd-5')`)
	require.NoError(t, err)

	stateHosts := func() []uint {
		var ids []uint
		require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &ids, `SELECT DISTINCT host_id FROM host_vpp_app_installs ORDER BY host_id`))
		return ids
	}

	// Four distinct keys, so two batches of two and a final empty one.
	cursor, n, err := ds.RefreshVPPAppInstallStateBatch(ctx, "", 2)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "1:android:app-b", cursor)
	require.Equal(t, []uint{1}, stateHosts())

	// A crash here leaves only the saved cursor, which is all the next run needs.
	cursor, n, err = ds.RefreshVPPAppInstallStateBatch(ctx, cursor, 2)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "3:ios:app-a", cursor)
	require.Equal(t, []uint{1, 2, 3}, stateHosts())

	next, n, err := ds.RefreshVPPAppInstallStateBatch(ctx, cursor, 2)
	require.NoError(t, err)
	require.Zero(t, n)
	require.Equal(t, cursor, next)

	assertVPPAppInstallStateMatchesRequests(t, ds)

	// A malformed cursor restarts the walk.
	_, n, err = ds.RefreshVPPAppInstallStateBatch(ctx, "garbage", 10)
	require.NoError(t, err)
	require.Equal(t, 4, n)
}

func testDeleteOrphanVPPAppInstallState(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	keep := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	gone := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('app-a', 'ios'), ('app-b', 'ios')`)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES
		(?, 'app-a', 'ios', 'cmd-1'), (?, 'app-a', 'ios', 'cmd-2'), (?, 'app-b', 'ios', 'cmd-3')`, keep.ID, gone.ID, gone.ID)
	require.NoError(t, err)
	lockAndRefreshVPPAppInstallState(t, ds,
		hostVPPAppKey{keep.ID, "app-a", "ios"}, hostVPPAppKey{gone.ID, "app-a", "ios"}, hostVPPAppKey{gone.ID, "app-b", "ios"})

	// A direct delete skips hostRefs, leaving the state behind as a missed cleanup would.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM hosts WHERE id = ?`, gone.ID)
	require.NoError(t, err)

	deleted, err := ds.DeleteOrphanVPPAppInstallState(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, deleted)

	var hostIDs []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.writer(ctx), &hostIDs, `SELECT host_id FROM host_vpp_app_installs`))
	require.Equal(t, []uint{keep.ID}, hostIDs)

	deleted, err = ds.DeleteOrphanVPPAppInstallState(ctx)
	require.NoError(t, err)
	require.Zero(t, deleted)
}
