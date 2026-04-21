package mysql

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// scdOpenSentinel is the end-of-time marker used for valid_to on currently-open
// snapshot rows. Also used as a filter to distinguish open rows from closed ones.
var scdOpenSentinel = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

// scdUpsertBatch caps how many entity rows are written per INSERT statement.
const scdUpsertBatch = 200

func (ds *Datastore) RecordBucketData(
	ctx context.Context,
	dataset string,
	bucketStart time.Time,
	bucketSize time.Duration,
	strategy api.SampleStrategy,
	entityBitmaps map[string][]byte,
) error {
	if len(entityBitmaps) == 0 {
		return nil
	}
	bucketStart = bucketStart.UTC()

	switch strategy {
	case api.SampleStrategyAccumulate:
		return ds.recordAccumulate(ctx, dataset, bucketStart, bucketSize, entityBitmaps)
	case api.SampleStrategySnapshot:
		return ds.recordSnapshot(ctx, dataset, bucketStart, entityBitmaps)
	default:
		return ctxerr.Errorf(ctx, "unknown sample strategy: %s", strategy)
	}
}

// recordAccumulate OR-merges each entity's new bitmap into the row keyed by
// (dataset, entity_id, bucketStart). Rows are always explicitly closed at
// bucketStart+bucketSize; there is no cross-bucket collapse. A new bucket
// always starts a fresh row (different valid_from, different unique key), so
// the first sample in a new bucket never inherits the prior bucket's bitmap.
func (ds *Datastore) recordAccumulate(
	ctx context.Context,
	dataset string,
	bucketStart time.Time,
	bucketSize time.Duration,
	entityBitmaps map[string][]byte,
) error {
	validTo := bucketStart.Add(bucketSize)

	entityIDs := make([]string, 0, len(entityBitmaps))
	for id := range entityBitmaps {
		entityIDs = append(entityIDs, id)
	}

	// Fetch the current in-bucket bitmaps so we can OR-merge before writing.
	// ODKU alone can't do this — VALUES(host_bitmap) would overwrite, not OR.
	existing := make(map[string][]byte, len(entityIDs))
	if len(entityIDs) > 0 {
		query, args, err := sqlx.In(
			`SELECT entity_id, host_bitmap FROM host_scd_data
			 WHERE dataset = ? AND valid_from = ? AND entity_id IN (?)`,
			dataset, bucketStart, entityIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand accumulate select args")
		}
		query = ds.rebind(query)

		type row struct {
			EntityID   string `db:"entity_id"`
			HostBitmap []byte `db:"host_bitmap"`
		}
		var rows []row
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &rows, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "fetch in-bucket bitmaps")
		}
		for _, r := range rows {
			existing[r.EntityID] = r.HostBitmap
		}
	}

	type upsertRow struct {
		entityID string
		bitmap   []byte
	}
	toUpsert := make([]upsertRow, 0, len(entityBitmaps))
	for entityID, newBitmap := range entityBitmaps {
		merged := chart.BlobOR(existing[entityID], newBitmap)
		toUpsert = append(toUpsert, upsertRow{entityID: entityID, bitmap: merged})
	}

	for i := 0; i < len(toUpsert); i += scdUpsertBatch {
		end := min(i+scdUpsertBatch, len(toUpsert))
		batch := toUpsert[i:end]

		placeholders := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*5)
		for _, r := range batch {
			placeholders = append(placeholders, "(?, ?, ?, ?, ?)")
			args = append(args, dataset, r.entityID, r.bitmap, bucketStart, validTo)
		}
		// Concatenating hardcoded "(?,?,?,?,?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from, valid_to) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ", ") +
			` ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert accumulate rows")
		}
	}
	return nil
}

// recordSnapshot reconciles the current per-entity bitmaps against open rows.
// Unchanged entities keep their open row (valid_to = sentinel extends naturally).
// Changed entities get their open row closed at bucketStart (if it opened in a
// prior bucket) and a new open row inserted for this bucket. Entities absent
// from the input whose open rows still exist are closed.
func (ds *Datastore) recordSnapshot(
	ctx context.Context,
	dataset string,
	bucketStart time.Time,
	entityBitmaps map[string][]byte,
) error {
	type openRow struct {
		EntityID   string    `db:"entity_id"`
		HostBitmap []byte    `db:"host_bitmap"`
		ValidFrom  time.Time `db:"valid_from"`
	}
	var openRows []openRow
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &openRows,
		`SELECT entity_id, host_bitmap, valid_from
		 FROM host_scd_data
		 WHERE dataset = ? AND valid_to = ?`,
		dataset, scdOpenSentinel); err != nil {
		return ctxerr.Wrap(ctx, err, "fetch open SCD rows")
	}

	openByEntity := make(map[string]openRow, len(openRows))
	for _, r := range openRows {
		openByEntity[r.EntityID] = r
	}

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
		if hasOpen && existing.ValidFrom.Before(bucketStart) {
			toClose = append(toClose, entityID)
		}
		toUpsert = append(toUpsert, upsertRow{entityID: entityID, bitmap: bitmap})
	}

	// Entities that disappeared entirely — close their open rows. If the row
	// opened this bucket the close leaves a zero-length historical record; that's
	// fine and callers can filter valid_from < valid_to if they care.
	for entityID := range openByEntity {
		if _, ok := entityBitmaps[entityID]; !ok {
			toClose = append(toClose, entityID)
		}
	}

	if len(toClose) > 0 {
		closeQuery, closeArgs, err := sqlx.In(
			`UPDATE host_scd_data SET valid_to = ?
			 WHERE dataset = ? AND valid_to = ? AND entity_id IN (?)`,
			bucketStart, dataset, scdOpenSentinel, toClose)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand close SCD query args")
		}
		closeQuery = ds.rebind(closeQuery)
		if _, err := ds.writer(ctx).ExecContext(ctx, closeQuery, closeArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "close stale SCD rows")
		}
	}

	// Snapshot inserts leave valid_to at its DEFAULT (the sentinel). ODKU on
	// uniq_entity_bucket means same-bucket overwrites collapse onto this bucket's
	// row; new-bucket writes create a fresh row whose predecessor (if any) was
	// just closed above.
	for i := 0; i < len(toUpsert); i += scdUpsertBatch {
		end := min(i+scdUpsertBatch, len(toUpsert))
		batch := toUpsert[i:end]

		placeholders := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*4)
		for _, r := range batch {
			placeholders = append(placeholders, "(?, ?, ?, ?)")
			args = append(args, dataset, r.entityID, r.bitmap, bucketStart)
		}
		// Concatenating hardcoded "(?,?,?,?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ", ") +
			` ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert snapshot rows")
		}
	}

	return nil
}

