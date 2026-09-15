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

// TestGetSCDDataMixedEncoding proves the lazy-migration premise: a dense legacy
// row and a roaring row for the same dataset are both decoded by the chart
// query path and contribute to the bucket's union. Without this, the day-1
// post-deploy story (mixed encodings coexisting until closed rows age out) is
// only covered by unit tests of DecodeBitmap, not by the wired-up read path.
func TestGetSCDDataMixedEncoding(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	startDate := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	endDate := startDate.Add(24 * time.Hour)
	// Rows must be open at bucketEnd (startDate + 2*bucketSize) for snapshot to
	// pick them, so seed them as open with valid_from comfortably before the
	// query window.
	validFrom := startDate.Add(-time.Hour)

	tdb.InsertSCDRowWithBlob(t, "cve", "CVE-A", testutils.DenseBlob([]uint{1, 2, 3}), validFrom, scdOpenSentinel)
	tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-B", []uint{3, 4, 5}, validFrom, scdOpenSentinel)

	pts, err := ds.GetSCDData(t.Context(), "cve",
		startDate, endDate, 24*time.Hour,
		api.SampleStrategySnapshot, nil, nil)
	require.NoError(t, err)
	require.Len(t, pts, 1)
	assert.Equal(t, 5, pts[0].Value, "union of dense {1,2,3} and roaring {3,4,5} = {1,2,3,4,5}")
}

// TestGetSCDDataEmptyEntityIDsReturnsZeroBuckets pins the non-nil empty
// entityIDs contract: a caller signaling "filter requested but resolved to
// nothing" must get zero-valued buckets across the date range — not an error
// from `IN ()` and not an empty slice. Rows are seeded that would match if the
// filter were nil; the empty-slice filter must exclude them.
func TestGetSCDDataEmptyEntityIDsReturnsZeroBuckets(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	startDate := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketSize := 24 * time.Hour
	endDate := startDate.Add(3 * bucketSize)
	validFrom := startDate.Add(-time.Hour)

	tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-A", []uint{1, 2, 3}, validFrom, scdOpenSentinel)
	tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-B", []uint{4, 5}, validFrom, scdOpenSentinel)

	pts, err := ds.GetSCDData(t.Context(), "cve",
		startDate, endDate, bucketSize,
		api.SampleStrategySnapshot, nil, []string{})
	require.NoError(t, err)
	require.Len(t, pts, 3, "one bucket per slot across the date range, not an empty slice")
	for i, dp := range pts {
		assert.Zero(t, dp.Value, "bucket %d must be zero — empty entityIDs filter excludes all rows", i)
	}
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

// recordTick writes one hourly collector tick: the full per-CVE state as of
// `at`. Mirrors what CVEDataset.Collect hands RecordBucketData.
func recordTick(t *testing.T, ds *Datastore, at time.Time, state map[string][]uint) {
	t.Helper()
	bitmaps := make(map[string]*roaring.Bitmap, len(state))
	for cve, ids := range state {
		bitmaps[cve] = chart.NewBitmap(ids)
	}
	require.NoError(t, ds.RecordBucketData(t.Context(), api.MetricCVE, at, time.Hour, api.SampleStrategySnapshot, bitmaps))
}

func values(points []api.DataPoint) []int {
	out := make([]int, len(points))
	for i, p := range points {
		out[i] = p.Value
	}
	return out
}

// seedCriticalFixture lays down three CVEs on tracked software (two critical,
// one low) and five hourly ticks in which the affected host sets change,
// shrink, and disappear. Returns the first tick's timestamp.
func seedCriticalFixture(t *testing.T, tdb *testutils.TestDB, ds *Datastore) time.Time {
	t.Helper()

	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-2026-0001")
	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-2026-0002")
	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-2026-0003")
	seedCVEMeta(t, tdb, "CVE-2026-0001", 9.5, 0.4, false)
	seedCVEMeta(t, tdb, "CVE-2026-0002", 10.0, 0.6, true)
	seedCVEMeta(t, tdb, "CVE-2026-0003", 3.0, 0.1, false)

	t0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	hr := func(n int) time.Time { return t0.Add(time.Duration(n) * time.Hour) }

	recordTick(t, ds, hr(0), map[string][]uint{"CVE-2026-0001": {1, 2}, "CVE-2026-0002": {2, 3}, "CVE-2026-0003": {9}})
	// Unchanged tick — the collector writes no new rows, so the aggregate must
	// not gain a row here either.
	recordTick(t, ds, hr(1), map[string][]uint{"CVE-2026-0001": {1, 2}, "CVE-2026-0002": {2, 3}, "CVE-2026-0003": {9}})
	// One CVE's host set shrinks, but the union is unchanged because the
	// dropped host is still affected by the other CVE.
	recordTick(t, ds, hr(2), map[string][]uint{"CVE-2026-0001": {1}, "CVE-2026-0002": {2, 3}, "CVE-2026-0003": {9}})
	recordTick(t, ds, hr(3), map[string][]uint{"CVE-2026-0001": {1}, "CVE-2026-0002": {2, 3, 4}, "CVE-2026-0003": {9}})
	recordTick(t, ds, hr(4), map[string][]uint{"CVE-2026-0001": {1}, "CVE-2026-0003": {9}})

	return t0
}

// retentionUnbounded is a horizon so far back it never floors a backfill, for
// tests that exercise the batching on its own.
var retentionUnbounded = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// backfillFully runs backfill batches until the series covers everything the
// dataset holds above horizon, and returns how many batches that took.
// Production spreads these across collection ticks.
func backfillFully(t *testing.T, ds *Datastore, aggID string, sourceIDs []string, now, horizon time.Time) int {
	t.Helper()
	for batches := 0; batches <= 200; batches++ {
		wrote, err := ds.BackfillAggregateEntity(t.Context(), api.MetricCVE, aggID, sourceIDs, now, horizon)
		require.NoError(t, err)
		if !wrote {
			return batches
		}
	}
	t.Fatal("backfill never reported completion")
	return 0
}

// If the two paths disagree, the chart silently reports a different population
// depending on which one a request happens to take.
func TestBackfillAggregateEntityMatchesPerCVEPath(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	t0 := seedCriticalFixture(t, tdb, ds)
	now := t0.Add(5 * time.Hour)

	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"CVE-2026-0001", "CVE-2026-0002"}, sourceIDs,
		"the low-severity CVE must stay out of the critical set")

	aggID := filter.AggregateEntityID()
	require.Positive(t, backfillFully(t, ds, aggID, sourceIDs, now, retentionUnbounded))

	for _, bucketHours := range []int{1, 3} {
		bucketSize := time.Duration(bucketHours) * time.Hour
		perCVE, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, bucketSize, api.SampleStrategySnapshot, nil, sourceIDs)
		require.NoError(t, err)
		agg, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, bucketSize, api.SampleStrategySnapshot, nil, []string{aggID})
		require.NoError(t, err)
		require.Equal(t, perCVE, agg, "aggregate and per-CVE series must agree at %dh buckets", bucketHours)
	}

	// Pin the values so the equality above can't pass on two all-zero results.
	perCVE, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, nil, sourceIDs)
	require.NoError(t, err)
	require.Equal(t, []int{3, 3, 4, 1, 1}, values(perCVE))
}

