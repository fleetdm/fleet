package mysql

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/require"
)

// oldTick writes one hour in the format collected before the change storage existed.
func oldTick(t *testing.T, ds *Datastore, hour time.Time, sets hostSets) {
	t.Helper()
	require.NoError(t, ds.RecordBucketData(t.Context(), "cve", hour, time.Hour, api.SampleStrategySnapshot, bitmapsOf(sets)))
}

// convertAll runs conversion steps until none remains and returns how many ran.
func convertAll(t *testing.T, ds *Datastore, dataset string) int {
	t.Helper()
	for n := 1; ; n++ {
		more, err := ds.ConvertSCDHistory(t.Context(), dataset)
		require.NoError(t, err)
		if !more {
			return n
		}
		require.Less(t, n, 10_000, "conversion does not finish")
	}
}

func loadConvertCursor(t *testing.T, tdb *testutils.TestDB, dataset string) *time.Time {
	t.Helper()
	var cursor *time.Time
	require.NoError(t, tdb.DB.GetContext(t.Context(), &cursor,
		`SELECT convert_cursor FROM host_scd_change_state WHERE dataset = ?`, dataset))
	if cursor != nil {
		*cursor = cursor.UTC()
	}
	return cursor
}

func setConvertCursor(t *testing.T, tdb *testutils.TestDB, dataset string, cursor *time.Time) {
	t.Helper()
	_, err := tdb.DB.ExecContext(t.Context(), `UPDATE host_scd_change_state SET convert_cursor = ? WHERE dataset = ?`, cursor, dataset)
	require.NoError(t, err)
}

// changesFromOn returns the changes at or after from.
func changesFromOn(changes []changeRec, from time.Time) []changeRec {
	out := []changeRec{}
	for _, c := range changes {
		if !c.at.Before(from) {
			out = append(out, c)
		}
	}
	return out
}

// requireNoHost fails if any change or current set of the dataset holds host.
func requireNoHost(t *testing.T, tdb *testutils.TestDB, dataset string, host uint) {
	t.Helper()
	for _, c := range loadChanges(t, tdb, dataset) {
		require.NotContains(t, c.added, host, "%+v", c)
		require.NotContains(t, c.removed, host, "%+v", c)
	}
	for e, ids := range loadCurrent(t, tdb, dataset) {
		require.NotContains(t, ids, host, e)
	}
}

func TestConvertSCDHistory(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"explicit history: gaps, an entity gone before the change storage, rows open from before the start", testConvertSCDHistoryExplicit},
		{"random histories replay to the snapshot read; one step and several give the same changes", testConvertSCDHistoryRandom},
		{"a repeated step changes nothing", testConvertSCDHistoryRepeatedStep},
		{"a new install completes on the first step", testConvertSCDHistoryNewInstall},
		{"history that cleanup partly removed", testConvertSCDHistoryAfterCleanup},
		{"a fleet scrub between two steps", testConvertSCDHistoryScrubBetweenSteps},
		{"an instant with more changes than one row holds", testConvertSCDHistoryManyChanges},
		{"a gap with nothing live, a zero-length row, overlapping rows of one entity", testConvertSCDHistoryOddRows},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

