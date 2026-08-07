package apple_mdm

import (
	"encoding/binary"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoundedPlistUnmarshalBinary(t *testing.T) {
	t.Run("flat dictionary decodes", func(t *testing.T) {
		data := buildFlatBinaryPlist(t, [][2]string{
			{"SERIAL", "ABC123"},
			{"UDID", "0000-1111"},
		})

		var info fleet.MDMAppleMachineInfo
		require.NoError(t, BoundedPlistUnmarshal(data, &info))
		assert.Equal(t, "ABC123", info.Serial)
		assert.Equal(t, "0000-1111", info.UDID)
	})

	t.Run("nesting beyond the depth limit is rejected", func(t *testing.T) {
		data := buildRefChain(t, 24)
		require.Less(t, len(data), 200)

		var info fleet.MDMAppleMachineInfo
		err := BoundedPlistUnmarshal(data, &info)
		require.ErrorIs(t, err, errPlistTooComplex)
	})

	t.Run("self-referential object is rejected", func(t *testing.T) {
		// Object 0 is an array referencing itself (a cycle).
		data := buildBinaryPlist(t, []byte{0xa2, 0x00, 0x00})

		var info fleet.MDMAppleMachineInfo
		err := BoundedPlistUnmarshal(data, &info)
		require.ErrorIs(t, err, errPlistTooComplex)
	})

	t.Run("string length beyond input is rejected", func(t *testing.T) {
		// ASCII string declaring a length larger than the input contains.
		obj := []byte{0x5f, 0x13, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}
		data := buildBinaryPlist(t, obj)

		var info fleet.MDMAppleMachineInfo
		err := BoundedPlistUnmarshal(data, &info)
		require.ErrorIs(t, err, errMalformedPlist)
	})

	t.Run("real size beyond input is rejected", func(t *testing.T) {
		// Real marker 0x2f declares more inline bytes than the input contains.
		data := buildBinaryPlist(t, []byte{0x2f})

		var info fleet.MDMAppleMachineInfo
		err := BoundedPlistUnmarshal(data, &info)
		require.ErrorIs(t, err, errMalformedPlist)
	})

	t.Run("object count beyond the limit is rejected", func(t *testing.T) {
		data := buildBinaryPlist(t, []byte{0x08}) // bool false
		trailer := data[len(data)-plistTrailerSize:]
		binary.BigEndian.PutUint64(trailer[8:16], maxPlistObjects+1)

		var info fleet.MDMAppleMachineInfo
		err := BoundedPlistUnmarshal(data, &info)
		require.ErrorIs(t, err, errPlistTooComplex)
	})
}

func TestBoundedPlistUnmarshalXML(t *testing.T) {
	// XML plists are not reference-encoded, so bounds checking is skipped.
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>SERIAL</key>
	<string>XML-SERIAL</string>
</dict>
</plist>`)

	var info fleet.MDMAppleMachineInfo
	require.NoError(t, BoundedPlistUnmarshal(xml, &info))
	assert.Equal(t, "XML-SERIAL", info.Serial)
}

// buildBinaryPlist wraps a single root object (placed immediately after the
// header at index 0) with a one-byte offset table and trailer.
func buildBinaryPlist(t *testing.T, rootObject []byte) []byte {
	t.Helper()
	body := append([]byte("bplist00"), rootObject...)
	offsetTableOffset := len(body)
	body = appendByte(body, len("bplist00")) // offset of object 0
	return append(body, makeTrailer(1, offsetTableOffset)...)
}

// buildRefChain lays out n array objects, each referencing the next object
// twice, terminated by a single leaf object.
func buildRefChain(t *testing.T, n int) []byte {
	t.Helper()
	require.Less(t, n, 250, "single-byte refs require < 250 objects")

	body := []byte("bplist00")
	offsets := make([]int, 0, n+1)
	for i := range n {
		offsets = append(offsets, len(body))
		body = appendByte(body, 0xa2) // array of 2 refs to the next object
		body = appendByte(body, i+1)
		body = appendByte(body, i+1)
	}
	offsets = append(offsets, len(body))
	body = appendByte(body, 0x08) // leaf: bool false

	offsetTableOffset := len(body)
	for _, off := range offsets {
		body = appendByte(body, off)
	}
	return append(body, makeTrailer(len(offsets), offsetTableOffset)...)
}

// buildFlatBinaryPlist builds a binary plist of a single dictionary whose keys
// and values are all short ASCII strings.
func buildFlatBinaryPlist(t *testing.T, pairs [][2]string) []byte {
	t.Helper()
	n := len(pairs)
	require.Less(t, n, 15, "builder uses inline dict counts")

	body := []byte("bplist00")
	offsets := []int{len(body)}

	// Object 0: dict. Key refs are objects 1..n, value refs are n+1..2n.
	body = appendByte(body, 0xd0|n)
	for i := range n {
		body = appendByte(body, 1+i)
	}
	for i := range n {
		body = appendByte(body, 1+n+i)
	}

	appendStr := func(s string) {
		require.Less(t, len(s), 15, "builder uses inline string counts")
		offsets = append(offsets, len(body))
		body = appendByte(body, 0x50|len(s))
		body = append(body, []byte(s)...)
	}
	for _, p := range pairs {
		appendStr(p[0])
	}
	for _, p := range pairs {
		appendStr(p[1])
	}

	offsetTableOffset := len(body)
	for _, off := range offsets {
		body = appendByte(body, off)
	}
	return append(body, makeTrailer(len(offsets), offsetTableOffset)...)
}

// makeTrailer builds a 32-byte trailer with single-byte offsets and refs.
func makeTrailer(numObjects, offsetTableOffset int) []byte {
	trailer := make([]byte, plistTrailerSize)
	trailer[6] = 1 // offset int size
	trailer[7] = 1 // object ref size
	putUint64(trailer[8:16], numObjects)
	putUint64(trailer[16:24], 0) // root object index
	putUint64(trailer[24:32], offsetTableOffset)
	return trailer
}

// appendByte and putUint64 keep the fixture's narrowing conversions in one place.
func appendByte(body []byte, v int) []byte {
	return append(body, byte(v)) //nolint:gosec // dismiss G115, fixture values are below 256
}

func putUint64(dst []byte, v int) {
	binary.BigEndian.PutUint64(dst, uint64(v)) //nolint:gosec // dismiss G115, fixture values are bounded
}
