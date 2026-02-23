package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// GetLatestCronStats returns a slice of no more than two cron stats records, where index 0 (if
// present) is the most recently created scheduled run, and index 1 (if present) represents a
// triggered run that is currently pending.
func (ds *Datastore) GetLatestCronStats(ctx context.Context, name string) ([]fleet.CronStats, error) {
	stmt := `
(
	SELECT
		id, name, instance, stats_type, status, created_at, updated_at
	FROM
		cron_stats
	WHERE
		name = ?
		AND stats_type = 'scheduled'
		AND (status = 'pending' OR status = 'completed')
	ORDER BY
		created_at DESC
	LIMIT 1)
UNION
(
	SELECT
		id, name, instance, stats_type, status, created_at, updated_at
	FROM
		cron_stats
	WHERE
		name = ?
		AND stats_type = 'triggered'
		AND (status = 'pending' OR status = 'completed')
	ORDER BY
		created_at DESC
	LIMIT 1)`

	var res []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, name, name)
	if err != nil {
		return []fleet.CronStats{}, ctxerr.Wrap(ctx, err, "select cron stats")
	}

	return res, nil
}

func (ds *Datastore) InsertCronStats(ctx context.Context, statsType fleet.CronStatsType, name string, instance string, status fleet.CronStatsStatus) (int, error) {
	stmt := `INSERT INTO cron_stats (stats_type, name, instance, status) VALUES (?, ?, ?, ?)`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, statsType, name, instance, status)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert cron stats")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert cron stats last insert id")
	}

	return int(id), nil
}

func (ds *Datastore) UpdateCronStats(ctx context.Context, id int, status fleet.CronStatsStatus, cronErrors *fleet.CronScheduleErrors) error {
	stmt := `UPDATE cron_stats SET status = ?, errors = ? WHERE id = ?`

	errorsJSON := sql.NullString{}
	if len(*cronErrors) > 0 {
		b, err := json.Marshal(cronErrors)
		if err == nil {
			errorsJSON.String = string(b)
			errorsJSON.Valid = true
		}
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, status, errorsJSON, id); err != nil {
		return ctxerr.Wrap(ctx, err, "update cron stats")
	}

	return nil
}

func (ds *Datastore) UpdateAllCronStatsForInstance(ctx context.Context, instance string, fromStatus fleet.CronStatsStatus, toStatus fleet.CronStatsStatus) error {
	stmt := `UPDATE cron_stats SET status = ? WHERE instance = ? AND status = ?`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, toStatus, instance, fromStatus); err != nil {
		return ctxerr.Wrap(ctx, err, "update all cron stats for instance")
	}

	return nil
}

func (ds *Datastore) CleanupCronStats(ctx context.Context) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Delete cron_stats entries that are older than two days.
		deleteStmt := `DELETE FROM cron_stats WHERE created_at < DATE_SUB(NOW(), INTERVAL 2 DAY)`
		if _, err := tx.ExecContext(ctx, deleteStmt); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting old cron stats")
		}
		// Mark cron_stats entries as expired if:
		// 1. Pending for >2 hours and no active lock (instance likely crashed), OR
		// 2. Pending for >12 hours regardless of lock state (hard cap for hung jobs).
		//
		// NOTE: The lock check assumes locks.name matches cron_stats.name. Schedules using
		// WithAltLockID (e.g., "leader", "worker") store locks under a different name, so
		// the NOT EXISTS check won't find their lock and they fall back to the 2-hour timeout.
		updateStmt := `
			UPDATE cron_stats cs
			SET cs.status = ?
			WHERE cs.status = ?
			AND (
				(cs.created_at < DATE_SUB(NOW(), INTERVAL 2 HOUR)
				AND NOT EXISTS (
					SELECT 1 FROM locks l
					WHERE l.name = cs.name
					AND l.expires_at >= CURRENT_TIMESTAMP
				))
				OR cs.created_at < DATE_SUB(NOW(), INTERVAL 12 HOUR)
			)`
		if _, err := tx.ExecContext(ctx, updateStmt, fleet.CronStatsStatusExpired, fleet.CronStatsStatusPending); err != nil {
			return ctxerr.Wrap(ctx, err, "updating expired cron stats")
		}

		return nil
	})
}
