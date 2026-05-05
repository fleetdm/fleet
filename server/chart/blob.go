// Package chart provides blob utility helpers, dataset implementations, and
// shared constants for the chart bounded context. Public API types live in
// server/chart/api; internal types (HostFilter, Datastore) live in
// server/chart/internal/types.
package chart

import (
	"encoding/binary"
	"math/bits"
)

// HostIDsToBlob builds a byte slice with bits set at positions corresponding to
// the given host IDs. Bit N of the blob = host ID N.
func HostIDsToBlob(ids []uint) []byte {
	if len(ids) == 0 {
		return nil
	}

	// Find the max ID to size the blob.
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

// BlobPopcount returns the number of set bits in the blob.
func BlobPopcount(blob []byte) int {
	count := 0
	// Process 8 bytes at a time for performance.
	i := 0
	for ; i+8 <= len(blob); i += 8 {
		v := binary.LittleEndian.Uint64(blob[i : i+8])
		count += bits.OnesCount64(v)
	}
	for ; i < len(blob); i++ {
		count += bits.OnesCount8(blob[i])
	}
	return count
}

// BlobAND returns a new blob that is the bitwise AND of a and b.
// The result length is min(len(a), len(b)) — bits beyond the shorter blob are implicitly zero.
func BlobAND(a, b []byte) []byte {
	if a == nil || b == nil {
		return nil
	}
	n := min(len(a), len(b))
	if n == 0 {
		return nil
	}
	result := make([]byte, n)
	a = a[:n]
	b = b[:n]
	for i := range n {
		result[i] = a[i] & b[i] //nolint:gosec // a and b are bounded to n via slicing above
	}
	return result
}

// BlobANDNOT returns a new blob equal to a with the bits set in mask cleared.
// Result length is len(a). If mask is shorter than a, it zero-extends — high
// bytes of a pass through unchanged. If mask is longer than a, the excess
// bytes of mask are ignored.
func BlobANDNOT(a, mask []byte) []byte {
	if len(a) == 0 {
		return nil
	}
	out := make([]byte, len(a))
	n := min(len(a), len(mask))
	bitsToMask := a[:n]
	sizedMask := mask[:n]
	for i := range n {
		out[i] = bitsToMask[i] &^ sizedMask[i] //nolint:gosec // bitsToMask and sizedMask are bounded to n via slicing above
	}
	// If mask is shorter than a, copy the remaining high bytes unchanged.
	if n < len(a) {
		copy(out[n:], a[n:])
	}
	return out
}

// BlobOR returns a new blob that is the bitwise OR of a and b.
// The result length is max(len(a), len(b)) — the shorter blob is zero-extended.
func BlobOR(a, b []byte) []byte {
	long, short := a, b
	if len(b) > len(a) {
		long, short = b, a
	}
	if len(long) == 0 {
		return nil
	}
	result := make([]byte, len(long))
	copy(result, long)
	for i := range short {
		result[i] |= short[i]
	}
	return result
}