// writeExplicitOldHistory writes host_scd_data history before changesFrom, then the first two collections of the
// change storage at and after it. start is changesFrom - 32 days, where conversion starts.
func writeExplicitOldHistory(t *testing.T, ds *Datastore, changesFrom time.Time) (start time.Time) {
	start = changesFrom.AddDate(0, 0, -32)
	day := func(n int) time.Time { return start.AddDate(0, 0, n) }
	for _, tick := range []struct {
		at   time.Time
		sets hostSets
	}{
		{day(-5), hostSets{"CVE-OPEN": {1, 2}}},
		{day(-2), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6}}},
		{start.Add(10 * time.Hour), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6, 7}}},
		{day(1), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6, 7}, "CVE-GAP": {3}}},
		{day(2), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6, 7}}},
		{day(3), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6, 7}, "CVE-GAP": {3, 4}}},
		{day(4), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6, 7}, "CVE-GAP": {3, 4}, "CVE-GONE": {5}}},
		{day(6), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {6, 7}, "CVE-GAP": {3, 4}}},
		{changesFrom.Add(-time.Hour), hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {7, 9}, "CVE-GAP": {3, 4}}},
	} {
		oldTick(t, ds, tick.at, tick.sets)
	}
	collectTick(t, ds, "cve", changesFrom, hostSets{"CVE-OPEN": {1, 2}, "CVE-CHG": {7, 9}, "CVE-GAP": {3, 4, 8}})
	collectTick(t, ds, "cve", changesFrom.Add(2*time.Hour), hostSets{"CVE-OPEN": {1}, "CVE-CHG": {7, 9}, "CVE-GAP": {3, 4, 8}})
	return start
}

// requireReplayMatchesSnapshotWindows compares the replay read with the snapshot read over [start, end) for
// hourly, 3-hourly and daily buckets, whole and half-hour time zones, with and without a mask and an entity
// list, for every bucket that ends at or after from.
func requireReplayMatchesSnapshotWindows(t *testing.T, ds *Datastore, start, end, from time.Time, mask *roaring.Bitmap, entityIDs []string) {
	t.Helper()
	for _, size := range []time.Duration{time.Hour, 3 * time.Hour, 24 * time.Hour} {
		for _, shift := range []time.Duration{0, -30 * time.Minute} {
			s := start.Add(shift)
			n := int(end.Sub(start)/size) + 1
			for _, m := range []*roaring.Bitmap{nil, mask} {
				for _, ids := range [][]string{nil, entityIDs} {
					requireReplayMatchesSnapshot(t, ds, replayQuery{start: s, end: s.Add(time.Duration(n) * size), size: size, mask: m, entityIDs: ids}, from)
				}
			}
		}
	}
}

func testConvertSCDHistoryExplicit(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	changesFrom := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	start := writeExplicitOldHistory(t, ds, changesFrom)
	collected := loadChanges(t, tdb, "cve")
	current := loadCurrent(t, tdb, "cve")

	convertAll(t, ds, "cve")

	day := func(n int) time.Time { return start.AddDate(0, 0, n) }
	// Changes before start are not written: CVE-OPEN and CVE-CHG start before it, and are held by the state
	// replay starts from.
	want := []changeRec{
		{at: start.Add(10 * time.Hour), entity: "CVE-CHG", added: []uint{7}, removed: []uint{}},
		{at: day(1), entity: "CVE-GAP", added: []uint{3}, removed: []uint{}},
		{at: day(2), entity: "CVE-GAP", added: []uint{}, removed: []uint{3}},
		{at: day(3), entity: "CVE-GAP", added: []uint{3, 4}, removed: []uint{}},
		{at: day(4), entity: "CVE-GONE", added: []uint{5}, removed: []uint{}},
		{at: day(6), entity: "CVE-GONE", added: []uint{}, removed: []uint{5}},
		{at: changesFrom.Add(-time.Hour), entity: "CVE-CHG", added: []uint{9}, removed: []uint{6}},
	}
	require.Equal(t, append(want, collected...), loadChanges(t, tdb, "cve"))
	require.Equal(t, current, loadCurrent(t, tdb, "cve"), "conversion does not touch the current sets")

	gotFrom, coveredFrom, found := loadChangeState(t, tdb, "cve")
	require.True(t, found)
	require.Equal(t, changesFrom, gotFrom)
	require.Equal(t, start, coveredFrom)
	require.Equal(t, &changesFrom, loadConvertCursor(t, tdb, "cve"))

	// Buckets that end before start hold the state at start.
	requireReplayMatchesSnapshotWindows(t, ds, day(-3), changesFrom.Add(24*time.Hour), start,
		chart.NewBitmap([]uint{2, 3, 5, 9}), []string{"CVE-GAP", "CVE-GONE"})

	more, err := ds.ConvertSCDHistory(t.Context(), "cve")
	require.NoError(t, err)
	require.False(t, more)
	require.Equal(t, append(want, collected...), loadChanges(t, tdb, "cve"), "a completed conversion writes nothing")
}