// The host mask is applied after the union, so scoping to a fleet or label must
// give the same answer on either path.
func TestBackfillAggregateEntityMatchesPerCVEPathUnderHostMask(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	t0 := seedCriticalFixture(t, tdb, ds)
	now := t0.Add(5 * time.Hour)

	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	aggID := filter.AggregateEntityID()
	backfillFully(t, ds, aggID, sourceIDs, now, retentionUnbounded)

	mask := chart.NewBitmap([]uint{2, 3, 4})
	perCVE, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, mask, sourceIDs)
	require.NoError(t, err)
	agg, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, mask, []string{aggID})
	require.NoError(t, err)
	require.Equal(t, perCVE, agg)
	require.Equal(t, []int{2, 2, 3, 0, 0}, values(perCVE), "host 1 is masked out")
}

// Runs on every tick, so a second pass must not rewrite history or stack
// duplicate rows on what it already built.
func TestBackfillAggregateEntityIsIdempotent(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	t0 := seedCriticalFixture(t, tdb, ds)
	now := t0.Add(5 * time.Hour)
	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	aggID := filter.AggregateEntityID()

	backfillFully(t, ds, aggID, sourceIDs, now, retentionUnbounded)
	first := tdb.CountSCDRows(t)

	wrote, err := ds.BackfillAggregateEntity(ctx, api.MetricCVE, aggID, sourceIDs, now, retentionUnbounded)
	require.NoError(t, err)
	require.False(t, wrote, "a series that already reaches the oldest data is left alone")
	require.Equal(t, first, tdb.CountSCDRows(t))
}

// Nothing collected yet means no window to backfill, and no bogus row left.
func TestBackfillAggregateEntityNoSourceData(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	filter := api.CVEFilter{CVSSMin: new(9.0)}
	wrote, err := ds.BackfillAggregateEntity(ctx, api.MetricCVE, filter.AggregateEntityID(), nil, time.Now().UTC(), retentionUnbounded)
	require.NoError(t, err)
	require.False(t, wrote)
	require.Equal(t, 0, tdb.CountSCDRows(t))
}

