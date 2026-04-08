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

	// Choose aggregation function.
	countExpr := "COUNT(*)"
	if hasEntityDimension {
		countExpr = "COUNT(DISTINCT hd.host_id)"
	}

	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	var query string
	if downsampleTo2h {
		// 2-hour blocks: check if either hour in the pair has a bit set.
		// hour_num goes 0,2,4,...,22 — we check (bitmap >> hour_num) & 3 > 0.
		query = fmt.Sprintf(`
			WITH RECURSIVE hour_numbers AS (
				SELECT 0 AS hour_num
				UNION ALL
				SELECT hour_num + 2 FROM hour_numbers WHERE hour_num < 22
			)
			SELECT
				hd.chart_date,
				hn.hour_num,
				%s AS value
			FROM host_hourly_data hd
			CROSS JOIN hour_numbers hn
			WHERE hd.dataset = ?
				AND hd.chart_date BETWEEN ? AND ?
				AND ((hd.hours_bitmap >> hn.hour_num) & 3) > 0
				%s
				%s
			GROUP BY hd.chart_date, hn.hour_num
			ORDER BY hd.chart_date, hn.hour_num`, countExpr, entityClause, hostSubquery)
	} else {
		query = fmt.Sprintf(`
			WITH RECURSIVE hour_numbers AS (
				SELECT 0 AS hour_num
				UNION ALL
				SELECT hour_num + 1 FROM hour_numbers WHERE hour_num < 23
			)
			SELECT
				hd.chart_date,
				hn.hour_num,
				%s AS value
			FROM host_hourly_data hd
			CROSS JOIN hour_numbers hn
			WHERE hd.dataset = ?
				AND hd.chart_date BETWEEN ? AND ?
				AND ((hd.hours_bitmap >> hn.hour_num) & 1) = 1
				%s
				%s
			GROUP BY hd.chart_date, hn.hour_num
			ORDER BY hd.chart_date, hn.hour_num`, countExpr, entityClause, hostSubquery)
	}

	var args []any
	args = append(args, dataset, startStr, endStr)
	args = append(args, entityArgs...)
	args = append(args, hostArgs...)

	// Expand sqlx.In placeholders.
	query, args, err := sqlx.In(query, args...)
	fmt.Printf("Expanded query: %s\nWith args: %v\n", query, args)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand chart data query args")
	}
	query = ds.reader(ctx).Rebind(query)

	type row struct {
		ChartDate time.Time `db:"chart_date"`
		HourNum   int       `db:"hour_num"`
		Value     int       `db:"value"`
	}

	var rows []row
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get chart data")
	}

	results := make([]fleet.ChartDataPoint, 0, len(rows))
	for _, r := range rows {
		ts := time.Date(r.ChartDate.Year(), r.ChartDate.Month(), r.ChartDate.Day(), r.HourNum, 0, 0, 0, time.UTC)
		results = append(results, fleet.ChartDataPoint{
			Timestamp: ts,
			Value:     r.Value,
		})
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
