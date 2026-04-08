package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) RecordHostHourlyData(ctx context.Context, hostID uint, dataset string, entityID uint, timestamp time.Time) error {
	utc := timestamp.UTC()
	dateOnly := utc.Format("2006-01-02")
	hour := utc.Hour()

	stmt := `
		INSERT INTO host_hourly_data (host_id, dataset, entity_id, chart_date, hours_bitmap)
		VALUES (?, ?, ?, ?, (1 << ?))
		ON DUPLICATE KEY UPDATE hours_bitmap = hours_bitmap | (1 << ?)`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, dataset, entityID, dateOnly, hour, hour)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "record host hourly data")
	}
	return nil
}

func (ds *Datastore) GetChartData(
	ctx context.Context,
	dataset string,
	startDate time.Time,
	endDate time.Time,
	hostFilter *fleet.ChartHostFilter,
	entityIDs []uint,
	hasEntityDimension bool,
	downsampleTo2h bool,
) ([]fleet.ChartDataPoint, error) {
	// Build the host filter subquery.
	hostSubquery, hostArgs := buildHostFilterSubquery(hostFilter)

	// Build entity filter clause.
	var entityClause string
	var entityArgs []any
	if len(entityIDs) > 0 {
		entityClause = " AND hd.entity_id IN (?)"
		entityArgs = append(entityArgs, entityIDs)
	}

	// Build per-hour SUM/COUNT expressions instead of a CTE cross join.
	// For non-entity datasets: SUM((bitmap >> h) & 1)
	// For entity datasets: COUNT(DISTINCT CASE WHEN bit set THEN host_id END)
	var hours []int
	var selectExprs []string

	step := 1
	maxHour := 23
	if downsampleTo2h {
		step = 2
		maxHour = 22
	}

	for h := 0; h <= maxHour; h += step {
		hours = append(hours, h)
		if downsampleTo2h {
			// 2-hour blocks: either hour in the pair has a bit set.
			if hasEntityDimension {
				selectExprs = append(selectExprs, fmt.Sprintf(
					"COUNT(DISTINCT CASE WHEN (hd.hours_bitmap >> %d) & 3 > 0 THEN hd.host_id END) AS h%d", h, h))
			} else {
				selectExprs = append(selectExprs, fmt.Sprintf(
					"SUM(CASE WHEN (hd.hours_bitmap >> %d) & 3 > 0 THEN 1 ELSE 0 END) AS h%d", h, h))
			}
		} else {
			if hasEntityDimension {
				selectExprs = append(selectExprs, fmt.Sprintf(
					"COUNT(DISTINCT CASE WHEN (hd.hours_bitmap >> %d) & 1 = 1 THEN hd.host_id END) AS h%d", h, h))
			} else {
				selectExprs = append(selectExprs, fmt.Sprintf(
					"SUM((hd.hours_bitmap >> %d) & 1) AS h%d", h, h))
			}
		}
	}

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	query := fmt.Sprintf(`
		SELECT hd.chart_date, %s
		FROM host_hourly_data hd
		WHERE hd.dataset = ?
			AND hd.chart_date BETWEEN ? AND ?
			%s
			%s
		GROUP BY hd.chart_date
		ORDER BY hd.chart_date`,
		strings.Join(selectExprs, ", "), entityClause, hostSubquery)

	var args []any
	args = append(args, dataset, startStr, endStr)
	args = append(args, entityArgs...)
	args = append(args, hostArgs...)

	// Expand sqlx.In placeholders.
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand chart data query args")
	}
	query = ds.reader(ctx).Rebind(query)

	dbRows, err := ds.reader(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get chart data")
	}
	defer dbRows.Close()

	var results []fleet.ChartDataPoint
	for dbRows.Next() {
		var chartDate time.Time
		hourVals := make([]int, len(hours))
		scanArgs := make([]any, len(hours)+1)
		scanArgs[0] = &chartDate
		for i := range hours {
			scanArgs[i+1] = &hourVals[i]
		}
		if err := dbRows.Scan(scanArgs...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scan chart data row")
		}
		for i, h := range hours {
			ts := time.Date(chartDate.Year(), chartDate.Month(), chartDate.Day(), h, 0, 0, 0, time.UTC)
			results = append(results, fleet.ChartDataPoint{
				Timestamp: ts,
				Value:     hourVals[i],
			})
		}
	}
	if err := dbRows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterate chart data rows")
	}

	return results, nil
}

func (ds *Datastore) CountHostsForChartFilter(ctx context.Context, hostFilter *fleet.ChartHostFilter) (int, error) {
	subquery, args := buildHostCountFilterClauses(hostFilter)

	query := fmt.Sprintf(`SELECT COUNT(*) FROM hosts h WHERE 1=1 %s`, subquery)

	// Expand sqlx.In placeholders.
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "expand count hosts query args")
	}
	query = ds.reader(ctx).Rebind(query)

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts for chart filter")
	}
	return count, nil
}

func (ds *Datastore) CleanupHostHourlyData(ctx context.Context, days int) error {
	_, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_hourly_data WHERE chart_date < CURDATE() - INTERVAL ? DAY`, days)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup host hourly data")
	}
	return nil
}

// buildHostFilterSubquery builds SQL clauses to filter host_hourly_data rows by host attributes.
// Uses "hd" as the table alias. Returns the clause (prefixed with AND) and args.
// Args may contain slices — caller must use sqlx.In to expand them.
func buildHostFilterSubquery(filter *fleet.ChartHostFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}

	var clauses []string
	var args []any

	if len(filter.LabelIDs) > 0 {
		clauses = append(clauses, "hd.host_id IN (SELECT DISTINCT host_id FROM label_membership WHERE label_id IN (?))")
		args = append(args, filter.LabelIDs)
	}

	if len(filter.Platforms) > 0 {
		clauses = append(clauses, "hd.host_id IN (SELECT id FROM hosts WHERE platform IN (?))")
		args = append(args, filter.Platforms)
	}

	if len(filter.IncludeHostIDs) > 0 {
		clauses = append(clauses, "hd.host_id IN (?)")
		args = append(args, filter.IncludeHostIDs)
	}

	if len(filter.ExcludeHostIDs) > 0 {
		clauses = append(clauses, "hd.host_id NOT IN (?)")
		args = append(args, filter.ExcludeHostIDs)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return " AND " + strings.Join(clauses, " AND "), args
}

// buildHostCountFilterClauses builds filter clauses for counting hosts directly from the hosts table.
// Uses "h" as the table alias. Args may contain slices — caller must use sqlx.In to expand them.
func buildHostCountFilterClauses(filter *fleet.ChartHostFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}

	var clauses []string
	var args []any

	if len(filter.LabelIDs) > 0 {
		clauses = append(clauses, "h.id IN (SELECT DISTINCT host_id FROM label_membership WHERE label_id IN (?))")
		args = append(args, filter.LabelIDs)
	}

	if len(filter.Platforms) > 0 {
		clauses = append(clauses, "h.platform IN (?)")
		args = append(args, filter.Platforms)
	}

	if len(filter.IncludeHostIDs) > 0 {
		clauses = append(clauses, "h.id IN (?)")
		args = append(args, filter.IncludeHostIDs)
	}

	if len(filter.ExcludeHostIDs) > 0 {
		clauses = append(clauses, "h.id NOT IN (?)")
		args = append(args, filter.ExcludeHostIDs)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return " AND " + strings.Join(clauses, " AND "), args
}