// GetSCDData walks buckets of bucketSize across [startDate, endDate] and, for
// each bucket, ORs the bitmaps of every row whose [valid_from, valid_to)
// interval overlaps that bucket. The caller's host filter, if any, is applied
// as a bitmap AND. Returns numBuckets = (endDate - startDate) / bucketSize data
// points, labeled by bucket *start* (the first label is startDate + bucketSize;
// the last label is endDate). Zero-valued buckets are emitted.
//
// The caller is responsible for passing bucket-aligned startDate/endDate (e.g.
// local-midnight-aligned for tz-sensitive rendering); the walker does not
// truncate.
func (ds *Datastore) GetSCDData(
	ctx context.Context,
	dataset string,
	startDate, endDate time.Time,
	bucketSize time.Duration,
	hostFilter *types.HostFilter,
	entityIDs []string,
) ([]api.DataPoint, error) {
	startDate = startDate.UTC()
	endDate = endDate.UTC()

	numBuckets := int(endDate.Sub(startDate) / bucketSize)
	if numBuckets <= 0 {
		return nil, nil
	}

	// Fetch every row whose validity interval overlaps any of the buckets. The
	// walker filters precisely per bucket; this just narrows the scan.
	firstBucketStart := startDate.Add(bucketSize)
	lastBucketEnd := endDate.Add(bucketSize)
	args := []any{dataset, lastBucketEnd, firstBucketStart}
	var entityClause string
	if len(entityIDs) > 0 {
		entityClause = " AND entity_id IN (?)"
		args = append(args, entityIDs)
	}

	query := fmt.Sprintf(`
		SELECT entity_id, host_bitmap, valid_from, valid_to
		FROM host_scd_data
		WHERE dataset = ?
			AND valid_from <  ?
			AND valid_to   >  ?%s`, entityClause)

	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand SCD query args")
	}
	expanded = ds.rebind(expanded)

	type scdRow struct {
		EntityID   string    `db:"entity_id"`
		HostBitmap []byte    `db:"host_bitmap"`
		ValidFrom  time.Time `db:"valid_from"`
		ValidTo    time.Time `db:"valid_to"`
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

	results := make([]api.DataPoint, numBuckets)
	for i := range numBuckets {
		bucketStart := startDate.Add(time.Duration(i+1) * bucketSize)
		bucketEnd := bucketStart.Add(bucketSize)
		var merged []byte
		for _, r := range rows {
			// Row overlaps bucket iff valid_from < bucketEnd AND valid_to > bucketStart.
			if !r.ValidFrom.Before(bucketEnd) || !r.ValidTo.After(bucketStart) {
				continue
			}
			merged = chart.BlobOR(merged, r.HostBitmap)
		}
		if filterMask != nil && merged != nil {
			merged = chart.BlobAND(merged, filterMask)
		}
		results[i] = api.DataPoint{
			Timestamp: bucketStart,
			Value:     chart.BlobPopcount(merged),
		}
	}
	return results, nil
}

// CleanupSCDData deletes closed SCD rows whose valid_to is older than the
// retention cutoff. Open rows (valid_to = sentinel) are always preserved.
func (ds *Datastore) CleanupSCDData(ctx context.Context, days int) error {
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
