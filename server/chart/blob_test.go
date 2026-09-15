package chart

import (
	"bytes"
	"testing"

	"context"
	"errors"
	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"time"
)

// chunkSize is the host-ID span covered by a single roaring container (2^16).
const chunkSize uint = 1 << 16

func TestNewBitmap(t *testing.T) {
	t.Run("empty input is empty bitmap", func(t *testing.T) {
		rb := NewBitmap(nil)
		assert.True(t, rb.IsEmpty())
		assert.Equal(t, uint64(0), rb.GetCardinality())
	})

	t.Run("host id 0 is skipped", func(t *testing.T) {
		rb := NewBitmap([]uint{0, 1, 2})
		assert.Equal(t, uint64(2), rb.GetCardinality())
		assert.False(t, rb.Contains(0))
		assert.True(t, rb.Contains(1))
		assert.True(t, rb.Contains(2))
	})

	t.Run("duplicates collapse", func(t *testing.T) {
		rb := NewBitmap([]uint{5, 5, 5, 10})
		assert.Equal(t, uint64(2), rb.GetCardinality())
	})

	t.Run("multi-chunk host ids", func(t *testing.T) {
		rb := NewBitmap([]uint{7, 99, chunkSize + 5, 3*chunkSize + 10})
		assert.Equal(t, uint64(4), rb.GetCardinality())
	})
}

func TestHostIDsToBlob(t *testing.T) {
	t.Run("empty input produces nil bytes tagged roaring", func(t *testing.T) {
		b := HostIDsToBlob(nil)
		assert.Nil(t, b.Bytes)
		assert.Equal(t, EncodingRoaring, b.Encoding)

		b = HostIDsToBlob([]uint{})
		assert.Nil(t, b.Bytes)
		assert.Equal(t, EncodingRoaring, b.Encoding)
	})

	t.Run("non-empty input always tagged roaring", func(t *testing.T) {
		b := HostIDsToBlob([]uint{7})
		assert.NotNil(t, b.Bytes)
		assert.Equal(t, EncodingRoaring, b.Encoding)
	})

	t.Run("round trip via DecodeBitmap matches input set", func(t *testing.T) {
		ids := []uint{1, 5, 10, 42, 100, 255, chunkSize + 7, 2*chunkSize + 3}
		blob := HostIDsToBlob(ids)
		rb, err := DecodeBitmap(blob)
		require.NoError(t, err)
		assert.Equal(t, uint64(len(ids)), rb.GetCardinality())
		for _, id := range ids {
			assert.Truef(t, rb.Contains(uint32(id)), "expected bit %d to be set", id) //nolint:gosec // G115: test IDs fit in uint32
		}
	})
}

func TestBitmapToBlob(t *testing.T) {
	t.Run("nil bitmap produces empty blob", func(t *testing.T) {
		b := BitmapToBlob(nil)
		assert.Nil(t, b.Bytes)
		assert.Equal(t, EncodingRoaring, b.Encoding)
	})

	t.Run("empty bitmap produces empty blob", func(t *testing.T) {
		b := BitmapToBlob(roaring.New())
		assert.Nil(t, b.Bytes)
		assert.Equal(t, EncodingRoaring, b.Encoding)
	})

	t.Run("non-empty bitmap produces non-nil bytes", func(t *testing.T) {
		rb := roaring.BitmapOf(1, 2, 3)
		b := BitmapToBlob(rb)
		assert.NotEmpty(t, b.Bytes)
		assert.Equal(t, EncodingRoaring, b.Encoding)
	})
}

