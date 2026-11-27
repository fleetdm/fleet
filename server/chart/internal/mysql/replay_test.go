package mysql

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/require"
)

type replayQuery struct {
	start, end time.Time
	size       time.Duration
	mask       *roaring.Bitmap
	entityIDs  []string
}

func (q replayQuery) String() string {
	mask := "nil"
	if q.mask != nil {
		mask = fmt.Sprint(chart.BitmapToHostIDs(q.mask))
	}
	ids := "nil"
	if q.entityIDs != nil {
		ids = fmt.Sprint(q.entityIDs)
	}
	return fmt.Sprintf("start=%s end=%s size=%s mask=%s entities=%s", q.start.Format(time.RFC3339), q.end.Format(time.RFC3339), q.size, mask, ids)
}

// requireReplayMatchesSnapshot checks that the replay read returns the same labels as the snapshot read of
// host_scd_data, and the same values for every bucket that ends at or after from. It returns the replay values.
func requireReplayMatchesSnapshot(t *testing.T, ds *Datastore, q replayQuery, from time.Time) []int {
	t.Helper()
	want, err := ds.GetSCDData(t.Context(), "cve", q.start, q.end, q.size, api.SampleStrategySnapshot, q.mask, q.entityIDs)
	require.NoError(t, err)
	got, err := ds.ReplaySCDChanges(t.Context(), "cve", q.start, q.end, q.size, q.mask, q.entityIDs)
	require.NoError(t, err)
	// Values of buckets that end before from are not compared.
	compared := func(pts []api.DataPoint) []api.DataPoint {
		out := make([]api.DataPoint, 0, len(pts))
		for _, p := range pts {
			if p.Timestamp.Add(q.size).Before(from) {
				p.Value = -1
			}
			out = append(out, p)
		}
		return out
	}
	require.Equal(t, compared(want), compared(got), "%s", q)
	values := make([]int, 0, len(got))
	for _, p := range got {
		values = append(values, p.Value)
	}
	return values
}

func TestReplaySCDChanges(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"explicit history", testReplaySCDChangesExplicit},
		{"random histories match the snapshot read", testReplaySCDChangesRandom},
		{"empty inputs and missing data", testReplaySCDChangesEmpty},
		{"covered from", testSCDChangesCoveredFrom},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

func testReplaySCDChangesExplicit(t *testing.T, _ *testutils.TestDB, ds *Datastore) {
	h0 := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	at := func(hours int) time.Time { return h0.Add(time.Duration(hours) * time.Hour) }
	collectTick(t, ds, "cve", at(2), hostSets{"CVE-A": {1, 2}, "CVE-B": {3}})
	collectTick(t, ds, "cve", at(5), hostSets{"CVE-A": {1, 2, 3, 7}, "CVE-B": {3}})
	collectTick(t, ds, "cve", at(9), hostSets{"CVE-B": {3, 4}})
	collectTick(t, ds, "cve", at(9), hostSets{"CVE-B": {3, 4}, "CVE-C": {5, 6, 8}}) // C appears in a re-run
	collectTick(t, ds, "cve", at(20), hostSets{"CVE-A": {1}, "CVE-B": {4}, "CVE-C": {5}})
	collectTick(t, ds, "cve", at(30), hostSets{"CVE-A": {1, 9}, "CVE-C": {5, 6}})

	// Twelve 3-hour buckets labeled h0, h0+3h, ..., ending h0+3h, ..., h0+36h. A change at hour t counts for
	// bucket ends after t.
	q := replayQuery{start: at(-3), end: at(33), size: 3 * time.Hour}
	require.Equal(t, []int{3, 4, 4, 5, 5, 5, 3, 3, 3, 3, 4, 4}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}))

	q.entityIDs = []string{"CVE-A"}
	require.Equal(t, []int{2, 4, 4, 0, 0, 0, 1, 1, 1, 1, 2, 2}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}))

	q.entityIDs = nil
	q.mask = chart.NewBitmap([]uint{1, 5, 9})
	require.Equal(t, []int{1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}))

	// A window that ends before the newest changes undoes them without recording a bucket for them.
	q = replayQuery{start: at(-1), end: at(10), size: time.Hour}
	require.Equal(t, []int{0, 0, 3, 3, 3, 4, 4, 4, 4, 5, 5}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}))
}