// A filter resolving to no CVEs is a legitimate empty chart, not a reason to
// fall back to the slow path forever.
func TestBackfillAggregateEntityEmptySourceSet(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	t0 := seedCriticalFixture(t, tdb, ds)
	now := t0.Add(5 * time.Hour)
	aggID := api.CVEFilter{CVSSMin: new(9.9), CVSSMax: new(9.95)}.AggregateEntityID()

	require.Positive(t, backfillFully(t, ds, aggID, nil, now, retentionUnbounded))

	agg, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, nil, []string{aggID})
	require.NoError(t, err)
	require.Equal(t, []int{0, 0, 0, 0, 0}, values(agg))
}

// A partly-built series read as complete would render its missing hours as zero.
func TestAggregateCoversFrom(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	t0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	hr := func(n int) time.Time { return t0.Add(time.Duration(n) * time.Hour) }

	covers, err := ds.AggregateCoversFrom(ctx, api.MetricCVE, "agg:whatever", t0, retentionUnbounded)
	require.NoError(t, err)
	require.False(t, covers)

	tdb.InsertSCDRowWithHostIDs(t, api.MetricCVE, "CVE-2026-0001", []uint{1}, t0, scdOpenSentinel)

	// A series that only reaches hour 4 does not cover a request from hour 0.
	tdb.InsertSCDRowWithHostIDs(t, api.MetricCVE, "agg:whatever", []uint{1}, hr(4), scdOpenSentinel)
	covers, err = ds.AggregateCoversFrom(ctx, api.MetricCVE, "agg:whatever", t0, retentionUnbounded)
	require.NoError(t, err)
	require.False(t, covers)

	covers, err = ds.AggregateCoversFrom(ctx, api.MetricCVE, "agg:whatever", hr(5), retentionUnbounded)
	require.NoError(t, err)
	require.True(t, covers)

	// Once it reaches the oldest data, it covers any window, including one
	// starting before the dataset itself begins.
	tdb.InsertSCDRowWithHostIDs(t, api.MetricCVE, "agg:whatever", []uint{1}, t0, hr(4))
	covers, err = ds.AggregateCoversFrom(ctx, api.MetricCVE, "agg:whatever", t0.Add(-100*time.Hour), retentionUnbounded)
	require.NoError(t, err)
	require.True(t, covers)

	// Datasets are separate namespaces — an uptime lookup must not see it.
	covers, err = ds.AggregateCoversFrom(ctx, "uptime", "agg:whatever", t0, retentionUnbounded)
	require.NoError(t, err)
	require.False(t, covers)
}

// A bounded slice per call, so a tick never stalls on one long rebuild.
func TestBackfillAggregateEntityWorksInBatches(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	ds.backfillWindow = 2 * time.Hour

	t0 := seedCriticalFixture(t, tdb, ds)
	now := t0.Add(5 * time.Hour)
	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	aggID := filter.AggregateEntityID()

	// The first batch covers only the most recent hours, so a request for the
	// whole window must still take the per-CVE path.
	wrote, err := ds.BackfillAggregateEntity(ctx, api.MetricCVE, aggID, sourceIDs, now, retentionUnbounded)
	require.NoError(t, err)
	require.True(t, wrote)
	covers, err := ds.AggregateCoversFrom(ctx, api.MetricCVE, aggID, t0, retentionUnbounded)
	require.NoError(t, err)
	require.False(t, covers, "one batch cannot cover six hours of history")

	batches := 1 + backfillFully(t, ds, aggID, sourceIDs, now, retentionUnbounded)
	require.Greater(t, batches, 1, "a two-hour window cannot rebuild six hours in one batch")

	covers, err = ds.AggregateCoversFrom(ctx, api.MetricCVE, aggID, t0, retentionUnbounded)
	require.NoError(t, err)
	require.True(t, covers)

	// However many batches, the result must match a single pass.
	perCVE, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, nil, sourceIDs)
	require.NoError(t, err)
	agg, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, nil, []string{aggID})
	require.NoError(t, err)
	require.Equal(t, perCVE, agg)
	require.Equal(t, []int{3, 3, 4, 1, 1}, values(perCVE))
}

