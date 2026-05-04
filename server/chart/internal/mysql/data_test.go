package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateBucketAccumulate(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// Three accumulate rows within the bucket, each observed during a different
	// hour. Accumulate semantics = union of all overlapping rows.
	rows := []scdRow{
		{EntityID: "", HostBitmap: chart.HostIDsToBlob([]uint{1, 2}), ValidFrom: bucketStart.Add(2 * time.Hour), ValidTo: bucketStart.Add(3 * time.Hour)},
		{EntityID: "", HostBitmap: chart.HostIDsToBlob([]uint{3}), ValidFrom: bucketStart.Add(10 * time.Hour), ValidTo: bucketStart.Add(11 * time.Hour)},
		{EntityID: "", HostBitmap: chart.HostIDsToBlob([]uint{2, 4}), ValidFrom: bucketStart.Add(15 * time.Hour), ValidTo: bucketStart.Add(16 * time.Hour)},
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategyAccumulate)
	assert.Equal(t, 4, chart.BlobPopcount(got), "union of {1,2}, {3}, {2,4} = {1,2,3,4}")
}

func TestAggregateBucketAccumulateMultiEntity(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 14, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(time.Hour)

	// Future-style multi-entity accumulate dataset (e.g. software usage):
	// entity = software name; bitmap = hosts that used that software this hour.
	// Bucket value = distinct hosts using any tracked software during the hour.
	rows := []scdRow{
		{EntityID: "slack", HostBitmap: chart.HostIDsToBlob([]uint{1, 2}), ValidFrom: bucketStart, ValidTo: bucketEnd},
		{EntityID: "zoom", HostBitmap: chart.HostIDsToBlob([]uint{2, 3}), ValidFrom: bucketStart, ValidTo: bucketEnd},
		{EntityID: "chrome", HostBitmap: chart.HostIDsToBlob([]uint{4}), ValidFrom: bucketStart, ValidTo: bucketEnd},
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategyAccumulate)
	assert.Equal(t, 4, chart.BlobPopcount(got), "union across entities = {1,2,3,4}")
}

func TestAggregateBucketSnapshotEndOfBucket(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// One entity "cve-A" changed state mid-bucket: affected hosts were {1,2,3}
	// from hr 0 to hr 14, then {1,2} from hr 14 onward (H3 patched).
	// End-of-bucket semantics should return only the *latest* state, not the OR.
	rows := []scdRow{
		{EntityID: "cve-A", HostBitmap: chart.HostIDsToBlob([]uint{1, 2, 3}), ValidFrom: bucketStart, ValidTo: bucketStart.Add(14 * time.Hour)},
		{EntityID: "cve-A", HostBitmap: chart.HostIDsToBlob([]uint{1, 2}), ValidFrom: bucketStart.Add(14 * time.Hour), ValidTo: time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)},
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, 2, chart.BlobPopcount(got), "end-of-bucket state is {1,2}, not union {1,2,3}")
}

func TestAggregateBucketSnapshotMultipleEntities(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	sentinel := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

	// Two entities, each with an end-of-bucket state; snapshot returns OR across
	// entities of each's latest row.
	rows := []scdRow{
		// cve-A: latest state {1,2}
		{EntityID: "cve-A", HostBitmap: chart.HostIDsToBlob([]uint{1, 2, 3}), ValidFrom: bucketStart, ValidTo: bucketStart.Add(14 * time.Hour)},
		{EntityID: "cve-A", HostBitmap: chart.HostIDsToBlob([]uint{1, 2}), ValidFrom: bucketStart.Add(14 * time.Hour), ValidTo: sentinel},
		// cve-B: latest state {3,4}
		{EntityID: "cve-B", HostBitmap: chart.HostIDsToBlob([]uint{3, 4}), ValidFrom: bucketStart.Add(5 * time.Hour), ValidTo: sentinel},
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, 4, chart.BlobPopcount(got), "union of cve-A end-state {1,2} and cve-B end-state {3,4}")
}

func TestAggregateBucketSnapshotEntityDisappears(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// Entity was active early in bucket but its row was closed mid-bucket with
	// no replacement (entity disappeared — e.g., last affected host patched).
	// End-of-bucket semantics exclude it: no row is active at bucketEnd.
	rows := []scdRow{
		{EntityID: "cve-A", HostBitmap: chart.HostIDsToBlob([]uint{1, 2, 3}), ValidFrom: bucketStart, ValidTo: bucketStart.Add(14 * time.Hour)},
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, 0, chart.BlobPopcount(got), "entity closed mid-bucket is absent at bucketEnd")
}

func TestAggregateBucketSnapshotRowClosedExactlyAtBucketEnd(t *testing.T) {
	bucketStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	bucketEnd := bucketStart.Add(24 * time.Hour)

	// Row's valid_to == bucketEnd. The row represents state up to (but not
	// including) bucketEnd — i.e., the state just before the bucket ends.
	// That's exactly what end-of-bucket semantics should pick.
	rows := []scdRow{
		{EntityID: "cve-A", HostBitmap: chart.HostIDsToBlob([]uint{1, 2}), ValidFrom: bucketStart, ValidTo: bucketEnd},
	}

	got := aggregateBucket(rows, bucketStart, bucketEnd, api.SampleStrategySnapshot)
	assert.Equal(t, 2, chart.BlobPopcount(got), "row whose valid_to equals bucketEnd covers bucketEnd-ε")
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
