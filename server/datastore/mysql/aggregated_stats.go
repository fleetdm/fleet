package mysql

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (d *Datastore) UpdateAggregatedStats(ctx context.Context) error {
	return d.withTx(ctx, func(tx sqlx.ExtContext) error {
		var lastUpdated time.Time
		err := sqlx.SelectContext(ctx, tx, &lastUpdated,
			`select updated_at from last_run where type='pack_stats'`)
		if err != nil {
			return errors.Wrap(err, "getting the latest updated at time")
		}

		var maxLastExecuted time.Time
		err = sqlx.SelectContext(ctx, tx, &maxLastExecuted, `select max(last_executed) from scheduled_query_stats`)
		if err != nil {
			return errors.Wrap(err, "getting the latest updated at time")
		}

		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO last_run(updated_at) VALUES(?) ON DUPLICATE KEY UPDATE updated_at=VALUES(updated_at)`,
			maxLastExecuted,
		)
		if err != nil {
			return errors.Wrap(err, "updating the last run time")
		}

		rows, err := tx.QueryxContext(ctx, `SELECT id FROM packs WHERE !disabled`)
		if err != nil {
			return errors.Wrap(err, "querying pack ids")
		}
		defer rows.Close()
		medialSQL := `
SELECT
	t1.user_time AS median_val
FROM (
	SELECT
		@rownum: = @rownum + 1 AS ` + "`row_number`" + `,
		d.user_time,
		sq.pack_id
	FROM
		scheduled_query_stats d
	JOIN scheduled_queries sq ON (d.scheduled_query_id = sq.id),
		(SELECT @rownum: = 0) r
	WHERE pack_id = ?
	ORDER BY d.user_time
) AS t1,
(
	SELECT
		count(*) AS total_rows
	FROM
		scheduled_query_stats d
	JOIN scheduled_queries sq ON (d.scheduled_query_id = sq.id)
	WHERE pack_id = ?
) AS t2
WHERE t1.row_number = floor(total_rows / 2) + 1;`
		for rows.Next() {
			var id int
			err := rows.Scan(&id)
			if err != nil {
				return errors.Wrap(err, "scanning id for pack")
			}
			medianVal
			sqlx.SelectContext(ctx, tx, nil, medialSQL)
		}

		return nil
	})
}