// The window is decoded a chunk of hours at a time, so a union that stays
// constant across a chunk boundary must not open a spurious row there.
func TestBackfillAggregateEntitySpansChunks(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	ds.backfillChunk = 2 * time.Hour

	t0 := seedCriticalFixture(t, tdb, ds)
	now := t0.Add(5 * time.Hour)
	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	aggID := filter.AggregateEntityID()
	backfillFully(t, ds, aggID, sourceIDs, now, retentionUnbounded)

	perCVE, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, nil, sourceIDs)
	require.NoError(t, err)
	agg, err := ds.GetSCDData(ctx, api.MetricCVE, t0, now, time.Hour, api.SampleStrategySnapshot, nil, []string{aggID})
	require.NoError(t, err)
	require.Equal(t, perCVE, agg)

	var rows int
	require.NoError(t, tdb.DB.GetContext(ctx, &rows,
		`SELECT COUNT(*) FROM host_scd_data WHERE dataset = ? AND entity_id = ?`, api.MetricCVE, aggID))
	// Six ticked hours collapse to three distinct unions: {1,2,3} through hour 2,
	// {1,2,3,4} at hour 3, {1} from hour 4. Chunking the walk must not split any
	// of them into extra rows.
	require.Equal(t, 3, rows)
}

// An aggregate whose filter matches nothing is an all-zero series the collector
// still has to store, and host_bitmap is NOT NULL.
func TestRecordBucketDataStoresEmptyBitmap(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, ds.RecordBucketData(ctx, api.MetricCVE, at, time.Hour, api.SampleStrategySnapshot,
		map[string]*roaring.Bitmap{"agg:empty": roaring.New()}))

	points, err := ds.GetSCDData(ctx, api.MetricCVE, at, at.Add(2*time.Hour), time.Hour,
		api.SampleStrategySnapshot, nil, []string{"agg:empty"})
	require.NoError(t, err)
	require.Equal(t, []int{0, 0}, values(points))
}

// End to end: backfill reconstructs the past and the tick's own write extends
// it. A gap or double-count would only show at the hour they share.
func TestCVEDatasetCollectAggregateMatchesPerCVEPath(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	seen := time.Now().UTC()
	hosts := seedHosts(t, tdb, []hostSeed{
		{seenTime: seen}, {seenTime: seen}, {seenTime: seen}, {seenTime: seen},
	})
	require.Len(t, hosts, 4)
	seedCVEMeta(t, tdb, "CVE-2026-0001", 9.5, 0.4, false)
	seedCVEMeta(t, tdb, "CVE-2026-0002", 10.0, 0.6, true)
	seedCVEMeta(t, tdb, "CVE-2026-0003", 3.0, 0.1, false)
	// Live state: the first critical CVE affects hosts 1-2, the second affects
	// hosts 2-4, and the low-severity one affects host 4 only.
	seedHostVulnSoftware(t, tdb, hosts[0], "Google Chrome", "apps", "CVE-2026-0001")
	seedHostVulnSoftware(t, tdb, hosts[1], "Google Chrome", "apps", "CVE-2026-0001", "CVE-2026-0002")
	seedHostVulnSoftware(t, tdb, hosts[2], "Google Chrome", "apps", "CVE-2026-0002")
	seedHostVulnSoftware(t, tdb, hosts[3], "Google Chrome", "apps", "CVE-2026-0002", "CVE-2026-0003")

	// Two ticks of history collected before any aggregate existed.
	t0 := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	h := func(n int) time.Time { return t0.Add(time.Duration(n) * time.Hour) }
	past := map[string][]uint{
		"CVE-2026-0001": {hosts[0], hosts[1]},
		"CVE-2026-0002": {hosts[1], hosts[2]},
		"CVE-2026-0003": {hosts[3]},
	}
	recordTick(t, ds, h(0), past)
	shrunk := map[string][]uint{
		"CVE-2026-0001": {hosts[0]},
		"CVE-2026-0002": {hosts[1], hosts[2]},
		"CVE-2026-0003": {hosts[3]},
	}
	recordTick(t, ds, h(1), shrunk)

	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	dataset := &chart.CVEDataset{PreaggregateFilters: []api.CVEFilter{filter}}
	require.NoError(t, dataset.Collect(ctx, ds, h(2), nil))

	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	aggID := filter.AggregateEntityID()

	perCVE, err := ds.GetSCDData(ctx, api.MetricCVE, t0, h(3), time.Hour, api.SampleStrategySnapshot, nil, sourceIDs)
	require.NoError(t, err)
	agg, err := ds.GetSCDData(ctx, api.MetricCVE, t0, h(3), time.Hour, api.SampleStrategySnapshot, nil, []string{aggID})
	require.NoError(t, err)

	require.Equal(t, perCVE, agg)
	// Labels start one bucket after startDate, so these read at hours 2, 3 and 4:
	// the backfilled state, then what this tick collected.
	require.Equal(t, []int{3, 4, 4}, values(perCVE))

	// A second tick must extend the series rather than reopen it.
	require.NoError(t, dataset.Collect(ctx, ds, h(3), nil))
	perCVE, err = ds.GetSCDData(ctx, api.MetricCVE, t0, h(4), time.Hour, api.SampleStrategySnapshot, nil, sourceIDs)
	require.NoError(t, err)
	agg, err = ds.GetSCDData(ctx, api.MetricCVE, t0, h(4), time.Hour, api.SampleStrategySnapshot, nil, []string{aggID})
	require.NoError(t, err)
	require.Equal(t, perCVE, agg)
	require.Equal(t, []int{3, 4, 4, 4}, values(perCVE))
}