// randomHistory writes hours of random host sets for a few entities, some hours several times. The first
// oldHours hours are written to host_scd_data only, as before the change storage existed.
func randomHistory(t *testing.T, ds *Datastore, rng *rand.Rand, h0 time.Time, hours, oldHours int) {
	t.Helper()
	for i := range hours {
		h := h0.Add(time.Duration(i) * time.Hour)
		runs := 1 + rng.IntN(3)*(rng.IntN(4)/3) // usually one run, sometimes up to three
		if i < oldHours {
			runs = 1
		}
		// recordSnapshot loses an entity that drops out and returns within the same hour's runs, so an entity
		// that drops out stays out for the rest of the hour.
		dropped := map[string]struct{}{}
		var prev hostSets
		for range runs {
			sets := hostSets{}
			for e := range 8 {
				name := fmt.Sprintf("CVE-%d", e)
				_, out := dropped[name]
				if out || rng.IntN(5) == 0 {
					if _, ok := prev[name]; ok {
						dropped[name] = struct{}{}
					}
					continue
				}
				ids := []uint{uint(1 + rng.IntN(12))}
				for host := uint(1); host <= 12; host++ {
					if rng.IntN(3) == 0 {
						ids = append(ids, host)
					}
				}
				if rng.IntN(4) == 0 {
					ids = append(ids, 70000+uint(rng.IntN(3))) // a second roaring container
				}
				sets[name] = ids
			}
			if i < oldHours {
				require.NoError(t, ds.RecordBucketData(t.Context(), "cve", h, time.Hour, api.SampleStrategySnapshot, bitmapsOf(sets)))
			} else {
				collectTick(t, ds, "cve", h, sets)
			}
			prev = sets
		}
	}
}

func testReplaySCDChangesRandom(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	defer func(changes, current, entities int) {
		scdChangeRowBytes, scdCurrentWriteBytes, scdChangeBatchEntities = changes, current, entities
	}(scdChangeRowBytes, scdCurrentWriteBytes, scdChangeBatchEntities)
	scdChangeRowBytes, scdCurrentWriteBytes, scdChangeBatchEntities = 200, 300, 2

	const hours = 60
	h0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	masks := []*roaring.Bitmap{nil, chart.NewBitmap([]uint{1, 3, 5, 7, 9, 11, 70001, 99999})}
	entityLists := []struct {
		ids       []string
		inListMax int
	}{
		{ids: nil},
		{ids: []string{"CVE-1", "CVE-4", "CVE-missing"}},
		{ids: []string{"CVE-0", "CVE-2", "CVE-3", "CVE-5", "CVE-7"}, inListMax: 2},
	}

	for i, oldHours := range []int{0, 0, 7, 20} {
		t.Run(fmt.Sprintf("seed %d, %d hours before the change storage", i, oldHours), func(t *testing.T) {
			defer tdb.TruncateTables(t)
			rng := rand.New(rand.NewPCG(uint64(i), 58928)) //nolint:gosec // deterministic test data
			randomHistory(t, ds, rng, h0, hours, oldHours)

			// Before the change storage starts, replay holds the state it started from, not the history.
			var from time.Time
			if oldHours > 0 {
				changesFrom, _, found := loadChangeState(t, tdb, "cve")
				require.True(t, found)
				require.Equal(t, h0.Add(time.Duration(oldHours)*time.Hour), changesFrom)
				from = changesFrom
			}

			for _, size := range []time.Duration{time.Hour, 3 * time.Hour, 24 * time.Hour} {
				// 0 and a half-hour time zone, whose bucket ends fall between hourly changes.
				for _, shift := range []time.Duration{0, -30 * time.Minute} {
					start := h0.Add(-2 * size).Add(shift)
					n := int((hours*time.Hour+4*size)/size) + 1
					windows := [][2]time.Time{
						{start, start.Add(time.Duration(n) * size)},                         // from before the first tick to after the last
						{start, start.Add(time.Duration(n/2+1) * size)},                     // ends before the newest changes
						{h0.Add(30*time.Hour + shift), h0.Add(31*time.Hour + shift + size)}, // one bucket in the middle
					}
					for _, w := range windows {
						for _, mask := range masks {
							for _, el := range entityLists {
								if el.inListMax > 0 {
									prev := scdEntityInListMax
									scdEntityInListMax = el.inListMax
									requireReplayMatchesSnapshot(t, ds, replayQuery{start: w[0], end: w[1], size: size, mask: mask, entityIDs: el.ids}, from)
									scdEntityInListMax = prev
									continue
								}
								requireReplayMatchesSnapshot(t, ds, replayQuery{start: w[0], end: w[1], size: size, mask: mask, entityIDs: el.ids}, from)
							}
						}
					}
				}
			}
		})
	}
}

