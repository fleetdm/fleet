package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// GetLatestCronStats returns a slice of no more than two cron stats records, where index 0 (if
// present) is the most recently created scheduled run, and index 1 (if present) represents a
// triggered run that is currently pending.
func (ds *Datastore) GetLatestCronStats(ctx context.Context, name string) ([]fleet.CronStats, error) {
	stmt := `
SELECT
	id, name, instance, stats_type, status, created_at, updated_at
FROM (
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
	LIMIT 1) cs1
UNION
SELECT
	id, name, instance, stats_type, status, created_at, updated_at
FROM (
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
	LIMIT 1) cs2
ORDER BY
	stats_type ASC`

	var res []fleet.CronStats
	err := sqlx.SelectContext(ctx, ds.reader, &res, stmt, name, name)
	if err != nil {
		return []fleet.CronStats{}, ctxerr.Wrap(ctx, err, "select cron stats")
	}

	return res, nil
}

func (ds *Datastore) InsertCronStats(ctx context.Context, statsType fleet.CronStatsType, name string, instance string, status fleet.CronStatsStatus) (int, error) {
	stmt := `INSERT INTO cron_stats (stats_type, name, instance, status) VALUES (?, ?, ?, ?)`

	res, err := ds.writer.ExecContext(ctx, stmt, statsType, name, instance, status)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert cron stats")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert cron stats last insert id")
	}

	return int(id), nil
}

func (ds *Datastore) UpdateCronStats(ctx context.Context, id int, status fleet.CronStatsStatus) error {
	stmt := `UPDATE cron_stats SET status = ? WHERE id = ?`

	if _, err := ds.writer.ExecContext(ctx, stmt, status, id); err != nil {
		return ctxerr.Wrap(ctx, err, "update cron stats")
	}

	return nil
}

func (ds *Datastore) UpdateAllCronStatsForInstance(ctx context.Context, instance string, fromStatus fleet.CronStatsStatus, toStatus fleet.CronStatsStatus) error {
	stmt := `UPDATE cron_stats SET status = ? WHERE instance = ? AND status = ?`

	if _, err := ds.writer.ExecContext(ctx, stmt, toStatus, instance, fromStatus); err != nil {
		return ctxerr.Wrap(ctx, err, "update all cron stats for instance")
	}

	return nil
}

func (ds *Datastore) CleanupCronStats(ctx context.Context) error {
	deleteStmt := `DELETE FROM cron_stats WHERE created_at < DATE_SUB(NOW(), INTERVAL ? DAY)`
	const MAX_DAYS_RETAINED = 14
	if _, err := ds.writer.ExecContext(ctx, deleteStmt, MAX_DAYS_RETAINED); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting old cron stats")
	}
	const MAX_HOURS_PENDING = 2
	updateStmt := `UPDATE cron_stats SET status = ? WHERE created_at < DATE_SUB(NOW(), INTERVAL ? HOUR) AND status = ?`
	if _, err := ds.writer.ExecContext(ctx, updateStmt, fleet.CronStatsStatusExpired, MAX_HOURS_PENDING, fleet.CronStatsStatusPending); err != nil {
		return ctxerr.Wrap(ctx, err, "updating expired cron stats")
	}

	return nil
}
