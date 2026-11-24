package mysql

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/testutils"
	"github.com/stretchr/testify/require"
)

type changeRec struct {
	at      time.Time
	entity  string
	added   []uint
	removed []uint
}

type hostSets map[string][]uint

func bitmapsOf(sets hostSets) map[string]*roaring.Bitmap {
	out := make(map[string]*roaring.Bitmap, len(sets))
	for e, ids := range sets {
		out[e] = chart.NewBitmap(ids)
	}
	return out
}

// collectTick writes one tick the way CVEDataset.Collect does: the full-copy rows first, then the changes.
func collectTick(t *testing.T, ds *Datastore, dataset string, hour time.Time, sets hostSets) {
	t.Helper()
	require.NoError(t, ds.RecordBucketData(t.Context(), dataset, hour, time.Hour, api.SampleStrategySnapshot, bitmapsOf(sets)))
	require.NoError(t, ds.RecordSCDChanges(t.Context(), dataset, hour, bitmapsOf(sets)))
}

func entityNames(t *testing.T, tdb *testutils.TestDB, dataset string) map[uint32]string {
	t.Helper()
	var rows []struct {
		ID       uint32 `db:"id"`
		EntityID string `db:"entity_id"`
	}
	require.NoError(t, tdb.DB.SelectContext(t.Context(), &rows, `SELECT id, entity_id FROM host_scd_entities WHERE dataset = ?`, dataset))
	out := make(map[uint32]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.EntityID
	}
	return out
}

// loadChanges returns the stored changes in write order.
func loadChanges(t *testing.T, tdb *testutils.TestDB, dataset string) []changeRec {
	t.Helper()
	names := entityNames(t, tdb, dataset)
	var rows []struct {
		ChangedAt time.Time `db:"changed_at"`
		Changes   []byte    `db:"changes"`
	}
	require.NoError(t, tdb.DB.SelectContext(t.Context(), &rows,
		`SELECT changed_at, changes FROM host_scd_changes WHERE dataset = ? ORDER BY changed_at, seq`, dataset))
	out := []changeRec{}
	for _, r := range rows {
		chs, err := decodeChanges(r.Changes)
		require.NoError(t, err)
		require.NotEmpty(t, chs, "a stored row must hold at least one change")
		for _, c := range chs {
			name, ok := names[c.key]
			require.True(t, ok, "change for unknown entity key %d", c.key)
			out = append(out, changeRec{at: r.ChangedAt.UTC(), entity: name, added: chart.BitmapToHostIDs(c.added), removed: chart.BitmapToHostIDs(c.removed)})
		}
	}
	return out
}

func loadCurrent(t *testing.T, tdb *testutils.TestDB, dataset string) hostSets {
	t.Helper()
	names := entityNames(t, tdb, dataset)
	var rows []struct {
		EntityKey    uint32 `db:"entity_key"`
		HostBitmap   []byte `db:"host_bitmap"`
		EncodingType uint8  `db:"encoding_type"`
	}
	require.NoError(t, tdb.DB.SelectContext(t.Context(), &rows,
		`SELECT entity_key, host_bitmap, encoding_type FROM host_scd_current WHERE dataset = ?`, dataset))
	out := hostSets{}
	for _, r := range rows {
		rb, err := chart.DecodeBitmap(chart.Blob{Bytes: r.HostBitmap, Encoding: r.EncodingType})
		require.NoError(t, err)
		require.False(t, rb.IsEmpty(), "current rows hold only non-empty sets")
		out[names[r.EntityKey]] = chart.BitmapToHostIDs(rb)
	}
	return out
}

func loadChangeState(t *testing.T, tdb *testutils.TestDB, dataset string) (changesFrom, coveredFrom time.Time, found bool) {
	t.Helper()
	var rows []struct {
		ChangesFrom time.Time `db:"changes_from"`
		CoveredFrom time.Time `db:"covered_from"`
	}
	require.NoError(t, tdb.DB.SelectContext(t.Context(), &rows,
		`SELECT changes_from, covered_from FROM host_scd_change_state WHERE dataset = ?`, dataset))
	if len(rows) == 0 {
		return time.Time{}, time.Time{}, false
	}
	return rows[0].ChangesFrom.UTC(), rows[0].CoveredFrom.UTC(), true
}

func countDatasetRows(t *testing.T, tdb *testutils.TestDB, dataset string) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"host_scd_data", "host_scd_entities", "host_scd_current", "host_scd_changes", "host_scd_change_state"} {
		var n int
		require.NoError(t, tdb.DB.GetContext(t.Context(), &n, `SELECT COUNT(*) FROM `+table+` WHERE dataset = ?`, dataset))
		out[table] = n
	}
	return out
}

