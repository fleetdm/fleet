package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
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
SELECT COALESCE((t1.%[1]s_total / t1.executions_total), 0)
FROM (SELECT (@rownum := @rownum + 1) AS row_number_value, sum1.*
      FROM (SELECT SUM(d.%[1]s) as %[1]s_total, SUM(d.executions) as executions_total
            FROM scheduled_query_stats d
            WHERE d.scheduled_query_id = ?
              AND d.executions > 0
            GROUP BY d.host_id) as sum1
      ORDER BY (%[1]s_total / executions_total)) AS t1,
     (SELECT @rownum := 0) AS r,
     (SELECT COUNT(*) AS total_rows
      FROM (SELECT COUNT(*)
            FROM scheduled_query_stats d
            WHERE d.scheduled_query_id = ?
              AND d.executions > 0
            GROUP BY d.host_id) as sum2) AS t2
WHERE t1.row_number_value = FLOOR(total_rows * %[2]s) + 1`

const (
	scheduledQueryTotalExecutions = `SELECT coalesce(sum(executions), 0) FROM scheduled_query_stats WHERE scheduled_query_id=?`
)

func getPercentileQuery(aggregate fleet.AggregatedStatsType, time string, percentile string) string {
	switch aggregate { //nolint:gocritic // ignore singleCaseSwitch
	case fleet.AggregatedStatsTypeScheduledQuery:
		return fmt.Sprintf(scheduledQueryPercentileQuery, time, percentile)
	}
	return ""
}

func setP50AndP95Map(
	ctx context.Context, tx sqlx.QueryerContext, aggregate fleet.AggregatedStatsType, time string, id uint, statsMap map[string]interface{},
) error {
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
	// Only process queries that actually have execution data in
	// scheduled_query_stats, instead of walking all query IDs (most of
	// which are saved/unscheduled queries with no stats). This avoids
	// running 5 expensive percentile queries per query that would all
	// return no rows.
	rows, err := ds.reader(ctx).QueryxContext(ctx,
		`SELECT DISTINCT scheduled_query_id FROM scheduled_query_stats WHERE executions > 0`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "querying query ids with execution data")
	}
	defer rows.Close()

	for rows.Next() {
		var queryID uint
		if err := rows.Scan(&queryID); err != nil {
			return ctxerr.Wrap(ctx, err, "scanning query id")
		}
		if err := ds.CalculateAggregatedPerfStatsPercentiles(
			ctx, fleet.AggregatedStatsTypeScheduledQuery, queryID,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "calculating stats for query")
		}
	}
	return rows.Err()
}

// CalculateAggregatedPerfStatsPercentiles calculates the aggregated user/system time performance statistics for the given query.
func (ds *Datastore) CalculateAggregatedPerfStatsPercentiles(ctx context.Context, aggregate fleet.AggregatedStatsType, queryID uint) error {
	// Before calling this method to update stats after a live query, we make sure the reader (replica) is up-to-date with the latest stats.
	// We are using the reader because the below SELECT queries are expensive, and we don't want to impact the performance of the writer.
	reader := ds.reader(ctx)
	var totalExecutions int
	statsMap := make(map[string]interface{})

	// many queries is not ideal, but getting both values and totals in the same query was a bit more complicated
	// so I went for the simpler approach first, we can optimize later
	if err := setP50AndP95Map(ctx, reader, aggregate, "user_time", queryID, statsMap); err != nil {
		return err
	}
	if err := setP50AndP95Map(ctx, reader, aggregate, "system_time", queryID, statsMap); err != nil {
		return err
	}

	err := sqlx.GetContext(ctx, reader, &totalExecutions, getTotalExecutionsQuery(aggregate), queryID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting total executions for %s %d", aggregate, queryID)
	}
	statsMap["total_executions"] = totalExecutions

	statsJson, err := json.Marshal(statsMap)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	// NOTE: this function gets called for query and scheduled_query, so the id
	// refers to a query/scheduled_query id, and it never computes "global"
	// stats. For that reason, we always set global_stats=0.
	_, err = ds.writer(ctx).ExecContext(
		ctx,
		`
		INSERT INTO aggregated_stats(id, type, global_stats, json_value)
		VALUES (?, ?, 0, ?)
		ON DUPLICATE KEY UPDATE json_value=VALUES(json_value)
		`,
		queryID, aggregate, statsJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for %s id %d", aggregate, queryID)
	}
	return nil
}

func getTotalExecutionsQuery(aggregate fleet.AggregatedStatsType) string {
	switch aggregate { //nolint:gocritic // ignore singleCaseSwitch
	case fleet.AggregatedStatsTypeScheduledQuery:
		return scheduledQueryTotalExecutions
	}
	return ""
}