// resetConversion removes the converted changes and the conversion's progress.
func resetConversion(t *testing.T, tdb *testutils.TestDB, dataset string) {
	t.Helper()
	_, err := tdb.DB.ExecContext(t.Context(), `DELETE c FROM host_scd_changes c JOIN host_scd_change_state s USING (dataset)
		WHERE c.dataset = ? AND c.changed_at < s.changes_from`, dataset)
	require.NoError(t, err)
	_, err = tdb.DB.ExecContext(t.Context(), `UPDATE host_scd_change_state SET convert_cursor = NULL, covered_from = changes_from
		WHERE dataset = ?`, dataset)
	require.NoError(t, err)
}

func testConvertSCDHistoryRandom(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	// The row size limits are left alone: other tests of the package run in parallel and change them.
	defer func(step int) { scdConvertStepHours = step }(scdConvertStepHours)

	const hours = 60
	h0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for i, oldHours := range []int{7, 20, 45} {
		t.Run(fmt.Sprintf("seed %d, %d hours before the change storage", i, oldHours), func(t *testing.T) {
			defer tdb.TruncateTables(t)
			rng := rand.New(rand.NewPCG(uint64(i), 589284)) //nolint:gosec // deterministic test data
			randomHistory(t, ds, rng, h0, hours, oldHours)
			changesFrom, _, found := loadChangeState(t, tdb, "cve")
			require.True(t, found)
			collected := loadChanges(t, tdb, "cve")
			current := loadCurrent(t, tdb, "cve")

			scdConvertStepHours = 5
			convertAll(t, ds, "cve")
			changes := loadChanges(t, tdb, "cve")
			require.Equal(t, collected, changesFromOn(changes, changesFrom), "no change at or after changes_from is written")
			require.Equal(t, current, loadCurrent(t, tdb, "cve"))
			_, coveredFrom, _ := loadChangeState(t, tdb, "cve")
			require.Equal(t, changesFrom.AddDate(0, 0, -32), coveredFrom)

			requireReplayMatchesSnapshotWindows(t, ds, h0.Add(-48*time.Hour), h0.Add((hours+24)*time.Hour), time.Time{},
				chart.NewBitmap([]uint{1, 3, 5, 7, 9, 11, 70001, 99999}), []string{"CVE-1", "CVE-4", "CVE-missing"})

			resetConversion(t, tdb, "cve")
			scdConvertStepHours = 40 * 24
			require.Equal(t, 1, convertAll(t, ds, "cve"))
			require.Equal(t, changes, loadChanges(t, tdb, "cve"), "one step gives the same changes as several")
		})
	}
}

func testConvertSCDHistoryRepeatedStep(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	defer func(step int) { scdConvertStepHours = step }(scdConvertStepHours)
	scdConvertStepHours = 6

	h0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	randomHistory(t, ds, rand.New(rand.NewPCG(7, 58928)), h0, 40, 30) //nolint:gosec // deterministic test data

	for range 2 {
		more, err := ds.ConvertSCDHistory(t.Context(), "cve")
		require.NoError(t, err)
		require.True(t, more)
	}
	afterTwo := loadChanges(t, tdb, "cve")
	cursorTwo := loadConvertCursor(t, tdb, "cve")
	require.NotNil(t, cursorTwo)

	// The second step again, as after a crash before its cursor was saved.
	back := cursorTwo.Add(-6 * time.Hour)
	setConvertCursor(t, tdb, "cve", &back)
	more, err := ds.ConvertSCDHistory(t.Context(), "cve")
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, afterTwo, loadChanges(t, tdb, "cve"))
	require.Equal(t, cursorTwo, loadConvertCursor(t, tdb, "cve"))

	// Starting over gives the same changes too.
	convertAll(t, ds, "cve")
	done := loadChanges(t, tdb, "cve")
	setConvertCursor(t, tdb, "cve", nil)
	convertAll(t, ds, "cve")
	require.Equal(t, done, loadChanges(t, tdb, "cve"))
}

