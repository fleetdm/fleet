package mysql

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CleanupVPPAppInstallAttempts(ctx context.Context, olderThan time.Time) (int64, error) {
	const (
		batchSize  = 500
		maxBatches = 200 // 100k rows per tick
	)
	return cleanupVPPAppInstallAttemptsDB(ctx, ds, olderThan, batchSize, maxBatches)
}

func cleanupVPPAppInstallAttemptsDB(ctx context.Context, ds *Datastore, olderThan time.Time, batchSize, maxBatches int) (int64, error) {
	// No cursor: every row older than the cutoff is deletable, so each batch starts at the oldest survivor on
	// idx_lvaia_finished. The id tie-break keeps the order deterministic for replication.
	const deleteStmt = `DELETE FROM log_vpp_app_install_attempts WHERE finished_at < ? ORDER BY finished_at, id LIMIT ?`

	var deleted int64
	hitCap := true
	for range maxBatches {
		res, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, olderThan, batchSize)
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "delete expired vpp app install attempts")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "count deleted vpp app install rows")
		}
		deleted += n
		if n < int64(batchSize) {
			hitCap = false
			break
		}
	}
	if hitCap {
		ds.logger.WarnContext(ctx, "cleanup vpp app install attempts hit its batch cap, any remaining rows are cleaned on the next run",
			"deleted", deleted, "max_batches", maxBatches)
	}
	return deleted, nil
}

// RefreshVPPAppInstallStateBatch refreshes the state of up to limit (host_id, adam_id, platform) keys of
// host_vpp_software_installs that sort after afterKey, an empty string for the first batch. It returns the last key
// it refreshed, which is the next call's afterKey, and how many keys it refreshed. Fewer than limit means the walk
// is complete.
func (ds *Datastore) RefreshVPPAppInstallStateBatch(ctx context.Context, afterKey string, limit int) (string, int, error) {
	after := parseVPPAppKeyCursor(afterKey)

	var keys []hostVPPAppKey
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &keys, `
SELECT DISTINCT host_id, adam_id, platform
FROM host_vpp_software_installs
WHERE (host_id, adam_id, platform) > (?, ?, ?)
ORDER BY host_id, adam_id, platform
LIMIT ?`, after.HostID, after.AdamID, after.Platform, limit); err != nil {
		return "", 0, ctxerr.Wrap(ctx, err, "select vpp app install keys to refresh")
	}

	// A key is committed on its own so the lock on its requests is held for one refresh, not the whole batch.
	for i, k := range keys {
		if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			if err := lockVPPAppInstallKeys(ctx, tx, []hostVPPAppKey{k}); err != nil {
				return err
			}
			return refreshVPPAppInstallState(ctx, tx, []hostVPPAppKey{k})
		}); err != nil {
			if i == 0 {
				return afterKey, 0, err
			}
			return formatVPPAppKeyCursor(keys[i-1]), i, err
		}
	}
	if len(keys) == 0 {
		return afterKey, 0, nil
	}
	return formatVPPAppKeyCursor(keys[len(keys)-1]), len(keys), nil
}

// DeleteOrphanVPPAppInstallState deletes state rows whose host no longer exists. The walk in
// RefreshVPPAppInstallStateBatch never reaches them, since their requests are deleted with the host.
func (ds *Datastore) DeleteOrphanVPPAppInstallState(ctx context.Context) (int64, error) {
	const (
		batchSize  = 500
		maxBatches = 200
	)
	var deleted int64
	for range maxBatches {
		var hostIDs []uint
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &hostIDs, `
SELECT DISTINCT s.host_id
FROM host_vpp_app_installs s
LEFT JOIN hosts h ON h.id = s.host_id
WHERE h.id IS NULL
LIMIT ?`, batchSize); err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "select orphan vpp app install state hosts")
		}
		if len(hostIDs) == 0 {
			return deleted, nil
		}

		stmt, args, err := sqlx.In(`
DELETE s FROM host_vpp_app_installs s
LEFT JOIN hosts h ON h.id = s.host_id
WHERE s.host_id IN (?) AND h.id IS NULL`, hostIDs)
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "build delete orphan vpp app install state")
		}
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "delete orphan vpp app install state")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return deleted, ctxerr.Wrap(ctx, err, "count deleted vpp app install rows")
		}
		deleted += n
	}
	return deleted, nil
}

// GetVPPAppInstallStateBackfillCursor and SetVPPAppInstallStateBackfillCursor persist the backfill cron's cursor.
// The bare datastore returns "" and drops writes; the mysqlredis wrapper backs them with Redis.
func (ds *Datastore) GetVPPAppInstallStateBackfillCursor(_ context.Context) (string, error) {
	return "", nil
}

func (ds *Datastore) SetVPPAppInstallStateBackfillCursor(_ context.Context, _ string) error {
	return nil
}

// formatVPPAppKeyCursor puts adam_id last, so SplitN keeps any colon it contains.
func formatVPPAppKeyCursor(k hostVPPAppKey) string {
	return strconv.FormatUint(uint64(k.HostID), 10) + ":" + k.Platform + ":" + k.AdamID
}

// parseVPPAppKeyCursor returns the zero key, which sorts before every real key, for "" and for a malformed cursor.
// Restarting the walk is safe, so a poisoned stored cursor can't wedge the job.
func parseVPPAppKeyCursor(cursor string) hostVPPAppKey {
	parts := strings.SplitN(cursor, ":", 3)
	if len(parts) != 3 {
		return hostVPPAppKey{}
	}
	hostID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return hostVPPAppKey{}
	}
	return hostVPPAppKey{HostID: uint(hostID), Platform: parts[1], AdamID: parts[2]}
}
