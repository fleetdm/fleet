// Package chart provides blob utility helpers, dataset implementations, and
// shared constants for the chart bounded context. Public API types live in
// server/chart/api; internal types (HostFilter, Datastore) live in
// server/chart/internal/types.
//
// # Bitmap encoding
//
// Host-set bitmaps are stored in host_scd_data.host_bitmap. Two on-disk
// formats are supported, discriminated by the host_scd_data.encoding_type
// column:
//
//   - EncodingDense (0): a raw bit-array sized to (max_id_in_set / 8) + 1.
//     Bit n set iff host n is in the set. The original format; legacy rows
//     written before this encoding was introduced read with encoding_type = 0
//     via the column DEFAULT.
//
//   - EncodingRoaring (1): the standard portable RoaringBitmap/roaring
//     serialization (Bitmap.ToBytes() output). All new writes use this
//     encoding; legacy dense rows are decoded into roaring at the I/O
//     boundary via DecodeBitmap and either age out via retention or are
//     overwritten on the next state transition.
//
// # Storage form vs op form
//
// Two distinct in-memory representations:
//
//   - Blob{Bytes, Encoding} — storage form. Used only at the database I/O
//     boundary. Constructed by HostIDsToBlob / BitmapToBlob. Consumed by
//     INSERT / UPDATE statements.
//
//   - *roaring.Bitmap — op form. Used for all bitwise operations
//     (BlobAND/OR/ANDNOT/Popcount) and in-memory bitmap manipulation.
//     Constructed by NewBitmap or DecodeBitmap. Encoding-awareness lives
//     in DecodeBitmap and BitmapToBlob only.
//
// All BitmapToBlob calls invoke RunOptimize before serializing, so the
// same host set always produces byte-equal Blob.Bytes. This is not
// load-bearing for correctness (change detection uses roaring.Equals on
// op-form bitmaps) but is a desirable storage property.
package chart

import (
	"github.com/RoaringBitmap/roaring"
)

// Encoding identifies the on-disk format of a host_bitmap blob. The constants
// here correspond directly to the host_scd_data.encoding_type column values.
const (
	EncodingDense   uint8 = 0
	EncodingRoaring uint8 = 1
)

// Blob is the storage form of a host-set bitmap. Bytes is the serialized
// payload as written to host_scd_data.host_bitmap; Encoding is the matching
// host_scd_data.encoding_type column value. A nil Bytes represents the empty
// host set regardless of Encoding.
type Blob struct {
	Bytes    []byte
	Encoding uint8
}

// NewBitmap builds a *roaring.Bitmap from a host ID list. Calls RunOptimize
// before returning so that subsequent serialization (via BitmapToBlob) is
// byte-deterministic for the input set. Host IDs of 0 are skipped — Fleet
// host IDs are AUTO_INCREMENT starting at 1.
func NewBitmap(ids []uint) *roaring.Bitmap {
	rb := roaring.New()
	for _, id := range ids {
		if id == 0 {
			continue
		}
		rb.Add(uint32(id))
	}
	rb.RunOptimize()
	return rb
}

// BitmapToBlob serializes a *roaring.Bitmap into the storage form. Always
// returns Encoding = EncodingRoaring. Calls RunOptimize defensively (safe to
// invoke multiple times) so callers do not need to remember to do so.
// Bitmaps with cardinality 0 serialize to a nil byte slice.
func BitmapToBlob(rb *roaring.Bitmap) Blob {
	if rb == nil || rb.IsEmpty() {
		return Blob{Encoding: EncodingRoaring}
	}
	rb.RunOptimize()
	return Blob{Bytes: serializeBitmap(rb), Encoding: EncodingRoaring}
}

// serializeBitmap wraps Bitmap.ToBytes; isolated so the encoder path has a
// single call site if we ever swap serialization formats.
func serializeBitmap(rb *roaring.Bitmap) []byte {
	out, err := rb.ToBytes()
	if err != nil {
		// Bitmap.ToBytes only errors on internal buffer issues that aren't
		// reachable for in-memory bitmaps; treat as a programmer error.
		panic("chart: roaring.Bitmap.ToBytes failed: " + err.Error())
	}
	return out
}