func testConvertSCDHistoryNewInstall(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	more, err := ds.ConvertSCDHistory(t.Context(), "cve")
	require.NoError(t, err)
	require.False(t, more, "nothing to convert before the change storage starts")
	_, _, found := loadChangeState(t, tdb, "cve")
	require.False(t, found)

	h := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}})
	before := loadChanges(t, tdb, "cve")

	require.Equal(t, 1, convertAll(t, ds, "cve"))
	require.Equal(t, before, loadChanges(t, tdb, "cve"))
	_, coveredFrom, _ := loadChangeState(t, tdb, "cve")
	require.Equal(t, h.AddDate(0, 0, -32), coveredFrom)
	require.Equal(t, &h, loadConvertCursor(t, tdb, "cve"))
}

func testConvertSCDHistoryAfterCleanup(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	now := time.Now().UTC().Truncate(time.Hour)
	changesFrom := now.Add(-24 * time.Hour)
	rng := rand.New(rand.NewPCG(3, 58928)) //nolint:gosec // deterministic test data
	randomSets := func() hostSets {
		sets := hostSets{}
		for e := range 5 {
			if rng.IntN(4) == 0 {
				continue
			}
			ids := []uint{uint(1 + rng.IntN(8))}
			for host := uint(1); host <= 8; host++ {
				if rng.IntN(3) == 0 {
					ids = append(ids, host)
				}
			}
			sets[fmt.Sprintf("CVE-%d", e)] = ids
		}
		return sets
	}
	for at := now.AddDate(0, 0, -36); at.Before(changesFrom); at = at.Add(9 * time.Hour) {
		oldTick(t, ds, at, randomSets())
	}
	require.NoError(t, ds.CleanupSCDData(t.Context(), 30))
	collectTick(t, ds, "cve", changesFrom, randomSets())
	collectTick(t, ds, "cve", changesFrom.Add(5*time.Hour), randomSets())

	convertAll(t, ds, "cve")

	// Cleanup ran a little after now, so a row that ended at now - 30 days may be gone.
	retained := now.AddDate(0, 0, -30).Add(time.Hour)
	requireReplayMatchesSnapshotWindows(t, ds, now.AddDate(0, 0, -31), now.Add(24*time.Hour), retained,
		chart.NewBitmap([]uint{2, 4, 6}), []string{"CVE-0", "CVE-3"})
}

func testConvertSCDHistoryScrubBetweenSteps(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	changesFrom := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	start := writeExplicitOldHistory(t, ds, changesFrom)

	// The first step converts the first day, which adds host 7 to CVE-CHG.
	more, err := ds.ConvertSCDHistory(t.Context(), "cve")
	require.NoError(t, err)
	require.True(t, more)
	require.Contains(t, loadChanges(t, tdb, "cve"),
		changeRec{at: start.Add(10 * time.Hour), entity: "CVE-CHG", added: []uint{7}, removed: []uint{}})

	require.NoError(t, ds.ApplyScrubMaskToDataset(t.Context(), "cve", chart.NewBitmap([]uint{7}), 0))
	convertAll(t, ds, "cve")

	requireNoHost(t, tdb, "cve", 7)
	requireReplayMatchesSnapshotWindows(t, ds, start, changesFrom.Add(24*time.Hour), start,
		chart.NewBitmap([]uint{6, 7, 9}), []string{"CVE-CHG"})
}