func TestDecodeBitmap(t *testing.T) {
	t.Run("nil bytes returns empty bitmap", func(t *testing.T) {
		rb, err := DecodeBitmap(Blob{Encoding: EncodingRoaring})
		require.NoError(t, err)
		assert.True(t, rb.IsEmpty())

		rb, err = DecodeBitmap(Blob{Encoding: EncodingDense})
		require.NoError(t, err)
		assert.True(t, rb.IsEmpty())
	})

	t.Run("roaring round trip", func(t *testing.T) {
		original := roaring.BitmapOf(1, 7, 99, 12345)
		original.RunOptimize()
		bytesData := serializeBitmap(original)

		rb, err := DecodeBitmap(Blob{Bytes: bytesData, Encoding: EncodingRoaring})
		require.NoError(t, err)
		assert.True(t, rb.Equals(original))
	})

	t.Run("dense round trip", func(t *testing.T) {
		ids := []uint{1, 7, 99, 1234}
		dense := hostIDsToDenseBlob(ids)

		rb, err := DecodeBitmap(Blob{Bytes: dense, Encoding: EncodingDense})
		require.NoError(t, err)
		assert.Equal(t, uint64(len(ids)), rb.GetCardinality())
		for _, id := range ids {
			assert.True(t, rb.Contains(uint32(id))) //nolint:gosec // G115: test IDs fit in uint32
		}
	})

	t.Run("single-byte dense", func(t *testing.T) {
		// 0x82 = bits 1 and 7 set
		rb, err := DecodeBitmap(Blob{Bytes: []byte{0x82}, Encoding: EncodingDense})
		require.NoError(t, err)
		assert.Equal(t, uint64(2), rb.GetCardinality())
		assert.True(t, rb.Contains(1))
		assert.True(t, rb.Contains(7))
	})

	t.Run("dense spanning chunk boundary", func(t *testing.T) {
		// Set a bit just below and one just above the 65536-bit chunk boundary.
		ids := []uint{chunkSize - 1, chunkSize, chunkSize + 1}
		dense := hostIDsToDenseBlob(ids)

		rb, err := DecodeBitmap(Blob{Bytes: dense, Encoding: EncodingDense})
		require.NoError(t, err)
		for _, id := range ids {
			assert.Truef(t, rb.Contains(uint32(id)), "expected bit %d to be set", id) //nolint:gosec // G115: test IDs fit in uint32
		}
	})

	t.Run("unknown encoding returns error", func(t *testing.T) {
		_, err := DecodeBitmap(Blob{Bytes: []byte{0xFF}, Encoding: 99})
		require.Error(t, err)
	})
}

func TestBitmapToHostIDs(t *testing.T) {
	t.Run("nil bitmap returns nil", func(t *testing.T) {
		assert.Nil(t, BitmapToHostIDs(nil))
	})

	t.Run("empty bitmap returns empty slice", func(t *testing.T) {
		out := BitmapToHostIDs(roaring.New())
		assert.Empty(t, out)
	})

	t.Run("populated bitmap returns sorted ids", func(t *testing.T) {
		rb := roaring.BitmapOf(99, 7, 1, 65540)
		out := BitmapToHostIDs(rb)
		assert.Equal(t, []uint{1, 7, 99, 65540}, out)
	})
}

func TestBlobPopcount(t *testing.T) {
	t.Run("nil is zero", func(t *testing.T) {
		assert.Equal(t, uint64(0), BlobPopcount(nil))
	})

	t.Run("empty bitmap is zero", func(t *testing.T) {
		assert.Equal(t, uint64(0), BlobPopcount(roaring.New()))
	})

	t.Run("counts set bits", func(t *testing.T) {
		assert.Equal(t, uint64(5), BlobPopcount(roaring.BitmapOf(1, 5, 9, 100, 65540)))
	})
}

func TestBlobAND(t *testing.T) {
	t.Run("nil operands produce empty", func(t *testing.T) {
		assert.True(t, BlobAND(nil, nil).IsEmpty())
		assert.True(t, BlobAND(roaring.BitmapOf(1, 2, 3), nil).IsEmpty())
		assert.True(t, BlobAND(nil, roaring.BitmapOf(1, 2, 3)).IsEmpty())
	})

	t.Run("intersection", func(t *testing.T) {
		a := roaring.BitmapOf(1, 5, 9, 15)
		b := roaring.BitmapOf(5, 9, 99)
		got := BlobAND(a, b)
		assert.True(t, got.Equals(roaring.BitmapOf(5, 9)))
	})

	t.Run("disjoint", func(t *testing.T) {
		a := roaring.BitmapOf(1, 2, 3)
		b := roaring.BitmapOf(10, 20, 30)
		assert.True(t, BlobAND(a, b).IsEmpty())
	})

	t.Run("idempotent", func(t *testing.T) {
		a := roaring.BitmapOf(3, 7, 11)
		assert.True(t, BlobAND(a, a).Equals(a))
	})

	t.Run("does not mutate operands", func(t *testing.T) {
		a := roaring.BitmapOf(1, 5, 9)
		b := roaring.BitmapOf(5, 9, 15)
		_ = BlobAND(a, b)
		assert.True(t, a.Equals(roaring.BitmapOf(1, 5, 9)))
		assert.True(t, b.Equals(roaring.BitmapOf(5, 9, 15)))
	})
}

