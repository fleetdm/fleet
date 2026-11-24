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

type scdSeedRow struct {
	entityID  string
	hostIDs   []uint
	validFrom time.Time
	validTo   time.Time
}

func TestGetSCDData(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	day := 24 * time.Hour
	// With startDate = b - 24h, bucket 0 is [b, b+24h) and bucket 1 is [b+24h, b+48h).
	b := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	before := b.Add(-time.Hour)
	open := scdOpenSentinel
	fourEntities := []scdSeedRow{
		{"CVE-A", []uint{1}, before, open},
		{"CVE-B", []uint{2}, before, open},
		{"CVE-C", []uint{3}, before, open},
		{"CVE-D", []uint{4}, before, open},
	}
	// A bucket per day starting at 18:30 UTC, as a UTC+5:30 client's midnight.
	tzStart := time.Date(2026, 4, 20, 18, 30, 0, 0, time.UTC)

	cases := []struct {
		name       string
		rows       []scdSeedRow
		strategy   api.SampleStrategy
		start, end time.Time
		bucketSize time.Duration
		mask       *roaring.Bitmap
		entityIDs  []string
		inListMax  int
		want       []int
	}{
		{
			name: "accumulate unions rows inside the bucket",
			rows: []scdSeedRow{
				{"", []uint{1, 2}, b.Add(2 * time.Hour), b.Add(3 * time.Hour)},
				{"", []uint{3}, b.Add(10 * time.Hour), b.Add(11 * time.Hour)},
				{"", []uint{2, 4}, b.Add(15 * time.Hour), b.Add(16 * time.Hour)},
			},
			strategy: api.SampleStrategyAccumulate, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{4},
		},
		{
			name: "accumulate unions across entities",
			rows: []scdSeedRow{
				{"slack", []uint{1, 2}, b, b.Add(time.Hour)},
				{"zoom", []uint{2, 3}, b, b.Add(time.Hour)},
				{"chrome", []uint{4}, b, b.Add(time.Hour)},
			},
			strategy: api.SampleStrategyAccumulate, start: b.Add(-time.Hour), end: b, bucketSize: time.Hour,
			want: []int{4},
		},
		{
			name: "snapshot takes the state at the bucket end, not the union",
			rows: []scdSeedRow{
				{"cve-A", []uint{1, 2, 3}, b, b.Add(14 * time.Hour)},
				{"cve-A", []uint{1, 2}, b.Add(14 * time.Hour), open},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{2},
		},
		{
			name: "snapshot unions the end state of each entity",
			rows: []scdSeedRow{
				{"cve-A", []uint{1, 2, 3}, b, b.Add(14 * time.Hour)},
				{"cve-A", []uint{1, 2}, b.Add(14 * time.Hour), open},
				{"cve-B", []uint{3, 4}, b.Add(5 * time.Hour), open},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{4},
		},
		{
			name: "snapshot excludes an entity closed mid-bucket",
			rows: []scdSeedRow{
				{"cve-A", []uint{1, 2, 3}, b, b.Add(14 * time.Hour)},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{0},
		},
		{
			name: "snapshot counts a row closed exactly at the bucket end",
			rows: []scdSeedRow{
				{"cve-A", []uint{1, 2}, b, b.Add(day)},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{2},
		},
		{
			name: "snapshot ignores a row that opens and closes inside one bucket",
			rows: []scdSeedRow{
				{"x", []uint{1, 2}, b.Add(2 * time.Hour), b.Add(5 * time.Hour)},
				{"y", []uint{3}, b.Add(20 * time.Hour), b.Add(30 * time.Hour)},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b.Add(day), bucketSize: day,
			want: []int{1, 0},
		},
		{
			name: "accumulate counts a row that opens and closes inside one bucket",
			rows: []scdSeedRow{
				{"x", []uint{1, 2}, b.Add(2 * time.Hour), b.Add(5 * time.Hour)},
				{"y", []uint{3}, b.Add(20 * time.Hour), b.Add(30 * time.Hour)},
			},
			strategy: api.SampleStrategyAccumulate, start: b.Add(-day), end: b.Add(day), bucketSize: day,
			want: []int{3, 1},
		},
		{
			name: "snapshot row boundaries on bucket ends",
			rows: []scdSeedRow{
				{"z", []uint{5, 6}, b, b.Add(day)},
				{"w", []uint{7}, b.Add(day), b.Add(2 * day)},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b.Add(day), bucketSize: day,
			want: []int{2, 1},
		},
		{
			name: "accumulate row boundaries on bucket ends",
			rows: []scdSeedRow{
				{"z", []uint{5, 6}, b, b.Add(day)},
				{"w", []uint{7}, b.Add(day), b.Add(2 * day)},
			},
			strategy: api.SampleStrategyAccumulate, start: b.Add(-day), end: b.Add(day), bucketSize: day,
			want: []int{2, 1},
		},
		{
			name: "accumulate across buckets with a row from before the window and an open row",
			rows: []scdSeedRow{
				{"p", []uint{1}, b.Add(-2 * day), b.Add(30 * time.Hour)},
				{"q", []uint{2}, b.Add(36 * time.Hour), open},
			},
			strategy: api.SampleStrategyAccumulate, start: b.Add(-day), end: b.Add(2 * day), bucketSize: day,
			want: []int{1, 2, 1},
		},
		{
			name: "snapshot excludes rows just outside the window",
			rows: []scdSeedRow{
				{"before", []uint{1}, b.Add(-2 * day), b},
				{"after", []uint{2}, b.Add(day), open},
				{"last", []uint{3}, b.Add(-2 * day), b.Add(day)},
			},
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{1},
		},
		{
			name: "accumulate excludes rows just outside the window",
			rows: []scdSeedRow{
				{"before", []uint{1}, b.Add(-2 * day), b},
				{"after", []uint{2}, b.Add(day), open},
				{"last", []uint{3}, b.Add(-2 * day), b.Add(day)},
			},
			strategy: api.SampleStrategyAccumulate, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{1},
		},
		{
			name: "half-hour time zone",
			rows: []scdSeedRow{
				{"cve-A", []uint{1, 2}, time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 22, 18, 0, 0, 0, time.UTC)},
				{"cve-A", []uint{1, 2, 3}, time.Date(2026, 4, 22, 18, 0, 0, 0, time.UTC), time.Date(2026, 4, 23, 19, 0, 0, 0, time.UTC)},
				{"cve-A", []uint{1}, time.Date(2026, 4, 23, 19, 0, 0, 0, time.UTC), open},
				{"cve-B", []uint{4}, time.Date(2026, 4, 23, 19, 0, 0, 0, time.UTC), open},
			},
			strategy: api.SampleStrategySnapshot, start: tzStart, end: tzStart.Add(3 * day), bucketSize: day,
			want: []int{3, 3, 2},
		},
		{
			name:     "nil entityIDs matches every entity",
			rows:     fourEntities,
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			want: []int{4},
		},
		{
			name:     "empty entityIDs matches nothing but still returns every bucket",
			rows:     fourEntities,
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b.Add(day), bucketSize: day,
			entityIDs: []string{},
			want:      []int{0, 0},
		},
		{
			name:     "entityIDs at or below the IN list limit",
			rows:     fourEntities,
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			entityIDs: []string{"CVE-A", "CVE-B", "CVE-C", "CVE-missing"},
			want:      []int{3},
		},
		{
			name:     "entityIDs exactly at the IN list limit",
			rows:     fourEntities,
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			entityIDs: []string{"CVE-A", "CVE-B", "CVE-C", "CVE-missing"},
			inListMax: 4,
			want:      []int{3},
		},
		{
			name:     "entityIDs above the IN list limit",
			rows:     fourEntities,
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			entityIDs: []string{"CVE-A", "CVE-B", "CVE-C", "CVE-missing"},
			inListMax: 2,
			want:      []int{3},
		},
		{
			name:     "filter mask",
			rows:     fourEntities,
			strategy: api.SampleStrategySnapshot, start: b.Add(-day), end: b, bucketSize: day,
			mask: chart.NewBitmap([]uint{1, 3, 99}),
			want: []int{2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			if c.inListMax > 0 {
				prev := scdEntityInListMax
				scdEntityInListMax = c.inListMax
				t.Cleanup(func() { scdEntityInListMax = prev })
			}
			for _, r := range c.rows {
				tdb.InsertSCDRowWithHostIDs(t, "cve", r.entityID, r.hostIDs, r.validFrom, r.validTo)
			}

			pts, err := ds.GetSCDData(t.Context(), "cve", c.start, c.end, c.bucketSize, c.strategy, c.mask, c.entityIDs)
			require.NoError(t, err)
			require.Len(t, pts, len(c.want))
			got := make([]int, len(pts))
			for i, p := range pts {
				got[i] = p.Value
				require.Equal(t, c.start.Add(time.Duration(i+1)*c.bucketSize), p.Timestamp, "bucket %d label", i)
			}
			require.Equal(t, c.want, got)
		})
	}
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
		{"RecordSnapshotClosesAcrossBatches", testRecordSnapshotClosesAcrossBatches},
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

func testRecordSnapshotClosesAcrossBatches(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()

	prev := scdCloseBatch
	scdCloseBatch = 2
	t.Cleanup(func() { scdCloseBatch = prev })

	bucket := time.Now().UTC().Truncate(time.Hour).Add(-time.Hour)
	open := map[string]*roaring.Bitmap{"e0": chart.NewBitmap([]uint{1}), "e1": chart.NewBitmap([]uint{1}), "e2": chart.NewBitmap([]uint{1})}
	require.NoError(t, ds.RecordBucketData(ctx, "cve", bucket, time.Hour, api.SampleStrategySnapshot, open))
	// An empty snapshot closes every open row: three closes span two batches.
	require.NoError(t, ds.RecordBucketData(ctx, "cve", bucket.Add(time.Hour), time.Hour, api.SampleStrategySnapshot, nil))

	var stillOpen int
	require.NoError(t, tdb.DB.GetContext(ctx, &stillOpen, `SELECT COUNT(*) FROM host_scd_data WHERE valid_to = ?`, scdOpenSentinel))
	assert.Zero(t, stillOpen)
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