// stateAt rebuilds every entity's host set at the end of hour h from the current sets by undoing every change
// written after h.
func stateAt(current hostSets, changes []changeRec, h time.Time) hostSets {
	state := map[string]*roaring.Bitmap{}
	for e, ids := range current {
		state[e] = chart.NewBitmap(ids)
	}
	for _, c := range slices.Backward(changes) {
		if !c.at.After(h) {
			break
		}
		bm := state[c.entity]
		if bm == nil {
			bm = roaring.New()
			state[c.entity] = bm
		}
		bm.AndNot(chart.NewBitmap(c.added))
		bm.Or(chart.NewBitmap(c.removed))
	}
	return nonEmpty(state)
}

func nonEmpty(state map[string]*roaring.Bitmap) hostSets {
	out := hostSets{}
	for e, bm := range state {
		if !bm.IsEmpty() {
			out[e] = chart.BitmapToHostIDs(bm)
		}
	}
	return out
}

func TestRecordSCDChanges(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "chart_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)

	cases := []struct {
		name string
		fn   func(t *testing.T, tdb *testutils.TestDB, ds *Datastore)
	}{
		{"first tick starts from the existing history", testRecordSCDChangesFirstTickWithHistory},
		{"from empty tables: hosts added and removed, an entity disappearing and returning", testRecordSCDChangesSequence},
		{"same-hour re-runs", testRecordSCDChangesSameHour},
		{"datasets are independent", testRecordSCDChangesDatasets},
		{"first tick unions overlapping history rows and clears leftovers", testRecordSCDChangesStartEdgeCases},
		{"random histories replay to every hour", testRecordSCDChangesRandom},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer tdb.TruncateTables(t)
			c.fn(t, tdb, ds)
		})
	}
}

func testRecordSCDChangesFirstTickWithHistory(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	ctx := t.Context()
	h := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	// History written before the change storage existed.
	require.NoError(t, ds.RecordBucketData(ctx, "cve", h.Add(-2*time.Hour), time.Hour, api.SampleStrategySnapshot,
		bitmapsOf(hostSets{"CVE-A": {1, 2}, "CVE-B": {3}, "CVE-C": {5}})))
	require.NoError(t, ds.RecordBucketData(ctx, "cve", h.Add(-time.Hour), time.Hour, api.SampleStrategySnapshot,
		bitmapsOf(hostSets{"CVE-A": {1, 2}, "CVE-B": {3, 4}, "CVE-C": {5}})))

	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}, "CVE-B": {3, 4, 6}, "CVE-D": {7}})

	require.ElementsMatch(t, []changeRec{
		{at: h, entity: "CVE-B", added: []uint{6}, removed: []uint{}},
		{at: h, entity: "CVE-C", added: []uint{}, removed: []uint{5}},
		{at: h, entity: "CVE-D", added: []uint{7}, removed: []uint{}},
	}, loadChanges(t, tdb, "cve"))
	require.Equal(t, hostSets{"CVE-A": {1, 2}, "CVE-B": {3, 4, 6}, "CVE-D": {7}}, loadCurrent(t, tdb, "cve"))
	changesFrom, coveredFrom, found := loadChangeState(t, tdb, "cve")
	require.True(t, found)
	require.Equal(t, h, changesFrom)
	require.Equal(t, h, coveredFrom)

	// The next tick diffs against the stored current sets, not the history.
	collectTick(t, ds, "cve", h.Add(time.Hour), hostSets{"CVE-A": {1}, "CVE-B": {3, 4, 6}, "CVE-D": {7}})
	changes := loadChanges(t, tdb, "cve")
	require.Equal(t, changeRec{at: h.Add(time.Hour), entity: "CVE-A", added: []uint{}, removed: []uint{2}}, changes[len(changes)-1])
	require.Len(t, changes, 4)
	changesFrom, _, _ = loadChangeState(t, tdb, "cve")
	require.Equal(t, h, changesFrom, "only the first tick sets changes_from")
}