func TestBlobOR(t *testing.T) {
	t.Run("both nil returns empty", func(t *testing.T) {
		assert.True(t, BlobOR(nil, nil).IsEmpty())
	})

	t.Run("one nil returns clone of the other", func(t *testing.T) {
		a := roaring.BitmapOf(1, 2, 3)
		got := BlobOR(a, nil)
		assert.True(t, got.Equals(a))

		// Mutating result should not affect the source.
		got.Remove(2)
		assert.True(t, a.Contains(2))
	})

	t.Run("union", func(t *testing.T) {
		a := roaring.BitmapOf(1, 5)
		b := roaring.BitmapOf(5, 9)
		assert.True(t, BlobOR(a, b).Equals(roaring.BitmapOf(1, 5, 9)))
	})

	t.Run("idempotent", func(t *testing.T) {
		a := roaring.BitmapOf(3, 7, 11)
		assert.True(t, BlobOR(a, a).Equals(a))
	})

	t.Run("does not mutate operands", func(t *testing.T) {
		a := roaring.BitmapOf(1, 5)
		b := roaring.BitmapOf(5, 9)
		_ = BlobOR(a, b)
		assert.True(t, a.Equals(roaring.BitmapOf(1, 5)))
		assert.True(t, b.Equals(roaring.BitmapOf(5, 9)))
	})
}

func TestBlobANDNOT(t *testing.T) {
	t.Run("nil a returns empty", func(t *testing.T) {
		assert.True(t, BlobANDNOT(nil, roaring.BitmapOf(1, 2, 3)).IsEmpty())
	})

	t.Run("nil mask returns clone of a", func(t *testing.T) {
		a := roaring.BitmapOf(1, 2, 3)
		got := BlobANDNOT(a, nil)
		assert.True(t, got.Equals(a))
		got.Remove(2)
		assert.True(t, a.Contains(2))
	})

	t.Run("subtraction", func(t *testing.T) {
		a := roaring.BitmapOf(1, 5, 9, 15)
		mask := roaring.BitmapOf(5, 15)
		assert.True(t, BlobANDNOT(a, mask).Equals(roaring.BitmapOf(1, 9)))
	})

	t.Run("mask covering a yields empty", func(t *testing.T) {
		a := roaring.BitmapOf(1, 2, 3)
		assert.True(t, BlobANDNOT(a, a).IsEmpty())
	})

	t.Run("disjoint mask is identity", func(t *testing.T) {
		a := roaring.BitmapOf(1, 2, 3)
		mask := roaring.BitmapOf(10, 20)
		assert.True(t, BlobANDNOT(a, mask).Equals(a))
	})

	t.Run("does not mutate operands", func(t *testing.T) {
		a := roaring.BitmapOf(1, 2, 3)
		mask := roaring.BitmapOf(2)
		_ = BlobANDNOT(a, mask)
		assert.True(t, a.Equals(roaring.BitmapOf(1, 2, 3)))
		assert.True(t, mask.Equals(roaring.BitmapOf(2)))
	})
}

// TestMixedEncoding exercises the transition case where a legacy dense row is
// decoded at the boundary and used alongside a roaring operand.
func TestMixedEncoding(t *testing.T) {
	ids := []uint{1, 5, 9}
	denseBlob := Blob{Bytes: hostIDsToDenseBlob(ids), Encoding: EncodingDense}
	roaringBlob := HostIDsToBlob([]uint{5, 9, 15})

	a, err := DecodeBitmap(denseBlob)
	require.NoError(t, err)
	b, err := DecodeBitmap(roaringBlob)
	require.NoError(t, err)

	t.Run("AND mixed-encoding", func(t *testing.T) {
		assert.True(t, BlobAND(a, b).Equals(roaring.BitmapOf(5, 9)))
	})

	t.Run("OR mixed-encoding", func(t *testing.T) {
		assert.True(t, BlobOR(a, b).Equals(roaring.BitmapOf(1, 5, 9, 15)))
	})

	t.Run("ANDNOT mixed-encoding both directions", func(t *testing.T) {
		assert.True(t, BlobANDNOT(a, b).Equals(roaring.BitmapOf(1)))
		assert.True(t, BlobANDNOT(b, a).Equals(roaring.BitmapOf(15)))
	})

	t.Run("popcount on decoded legacy dense", func(t *testing.T) {
		assert.Equal(t, uint64(3), BlobPopcount(a))
	})
}

