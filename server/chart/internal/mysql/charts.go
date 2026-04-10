// Package mysql provides the MySQL datastore implementation for the chart bounded context.
package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// Datastore is the MySQL implementation of the chart datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

// NewDatastore creates a new MySQL datastore for the chart bounded context.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger}
}

// Ensure Datastore implements types.Datastore at compile time.
var _ types.Datastore = (*Datastore)(nil)

func (ds *Datastore) reader(ctx context.Context) sqlx.QueryerContext {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

func (ds *Datastore) writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

// rebind rewrites a query from ? placeholders to the driver-specific format.
func (ds *Datastore) rebind(query string) string {
	return ds.primary.Rebind(query)
}

func (ds *Datastore) RecordHostHourlyData(ctx context.Context, hostID uint, dataset string, entityID uint, timestamp time.Time) error {
	utc := timestamp.UTC()
	dateOnly := utc.Format("2006-01-02")
	hour := utc.Hour()

	stmt := `
		INSERT INTO host_daily_data_bitmaps (host_id, dataset, entity_id, chart_date, hours_bitmap)
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
	hostFilter *chart.HostFilter,
	entityIDs []uint,
	hasEntityDimension bool,
	downsample int,
) ([]chart.DataPoint, error) {
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
	// For non-entity datasets: SUM((bitmap >> h) & mask)
	// For entity datasets: COUNT(DISTINCT CASE WHEN masked bits set THEN host_id END)
	step := 1
	if downsample > 0 {
		step = downsample
	}
	// Bitmask covers `step` consecutive bits: e.g. step=1 → 0x1, step=2 → 0x3, step=4 → 0xF, step=8 → 0xFF.
	mask := (1 << step) - 1

	var hours []int
	var selectExprs []string
	for h := 0; h+step <= 24; h += step {
		hours = append(hours, h)
		if downsample > 0 {
			if hasEntityDimension {
				selectExprs = append(selectExprs, fmt.Sprintf(
					"COUNT(DISTINCT CASE WHEN (hd.hours_bitmap >> %d) & %d > 0 THEN hd.host_id END) AS h%d", h, mask, h))
			} else {
				selectExprs = append(selectExprs, fmt.Sprintf(
					"SUM(CASE WHEN (hd.hours_bitmap >> %d) & %d > 0 THEN 1 ELSE 0 END) AS h%d", h, mask, h))
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
		FROM host_daily_data_bitmaps hd
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
	query = ds.rebind(query)

	dbRows, err := ds.reader(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get chart data")
	}
	defer dbRows.Close()

	var results []chart.DataPoint
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
			results = append(results, chart.DataPoint{
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

func (ds *Datastore) CountHostsForChartFilter(ctx context.Context, hostFilter *chart.HostFilter) (int, error) {
	subquery, args := buildHostCountFilterClauses(hostFilter)

	query := fmt.Sprintf(`SELECT COUNT(*) FROM hosts h WHERE 1=1 %s`, subquery)

	// Expand sqlx.In placeholders.
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "expand count hosts query args")
	}
	query = ds.rebind(query)

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts for chart filter")
	}
	return count, nil
}

// collectChartDataIntervalMinutes is the lookback window for determining which hosts
// have recently checked in. Should match the cron schedule cadence.
const collectChartDataIntervalMinutes = 10

func (ds *Datastore) CollectUptimeChartData(ctx context.Context, now time.Time) error {
	utc := now.UTC()
	hour := utc.Hour()
	dateStr := utc.Format("2006-01-02")

	// Query host IDs that have recently checked in.
	var hostIDs []uint
	query := fmt.Sprintf(`
		SELECT h.id
		FROM hosts h
			LEFT JOIN host_seen_times hst ON h.id = hst.host_id
			LEFT JOIN nano_enrollments ne ON ne.id = h.uuid
				AND ne.type IN ('Device', 'User Enrollment (Device)')
		WHERE COALESCE(
			GREATEST(
				COALESCE(hst.seen_time, ne.last_seen_at),
				COALESCE(ne.last_seen_at, hst.seen_time)
			),
			NULLIF(h.detail_updated_at, '2000-01-01 00:00:00'),
			h.created_at
		) >= NOW() - INTERVAL %d MINUTE`,
		collectChartDataIntervalMinutes)

	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &hostIDs, query); err != nil {
		return ctxerr.Wrap(ctx, err, "query recently seen hosts for uptime")
	}
	if len(hostIDs) == 0 {
		return nil
	}

	// Build a blob from the host IDs.
	newBlob := chart.HostIDsToBlob(hostIDs)

	// Read the existing blob for this hour (if any) and OR it with the new data.
	var existing []byte
	err := sqlx.GetContext(ctx, ds.writer(ctx), &existing,
		`SELECT host_bitmap FROM host_hourly_data_blobs WHERE dataset = 'uptime' AND entity_id = 0 AND chart_date = ? AND hour = ?`,
		dateStr, hour)
	if err == nil {
		newBlob = chart.BlobOR(existing, newBlob)
	}
	// If err is sql.ErrNoRows, that's fine — no existing blob to merge.

	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
		 VALUES ('uptime', 0, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`,
		dateStr, hour, newBlob)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "write uptime blob data")
	}
	return nil
}

