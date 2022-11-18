package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetLatestCronStats(ctx context.Context, name string) (fleet.CronStats, error) {
	stmt := `SELECT id, name, instance, created_at, updated_at, stats_type, status FROM cron_stats WHERE name = ? ORDER BY created_at DESC LIMIT 1`

	var res fleet.CronStats
	err := sqlx.GetContext(ctx, ds.reader, &res, stmt, name)
	switch {
	case err == sql.ErrNoRows:
		return fleet.CronStats{}, nil
	case err != nil:
		return fleet.CronStats{}, ctxerr.Wrap(ctx, err, "select cron stats")
	default:
		return res, nil
	}
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