// TestContainerTypes builds bitmaps that force each roaring container type
// (array, bitmap, run) and a multi-chunk bitmap, then exercises all ops over
// the fixture matrix. Without this the bitmap and run paths are silently
// untested when the rest of the suite uses sparse-shaped inputs.
func TestContainerTypes(t *testing.T) {
	// Array container: 50 scattered ids within one chunk (cardinality << 4096).
	arrayIDs := make([]uint, 0, 50)
	for i := range uint(50) {
		arrayIDs = append(arrayIDs, 1000+i*7)
	}
	array := NewBitmap(arrayIDs)

	// Bitmap container: 5000 ids in one chunk (cardinality > 4096 forces bitmap).
	bitmapIDs := make([]uint, 0, 5000)
	for i := range uint(5000) {
		bitmapIDs = append(bitmapIDs, 10000+i)
	}
	bitmapRB := NewBitmap(bitmapIDs)

	// Run container: a contiguous range of 10000 ids — RunOptimize will pick
	// a run container as the compact representation.
	runIDs := make([]uint, 0, 10000)
	for i := range uint(10000) {
		runIDs = append(runIDs, 100+i)
	}
	run := NewBitmap(runIDs)

	// Multi-chunk: ids spanning ≥3 chunks across the 65,536-bit boundary.
	multiIDs := []uint{
		7, 99, chunkSize / 2,
		chunkSize + 7, chunkSize + 99,
		2*chunkSize + 7, 2*chunkSize + 99,
	}
	multi := NewBitmap(multiIDs)

	fixtures := map[string]*roaring.Bitmap{
		"array":  array,
		"bitmap": bitmapRB,
		"run":    run,
		"multi":  multi,
	}

	for nameA, a := range fixtures {
		for nameB, b := range fixtures {
			t.Run("AND/"+nameA+"_x_"+nameB, func(t *testing.T) {
				got := BlobAND(a, b)
				want := roaring.And(a, b)
				assert.True(t, got.Equals(want))
			})
			t.Run("OR/"+nameA+"_x_"+nameB, func(t *testing.T) {
				got := BlobOR(a, b)
				want := roaring.Or(a, b)
				assert.True(t, got.Equals(want))
			})
			t.Run("ANDNOT/"+nameA+"_x_"+nameB, func(t *testing.T) {
				got := BlobANDNOT(a, b)
				want := roaring.AndNot(a, b)
				assert.True(t, got.Equals(want))
			})
		}
	}
}

// TestSerializationDeterminism asserts that the same host set produces
// byte-equal output regardless of which code path built the bitmap. Catches
// any missed RunOptimize call in the encoder chain.
func TestSerializationDeterminism(t *testing.T) {
	ids := []uint{2, 100, chunkSize + 4, 2 * chunkSize}

	// Path A: build directly.
	bytesA := BitmapToBlob(NewBitmap(ids)).Bytes

	// Path B: round-trip through dense.
	denseBlob := Blob{Bytes: hostIDsToDenseBlob(ids), Encoding: EncodingDense}
	rbFromDense, err := DecodeBitmap(denseBlob)
	require.NoError(t, err)
	bytesB := BitmapToBlob(rbFromDense).Bytes

	// Path C: OR an empty with the source bitmap.
	bytesC := BitmapToBlob(BlobOR(roaring.New(), NewBitmap(ids))).Bytes

	require.True(t, bytes.Equal(bytesA, bytesB), "BitmapToBlob(NewBitmap) vs BitmapToBlob(DecodeBitmap(dense)) differ:\nA=%x\nB=%x", bytesA, bytesB)
	require.True(t, bytes.Equal(bytesA, bytesC), "BitmapToBlob(NewBitmap) vs BitmapToBlob(BlobOR(empty, ...)) differ:\nA=%x\nC=%x", bytesA, bytesC)
}

// fakeDatasetStore records what the collector hands the storage layer.
type fakeDatasetStore struct {
	collectible []string
	affected    map[string][]uint
	resolved    map[string][]string // keyed by the filter's aggregate entity ID
	resolveErr  error

	recordedCalls  int
	recorded       map[string]*roaring.Bitmap
	backfillCalls  []string
	backfillSource map[string][]string
	// backfillDone lists aggregates with nothing left to rebuild.
	backfillDone map[string]bool
}

func (f *fakeDatasetStore) FindOnlineHostIDs(context.Context, time.Time, []uint) ([]uint, error) {
	return nil, nil
}