func (ds *Datastore) GetBlobData(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []uint) ([]chart.BlobDataPoint, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	var entityClause string
	var args []any
	args = append(args, dataset, startStr, endStr)

	if len(entityIDs) > 0 {
		entityClause = " AND entity_id IN (?)"
		args = append(args, entityIDs)
	}

	query := fmt.Sprintf(`
		SELECT chart_date, hour, host_bitmap
		FROM host_hourly_data_blobs
		WHERE dataset = ? AND chart_date BETWEEN ? AND ?%s
		ORDER BY chart_date, hour`, entityClause)

	// Expand sqlx.In placeholders.
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand blob data query args")
	}
	query = ds.rebind(query)

	type blobRow struct {
		ChartDate  time.Time `db:"chart_date"`
		Hour       int       `db:"hour"`
		HostBitmap []byte    `db:"host_bitmap"`
	}

	var rows []blobRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get blob data")
	}

	results := make([]chart.BlobDataPoint, len(rows))
	for i, r := range rows {
		results[i] = chart.BlobDataPoint{
			ChartDate:  r.ChartDate,
			Hour:       r.Hour,
			HostBitmap: r.HostBitmap,
		}
	}
	return results, nil
}

func (ds *Datastore) GetHostIDsForFilter(ctx context.Context, hostFilter *chart.HostFilter) ([]uint, error) {
	subquery, args := buildHostCountFilterClauses(hostFilter)

	query := fmt.Sprintf(`SELECT h.id FROM hosts h WHERE 1=1 %s`, subquery)

	// Expand sqlx.In placeholders.
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand host IDs filter query args")
	}
	query = ds.rebind(query)

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs for filter")
	}
	return ids, nil
}

func (ds *Datastore) CleanupHostDailyBitmapData(ctx context.Context, days int) error {
	_, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_daily_data_bitmaps WHERE chart_date < CURDATE() - INTERVAL ? DAY`, days)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup host daily bitmap data")
	}
	return nil
}

func (ds *Datastore) CleanupBlobData(ctx context.Context, days int) error {
	_, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_hourly_data_blobs WHERE chart_date < CURDATE() - INTERVAL ? DAY`, days)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup blob data")
	}
	return nil
}

// buildHostFilterSubquery builds SQL clauses to filter host_daily_data_bitmaps rows by host attributes.
// Uses "hd" as the table alias. Returns the clause (prefixed with AND) and args.
// Args may contain slices — caller must use sqlx.In to expand them.
func buildHostFilterSubquery(filter *chart.HostFilter) (string, []any) {
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
func buildHostCountFilterClauses(filter *chart.HostFilter) (string, []any) {
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