// HostIDsToBlob is the convenience composition of NewBitmap + BitmapToBlob for
// callers going directly from a host-id list to storage form. Empty input
// returns Blob{Bytes: nil, Encoding: EncodingRoaring}.
func HostIDsToBlob(ids []uint) Blob {
	return BitmapToBlob(NewBitmap(ids))
}

// hostIDsToDenseBlob is the pre-change dense encoder, retained for tests and
// for constructing legacy-row fixtures in the migration tests. Production
// writes go through HostIDsToBlob (which produces roaring) instead.
func hostIDsToDenseBlob(ids []uint) []byte {
	if len(ids) == 0 {
		return nil
	}
	var maxID uint
	for _, id := range ids {
		if id > maxID {
			maxID = id
		}
	}
	blob := make([]byte, maxID/8+1)
	for _, id := range ids {
		blob[id/8] |= 1 << (id % 8)
	}
	return blob
}

// DecodeBitmap converts storage form to op form. Dispatches on Blob.Encoding:
// roaring blobs are deserialized via the library; legacy dense blobs are
// walked byte-by-byte and each set bit added to a fresh roaring bitmap.
// A nil or empty Bytes slice returns an empty bitmap regardless of Encoding.
// An unknown encoding value returns an error.
func DecodeBitmap(b Blob) (*roaring.Bitmap, error) {
	if len(b.Bytes) == 0 {
		return roaring.New(), nil
	}
	switch b.Encoding {
	case EncodingRoaring:
		rb := roaring.New()
		if _, err := rb.FromBuffer(b.Bytes); err != nil {
			return nil, err
		}
		return rb, nil
	case EncodingDense:
		return decodeDense(b.Bytes), nil
	default:
		return nil, errUnknownEncoding(b.Encoding)
	}
}

// decodeDense walks a dense bitmap byte-by-byte and inserts each set bit's
// position as a uint32 into a fresh roaring bitmap. O(byte count) work.
func decodeDense(blob []byte) *roaring.Bitmap {
	rb := roaring.New()
	for i, byteVal := range blob {
		if byteVal == 0 {
			continue
		}
		base := uint32(i) * 8
		for bit := uint32(0); bit < 8; bit++ {
			if byteVal&(1<<bit) != 0 {
				rb.Add(base + bit)
			}
		}
	}
	return rb
}

type errUnknownEncoding uint8

func (e errUnknownEncoding) Error() string {
	return "chart: unknown bitmap encoding " + hexByte(uint8(e))
}

func hexByte(v uint8) string {
	const hex = "0123456789abcdef"
	return "0x" + string([]byte{hex[v>>4], hex[v&0xF]})
}

// BitmapToHostIDs returns the set bits of a *roaring.Bitmap as a sorted []uint.
// Thin convenience over roaring.Bitmap.ToArray (which returns []uint32) for
// callers that work in uint at the Fleet boundary.
func BitmapToHostIDs(rb *roaring.Bitmap) []uint {
	if rb == nil {
		return nil
	}
	arr := rb.ToArray()
	out := make([]uint, len(arr))
	for i, v := range arr {
		out[i] = uint(v)
	}
	return out
}

// BlobPopcount returns the cardinality of the bitmap. A nil bitmap is treated
// as the empty set.
func BlobPopcount(rb *roaring.Bitmap) uint64 {
	if rb == nil {
		return 0
	}
	return rb.GetCardinality()
}

// BlobAND returns the intersection of a and b as a new bitmap. nil operands
// are treated as the empty set; the result is the empty set.
func BlobAND(a, b *roaring.Bitmap) *roaring.Bitmap {
	if a == nil || b == nil {
		return roaring.New()
	}
	return roaring.And(a, b)
}

// BlobOR returns the union of a and b as a new bitmap. nil operands are
// treated as the empty set.
func BlobOR(a, b *roaring.Bitmap) *roaring.Bitmap {
	switch {
	case a == nil && b == nil:
		return roaring.New()
	case a == nil:
		return b.Clone()
	case b == nil:
		return a.Clone()
	}
	return roaring.Or(a, b)
}

// BlobANDNOT returns a \ mask: the bits set in a but not in mask, as a new
// bitmap. nil a returns the empty set; nil mask returns a clone of a.
func BlobANDNOT(a, mask *roaring.Bitmap) *roaring.Bitmap {
	if a == nil {
		return roaring.New()
	}
	if mask == nil {
		return a.Clone()
	}
	return roaring.AndNot(a, mask)
}
