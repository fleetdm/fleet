package mysql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (d *Datastore) UpdateAggregatedStats(ctx context.Context) error {
	statsTypeScheduledQuery := "scheduled_query"

	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// we are going to get stats by scheduled query id, and then we'll use that to get the pack and query stats
		// we'll store:
		// id -> scheduled_query_id.id
		// type -> "scheduled_query_id"
		// json_value -> {p50: <calculated median value>, p95: <calculated 95th percentile>, executions: <total amount of executions>}
		// to get each pack stats, we will get the stats for each scheduled query id for the pack and then add all the p50s??
		// to get each query stats, we get the p50 across all the scheduled query p50s for that query

		rows, err := tx.QueryxContext(ctx, `SELECT id FROM scheduled_queries`)
		if err != nil {
			return errors.Wrap(err, "querying pack ids")
		}
		defer rows.Close()
		psql := `
SELECT
	(t1.user_time / t1.executions)
FROM (
	SELECT
		@rownum := @rownum + 1 AS ` + "`" + `row_number` + "`" + `,
		d.scheduled_query_id,
		d.user_time,
		d.executions
	FROM
		scheduled_query_stats d,
		(SELECT @rownum := 0) AS r
	WHERE d.scheduled_query_id=?
	ORDER BY (d.user_time / d.executions) ASC
) AS t1,
(
	SELECT
		count(*) AS total_rows
	FROM
		scheduled_query_stats d
	WHERE d.scheduled_query_id=?
) AS t2
WHERE t1.row_number = floor(total_rows * %s) + 1;`

		var scheduledQueryIDs []uint
		for rows.Next() {
			var id uint
			err := rows.Scan(&id)
			if err != nil {
				return errors.Wrap(err, "scanning id for scheduled_query")
			}
			scheduledQueryIDs = append(scheduledQueryIDs, id)
		}
		for _, id := range scheduledQueryIDs {
			var p50, p95 float64
			var totalExecutions int

			// 3 queries is not ideal, but getting both values and totals in the same query was a bit more complicated
			// so I went for the simpler approach first, we can optimize later
			err = sqlx.GetContext(ctx, tx, &p50, fmt.Sprintf(psql, "0.5"), id, id)
			if err != nil {
				return errors.Wrapf(err, "getting median for scheduled query %d", id)
			}
			err = sqlx.GetContext(ctx, tx, &p95, fmt.Sprintf(psql, "0.95"), id, id)
			if err != nil {
				return errors.Wrapf(err, "getting median for scheduled query %d", id)
			}
			err = sqlx.GetContext(ctx, tx, &totalExecutions, `SELECT sum(executions) FROM scheduled_query_stats WHERE scheduled_query_id=?`, id)
			if err != nil {
				return errors.Wrapf(err, "getting median for scheduled query %d", id)
			}

			statsMap := make(map[string]interface{})
			statsMap["p50"] = p50
			statsMap["p95"] = p95
			statsMap["total_executions"] = totalExecutions

			statsJson, err := json.Marshal(statsMap)
			if err != nil {
				return errors.Wrap(err, "marshaling stats")
			}

			_, err = tx.ExecContext(ctx,
				`INSERT INTO aggregated_stats(id, type, json_value) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE json_value=VALUES(json_value)`,
				id, statsTypeScheduledQuery, statsJson,
			)
			if err != nil {
				return errors.Wrapf(err, "inserting stats for scheduled query id %d", id)
			}
		}

		// TODO: update last_run

		return nil
	})
}