func testRecordSCDChangesSequence(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}, "CVE-B": {3}})
	changesFrom, coveredFrom, found := loadChangeState(t, tdb, "cve")
	require.True(t, found)
	require.Equal(t, h, changesFrom)
	require.Equal(t, h, coveredFrom)

	collectTick(t, ds, "cve", h.Add(time.Hour), hostSets{"CVE-A": {2, 3}, "CVE-B": {3}})
	collectTick(t, ds, "cve", h.Add(2*time.Hour), hostSets{"CVE-A": {2, 3}, "CVE-B": {3}})
	collectTick(t, ds, "cve", h.Add(3*time.Hour), hostSets{"CVE-B": {3}})
	require.Equal(t, hostSets{"CVE-B": {3}}, loadCurrent(t, tdb, "cve"), "an entity without hosts has no current row")
	collectTick(t, ds, "cve", h.Add(4*time.Hour), hostSets{"CVE-A": {1}, "CVE-B": {3}})

	require.ElementsMatch(t, []changeRec{
		{at: h, entity: "CVE-A", added: []uint{1, 2}, removed: []uint{}},
		{at: h, entity: "CVE-B", added: []uint{3}, removed: []uint{}},
		{at: h.Add(time.Hour), entity: "CVE-A", added: []uint{3}, removed: []uint{1}},
		{at: h.Add(3 * time.Hour), entity: "CVE-A", added: []uint{}, removed: []uint{2, 3}},
		{at: h.Add(4 * time.Hour), entity: "CVE-A", added: []uint{1}, removed: []uint{}},
	}, loadChanges(t, tdb, "cve"))
	require.Equal(t, hostSets{"CVE-A": {1}, "CVE-B": {3}}, loadCurrent(t, tdb, "cve"))
}

func testRecordSCDChangesSameHour(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	h1 := h.Add(time.Hour)
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}})
	collectTick(t, ds, "cve", h1, hostSets{"CVE-A": {1, 2, 3}})
	collectTick(t, ds, "cve", h1, hostSets{"CVE-A": {1}})
	// The last run of the hour undoes the earlier ones: the hour's changes net to nothing.
	collectTick(t, ds, "cve", h1, hostSets{"CVE-A": {1, 2}})

	changes := loadChanges(t, tdb, "cve")
	current := loadCurrent(t, tdb, "cve")
	require.Equal(t, hostSets{"CVE-A": {1, 2}}, current)
	require.Equal(t, hostSets{"CVE-A": {1, 2}}, stateAt(current, changes, h), "state before the re-run hour")
	require.Equal(t, hostSets{"CVE-A": {1, 2}}, stateAt(current, changes, h1), "state at the end of the re-run hour")
	changesFrom, _, _ := loadChangeState(t, tdb, "cve")
	require.Equal(t, h, changesFrom)

	// A re-run of the first hour keeps changes_from.
	collectTick(t, ds, "cve", h, hostSets{"CVE-A": {1, 2}})
	changesFrom, _, _ = loadChangeState(t, tdb, "cve")
	require.Equal(t, h, changesFrom)
}

func testRecordSCDChangesStartEdgeCases(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-A", []uint{1}, h.Add(-3*time.Hour), scdOpenSentinel)
	tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-A", []uint{2}, h.Add(-2*time.Hour), h)
	tdb.InsertSCDRowWithHostIDs(t, "cve", "CVE-B", []uint{9}, h, scdOpenSentinel) // opens at h: not live before it
	// Left by an earlier attempt that failed before writing the state row.
	require.NoError(t, ds.ensureEntityKeys(t.Context(), "cve", []string{"CVE-Z"}, map[string]uint32{}))
	_, err := tdb.DB.ExecContext(t.Context(),
		`INSERT INTO host_scd_current (dataset, entity_key, host_bitmap, encoding_type)
		 SELECT 'cve', id, ?, ? FROM host_scd_entities WHERE entity_id = 'CVE-Z'`,
		chart.HostIDsToBlob([]uint{7}).Bytes, chart.EncodingRoaring)
	require.NoError(t, err)

	require.NoError(t, ds.RecordSCDChanges(t.Context(), "cve", h, bitmapsOf(hostSets{"CVE-A": {1, 2}, "CVE-B": {9}})))

	require.Equal(t, []changeRec{{at: h, entity: "CVE-B", added: []uint{9}, removed: []uint{}}}, loadChanges(t, tdb, "cve"))
	require.Equal(t, hostSets{"CVE-A": {1, 2}, "CVE-B": {9}}, loadCurrent(t, tdb, "cve"))
}

func TestDecodeChanges(t *testing.T) {
	chs := []entityChange{
		{key: 1, added: chart.NewBitmap([]uint{1, 2}), removed: roaring.New()},
		{key: 1 << 31, added: roaring.New(), removed: chart.NewBitmap([]uint{70000})},
	}
	b := encodeChanges(chs)
	got, err := decodeChanges(b)
	require.NoError(t, err)
	require.Len(t, got, 2)
	for i := range chs {
		require.Equal(t, chs[i].key, got[i].key)
		require.True(t, chs[i].added.Equals(got[i].added))
		require.True(t, chs[i].removed.Equals(got[i].removed))
	}

	empty, err := decodeChanges(nil)
	require.NoError(t, err)
	require.Empty(t, empty)

	// Cutting at the end of the first entity leaves a valid list; any other cut is an error.
	first := len(encodeChanges(chs[:1]))
	for i := 1; i < len(b); i++ {
		_, err := decodeChanges(b[:i])
		if i == first {
			require.NoError(t, err)
			continue
		}
		require.Error(t, err, "truncated at %d of %d bytes", i, len(b))
	}
	tooBig := binary.AppendUvarint(nil, 1<<32)
	tooBig = binary.AppendUvarint(tooBig, 0)
	tooBig = binary.AppendUvarint(tooBig, 0)
	_, err = decodeChanges(tooBig)
	require.ErrorContains(t, err, "out of range")
}