func testConvertSCDHistoryManyChanges(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	changesFrom := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	at := changesFrom.Add(-48 * time.Hour)
	sets := hostSets{}
	// One more than the default scdChangeBatchEntities.
	for i := range 1001 {
		sets[fmt.Sprintf("CVE-%05d", i)] = []uint{uint(1 + i%5)}
	}
	oldTick(t, ds, at, sets)
	collectTick(t, ds, "cve", changesFrom, sets)

	convertAll(t, ds, "cve")

	var rows int
	require.NoError(t, tdb.DB.GetContext(t.Context(), &rows,
		`SELECT COUNT(*) FROM host_scd_changes WHERE dataset = 'cve' AND changed_at = ?`, at))
	require.GreaterOrEqual(t, rows, 2)
	require.Len(t, loadChanges(t, tdb, "cve"), len(sets))
	requireReplayMatchesSnapshot(t, ds, replayQuery{start: at.Add(-3 * time.Hour), end: at.Add(3 * time.Hour), size: time.Hour}, time.Time{})
	requireReplayMatchesSnapshot(t, ds, replayQuery{start: at.Add(-3 * time.Hour), end: at.Add(3 * time.Hour), size: time.Hour,
		entityIDs: []string{"CVE-00003", "CVE-01000"}}, time.Time{})
}

func testConvertSCDHistoryOddRows(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	changesFrom := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	start := changesFrom.AddDate(0, 0, -32)
	at := func(hours int) time.Time { return start.Add(time.Duration(hours) * time.Hour) }
	row := func(entity string, hosts []uint, from, to time.Time) {
		tdb.InsertSCDRowWithBlob(t, "cve", entity, chart.BitmapToBlob(chart.NewBitmap(hosts)), from, to)
	}
	row("CVE-A", []uint{1, 2}, at(2), at(5))
	// Nothing is live from hour 5 to hour 72, across two step boundaries.
	row("CVE-B", []uint{3}, at(72), scdOpenSentinel)
	row("CVE-B", []uint{9}, at(96), at(96)) // zero length: never counted
	row("CVE-C", []uint{4, 5}, at(74), at(82))
	row("CVE-C", []uint{5, 6}, at(77), at(92)) // overlaps the row before
	collectTick(t, ds, "cve", changesFrom, hostSets{"CVE-B": {3}})
	collected := loadChanges(t, tdb, "cve")

	convertAll(t, ds, "cve")

	want := []changeRec{
		{at: at(2), entity: "CVE-A", added: []uint{1, 2}, removed: []uint{}},
		{at: at(5), entity: "CVE-A", added: []uint{}, removed: []uint{1, 2}},
		{at: at(72), entity: "CVE-B", added: []uint{3}, removed: []uint{}},
		{at: at(74), entity: "CVE-C", added: []uint{4, 5}, removed: []uint{}},
		{at: at(77), entity: "CVE-C", added: []uint{6}, removed: []uint{}},
		{at: at(82), entity: "CVE-C", added: []uint{}, removed: []uint{4}},
		{at: at(92), entity: "CVE-C", added: []uint{}, removed: []uint{5, 6}},
	}
	require.Equal(t, append(want, collected...), loadChanges(t, tdb, "cve"))
	requireReplayMatchesSnapshotWindows(t, ds, start, changesFrom.Add(24*time.Hour), start,
		chart.NewBitmap([]uint{2, 5, 9}), []string{"CVE-B", "CVE-C"})
}

// waitForLockWaiter waits until a session of this database waits for a named lock, or until done is closed.
func waitForLockWaiter(t *testing.T, tdb *testutils.TestDB, done <-chan struct{}) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-done:
			return
		default:
		}
		var n int
		require.NoError(t, tdb.DB.GetContext(t.Context(), &n,
			`SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE DB = DATABASE() AND STATE = 'User lock'`))
		if n > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no session waits for the lock")
}

func TestSCDChangeLock(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"collection and conversion skip while the lock is held", testSCDChangeLockBusy},
		{"a fleet scrub waits for a collection", testSCDChangeLockScrubWaitsForCollection},
		{"a global scrub waits before deleting the change storage", testSCDChangeLockGlobalScrubWaits},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

