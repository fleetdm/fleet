package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rowFixture is a compact way to declare a decodedSCDRow in tests.
func rowFixture(entityID string, ids []uint, validFrom, validTo time.Time) decodedSCDRow {
	return decodedSCDRow{
		entityID:  entityID,
		bitmap:    chart.NewBitmap(ids),
		validFrom: validFrom,
		validTo:   validTo,
	}
}

func TestAggregateBucketAccumulate(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// Three accumulate rows within the bucket, each observed during a different
	// hour. Accumulate semantics = union of all overlapping rows.
	rows := []decodedSCDRow{
		rowFixture("", []uint{1, 2}, bucketStart.Add(2*time.Hour), bucketStart.Add(3*time.Hour)),
		rowFixture("", []uint{3}, bucketStart.Add(10*time.Hour), bucketStart.Add(11*time.Hour)),
		rowFixture("", []uint{2, 4}, bucketStart.Add(15*time.Hour), bucketStart.Add(16*time.Hour)),
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategyAccumulate)
	assert.Equal(t, uint64(4), chart.BlobPopcount(got), "union of {1,2}, {3}, {2,4} = {1,2,3,4}")
}

func TestAggregateBucketAccumulateMultiEntity(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 14, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(time.Hour)

	// Future-style multi-entity accumulate dataset (e.g. software usage):
	// entity = software name; bitmap = hosts that used that software this hour.
	// Bucket value = distinct hosts using any tracked software during the hour.
	rows := []decodedSCDRow{
		rowFixture("slack", []uint{1, 2}, bucketStart, bucketEnd),
		rowFixture("zoom", []uint{2, 3}, bucketStart, bucketEnd),
		rowFixture("chrome", []uint{4}, bucketStart, bucketEnd),
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategyAccumulate)
	assert.Equal(t, uint64(4), chart.BlobPopcount(got), "union across entities = {1,2,3,4}")
}

func TestAggregateBucketSnapshotEndOfBucket(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// One entity "cve-A" changed state mid-bucket: affected hosts were {1,2,3}
	// from hr 0 to hr 14, then {1,2} from hr 14 onward (H3 patched).
	// End-of-bucket semantics should return only the *latest* state, not the OR.
	rows := []decodedSCDRow{
		rowFixture("cve-A", []uint{1, 2, 3}, bucketStart, bucketStart.Add(14*time.Hour)),
		rowFixture("cve-A", []uint{1, 2}, bucketStart.Add(14*time.Hour), time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)),
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, uint64(2), chart.BlobPopcount(got), "end-of-bucket state is {1,2}, not union {1,2,3}")
}

func TestAggregateBucketSnapshotMultipleEntities(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	sentinel := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

	// Two entities, each with an end-of-bucket state; snapshot returns OR across
	// entities of each's latest row.
	rows := []decodedSCDRow{
		// cve-A: latest state {1,2}
		rowFixture("cve-A", []uint{1, 2, 3}, bucketStart, bucketStart.Add(14*time.Hour)),
		rowFixture("cve-A", []uint{1, 2}, bucketStart.Add(14*time.Hour), sentinel),
		// cve-B: latest state {3,4}
		rowFixture("cve-B", []uint{3, 4}, bucketStart.Add(5*time.Hour), sentinel),
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, uint64(4), chart.BlobPopcount(got), "union of cve-A end-state {1,2} and cve-B end-state {3,4}")
}

func TestAggregateBucketSnapshotEntityDisappears(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// Entity was active early in bucket but its row was closed mid-bucket with
	// no replacement (entity disappeared — e.g., last affected host patched).
	// End-of-bucket semantics exclude it: no row is active at bucketEnd.
	rows := []decodedSCDRow{
		rowFixture("cve-A", []uint{1, 2, 3}, bucketStart, bucketStart.Add(14*time.Hour)),
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, uint64(0), chart.BlobPopcount(got), "entity closed mid-bucket is absent at bucketEnd")
}