func (f *fakeDatasetStore) CollectibleCVEs(context.Context) ([]string, error) {
	return f.collectible, nil
}

func (f *fakeDatasetStore) AffectedHostIDsByCVE(_ context.Context, _ []uint, cves []string) (map[string]*roaring.Bitmap, error) {
	out := make(map[string]*roaring.Bitmap)
	for _, cve := range cves {
		if ids, ok := f.affected[cve]; ok {
			out[cve] = NewBitmap(ids)
		}
	}
	return out, nil
}

func (f *fakeDatasetStore) ResolveCVEChartEntities(_ context.Context, filter api.CVEFilter) ([]string, error) {
	if f.resolveErr != nil {
		return nil, f.resolveErr
	}
	return f.resolved[filter.AggregateEntityID()], nil
}

func (f *fakeDatasetStore) BackfillAggregateEntity(_ context.Context, _, aggregateID string, sourceIDs []string, _, _ time.Time) (bool, error) {
	f.backfillCalls = append(f.backfillCalls, aggregateID)
	if f.backfillSource == nil {
		f.backfillSource = map[string][]string{}
	}
	f.backfillSource[aggregateID] = sourceIDs
	return !f.backfillDone[aggregateID], nil
}

func (f *fakeDatasetStore) RecordBucketData(_ context.Context, _ string, _ time.Time, _ time.Duration, _ api.SampleStrategy, entityBitmaps map[string]*roaring.Bitmap) error {
	f.recordedCalls++
	f.recorded = entityBitmaps
	return nil
}

func hostIDs(rb *roaring.Bitmap) []uint {
	if rb == nil {
		return nil
	}
	return BitmapToHostIDs(rb)
}

// criticalFilter is the shape the dashboard's default request carries.
func criticalFilter() api.CVEFilter {
	return api.CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)}
}

func newFakeStore() *fakeDatasetStore {
	crit := criticalFilter()
	return &fakeDatasetStore{
		collectible: []string{"CVE-1", "CVE-2", "CVE-3"},
		affected: map[string][]uint{
			"CVE-1": {1, 2},
			"CVE-2": {2, 3},
			"CVE-3": {9},
		},
		resolved: map[string][]string{
			crit.AggregateEntityID(): {"CVE-1", "CVE-2"},
		},
	}
}

func TestCVEDatasetCollectWithoutPreaggregation(t *testing.T) {
	store := newFakeStore()
	require.NoError(t, (&CVEDataset{}).Collect(t.Context(), store, time.Now(), nil))

	require.Equal(t, 1, store.recordedCalls)
	require.Len(t, store.recorded, 3, "only the per-CVE entities are written")
	require.Empty(t, store.backfillCalls)
}

func TestCVEDatasetCollectWritesAggregate(t *testing.T) {
	store := newFakeStore()
	crit := criticalFilter()
	ds := &CVEDataset{PreaggregateFilters: []api.CVEFilter{crit}}

	require.NoError(t, ds.Collect(t.Context(), store, time.Now(), nil))

	// One call, carrying both the per-CVE rows and the aggregate: snapshot
	// semantics close every entity missing from the map, so a second call would
	// close whatever the first one wrote.
	require.Equal(t, 1, store.recordedCalls)
	require.Len(t, store.recorded, 4)

	agg, ok := store.recorded[crit.AggregateEntityID()]
	require.True(t, ok, "the aggregate entity must be written alongside the per-CVE rows")
	require.Equal(t, []uint{1, 2, 3}, hostIDs(agg), "union of the critical CVEs, excluding the low-severity one")

	// The per-CVE rows must still be written — custom filters read them.
	require.Equal(t, []uint{1, 2}, hostIDs(store.recorded["CVE-1"]))
	require.Equal(t, []uint{9}, hostIDs(store.recorded["CVE-3"]))
}

func TestCVEDatasetCollectBackfillsAggregate(t *testing.T) {
	store := newFakeStore()
	crit := criticalFilter()
	ds := &CVEDataset{PreaggregateFilters: []api.CVEFilter{crit}}

	require.NoError(t, ds.Collect(t.Context(), store, time.Now(), nil))

	require.Equal(t, []string{crit.AggregateEntityID()}, store.backfillCalls)
	require.Equal(t, []string{"CVE-1", "CVE-2"}, store.backfillSource[crit.AggregateEntityID()],
		"backfill must reconstruct history from the same CVEs the aggregate unions")
}

