package file

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeStringsMemoryAmplification(t *testing.T) {
	t.Parallel()

	// Build a _StringPool that claims a single string of 64 MB.
	// The pool format is: 4-byte header (codepage + unknown), then 4-byte entries (size uint16 + refcount uint16).
	var pool bytes.Buffer

	// Pool header: codepage=0, unknown=0
	require.NoError(t, binary.Write(&pool, binary.LittleEndian, uint16(0))) // codepage
	require.NoError(t, binary.Write(&pool, binary.LittleEndian, uint16(0))) // unknown

	// One entry claiming a huge size. The "large string" path is triggered
	// when Size==0 and RefCount!=0, then reads a uint32 for the actual size.
	require.NoError(t, binary.Write(&pool, binary.LittleEndian, uint16(0))) // Size=0 triggers large-string path
	require.NoError(t, binary.Write(&pool, binary.LittleEndian, uint16(1))) // RefCount!=0
	const claimedSize = 64 * 1024 * 1024                                   // 64 MB
	require.NoError(t, binary.Write(&pool, binary.LittleEndian, uint32(claimedSize)))

	// _StringData is empty: zero actual bytes of string data.
	var data bytes.Buffer

	// Measure memory before
	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	// decodeStrings should fail because there is no data to read,
	// but it must NOT allocate 64 MB first.
	_, err := decodeStrings(&data, &pool)
	require.Error(t, err, "expected an error because string data is empty")

	// Measure memory after
	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)

	// With the fix, TotalAlloc should increase by well under 1 MB.
	// Without the fix, it would jump by ~64 MB from the speculative Grow call.
	allocated := after.TotalAlloc - before.TotalAlloc
	const maxAllowed = 1024 * 1024 // 1 MB
	require.Less(t, allocated, uint64(maxAllowed),
		"decodeStrings allocated %d bytes; expected less than %d (memory amplification detected)", allocated, maxAllowed)
}
