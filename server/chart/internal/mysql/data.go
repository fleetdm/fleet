package mysql

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// scdOpenSentinel is the end-of-time marker used for valid_to on currently-open
// snapshot rows. Also used as a filter to distinguish open rows from closed ones.
var scdOpenSentinel = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

// scdUpsertBatch caps how many entity rows are written per INSERT statement.
const scdUpsertBatch = 200

// scdCleanupBatch caps how many rows CleanupSCDData deletes per statement, so
// each batch's lock window is short and concurrent writers can interleave.
// var (not const) so tests can shrink it to exercise multi-batch behavior.
var scdCleanupBatch = 1000

// scdScrubWriteByteBudget targets the maximum payload size of one CASE/WHEN
// UPDATE statement emitted by ApplyScrubMaskToDataset. Bitmaps are sized in
// bytes, so the per-statement row count derives from this budget divided by
// the mask length. Sized well under MySQL's default max_allowed_packet
// (16-64 MB) to keep replication binlog events small.
const scdScrubWriteByteBudget = 2_000_000

// scdScrubWriteBatchCap bounds the row count per CASE/WHEN UPDATE statement
// regardless of mask size, capping parser/optimizer cost. var (not const) so
// tests can shrink it to exercise multi-batch behavior.
var scdScrubWriteBatchCap = 1000

// scdRow is a single row of host_scd_data as fetched by GetSCDData.
type scdRow struct {
	EntityID   string    `db:"entity_id"`
	HostBitmap []byte    `db:"host_bitmap"`
	ValidFrom  time.Time `db:"valid_from"`
	ValidTo    time.Time `db:"valid_to"`
}