// Filters resolving identically share one series, not two copies of a union.
func TestCVEDatasetCollectDeduplicatesEquivalentFilters(t *testing.T) {
	store := newFakeStore()
	ds := &CVEDataset{PreaggregateFilters: []api.CVEFilter{
		{CVSSMin: new(9.0), CVSSMax: new(10.0)},
		{CVSSMin: new(9.0), CVSSMax: new(10.0), Categories: []string{
			api.CVECategoryOS, api.CVECategoryBrowsers, api.CVECategoryOffice, api.CVECategoryAdobe,
		}},
	}}

	require.NoError(t, ds.Collect(t.Context(), store, time.Now(), nil))
	require.Len(t, store.recorded, 4, "the two filters collapse onto one aggregate")
	require.Len(t, store.backfillCalls, 1)
}

// A filter matching nothing still needs a row: a missing entity closes the
// series and drops back to the slow path.
func TestCVEDatasetCollectWritesEmptyAggregate(t *testing.T) {
	store := newFakeStore()
	empty := api.CVEFilter{CVSSMin: new(9.9), CVSSMax: new(9.95)}
	ds := &CVEDataset{PreaggregateFilters: []api.CVEFilter{empty}}

	require.NoError(t, ds.Collect(t.Context(), store, time.Now(), nil))

	agg, ok := store.recorded[empty.AggregateEntityID()]
	require.True(t, ok)
	require.NotNil(t, agg)
	require.True(t, agg.IsEmpty())
}

// A partial map closes every aggregate the previous tick opened, so an
// unresolvable filter has to abort the tick.
func TestCVEDatasetCollectAbortsWhenFiltersUnavailable(t *testing.T) {
	store := newFakeStore()
	store.resolveErr = errors.New("resolve failed")
	ds := &CVEDataset{PreaggregateFilters: []api.CVEFilter{criticalFilter()}}
	require.Error(t, ds.Collect(t.Context(), store, time.Now(), nil))
	require.Zero(t, store.recordedCalls)
}

// severityBandFilters stands in for the set cmd/fleet precomputes.
func severityBandFilters() []api.CVEFilter {
	return []api.CVEFilter{
		{CVSSMin: new(9.0), CVSSMax: new(10.0)},
		{},
		{CVSSMin: new(7.0), CVSSMax: new(8.9)},
		{CVSSMin: new(4.0), CVSSMax: new(6.9)},
		{CVSSMin: new(0.1), CVSSMax: new(3.9)},
	}
}

// Rebuilding is the expensive part, so a tick does a bounded amount of it.
func TestCVEDatasetCollectBudgetsBackfillPerTick(t *testing.T) {
	store := newFakeStore()
	filters := severityBandFilters()
	for _, f := range filters {
		store.resolved[f.AggregateEntityID()] = []string{"CVE-1", "CVE-2"}
	}
	ds := &CVEDataset{PreaggregateFilters: filters}

	require.NoError(t, ds.Collect(t.Context(), store, time.Now(), nil))

	require.Len(t, store.backfillCalls, aggregateBackfillBatchesPerTick)
	// Every series still gets its current row: one left out of the snapshot
	// would be closed and lose the history already rebuilt for it.
	require.Len(t, store.recorded, len(store.collectible)+len(filters))
	for _, f := range filters {
		require.Contains(t, store.recorded, f.AggregateEntityID())
	}
}

// A finished series must not consume the budget and starve the others.
func TestCVEDatasetCollectBudgetSkipsCompletedBackfills(t *testing.T) {
	store := newFakeStore()
	filters := severityBandFilters()
	store.backfillDone = map[string]bool{}
	for i, f := range filters {
		store.resolved[f.AggregateEntityID()] = []string{"CVE-1"}
		if i < 3 {
			store.backfillDone[f.AggregateEntityID()] = true
		}
	}
	ds := &CVEDataset{PreaggregateFilters: filters}

	require.NoError(t, ds.Collect(t.Context(), store, time.Now(), nil))

	// The three finished series are asked and decline, costing no budget; it is
	// then spent on the ones that still have history to rebuild.
	require.Len(t, store.backfillCalls, 3+aggregateBackfillBatchesPerTick)
	for i := range aggregateBackfillBatchesPerTick {
		require.Equal(t, filters[3+i].AggregateEntityID(), store.backfillCalls[3+i])
	}
}
