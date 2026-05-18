package chart

import (
	"bytes"
	"testing"

	"github.com/RoaringBitmap/roaring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Truef(t, rb.Contains(uint32(id)), "expected bit %d to be set", id)
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
			assert.True(t, rb.Contains(uint32(id)))
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
			assert.Truef(t, rb.Contains(uint32(id)), "expected bit %d to be set", id)
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
	for i := uint(0); i < 50; i++ {
		arrayIDs = append(arrayIDs, 1000+i*7)
	}
	array := NewBitmap(arrayIDs)

	// Bitmap container: 5000 ids in one chunk (cardinality > 4096 forces bitmap).
	bitmapIDs := make([]uint, 0, 5000)
	for i := uint(0); i < 5000; i++ {
		bitmapIDs = append(bitmapIDs, 10000+i)
	}
	bitmapRB := NewBitmap(bitmapIDs)

	// Run container: a contiguous range of 10000 ids — RunOptimize will pick
	// a run container as the compact representation.
	runIDs := make([]uint, 0, 10000)
	for i := uint(0); i < 10000; i++ {
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
