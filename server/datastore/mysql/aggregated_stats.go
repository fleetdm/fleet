package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

type aggregatedStatsType string

const (
	aggregatedStatsTypeScheduledQuery       = "scheduled_query"
	aggregatedStatsTypeMunkiVersions        = "munki_versions"
	aggregatedStatsTypeMunkiIssues          = "munki_issues"
	aggregatedStatsTypeOSVersions           = "os_versions"
	aggregatedStatsTypePolicyViolationsDays = "policy_violation_days"
	// those types are partial because the actual stats type is by platform,
	// which is computed with this stats type and the platform type (see
	// platformKey function).
	aggregatedStatsTypeMDMStatusPartial    = "mdm_status"
	aggregatedStatsTypeMDMSolutionsPartial = "mdm_solutions"
)

// These queries are a bit annoyingly written. The main reason they are this way is that we want rownum sorted. There's
// a slightly simpler version but that adds the rownum before sorting.

const scheduledQueryPercentileQuery = `
SELECT
	coalesce((t1.%s / t1.executions), 0)
FROM (
	SELECT (@rownum := @rownum + 1) AS row_number_value, mm.* FROM (
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
WHERE t1.row_number_value = floor(total_rows * %s) + 1;`

const (
	scheduledQueryTotalExecutions = `SELECT coalesce(sum(executions), 0) FROM scheduled_query_stats WHERE scheduled_query_id=?`
)

func getPercentileQuery(aggregate aggregatedStatsType, time string, percentile string) string {
	switch aggregate {
	case aggregatedStatsTypeScheduledQuery:
		return fmt.Sprintf(scheduledQueryPercentileQuery, time, time, time, percentile)
	}
	return ""
}

func setP50AndP95Map(ctx context.Context, tx sqlx.QueryerContext, aggregate aggregatedStatsType, time string, id uint, statsMap map[string]interface{}) error {
	var p50, p95 float64

	err := sqlx.GetContext(ctx, tx, &p50, getPercentileQuery(aggregate, time, "0.5"), id, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return ctxerr.Wrapf(ctx, err, "getting %s p50 for %s %d", time, aggregate, id)
	}
	statsMap[time+"_p50"] = p50
	err = sqlx.GetContext(ctx, tx, &p95, getPercentileQuery(aggregate, time, "0.95"), id, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return ctxerr.Wrapf(ctx, err, "getting %s p95 for %s %d", time, aggregate, id)
	}
	statsMap[time+"_p95"] = p95

	return nil
}

func (ds *Datastore) UpdateQueryAggregatedStats(ctx context.Context) error {
	err := walkIdsInTable(ctx, ds.reader(ctx), "queries", func(id uint) error {
		return calculatePercentiles(ctx, ds.writer(ctx), aggregatedStatsTypeScheduledQuery, id)
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "looping through query ids")
	}

	return nil
}

func calculatePercentiles(ctx context.Context, tx sqlx.ExtContext, aggregate aggregatedStatsType, id uint) error {
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
		return ctxerr.Wrapf(ctx, err, "getting total executions for %s %d", aggregate, id)
	}
	statsMap["total_executions"] = totalExecutions

	statsJson, err := json.Marshal(statsMap)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	// NOTE: this function gets called for query and scheduled_query, so the id
	// refers to a query/scheduled_query id, and it never computes "global"
	// stats. For that reason, we always set global_stats=0.
	_, err = tx.ExecContext(ctx,
		`
		INSERT INTO aggregated_stats(id, type, global_stats, json_value)
		VALUES (?, ?, 0, ?)
		ON DUPLICATE KEY UPDATE json_value=VALUES(json_value)
		`,
		id, aggregate, statsJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for %s id %d", aggregate, id)
	}
	return nil
}

func getTotalExecutionsQuery(aggregate aggregatedStatsType) string {
	switch aggregate {
	case aggregatedStatsTypeScheduledQuery:
		return scheduledQueryTotalExecutions
	}
	return ""
}

func walkIdsInTable(
	ctx context.Context,
	tx sqlx.QueryerContext,
	table string,
	visitFunc func(id uint) error,
) error {
	rows, err := tx.QueryxContext(ctx, fmt.Sprintf(`SELECT id FROM %s`, table))
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "querying %s ids", table)
	}
	defer rows.Close()

	for rows.Next() {
		var id uint

		if err := rows.Scan(&id); err != nil {
			return ctxerr.Wrapf(ctx, err, "scanning id for %s", table)
		}
		if err := visitFunc(id); err != nil {
			return ctxerr.Wrapf(ctx, err, "running visitFunc for %s", table)
		}
	}
	return nil
}