func testRecordSCDChangesDatasets(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	h := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	collectTick(t, ds, "cve", h, hostSets{"X": {1}})
	collectTick(t, ds, "other", h.Add(time.Hour), hostSets{"X": {2}})
	collectTick(t, ds, "cve", h.Add(2*time.Hour), hostSets{"X": {1, 3}})

	require.Equal(t, hostSets{"X": {1, 3}}, loadCurrent(t, tdb, "cve"))
	require.Equal(t, hostSets{"X": {2}}, loadCurrent(t, tdb, "other"))
	require.Equal(t, []changeRec{{at: h.Add(time.Hour), entity: "X", added: []uint{2}, removed: []uint{}}}, loadChanges(t, tdb, "other"))
	cveFrom, _, _ := loadChangeState(t, tdb, "cve")
	otherFrom, _, _ := loadChangeState(t, tdb, "other")
	require.Equal(t, h, cveFrom)
	require.Equal(t, h.Add(time.Hour), otherFrom)
}

// testRecordSCDChangesRandom writes random histories, some hours several times, with small row budgets so writes
// split into several rows, and checks that undoing the changes from the current sets gives back the last input of
// every hour since the change storage started, including the history it started from.
func testRecordSCDChangesRandom(t *testing.T, tdb *testutils.TestDB, ds *Datastore) {
	defer func(changes, current, entities int) {
		scdChangeRowBytes, scdCurrentWriteBytes, scdChangeBatchEntities = changes, current, entities
	}(scdChangeRowBytes, scdCurrentWriteBytes, scdChangeBatchEntities)
	scdChangeRowBytes, scdCurrentWriteBytes, scdChangeBatchEntities = 200, 300, 2

	// oldHours is how many hours are written before the change storage exists.
	for oldHours, seed := range []uint64{0, 1, 2, 3} {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			defer tdb.TruncateTables(t)
			rng := rand.New(rand.NewPCG(seed, 58928)) //nolint:gosec // deterministic test data
			h0 := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
			last := map[time.Time]hostSets{}
			for i := range 30 {
				h := h0.Add(time.Duration(i) * time.Hour)
				runs := 1 + rng.IntN(3)*(rng.IntN(4)/3) // usually one run, sometimes up to three
				if i < oldHours {
					// A same-hour re-run in host_scd_data can lose an entity that drops out and returns
					// within the hour, so the history the change storage starts from is written once per hour.
					runs = 1
				}
				for range runs {
					sets := hostSets{}
					for e := range 6 {
						if rng.IntN(5) == 0 {
							continue
						}
						ids := []uint{uint(1 + rng.IntN(12))}
						for host := uint(1); host <= 12; host++ {
							if rng.IntN(3) == 0 {
								ids = append(ids, host)
							}
						}
						sets[fmt.Sprintf("CVE-%d", e)] = ids
					}
					if i < oldHours {
						require.NoError(t, ds.RecordBucketData(t.Context(), "cve", h, time.Hour, api.SampleStrategySnapshot, bitmapsOf(sets)))
					} else {
						collectTick(t, ds, "cve", h, sets)
					}
					last[h] = sets
				}
			}

			changes := loadChanges(t, tdb, "cve")
			current := loadCurrent(t, tdb, "cve")
			start := h0.Add(time.Duration(max(oldHours-1, 0)) * time.Hour)
			if oldHours == 0 {
				require.Empty(t, stateAt(current, changes, h0.Add(-time.Hour)), "a new install starts from nothing")
			}
			for i := range 30 {
				h := h0.Add(time.Duration(i) * time.Hour)
				if h.Before(start) {
					continue
				}
				want := hostSets{}
				for e, ids := range last[h] {
					if len(ids) > 0 {
						want[e] = chart.BitmapToHostIDs(chart.NewBitmap(ids))
					}
				}
				require.Equal(t, want, stateAt(current, changes, h), "hour %d", i)
			}
			require.True(t, slices.IsSortedFunc(changes, func(a, b changeRec) int { return a.at.Compare(b.at) }))
		})
	}
}
