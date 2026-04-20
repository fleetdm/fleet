package msi

import (
	"bytes"
	"encoding/binary"
)

// MSI column type constants (on-disk format, verified against production MSI reader).
// The type field in _Columns is: width(bits 0-7) | flags(bits 8-15).
// For integers, width = byte size (2 or 4).
// For strings, width = max character length (0 = unlimited, 72 = s72, 255 = l255).
const (
	colTypeLong  uint16 = 0x0104 // 32-bit integer, width=4
	colTypeShort uint16 = 0x0502 // 16-bit integer, width=2

	// String base types (add max-length in lower byte when defining columns).
	colString          uint16 = 0x0D00 // String (non-localizable)
	colStringLocalized uint16 = 0x0F00 // Localizable string
	colBinary          uint16 = 0x0900 // Binary stream reference

	// Column attribute modifier bits.
	colNullable   uint16 = 0x1000
	colPrimaryKey uint16 = 0x2000
)

// Helper functions to create column types with proper string widths.
func colStr(maxLen uint8) uint16    { return colString | uint16(maxLen) }          // s<N> (e.g. s72)
func colStrL(maxLen uint8) uint16   { return colStringLocalized | uint16(maxLen) } // l<N> (e.g. l255)
func colStrPK(maxLen uint8) uint16  { return colStr(maxLen) | colPrimaryKey }      // s<N> primary key
func colStrN(maxLen uint8) uint16   { return colStr(maxLen) | colNullable }        // S<N> nullable
func colStrLN(maxLen uint8) uint16  { return colStrL(maxLen) | colNullable }       // L<N> nullable

// ColumnDef defines a single column in an MSI table.
type ColumnDef struct {
	Name string
	Type uint16 // Base type (colType*) | optional modifiers (colNullable, colPrimaryKey)
}

// TableSchema defines the schema for an MSI table.
type TableSchema struct {
	Name    string
	Columns []ColumnDef
}

// TableData holds the rows for one MSI table. Each row is a slice of cell
// values, one per column. Cell types:
//   - string: encoded as a string pool index
//   - int16/int32: encoded with XOR masking
//   - nil: NULL value
//   - []byte: binary stream data (handled separately)
type TableData struct {
	Schema TableSchema
	Rows   [][]any
}

// isStringType returns true if the column type is string-based.
func isStringType(colType uint16) bool {
	base := colType & 0x0F00
	return base == colString || base == colStringLocalized || base == colBinary
}

// isLongType returns true if the column type is a 32-bit integer.
func isLongType(colType uint16) bool {
	return colType&0x0F00 == (colTypeLong & 0x0F00)
}

// EncodeTableData encodes a table's row data in MSI column-major format.
// String values are resolved through the provided string pool.
// Returns the raw bytes for the table's data stream.
func EncodeTableData(td *TableData, pool *StringPool) []byte {
	if len(td.Rows) == 0 {
		return nil
	}

	numCols := len(td.Schema.Columns)
	numRows := len(td.Rows)

	var buf bytes.Buffer

	// Column-major: for each column, write all rows' values for that column.
	for col := range numCols {
		colDef := td.Schema.Columns[col]
		for row := range numRows {
			cell := td.Rows[row][col]
			writeCell(&buf, cell, colDef.Type, pool)
		}
	}

	return buf.Bytes()
}

// writeCell writes a single cell value in MSI binary format.
func writeCell(buf *bytes.Buffer, cell any, colType uint16, pool *StringPool) {
	if isStringType(colType) {
		// String/binary column: write as uint16 string pool index.
		var idx uint16
		if cell != nil {
			if s, ok := cell.(string); ok && s != "" {
				idx = pool.Add(s)
			}
		}
		binary.Write(buf, binary.LittleEndian, idx) //nolint:errcheck
		return
	}

	if isLongType(colType) {
		// Long (int32): XOR with 0x80000000. NULL = 0.
		var val uint32
		if cell != nil {
			switch v := cell.(type) {
			case int32:
				val = uint32(v) ^ 0x80000000 //nolint:gosec // G115
			case int:
				val = uint32(int32(v)) ^ 0x80000000 //nolint:gosec // G115
			}
		}
		binary.Write(buf, binary.LittleEndian, val) //nolint:errcheck
		return
	}

	// Short (int16): XOR with 0x8000. NULL = 0.
	var val uint16
	if cell != nil {
		switch v := cell.(type) {
		case int16:
			val = uint16(v) ^ 0x8000 //nolint:gosec // G115
		case int:
			val = uint16(int16(v)) ^ 0x8000 //nolint:gosec // G115
		}
	}
	binary.Write(buf, binary.LittleEndian, val) //nolint:errcheck
}

// EncodeTablesStream produces the _Tables stream: a list of table names
// as string pool indices (uint16 each), one per table.
func EncodeTablesStream(tables []*TableData, pool *StringPool) []byte {
	var buf bytes.Buffer
	for _, t := range tables {
		idx := pool.Add(t.Schema.Name)
		binary.Write(&buf, binary.LittleEndian, idx) //nolint:errcheck
	}
	return buf.Bytes()
}

// EncodeColumnsStream produces the _Columns stream: 4 columns per row
// (tableName, colNumber, colName, colAttributes), all in column-major order.
func EncodeColumnsStream(tables []*TableData, pool *StringPool) []byte {
	// Count total rows.
	totalRows := 0
	for _, t := range tables {
		totalRows += len(t.Schema.Columns)
	}

	// Prepare column arrays (4 columns, each with totalRows entries).
	tableNameIDs := make([]uint16, 0, totalRows)
	colNumbers := make([]uint16, 0, totalRows)
	colNameIDs := make([]uint16, 0, totalRows)
	colAttrs := make([]uint16, 0, totalRows)

	for _, t := range tables {
		for i, col := range t.Schema.Columns {
			tableNameIDs = append(tableNameIDs, pool.Add(t.Schema.Name))
			colNumbers = append(colNumbers, uint16(i+1)^0x8000) //nolint:gosec // G115
			colNameIDs = append(colNameIDs, pool.Add(col.Name))
			colAttrs = append(colAttrs, col.Type^0x8000)
		}
	}

	// Write in column-major order.
	var buf bytes.Buffer
	for _, v := range tableNameIDs {
		binary.Write(&buf, binary.LittleEndian, v) //nolint:errcheck
	}
	for _, v := range colNumbers {
		binary.Write(&buf, binary.LittleEndian, v) //nolint:errcheck
	}
	for _, v := range colNameIDs {
		binary.Write(&buf, binary.LittleEndian, v) //nolint:errcheck
	}
	for _, v := range colAttrs {
		binary.Write(&buf, binary.LittleEndian, v) //nolint:errcheck
	}

	return buf.Bytes()
}