func (ds *Datastore) RecordBucketData(
	ctx context.Context,
	dataset string,
	bucketStart time.Time,
	bucketSize time.Duration,
	strategy api.SampleStrategy,
	entityBitmaps map[string][]byte,
) error {
	bucketStart = bucketStart.UTC()

	switch strategy {
	case api.SampleStrategyAccumulate:
		// Accumulate with an empty map has nothing to OR-merge and no prior
		// state to reconcile — skip the round trip.
		if len(entityBitmaps) == 0 {
			return nil
		}
		return ds.recordAccumulate(ctx, dataset, bucketStart, bucketSize, entityBitmaps)
	case api.SampleStrategySnapshot:
		// Do NOT short-circuit on empty: empty means "no entities are
		// currently in the tracked state." For snapshot semantics this is a
		// meaningful write because any previously-open rows for this dataset
		// need to be closed at bucketStart. recordSnapshot handles empty
		// input correctly (the "absent entities" branch closes every open
		// row).
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
		// Using writer here since a stale read would OR-merge against an older
		// bitmap, then ODKU would overwrite the row with the partial merge — silently
		// dropping hosts from any sample the replica hadn't replicated yet.
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
	// Reader is safe here: the close UPDATE filters by valid_to = sentinel and the
	// insert uses ODKU on uniq_entity_bucket, so a stale read at worst produces
	// idempotent re-work (a no-op close or a same-bucket overwrite).
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &openRows,
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
// each bucket, aggregates the rows whose interval touches the bucket according
// to the sample strategy:
//   - Accumulate: OR every overlapping row (across all entities, unless the
//     caller restricted entityIDs). For single-entity datasets like uptime this
//     is "hosts observed at any point in bucket." For multi-entity datasets
//     like (future) software usage it's "distinct hosts seen doing anything
//     tracked during the bucket" — the entity dimension collapses into the
//     union of hosts touching any entity.
//   - Snapshot: per entity, take the row active at bucketEnd; OR across
//     entities. "State as of the end of the bucket" — for multi-entity datasets
//     like CVE, this is the union of hosts affected by any tracked entity at
//     bucketEnd.
//
// filterMask is AND-ed into every bucket's merged bitmap so results reflect
// only hosts visible to the caller. Returns numBuckets =
// (endDate - startDate) / bucketSize data points, labeled by bucket *start*
// (the first label is startDate + bucketSize; the last label is endDate).
// Zero-valued buckets are included with value 0, not omitted.
//
// The caller is responsible for passing bucket-aligned startDate/endDate (e.g.
// local-midnight-aligned for tz-sensitive rendering); the walker does not
// truncate.
func (ds *Datastore) GetSCDData(
	ctx context.Context,
	dataset string,
	startDate, endDate time.Time,
	bucketSize time.Duration,
	strategy api.SampleStrategy,
	filterMask []byte,
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
	switch {
	case entityIDs == nil:
		// no clause — match every entity for this dataset
	case len(entityIDs) == 0:
		// explicit empty set — match nothing; avoids MySQL syntax error from `IN ()`
		entityClause = " AND 1=0"
	default:
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

	var rows []scdRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, expanded, expandedArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get SCD data")
	}

	results := make([]api.DataPoint, numBuckets)
	for i := range numBuckets {
		bucketStart := startDate.Add(time.Duration(i+1) * bucketSize)
		bucketEnd := bucketStart.Add(bucketSize)
		merged := aggregateBucket(rows, bucketStart, bucketEnd, strategy)
		if merged != nil {
			merged = chart.BlobAND(merged, filterMask)
		}
		results[i] = api.DataPoint{
			Timestamp: bucketStart,
			Value:     chart.BlobPopcount(merged),
		}
	}
	return results, nil
}

// aggregateBucket returns the merged bitmap for a single bucket given the
// sample strategy. For Accumulate, ORs every overlapping row (entity dimension
// collapses into the union — correct for "distinct hosts seen doing anything
// tracked"). For Snapshot, picks the row active at bucketEnd per entity and
// ORs across entities.
func aggregateBucket(rows []scdRow, bucketStart, bucketEnd time.Time, strategy api.SampleStrategy) []byte {
	if strategy == api.SampleStrategySnapshot {
		// Per entity, the row "active at bucketEnd" is the one whose
		// [valid_from, valid_to) covers the instant bucketEnd-ε. For interval
		// boundaries, that's valid_from < bucketEnd AND valid_to >= bucketEnd.
		// Write semantics ensure at most one such row per (entity, moment).
		var merged []byte
		seen := make(map[string]struct{})
		for _, r := range rows {
			if !r.ValidFrom.Before(bucketEnd) || r.ValidTo.Before(bucketEnd) {
				continue
			}
			if _, dup := seen[r.EntityID]; dup {
				continue
			}
			seen[r.EntityID] = struct{}{}
			merged = chart.BlobOR(merged, r.HostBitmap)
		}
		return merged
	}

	// Accumulate: OR every row that overlaps the bucket.
	var merged []byte
	for _, r := range rows {
		if !r.ValidFrom.Before(bucketEnd) || !r.ValidTo.After(bucketStart) {
			continue
		}
		merged = chart.BlobOR(merged, r.HostBitmap)
	}
	return merged
}

// CleanupSCDData deletes closed SCD rows whose valid_to is older than the
// retention cutoff. Open rows (valid_to = sentinel) are always preserved.
// Deletes in batches so each statement holds locks briefly and the concurrent
// collection cron can interleave writes.
func (ds *Datastore) CleanupSCDData(ctx context.Context, days int) error {
	// Compute the cutoff in Go (UTC) so the retention boundary doesn't depend
	// on the MySQL session time zone — all valid_to writes are UTC.
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	for {
		if err := ctx.Err(); err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup SCD data")
		}
		res, err := ds.writer(ctx).ExecContext(ctx,
			`DELETE FROM host_scd_data
			 WHERE valid_to < ?
			   AND valid_to <> ?
			 ORDER BY valid_to
			 LIMIT ?`,
			cutoff, scdOpenSentinel, scdCleanupBatch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup SCD data")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup SCD data rows affected")
		}
		if n < int64(scdCleanupBatch) {
			return nil
		}
	}
}

// DeleteAllForDataset removes every host_scd_data row for the given dataset in
// batches. Used by the global scrub worker. Loops until a DELETE affects zero
// rows; the loop is naturally idempotent on retry.
func (ds *Datastore) DeleteAllForDataset(ctx context.Context, dataset string, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 5000
	}
	for {
		res, err := ds.writer(ctx).ExecContext(ctx,
			`DELETE FROM host_scd_data WHERE dataset = ? LIMIT ?`,
			dataset, batchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete SCD rows for dataset")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "rows affected for dataset delete")
		}
		if n == 0 {
			return nil
		}
	}
}

// HostIDsInFleets returns host IDs whose team_id is one of the given fleet IDs.
// Used by the per-fleet scrub worker to build a bitmap of hosts to clear from
// existing host_scd_data rows.
//
// Reads from the primary: the result drives an immediately-following UPDATE
// (ApplyScrubMaskToDataset). Replica lag could yield a stale membership set
// — scrubbing the wrong hosts, or missing hosts that just moved into the
// fleet — within the same job invocation.
func (ds *Datastore) HostIDsInFleets(ctx context.Context, fleetIDs []uint) ([]uint, error) {
	if len(fleetIDs) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`SELECT id FROM hosts WHERE team_id IN (?)`, fleetIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand host-IDs-in-fleets args")
	}
	query = ds.rebind(query)

	ctx = ctxdb.RequirePrimary(ctx, true)
	var ids []uint
	// reader(ctx) honors RequirePrimary set above and returns the writer connection.
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host IDs in fleets")
	}
	return ids, nil
}

