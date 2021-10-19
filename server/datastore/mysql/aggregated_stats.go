package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// These queries are a bit annoyingly written. The main reason they are this way is that we want rownum sorted. There's
// a slightly simpler version but that adds the rownum before sorting.

const scheduledQueryPercentileQuery = `
SELECT
	(t1.%s / t1.executions)
FROM (
	SELECT @rownum := @rownum + 1 AS row_number, mm.* FROM (
		SELECT d.scheduled_query_id, d.%s, d.executions
		FROM scheduled_query_stats d 
		WHERE d.scheduled_query_id=?
		ORDER BY (d.%s / d.executions) ASC
	) AS mm
) AS t1,
(SELECT @rownum := 0) AS r,
(
	SELECT count(*) AS total_rows
	FROM scheduled_query_stats d
	WHERE d.scheduled_query_id=?
) AS t2
WHERE t1.row_number = floor(total_rows * %s) + 1;`

const queryPercentileQuery = `
SELECT
	(t1.%s / t1.executions)
FROM (
	SELECT @rownum := @rownum + 1 AS row_number, mm.* FROM (
		SELECT d.scheduled_query_id, d.%s, d.executions
		FROM scheduled_query_stats d 
		JOIN scheduled_queries sq ON (sq.id=d.scheduled_query_id)
		WHERE sq.query_id=?
		ORDER BY (d.%s / d.executions) ASC
	) AS mm
) AS t1,
(SELECT @rownum := 0) AS r,
(
	SELECT count(*) AS total_rows
	FROM scheduled_query_stats d
	JOIN scheduled_queries sq ON (sq.id=d.scheduled_query_id)
	WHERE sq.query_id=?
) AS t2
WHERE t1.row_number = floor(total_rows * %s) + 1;`

const scheduledQueryTotalExecutions = `SELECT coalesce(sum(executions), 0) FROM scheduled_query_stats WHERE scheduled_query_id=?`
const queryTotalExecutions = `SELECT coalesce(sum(executions), 0) FROM scheduled_query_stats sqs JOIN scheduled_queries sq ON (sqs.scheduled_query_id=sq.id) JOIN queries q ON (q.id=sq.query_id) WHERE sq.query_id=?`

func getPercentileQuery(aggregate string, time string, percentile string) string {
	switch aggregate {
	case "scheduled_query":
		return fmt.Sprintf(scheduledQueryPercentileQuery, time, time, time, percentile)
	case "query":
		return fmt.Sprintf(queryPercentileQuery, time, time, time, percentile)
	}
	return ""
}

func setP50AndP95Map(ctx context.Context, tx sqlx.QueryerContext, aggregate string, time string, id uint, statsMap map[string]interface{}) error {
	var p50, p95 float64

	err := sqlx.GetContext(ctx, tx, &p50, getPercentileQuery(aggregate, time, "0.5"), id, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.Wrapf(err, "getting %s p50 for %s %d", time, aggregate, id)
	}
	statsMap[time+"_p50"] = p50
	err = sqlx.GetContext(ctx, tx, &p95, getPercentileQuery(aggregate, time, "0.95"), id, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return errors.Wrapf(err, "getting %s p95 for %s %d", time, aggregate, id)
	}
	statsMap[time+"_p95"] = p95

	return nil
}

func (d *Datastore) UpdateScheduledQueryAggregatedStats(ctx context.Context) error {
	statsTypeScheduledQuery := "scheduled_query"

	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		ids, err := getIdsForTable(ctx, tx, "scheduled_queries")
		if err != nil {
			return errors.Wrap(err, "getting ids")
		}
		if err := calculatePercentiles(ctx, tx, statsTypeScheduledQuery, ids); err != nil {
			return err
		}

		return nil
	})
}

func (d *Datastore) UpdateQueryAggregatedStats(ctx context.Context) error {
	statsTypeQuery := "query"

	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		ids, err := getIdsForTable(ctx, tx, "queries")
		if err != nil {
			return errors.Wrap(err, "getting ids")
		}
		if err := calculatePercentiles(ctx, tx, statsTypeQuery, ids); err != nil {
			return err
		}

		return nil
	})
}

func calculatePercentiles(ctx context.Context, tx sqlx.ExtContext, aggregate string, ids []uint) error {
	for _, id := range ids {
		var totalExecutions int
		statsMap := make(map[string]interface{})

		// many queries is not ideal, but getting both values and totals in the same query was a bit more complicated
		// so I went for the simpler approach first, we can optimize later
		if err := setP50AndP95Map(ctx, tx, aggregate, "user_time", id, statsMap); err != nil {
			return err
		}
		if err := setP50AndP95Map(ctx, tx, aggregate, "system_time", id, statsMap); err != nil {
			return err
		}

		err := sqlx.GetContext(ctx, tx, &totalExecutions, getTotalExecutionsQuery(aggregate), id)
		if err != nil {
			return errors.Wrapf(err, "getting total executions for %s %d", aggregate, id)
		}
		statsMap["total_executions"] = totalExecutions

		statsJson, err := json.Marshal(statsMap)
		if err != nil {
			return errors.Wrap(err, "marshaling stats")
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO aggregated_stats(id, type, json_value) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE json_value=VALUES(json_value)`,
			id, aggregate, statsJson,
		)
		if err != nil {
			return errors.Wrapf(err, "inserting stats for %s id %d", aggregate, id)
		}
	}
	return nil
}

func getTotalExecutionsQuery(aggregate string) string {
	switch aggregate {
	case "scheduled_query":
		return scheduledQueryTotalExecutions
	case "query":
		return queryTotalExecutions
	}
	return ""
}

func getIdsForTable(ctx context.Context, tx sqlx.QueryerContext, table string) ([]uint, error) {
	rows, err := tx.QueryxContext(ctx, fmt.Sprintf(`SELECT id FROM %s`, table))
	if err != nil {
		return nil, errors.Wrapf(err, "querying %s ids", table)
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		err := rows.Scan(&id)
		if err != nil {
			return nil, errors.Wrap(err, "scanning id for scheduled_query")
		}
		ids = append(ids, id)
	}
	return ids, nil
}