func TestAggregateBucketSnapshotRowClosedExactlyAtBucketEnd(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// Row's valid_to == bucketEnd. The row represents state up to (but not
	// including) bucketEnd — i.e., the state just before the bucket ends.
	// That's exactly what end-of-bucket semantics should pick.
	rows := []decodedSCDRow{
		rowFixture("cve-A", []uint{1, 2}, bucketStart, bucketEnd),
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, uint64(2), chart.BlobPopcount(got), "row whose valid_to equals bucketEnd covers bucketEnd-ε")
}

func TestCleanupSCDData(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"PreservesOpenAndRecent", testCleanupPreservesOpenAndRecent},
		{"MultipleBatches", testCleanupMultipleBatches},
		{"HonorsCtxCancellation", testCleanupHonorsCtxCancellation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

func testCleanupPreservesOpenAndRecent(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC()

	// Old closed row — should be deleted (valid_to is 40 days ago, retention 30).
	tdb.InsertSCDRow(t, "cve", "old", now.AddDate(0, 0, -45), now.AddDate(0, 0, -40))
	// Recent closed row — within retention window, should be preserved.
	tdb.InsertSCDRow(t, "cve", "recent", now.AddDate(0, 0, -10), now.AddDate(0, 0, -5))
	// Open row (sentinel valid_to) — must always be preserved.
	tdb.InsertSCDRow(t, "cve", "open", now.AddDate(0, 0, -45), scdOpenSentinel)

	require.NoError(t, ds.CleanupSCDData(ctx, 30))

	assert.Equal(t, 2, tdb.CountSCDRows(t), "only the old closed row should be deleted")

	var entities []string
	require.NoError(t, tdb.DB.SelectContext(ctx, &entities, `SELECT entity_id FROM host_scd_data ORDER BY entity_id`))
	assert.Equal(t, []string{"open", "recent"}, entities)
}

func testCleanupMultipleBatches(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	now := time.Now().UTC()

	// Shrink batch size so we can prove the loop iterates without inserting
	// thousands of rows.
	prev := scdCleanupBatch
	scdCleanupBatch = 3
	t.Cleanup(func() { scdCleanupBatch = prev })

	// Insert 10 expired closed rows — that's 4 iterations at batch size 3
	// (3 + 3 + 3 + 1, where the final partial batch terminates the loop).
	for i := range 10 {
		validFrom := now.AddDate(0, 0, -45).Add(time.Duration(i) * time.Minute)
		validTo := now.AddDate(0, 0, -40).Add(time.Duration(i) * time.Minute)
		tdb.InsertSCDRow(t, "cve", fmt.Sprintf("e%d", i), validFrom, validTo)
	}

	require.NoError(t, ds.CleanupSCDData(ctx, 30))

	assert.Equal(t, 0, tdb.CountSCDRows(t), "all expired rows should be drained across batches")
}

func testCleanupHonorsCtxCancellation(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	now := time.Now().UTC()

	// Insert a single expired row so a non-canceled call would have something
	// to delete — confirms that nothing was removed because of cancellation.
	tdb.InsertSCDRow(t, "cve", "old", now.AddDate(0, 0, -45), now.AddDate(0, 0, -40))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := ds.CleanupSCDData(ctx, 30)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, tdb.CountSCDRows(t), "no rows should be deleted when ctx was canceled before the first batch")
}

func TestApplyScrubMaskToDataset(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"EmptyMaskNoOp", testScrubEmptyMaskNoOp},
		{"ClearsAffectedBits", testScrubClearsAffectedBits},
		{"SkipsRowsMaskDoesNotTouch", testScrubSkipsRowsMaskDoesNotTouch},
		{"ChunkedAcrossWriteBatches", testScrubChunkedAcrossWriteBatches},
		{"HonorsCtxCancellation", testScrubHonorsCtxCancellation},
		{"OtherDatasetUnaffected", testScrubOtherDatasetUnaffected},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

