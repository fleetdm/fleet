package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// scdOpenSentinel is the end-of-time marker used for the valid_to column of
// currently-active SCD rows. A fixed sentinel lets the unique key on
// (dataset, host_id, entity_id, valid_to) enforce "at most one open row per tuple"
// and makes upserts a single INSERT ... ON DUPLICATE KEY UPDATE.
const scdOpenSentinel = "9999-12-31 23:59:59"

func (ds *Datastore) RecordSCDData(ctx context.Context, dataset string, hostID uint, entityIDs []string, now time.Time) error {
	// Step 1: close any currently-open rows for this host/dataset whose entity
	// is no longer in the active set. If the active set is empty, close them all.
	nowStr := now.UTC().Format("2006-01-02 15:04:05")

	closeQuery := `UPDATE host_scd_data
		SET valid_to = ?
		WHERE dataset = ? AND host_id = ? AND valid_to = ?`
	closeArgs := []any{nowStr, dataset, hostID, scdOpenSentinel}

	if len(entityIDs) > 0 {
		closeQuery += ` AND entity_id NOT IN (?)`
		closeArgs = append(closeArgs, entityIDs)
	}

	q, args, err := sqlx.In(closeQuery, closeArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "expand close SCD query args")
	}
	q = ds.rebind(q)
	if _, err := ds.writer(ctx).ExecContext(ctx, q, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "close stale SCD rows")
	}

	if len(entityIDs) == 0 {
		return nil
	}

	// Step 2: upsert an open row for each currently-active entity. If the row
	// already exists (unique key hit), the ON DUPLICATE KEY UPDATE is a no-op that
	// preserves the original valid_from.
	placeholders := make([]string, 0, len(entityIDs))
	insertArgs := make([]any, 0, len(entityIDs)*4)
	for _, entityID := range entityIDs {
		placeholders = append(placeholders, "(?, ?, ?, ?)")
		insertArgs = append(insertArgs, dataset, hostID, entityID, nowStr)
	}
	insertQuery := fmt.Sprintf(`INSERT INTO host_scd_data (dataset, host_id, entity_id, valid_from)
		VALUES %s
		ON DUPLICATE KEY UPDATE valid_from = valid_from`,
		strings.Join(placeholders, ", "))

	if _, err := ds.writer(ctx).ExecContext(ctx, insertQuery, insertArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert SCD rows")
	}
	return nil
}

func (ds *Datastore) GetSCDData(
	ctx context.Context,
	dataset string,
	startDate, endDate time.Time,
	bucketIntervalHours int,
	hostFilter *chart.HostFilter,
	entityIDs []string,
) ([]chart.DataPoint, error) {
	if bucketIntervalHours <= 0 {
		bucketIntervalHours = 24
	}

	hostClause, hostArgs := buildHostFilterSubqueryForAlias(hostFilter, "s")

	var entityClause string
	var entityArgs []any
	if len(entityIDs) > 0 {
		entityClause = " AND s.entity_id IN (?)"
		entityArgs = append(entityArgs, entityIDs)
	}

	// Align start to bucket boundary (truncate to the bucket-aligned hour).
	startAligned := startDate.UTC().Truncate(time.Duration(bucketIntervalHours) * time.Hour)
	endAligned := endDate.UTC()

	query := fmt.Sprintf(`
		WITH RECURSIVE buckets AS (
			SELECT ? AS ts
			UNION ALL
			SELECT ts + INTERVAL ? HOUR FROM buckets WHERE ts + INTERVAL ? HOUR <= ?
		)
		SELECT b.ts, COUNT(DISTINCT s.host_id) AS host_count
		FROM buckets b
		LEFT JOIN host_scd_data s
			ON s.dataset = ?
			AND s.valid_from <= b.ts
			AND s.valid_to > b.ts
			%s
			%s
		GROUP BY b.ts
		ORDER BY b.ts`,
		entityClause, hostClause)

	args := []any{
		startAligned, bucketIntervalHours, bucketIntervalHours, endAligned,
		dataset,
	}
	args = append(args, entityArgs...)
	args = append(args, hostArgs...)

	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand SCD query args")
	}
	query = ds.rebind(query)

	type scdRow struct {
		Ts        time.Time `db:"ts"`
		HostCount int       `db:"host_count"`
	}
	var rows []scdRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get SCD data")
	}

	results := make([]chart.DataPoint, len(rows))
	for i, r := range rows {
		results[i] = chart.DataPoint{Timestamp: r.Ts, Value: r.HostCount}
	}
	return results, nil
}

func (ds *Datastore) CleanupSCDData(ctx context.Context, days int) error {
	// Only delete fully-closed rows whose valid_to is older than the cutoff.
	// Open rows (valid_to = sentinel) are always preserved.
	_, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_scd_data
		 WHERE valid_to < NOW() - INTERVAL ? DAY
		   AND valid_to <> ?`,
		days, scdOpenSentinel)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup SCD data")
	}
	return nil
}