// holdSCDChangeLock holds the dataset's change storage lock until the returned release is called.
func holdSCDChangeLock(t *testing.T, ds *Datastore, dataset string) (release func()) {
	t.Helper()
	held, stop, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- ds.withSCDChangeLock(t.Context(), dataset, true, func() error {
			close(held)
			<-stop
			return nil
		})
	}()
	select {
	case <-held:
	case err := <-done:
		t.Fatalf("lock not acquired: %v", err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			close(stop)
			require.NoError(t, <-done)
		})
	}
	// A failed test must not leave the lock held for the next one.
	t.Cleanup(release)
	return release
}

func testSCDChangeLockBusy(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1}})
	changes, current := loadChanges(t, tdb, "cve"), loadCurrent(t, tdb, "cve")

	release := holdSCDChangeLock(t, ds, "cve")
	err := ds.RecordSCDChanges(t.Context(), "cve", h.Add(time.Hour), bitmapsOf(hostSets{"CVE-A": {1, 2}}))
	require.ErrorIs(t, err, api.ErrChangeStorageBusy)
	_, err = ds.ConvertSCDHistory(t.Context(), "cve")
	require.ErrorIs(t, err, api.ErrChangeStorageBusy)
	require.Equal(t, changes, loadChanges(t, tdb, "cve"))
	require.Equal(t, current, loadCurrent(t, tdb, "cve"))
	require.Nil(t, loadConvertCursor(t, tdb, "cve"))

	require.NoError(t, ds.RecordSCDChanges(t.Context(), "other", h, bitmapsOf(hostSets{"X": {1}})), "each dataset has its own lock")
	release()

	require.NoError(t, ds.RecordSCDChanges(t.Context(), "cve", h.Add(time.Hour), bitmapsOf(hostSets{"CVE-A": {1, 2}})))
	require.Equal(t, hostSets{"CVE-A": {1, 2}}, loadCurrent(t, tdb, "cve"))
}

func testSCDChangeLockScrubWaitsForCollection(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 3}, "CVE-B": {3, 4}})

	// The scrub starts after the collection read the current sets and before it writes.
	scrubbed := make(chan struct{})
	var scrubErr error
	defer func() { ds.testHookSCDChangesBeforeWrite = nil }()
	ds.testHookSCDChangesBeforeWrite = func() {
		ds.testHookSCDChangesBeforeWrite = nil
		go func() {
			defer close(scrubbed)
			scrubErr = ds.ApplyScrubMaskToDataset(t.Context(), "cve", chart.NewBitmap([]uint{3}), 0)
		}()
		waitForLockWaiter(t, tdb, scrubbed)
	}
	// The collection still sees host 3, as when its fleet was disabled after the collection started.
	collectTick(t, ds, "cve", h.Add(time.Hour), hostSets{"CVE-A": {1, 3, 5}, "CVE-B": {3, 4, 6}})
	<-scrubbed
	require.NoError(t, scrubErr)

	require.Equal(t, hostSets{"CVE-A": {1, 5}, "CVE-B": {4, 6}}, loadCurrent(t, tdb, "cve"))
	requireNoHost(t, tdb, "cve", 3)
	requireReplayMatchesSnapshot(t, ds, replayQuery{start: h.Add(-time.Hour), end: h.Add(3 * time.Hour), size: time.Hour}, time.Time{})
}

func testSCDChangeLockGlobalScrubWaits(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1}})

	release := holdSCDChangeLock(t, ds, "cve")
	deleted := make(chan struct{})
	var deleteErr error
	go func() {
		defer close(deleted)
		deleteErr = ds.DeleteAllForDataset(t.Context(), "cve", 0)
	}()
	waitForLockWaiter(t, tdb, deleted)
	_, _, found := loadChangeState(t, tdb, "cve")
	require.True(t, found, "the change storage is not deleted while the lock is held")

	release()
	<-deleted
	require.NoError(t, deleteErr)
	require.Equal(t, map[string]int{"host_scd_data": 0, "host_scd_entities": 0, "host_scd_current": 0, "host_scd_changes": 0, "host_scd_change_state": 0},
		countDatasetRows(t, tdb, "cve"))
}