// ApplyScrubMaskToDataset pages through host_scd_data rows for the given
// dataset in id-order, applies BlobANDNOT(host_bitmap, mask) to each, and
// writes the result back. Idempotent: re-running with the same mask is a
// no-op for already-scrubbed rows.
//
// An empty mask is a no-op (ANDNOT with empty leaves the bitmap unchanged).
// The walk happens once for the dataset regardless of how many fleets the
// mask was built from.
//
// Two efficiency mechanics keep this cheap on large datasets:
//   - Rows the mask doesn't touch (BlobANDNOT returns the same bytes) are
//     skipped — for sparse fleet-scoped masks, most rows generate no UPDATE.
//   - Surviving updates are flushed in chunked CASE/WHEN UPDATE statements
//     so a read-page of N rows costs O(N / writeBatch) round trips instead
//     of O(N).
func (ds *Datastore) ApplyScrubMaskToDataset(ctx context.Context, dataset string, mask []byte, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 5000
	}
	if len(mask) == 0 {
		// Nothing to clear; avoid the row walk entirely.
		return nil
	}

	// Size each CASE/WHEN UPDATE so its payload (~writeBatch * len(mask) bytes
	// of new bitmap data) stays under scdScrubWriteByteBudget. Bounded above
	// by scdScrubWriteBatchCap to keep parser cost predictable.
	writeBatch := min(max(scdScrubWriteByteBudget/len(mask), 1), scdScrubWriteBatchCap)

	type row struct {
		ID         uint   `db:"id"`
		HostBitmap []byte `db:"host_bitmap"`
	}
	type pendingRow struct {
		id       uint
		scrubbed []byte
	}

	// Paging select reads from the primary: the loop terminates on
	// `len(rows) == 0`, so replica lag could end the scrub early while
	// rows still exist on the primary, leaving disabled-scope bits behind
	// that the next disable won't re-enqueue.
	ctx = ctxdb.RequirePrimary(ctx, true)

	var lastID uint
	for {
		if err := ctx.Err(); err != nil {
			return ctxerr.Wrap(ctx, err, "scrub dataset")
		}

		var rows []row
		// reader(ctx) honors RequirePrimary set above and returns the writer connection.
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
			`SELECT id, host_bitmap FROM host_scd_data
			 WHERE dataset = ? AND id > ?
			 ORDER BY id LIMIT ?`,
			dataset, lastID, batchSize); err != nil {
			return ctxerr.Wrap(ctx, err, "read scrub batch")
		}
		if len(rows) == 0 {
			return nil
		}

		// Compute scrubbed bitmaps in Go; rows the mask doesn't touch
		// produce no UPDATE.
		pending := make([]pendingRow, 0, len(rows))
		for _, r := range rows {
			scrubbed := chart.BlobANDNOT(r.HostBitmap, mask)
			if !bytes.Equal(scrubbed, r.HostBitmap) {
				pending = append(pending, pendingRow{id: r.ID, scrubbed: scrubbed})
			}
			lastID = r.ID
		}

		for i := 0; i < len(pending); i += writeBatch {
			end := min(i+writeBatch, len(pending))
			chunk := pending[i:end]

			caseClauses := make([]string, 0, len(chunk))
			inPlaceholders := make([]string, 0, len(chunk))
			args := make([]any, 0, len(chunk)*3)
			for _, p := range chunk {
				caseClauses = append(caseClauses, "WHEN ? THEN ?")
				args = append(args, p.id, p.scrubbed)
			}
			for _, p := range chunk {
				inPlaceholders = append(inPlaceholders, "?")
				args = append(args, p.id)
			}
			// Concatenating hardcoded "WHEN ? THEN ?" / "?" placeholders, not user input.
			stmt := `UPDATE host_scd_data SET host_bitmap = CASE id ` + //nolint:gosec // G202
				strings.Join(caseClauses, " ") +
				` END WHERE id IN (` +
				strings.Join(inPlaceholders, ", ") + `)`
			if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "scrub batch")
			}
		}
	}
}