func testScrubEmptyMaskNoOp(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	now := time.Now().UTC()
	id := tdb.InsertSCDRowWithHostIDs(t, "uptime", "", []uint{1, 2, 3}, now.Add(-time.Hour), now)
	before := tdb.SCDBlob(t, id)

	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "uptime", nil, 0))
	assert.Equal(t, before, tdb.SCDBlob(t, id), "nil mask must not modify the row")

	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "uptime", roaring.New(), 0))
	assert.Equal(t, before, tdb.SCDBlob(t, id), "empty mask must not modify the row")
}

func testScrubClearsAffectedBits(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	now := time.Now().UTC()
	id := tdb.InsertSCDRowWithHostIDs(t, "uptime", "", []uint{1, 2, 3, 4, 5}, now.Add(-time.Hour), now)

	mask := chart.NewBitmap([]uint{2, 4})
	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "uptime", mask, 0))

	assert.Equal(t, []uint{1, 3, 5}, tdb.SCDHostIDs(t, id))
}

func testScrubSkipsRowsMaskDoesNotTouch(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	// Two rows: one with hosts the mask hits, one with hosts it doesn't. The
	// untouched row's bitmap MUST be byte-for-byte identical post-scrub —
	// this is the contract the skip-noop optimization promises.
	now := time.Now().UTC()
	hitID := tdb.InsertSCDRowWithHostIDs(t, "uptime", "a", []uint{1, 2, 3}, now.Add(-time.Hour), now)
	missID := tdb.InsertSCDRowWithHostIDs(t, "uptime", "b", []uint{10, 11, 12}, now.Add(-time.Hour), now)
	missBefore := tdb.SCDBlob(t, missID)

	mask := chart.NewBitmap([]uint{2})
	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "uptime", mask, 0))

	assert.Equal(t, []uint{1, 3}, tdb.SCDHostIDs(t, hitID))
	assert.Equal(t, missBefore, tdb.SCDBlob(t, missID), "mask doesn't intersect — row must remain unchanged")
}

func testScrubChunkedAcrossWriteBatches(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	// Shrink the write-batch cap so a small number of rows still exercises
	// the multi-chunk path.
	prev := scdScrubWriteBatchCap
	scdScrubWriteBatchCap = 3
	t.Cleanup(func() { scdScrubWriteBatchCap = prev })

	now := time.Now().UTC()
	mask := chart.NewBitmap([]uint{1})

	// 7 rows, all containing host 1 → 7 affected rows → 3+3+1 across chunks.
	// Read batch of 4 forces two read pages, each splitting into multiple
	// CASE/WHEN UPDATEs.
	ids := make([]uint, 7)
	for i := range ids {
		ids[i] = tdb.InsertSCDRowWithHostIDs(t, "uptime", fmt.Sprintf("e%d", i),
			[]uint{1, 2}, now.Add(-time.Hour), now)
	}

	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "uptime", mask, 4))

	for _, id := range ids {
		assert.Equal(t, []uint{2}, tdb.SCDHostIDs(t, id), "row %d", id)
	}
}

func testScrubHonorsCtxCancellation(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	now := time.Now().UTC()
	id := tdb.InsertSCDRowWithHostIDs(t, "uptime", "", []uint{1, 2}, now.Add(-time.Hour), now)
	before := tdb.SCDBlob(t, id)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := ds.ApplyScrubMaskToDataset(ctx, "uptime", chart.NewBitmap([]uint{1}), 0)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, before, tdb.SCDBlob(t, id), "row must be untouched when ctx was canceled before the first read")
}

func testScrubOtherDatasetUnaffected(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	now := time.Now().UTC()

	uptimeID := tdb.InsertSCDRowWithHostIDs(t, "uptime", "", []uint{1, 2, 3}, now.Add(-time.Hour), now)
	cveID := tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-1", []uint{1, 2, 3}, now.Add(-time.Hour), now)
	cveBefore := tdb.SCDBlob(t, cveID)

	mask := chart.NewBitmap([]uint{2})
	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "uptime", mask, 0))

	assert.Equal(t, []uint{1, 3}, tdb.SCDHostIDs(t, uptimeID))
	assert.Equal(t, cveBefore, tdb.SCDBlob(t, cveID), "cve dataset must not be touched by an uptime scrub")
}
