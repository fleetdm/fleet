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
		`SELECT host_bitmap FROM host_hourly_data_blobs WHERE dataset = 'uptime' AND entity_id = '' AND chart_date = ? AND hour = ?`,
		dateStr, hour)
	if err == nil {
		newBlob = chart.BlobOR(existing, newBlob)
	}
	// If err is sql.ErrNoRows, that's fine — no existing blob to merge.

	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
		 VALUES ('uptime', '', ?, ?, ?)
		 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`,
		dateStr, hour, newBlob)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "write uptime blob data")
	}
	return nil
}

func (ds *Datastore) GetBlobData(ctx context.Context, dataset string, startDate, endDate time.Time, entityIDs []string) ([]chart.BlobDataPoint, error) {
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
