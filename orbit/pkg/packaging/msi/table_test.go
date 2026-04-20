package msi

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeTableData_Property(t *testing.T) {
	// Build a Property table (2 string columns) and verify round-trip
	// with the decoder from pkg/file/msi.go.
	pool := NewStringPool()

	schema := TableSchema{
		Name: "Property",
		Columns: []ColumnDef{
			{Name: "Property", Type: colStrPK(72)},
			{Name: "Value", Type: colStrL(255)},
		},
	}

	td := &TableData{
		Schema: schema,
		Rows: [][]any{
			{"ProductName", "Fleet osquery"},
			{"ProductVersion", "1.0.0"},
			{"Manufacturer", "Fleet Device Management"},
		},
	}

	data := EncodeTableData(td, pool)
	require.NotEmpty(t, data)

	// Decode: the Property table has 2 uint16 columns, so each row is 4 bytes.
	// Column-major: first 3 uint16s are column 1, then 3 uint16s for column 2.
	reader := bytes.NewReader(data)
	rowCount := len(data) / 4 // 2 columns × 2 bytes each
	assert.Equal(t, 3, rowCount)

	cols := [2][]uint16{{}, {}}
	for i := range 2 {
		for range rowCount {
			var v uint16
			require.NoError(t, binary.Read(reader, binary.LittleEndian, &v))
			cols[i] = append(cols[i], v)
		}
	}

	// Decode string pool.
	poolData := pool.EncodePool()
	stringData := pool.EncodeData()
	allStrings, err := testDecodeStrings(bytes.NewReader(stringData), bytes.NewReader(poolData))
	require.NoError(t, err)
	require.NotEmpty(t, allStrings)

	// Verify: row 0 = ("ProductName", "Fleet osquery")
	assert.Equal(t, "ProductName", allStrings[cols[0][0]-1])
	assert.Equal(t, "Fleet osquery", allStrings[cols[1][0]-1])

	// Verify: row 1 = ("ProductVersion", "1.0.0")
	assert.Equal(t, "ProductVersion", allStrings[cols[0][1]-1])
	assert.Equal(t, "1.0.0", allStrings[cols[1][1]-1])
}

func TestEncodeTableData_IntegerColumns(t *testing.T) {
	pool := NewStringPool()

	schema := TableSchema{
		Name: "TestInt",
		Columns: []ColumnDef{
			{Name: "Name", Type: colStrPK(72)},
			{Name: "Value", Type: colTypeLong},
		},
	}

	td := &TableData{
		Schema: schema,
		Rows: [][]any{
			{"key1", int32(42)},
			{"key2", int32(0)},
			{"key3", int32(-1)},
		},
	}

	data := EncodeTableData(td, pool)
	require.NotEmpty(t, data)

	// Column 1: 3 × uint16 (string indices) = 6 bytes
	// Column 2: 3 × uint32 (XOR-masked int32) = 12 bytes
	// Total: 18 bytes
	assert.Len(t, data, 18)

	reader := bytes.NewReader(data)

	// Skip string column (3 × uint16 = 6 bytes).
	_, err := reader.Seek(6, io.SeekStart)
	require.NoError(t, err)

	// Read int32 values (XOR 0x80000000).
	var v1, v2, v3 uint32
	require.NoError(t, binary.Read(reader, binary.LittleEndian, &v1))
	require.NoError(t, binary.Read(reader, binary.LittleEndian, &v2))
	require.NoError(t, binary.Read(reader, binary.LittleEndian, &v3))

	assert.Equal(t, int32(42), int32(v1^0x80000000))  //nolint:gosec // G115: XOR mask result is intentional for MSI integer decoding
	assert.Equal(t, int32(0), int32(v2^0x80000000))  //nolint:gosec // G115: XOR mask result is intentional for MSI integer decoding
	assert.Equal(t, int32(-1), int32(v3^0x80000000)) //nolint:gosec // G115: XOR mask result is intentional for MSI integer decoding
}

func TestEncodeColumnsStream(t *testing.T) {
	pool := NewStringPool()

	tables := []*TableData{
		{
			Schema: TableSchema{
				Name: "Property",
				Columns: []ColumnDef{
					{Name: "Property", Type: colStrPK(72)},
					{Name: "Value", Type: colStrL(255)},
				},
			},
		},
	}

	data := EncodeColumnsStream(tables, pool)
	require.NotEmpty(t, data)

	// 2 columns × 4 fields × 2 bytes = 16 bytes in column-major format.
	// But column-major across 4 meta-columns: 2 rows × 4 columns = 8 uint16s = 16 bytes.
	assert.Len(t, data, 16)

	// Decode and verify.
	reader := bytes.NewReader(data)
	// 4 meta-columns, 2 rows each.
	vals := make([][]uint16, 4)
	for i := range 4 {
		for range 2 {
			var v uint16
			require.NoError(t, binary.Read(reader, binary.LittleEndian, &v))
			vals[i] = append(vals[i], v)
		}
	}

	// Column 0 (table name): both should be "Property" index.
	assert.Equal(t, vals[0][0], vals[0][1]) // same table

	// Column 1 (col number): short integers XOR-masked with 0x8000.
	assert.Equal(t, uint16(1)^0x8000, vals[1][0])
	assert.Equal(t, uint16(2)^0x8000, vals[1][1])

	// Column 3 (attributes): short integers XOR-masked with 0x8000.
	assert.Equal(t, colStrPK(72)^0x8000, vals[3][0])
	assert.Equal(t, colStrL(255)^0x8000, vals[3][1])
}

func TestEncodeTablesStream(t *testing.T) {
	pool := NewStringPool()

	tables := []*TableData{
		{Schema: TableSchema{Name: "Property"}},
		{Schema: TableSchema{Name: "File"}},
	}

	data := EncodeTablesStream(tables, pool)
	assert.Len(t, data, 4) // 2 × uint16
}
