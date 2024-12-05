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
		id, name, instance, stats_type, status, coalesce(errors, '') as errors, created_at, updated_at
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
		id, name, instance, stats_type, status, coalesce(errors, '') as errors, created_at, updated_at
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
		// Delete cron_stats entries that have been in pending state for more than two hours.
		//
		// NOTE(lucas): We don't know of any job that is taking longer than two hours. This value might need changing
		// if that is not true anymore in the future.
		updateStmt := `UPDATE cron_stats SET status = ? WHERE created_at < DATE_SUB(NOW(), INTERVAL 2 HOUR) AND status = ?`
		if _, err := tx.ExecContext(ctx, updateStmt, fleet.CronStatsStatusExpired, fleet.CronStatsStatusPending); err != nil {
			return ctxerr.Wrap(ctx, err, "updating expired cron stats")
		}

		return nil
	})
}