func testReplaySCDChangesEmpty(t *testing.T, _ *testutils.TestDB, ds *Datastore) {
	h0 := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	// Buckets end at h0, h0+1h, h0+2h and h0+3h.
	q := replayQuery{start: h0.Add(-2 * time.Hour), end: h0.Add(2 * time.Hour), size: time.Hour}

	// Nothing collected yet.
	require.Equal(t, []int{0, 0, 0, 0}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}))

	collectTick(t, ds, "cve", h0, hostSets{"CVE-A": {1, 2}})
	collectTick(t, ds, "other", h0, hostSets{"CVE-B": {3}})
	require.Equal(t, []int{0, 2, 2, 2}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}), "other datasets are not read")

	q.entityIDs = []string{}
	require.Equal(t, []int{0, 0, 0, 0}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}), "an empty entity list matches nothing")
	q.entityIDs = []string{"CVE-missing"}
	require.Equal(t, []int{0, 0, 0, 0}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}))
	q.entityIDs = nil
	q.mask = roaring.New()
	require.Equal(t, []int{0, 0, 0, 0}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}), "an empty mask matches nothing")
	q.mask = chart.NewBitmap([]uint{2, 1000})
	require.Equal(t, []int{0, 1, 1, 1}, requireReplayMatchesSnapshot(t, ds, q, time.Time{}), "hosts outside the mask are not counted")

	got, err := ds.ReplaySCDChanges(t.Context(), "cve", h0, h0.Add(30*time.Minute), time.Hour, nil, nil)
	require.NoError(t, err)
	require.Nil(t, got, "a range shorter than one bucket has no buckets")
}

func testSCDChangesCoveredFrom(t *testing.T, _ *testutils.TestDB, ds *Datastore) {
	_, ok, err := ds.SCDChangesCoveredFrom(t.Context(), "cve")
	require.NoError(t, err)
	require.False(t, ok, "the change storage has not started")

	h := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	oldTick(t, ds, h.Add(-48*time.Hour), hostSets{"CVE-A": {1}})
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}})
	collectTick(t, ds, "uptime", h, hostSets{"": {1}})

	coveredFrom, ok, err := ds.SCDChangesCoveredFrom(t.Context(), "cve")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, h, coveredFrom, "before conversion, the changes cover from the first collection")

	convertAll(t, ds, "cve")
	coveredFrom, ok, err = ds.SCDChangesCoveredFrom(t.Context(), "cve")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, h.AddDate(0, 0, -scdConvertDays), coveredFrom)

	require.NoError(t, ds.DeleteAllForDataset(t.Context(), "cve", 100))
	_, ok, err = ds.SCDChangesCoveredFrom(t.Context(), "cve")
	require.NoError(t, err)
	require.False(t, ok, "a global scrub removes the change storage")
	_, ok, err = ds.SCDChangesCoveredFrom(t.Context(), "uptime")
	require.NoError(t, err)
	require.True(t, ok)
}