// seedRetentionFixture records 60 days of 12-hourly ticks ending at `now`. One
// critical CVE is stable, so its open row is never cleaned and pins the
// dataset's oldest row far behind retention; the other churns every tick so
// its closed rows age out.
func seedRetentionFixture(t *testing.T, tdb *testutils.TestDB, ds *Datastore, now time.Time) {
	t.Helper()
	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-2026-0001")
	seedCVEMeta(t, tdb, "CVE-2026-0001", 9.5, 0.4, false)
	seedSoftware(t, tdb, "Google Chrome", "apps", "CVE-2026-0002")
	seedCVEMeta(t, tdb, "CVE-2026-0002", 9.0, 0.4, false)

	t0 := now.Add(-60 * 24 * time.Hour)
	for at := t0; !at.After(now); at = at.Add(12 * time.Hour) {
		churn := uint(at.Sub(t0)/(12*time.Hour))%3 + 3 //nolint:gosec // small fixture values
		recordTick(t, ds, at, map[string][]uint{"CVE-2026-0001": {1, 2}, "CVE-2026-0002": {churn}})
	}
}

// A completed series must stay completed after retention cleanup. Cleanup
// deletes the series' rows below the horizon but keeps the stable CVE's open
// row from day one, so "reach the dataset's oldest row" is never satisfied
// again. Without the horizon floor, every tick would rebuild a batch that the
// next cleanup deletes, burning the tick's whole backfill budget forever.
func TestBackfillAggregateEntityStopsAtRetentionHorizon(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Hour)
	seedRetentionFixture(t, tdb, ds, now)
	horizon := now.AddDate(0, 0, -api.RetentionDays)

	filter := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, filter)
	require.NoError(t, err)
	aggID := filter.AggregateEntityID()
	backfillFully(t, ds, aggID, sourceIDs, now, horizon)

	for range 3 {
		require.NoError(t, ds.CleanupSCDData(ctx, api.RetentionDays))
		wrote, err := ds.BackfillAggregateEntity(ctx, api.MetricCVE, aggID, sourceIDs, now, horizon)
		require.NoError(t, err)
		require.False(t, wrote, "series rebuilt a batch below the retention horizon")
	}
}

// The dashboard asks for one day more than retention keeps. Once a series
// reaches the horizon it holds everything any path could answer with, so it
// must qualify even though it does not reach the requested start.
func TestAggregateCoversFromAtRetentionHorizon(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	defer tdb.TruncateTables(t)
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Hour)
	seedRetentionFixture(t, tdb, ds, now)
	horizon := now.AddDate(0, 0, -api.RetentionDays)
	from := now.AddDate(0, 0, -(api.RetentionDays + 1))

	critical := api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
	sourceIDs, err := ds.ResolveCVEChartEntities(ctx, critical)
	require.NoError(t, err)
	criticalID := critical.AggregateEntityID()
	backfillFully(t, ds, criticalID, sourceIDs, now, horizon)
	require.NoError(t, ds.CleanupSCDData(ctx, api.RetentionDays))

	covers, err := ds.AggregateCoversFrom(ctx, api.MetricCVE, criticalID, from, horizon)
	require.NoError(t, err)
	require.True(t, covers, "a series reaching the retention horizon answers the dashboard's default window")

	// One that is still catching up does not.
	anySeverity := api.CVEFilter{}
	anyID := anySeverity.AggregateEntityID()
	wrote, err := ds.BackfillAggregateEntity(ctx, api.MetricCVE, anyID, sourceIDs, now, horizon)
	require.NoError(t, err)
	require.True(t, wrote)
	covers, err = ds.AggregateCoversFrom(ctx, api.MetricCVE, anyID, from, horizon)
	require.NoError(t, err)
	require.False(t, covers, "one batch does not reach the horizon")
}
