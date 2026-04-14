package mysql

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// scdOpenSentinel is the end-of-time marker used for valid_to on currently-active
// SCD rows. A fixed sentinel lets the unique key on (dataset, entity_id, valid_from)
// enforce "at most one row per entity per day" and makes same-day bitmap updates
// a natural INSERT ... ON DUPLICATE KEY UPDATE.
const scdOpenSentinel = "9999-12-31"

// scdDateFormat is the DATE format used in the SCD table.
const scdDateFormat = "2006-01-02"

// scdUpsertBatch caps how many entity rows are written per INSERT statement.
const scdUpsertBatch = 200

func (ds *Datastore) RecordSCDData(ctx context.Context, dataset string, entityBitmaps map[string][]byte, now time.Time) error {
	today := now.UTC().Format(scdDateFormat)

	// Fetch all currently-open rows for this dataset so we can diff against the
	// desired state in Go. Small table (# entities at most), fast.
	type openRow struct {
		EntityID   string `db:"entity_id"`
		HostBitmap []byte `db:"host_bitmap"`
		ValidFrom  string `db:"valid_from"`
	}
	var openRows []openRow
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &openRows,
		`SELECT entity_id, host_bitmap, DATE_FORMAT(valid_from, '%Y-%m-%d') AS valid_from
		 FROM host_scd_data
		 WHERE dataset = ? AND valid_to = ?`,
		dataset, scdOpenSentinel); err != nil {
		return ctxerr.Wrap(ctx, err, "fetch open SCD rows")
	}

	openByEntity := make(map[string]openRow, len(openRows))
	for _, r := range openRows {
		openByEntity[r.EntityID] = r
	}

	// Partition the work into closes (entities whose open row is stale and must be
	// sealed at today) and upserts (entities whose current bitmap needs to land in
	// today's row). Closes only apply when the existing open row's valid_from
	// predates today; same-day updates collapse onto today's row via ODKU.
	var toClose []string
	type upsertRow struct {
		entityID string
		bitmap   []byte
	}
	var toUpsert []upsertRow

	for entityID, bitmap := range entityBitmaps {
		existing, hasOpen := openByEntity[entityID]
		if hasOpen && bytes.Equal(existing.HostBitmap, bitmap) {
			continue // unchanged state — leave the row alone
		}
		if hasOpen && existing.ValidFrom < today {
			toClose = append(toClose, entityID)
		}
		toUpsert = append(toUpsert, upsertRow{entityID: entityID, bitmap: bitmap})
	}

	// Entities that disappeared entirely — close their open rows. If the row
	// opened today the close leaves a zero-length historical record; that's fine
	// and callers can filter `valid_from < valid_to` if they care.
	for entityID := range openByEntity {
		if _, ok := entityBitmaps[entityID]; !ok {
			toClose = append(toClose, entityID)
		}
	}

	if len(toClose) > 0 {
		closeQuery, closeArgs, err := sqlx.In(
			`UPDATE host_scd_data SET valid_to = ?
			 WHERE dataset = ? AND valid_to = ? AND entity_id IN (?)`,
			today, dataset, scdOpenSentinel, toClose)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand close SCD query args")
		}
		closeQuery = ds.rebind(closeQuery)
		if _, err := ds.writer(ctx).ExecContext(ctx, closeQuery, closeArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "close stale SCD rows")
		}
	}

	// Batched upserts: ODKU on uniq_entity_day means same-day overwrites collapse
	// onto today's row; new-day writes create a fresh row whose predecessor (if
	// any) was just closed above.
	for i := 0; i < len(toUpsert); i += scdUpsertBatch {
		end := min(i+scdUpsertBatch, len(toUpsert))
		batch := toUpsert[i:end]

		placeholders := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*4)
		for _, r := range batch {
			placeholders = append(placeholders, "(?, ?, ?, ?)")
			args = append(args, dataset, r.entityID, r.bitmap, today)
		}
		// Concatenating hardcoded "(?,?,?,?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ", ") +
			` ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert SCD rows")
		}
	}

	return nil
}

func (ds *Datastore) GetSCDData(
	ctx context.Context,
	dataset string,
	startDate, endDate time.Time,
	hostFilter *chart.HostFilter,
	entityIDs []string,
) ([]chart.DataPoint, error) {
	startDay := startDate.UTC().Truncate(24 * time.Hour)
	endDay := endDate.UTC().Truncate(24 * time.Hour)

	// Fetch every row whose validity interval overlaps the requested range.
	// One query — we iterate buckets in Go.
	var entityClause string
	args := []any{dataset, endDay.Format(scdDateFormat), startDay.Format(scdDateFormat)}
	if len(entityIDs) > 0 {
		entityClause = " AND entity_id IN (?)"
		args = append(args, entityIDs)
	}

	// Rows that overlap [startDay, endDay]: valid_from <= endDay AND valid_to > startDay.
	query := fmt.Sprintf(`
		SELECT entity_id, host_bitmap,
			DATE_FORMAT(valid_from, '%%Y-%%m-%%d') AS valid_from,
			DATE_FORMAT(valid_to,   '%%Y-%%m-%%d') AS valid_to
		FROM host_scd_data
		WHERE dataset = ?
			AND valid_from <= ?
			AND valid_to   >  ?%s`, entityClause)

	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand SCD query args")
	}
	expanded = ds.rebind(expanded)

	type scdRow struct {
		EntityID   string `db:"entity_id"`
		HostBitmap []byte `db:"host_bitmap"`
		ValidFrom  string `db:"valid_from"`
		ValidTo    string `db:"valid_to"`
	}
	var rows []scdRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, expanded, expandedArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get SCD data")
	}

	// Build the optional host-filter mask once.
	var filterMask []byte
	if hostFilter != nil {
		hostIDs, err := ds.GetHostIDsForFilter(ctx, hostFilter)
		if err != nil {
			return nil, err
		}
		filterMask = chart.HostIDsToBlob(hostIDs)
	}

	// Walk buckets from startDay..endDay inclusive. For each bucket, OR the
	// bitmaps of rows whose interval covers that day, AND with filter, popcount.
	var results []chart.DataPoint
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		dayStr := d.Format(scdDateFormat)
		var merged []byte
		for _, r := range rows {
			if r.ValidFrom > dayStr || r.ValidTo <= dayStr {
				continue
			}
			merged = chart.BlobOR(merged, r.HostBitmap)
		}
		if filterMask != nil && merged != nil {
			merged = chart.BlobAND(merged, filterMask)
		}
		results = append(results, chart.DataPoint{
			Timestamp: d,
			Value:     chart.BlobPopcount(merged),
		})
	}
	return results, nil
}

func (ds *Datastore) CleanupSCDData(ctx context.Context, days int) error {
	// Only delete fully-closed rows whose valid_to is older than the cutoff.
	// Open rows (valid_to = sentinel) are always preserved.
	_, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_scd_data
		 WHERE valid_to < CURDATE() - INTERVAL ? DAY
		   AND valid_to <> ?`,
		days, scdOpenSentinel)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup SCD data")
	}
	return nil
}
