package file

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeStringsMemoryAmplification(t *testing.T) {
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
	const claimedSize = 64 * 1024 * 1024                                    // 64 MB
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
	runtime.ReadMemStats(&after)

	// With the fix, TotalAlloc should increase by well under 1 MB.
	// Without the fix, it would jump by ~64 MB from the speculative Grow call.
	allocated := after.TotalAlloc - before.TotalAlloc
	const maxAllowed = 1024 * 1024 // 1 MB
	require.Less(t, allocated, uint64(maxAllowed),
		"decodeStrings allocated %d bytes; expected less than %d (memory amplification detected)", allocated, maxAllowed)
}

func TestDecodeMSITablesRejectOutOfRangeStringIDs(t *testing.T) {
	le := func(vals ...uint16) *bytes.Reader {
		var b bytes.Buffer
		require.NoError(t, binary.Write(&b, binary.LittleEndian, vals))
		return bytes.NewReader(b.Bytes())
	}
	strs := []string{"Property", "Name", "Value", "ProductName", "App"}

	t.Run("columns", func(t *testing.T) {
		// column-major rows: table id, column number, column name id, attributes
		tbl, err := decodePropertyTableColumns(le(1, 1, 2, 3, 1, 2, 0xD00, 0xF00), strs)
		require.NoError(t, err)
		require.Len(t, tbl.Cols, 2)

		for _, tc := range []struct {
			name string
			data []uint16
		}{
			{"table id zero", []uint16{0, 0, 1, 2, 2, 3, 0xD00, 0xF00}},
			{"table id past end", []uint16{1, 6, 1, 2, 2, 3, 0xD00, 0xF00}},
			{"column name id zero", []uint16{1, 1, 1, 2, 0, 3, 0xD00, 0xF00}},
			{"column name id past end", []uint16{1, 1, 1, 2, 2, 600, 0xD00, 0xF00}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := decodePropertyTableColumns(le(tc.data...), strs)
				require.ErrorIs(t, err, ErrInvalidInstaller)
			})
		}
	})

	t.Run("property", func(t *testing.T) {
		tbl := &msiTable{Cols: []msiColumn{{Number: 1, Attributes: 0xD00}, {Number: 2, Attributes: 0xF00}}}

		kv, err := decodePropertyTable(le(4, 5), tbl, strs)
		require.NoError(t, err)
		require.Equal(t, map[string]string{"ProductName": "App"}, kv)

		for name, data := range map[string][]uint16{
			"key id zero":       {0, 5},
			"key id past end":   {6, 5},
			"value id zero":     {4, 0},
			"value id past end": {4, 65535},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := decodePropertyTable(le(data...), tbl, strs)
				require.ErrorIs(t, err, ErrInvalidInstaller)
			})
		}
	})
}
