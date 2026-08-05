package msi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
)

// database.go implements the MSI installer-database encoding: the string
// pool (!_StringPool / !_StringData), the catalog streams (!_Tables,
// !_Columns), and the per-table row streams.
//
// String IDs are assigned in first-use order. To stay byte-compatible with
// WiX output, rows must be fed to the database in the same order WiX inserts
// them (see model.go); rows are then physically stored sorted by their
// primary key columns, where string cells compare by string ID and integer
// cells by value — exactly how Windows Installer stores them, which is why
// msidump shows rows in that order.

type stringPool struct {
	ids     map[string]int
	strings []string
	refs    []int
}

func newStringPool() *stringPool {
	// The pool always starts with an unused empty entry at ID 1 (WiX/msi.dll
	// reserve it); null string cells are stored as ID 0.
	return &stringPool{ids: map[string]int{}, strings: []string{""}, refs: []int{0}}
}

// add interns s and returns its ID (adding a reference). The empty string is
// ID 0 (stored as a null cell, never in the pool).
func (p *stringPool) add(s string) int {
	if s == "" {
		return 0
	}
	id, ok := p.ids[s]
	if !ok {
		p.strings = append(p.strings, s)
		p.refs = append(p.refs, 0)
		id = len(p.strings) // IDs are 1-based
		p.ids[s] = id
	}
	p.refs[id-1]++
	return id
}

// streams serializes the pool. All fleetd strings are ASCII; anything else
// would change the codepage handling, so reject it.
func (p *stringPool) streams() (poolStream, dataStream []byte, err error) {
	if len(p.strings) > 0xFFFF {
		return nil, nil, fmt.Errorf("too many strings in MSI database: %d (max 65535)", len(p.strings))
	}
	var pool, data bytes.Buffer
	// header: codepage 0 (neutral), flags 0
	pool.Write([]byte{0, 0, 0, 0})
	le := binary.LittleEndian
	for i, s := range p.strings {
		if !isASCII(s) {
			return nil, nil, fmt.Errorf("non-ASCII string in MSI database: %q", s)
		}
		var entry [4]byte
		if len(s) >= 1<<16 {
			// long string: zero size with the real size in the next slot
			le.PutUint16(entry[2:], uint16(p.refs[i])) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
			pool.Write(entry[:])
			var size [4]byte
			le.PutUint32(size[:], uint32(len(s))) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
			pool.Write(size[:])
		} else {
			le.PutUint16(entry[0:], uint16(len(s)))    //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
			le.PutUint16(entry[2:], uint16(p.refs[i])) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
			pool.Write(entry[:])
		}
		data.WriteString(s)
	}
	return pool.Bytes(), data.Bytes(), nil
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7F {
			return false
		}
	}
	return true
}

// cell is one table cell: a string, an integer, or nil.
type cell any

type table struct {
	schema tableSchema
	rows   [][]cell
}

// database accumulates tables and interns row strings in insertion order.
type database struct {
	pool   *stringPool
	tables map[string]*table
}

func newDatabase() *database {
	db := &database{pool: newStringPool(), tables: map[string]*table{}}
	for _, s := range fleetdSchemas {
		db.tables[s.name] = &table{schema: s}
	}
	return db
}

// insert appends a row (cells in column order) and interns its strings
// left to right, mirroring the WiX/msi.dll insertion behavior that
// determines string IDs.
func (db *database) insert(tableName string, cells ...cell) error {
	t, ok := db.tables[tableName]
	if !ok {
		return fmt.Errorf("unknown table %q", tableName)
	}
	if len(cells) != len(t.schema.cols) {
		return fmt.Errorf("table %s: got %d cells, want %d", tableName, len(cells), len(t.schema.cols))
	}
	for i, c := range cells {
		col := t.schema.cols[i]
		switch v := c.(type) {
		case nil:
		case string:
			if !col.isString() {
				return fmt.Errorf("table %s column %s: string cell in integer column", tableName, col.name)
			}
			if !isASCII(v) {
				// The string pool is written with a neutral codepage;
				// non-ASCII values would need codepage handling that the
				// fleetd installer has never used (WiX builds had the same
				// effective limitation).
				return fmt.Errorf("table %s column %s: value %q contains non-ASCII characters, which the fleetd MSI does not support", tableName, col.name, v)
			}
			db.pool.add(v)
		case int:
			if col.isString() && !col.isObject() {
				return fmt.Errorf("table %s column %s: integer cell in string column", tableName, col.name)
			}
		default:
			return fmt.Errorf("table %s column %s: unsupported cell type %T", tableName, col.name, c)
		}
	}
	t.rows = append(t.rows, cells)
	return nil
}

// sortRows orders a table's rows by its key columns (string cells by pool
// ID, integer cells by value), which is the physical row order of the table
// stream.
func (db *database) sortRows(t *table) {
	keyIdx := []int{}
	for i, c := range t.schema.cols {
		if c.isKey() {
			keyIdx = append(keyIdx, i)
		}
	}
	cellOrd := func(c cell, col column) int64 {
		switch v := c.(type) {
		case string:
			return int64(db.pool.ids[v])
		case int:
			return int64(v)
		}
		return 0
	}
	sort.SliceStable(t.rows, func(a, b int) bool {
		for _, k := range keyIdx {
			av := cellOrd(t.rows[a][k], t.schema.cols[k])
			bv := cellOrd(t.rows[b][k], t.schema.cols[k])
			if av != bv {
				return av < bv
			}
		}
		return false
	})
}

// encodeTable serializes a table stream: cells stored column-major, strings
// as 16-bit pool IDs, 16-bit integers biased by 0x8000, 32-bit integers
// biased by 0x80000000, and nulls as zero.
func (db *database) encodeTable(t *table) []byte {
	var out bytes.Buffer
	le := binary.LittleEndian
	for ci, col := range t.schema.cols {
		for _, row := range t.rows {
			c := row[ci]
			if col.isObject() {
				// object (stream) columns store a raw index, in practice 1
				var b [2]byte
				if v, ok := c.(int); ok {
					le.PutUint16(b[:], uint16(v)) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
				}
				out.Write(b[:])
				continue
			}
			if col.isString() {
				id := 0
				if s, ok := c.(string); ok {
					id = db.pool.ids[s]
				}
				var b [2]byte
				le.PutUint16(b[:], uint16(id))
				out.Write(b[:])
				continue
			}
			switch col.cellBytes() {
			case 2:
				var b [2]byte
				if v, ok := c.(int); ok {
					le.PutUint16(b[:], uint16(v)^0x8000) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
				}
				out.Write(b[:])
			case 4:
				var b [4]byte
				if v, ok := c.(int); ok {
					le.PutUint32(b[:], uint32(v)^0x80000000) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
				}
				out.Write(b[:])
			}
		}
	}
	return out.Bytes()
}

// isRealTable reports whether the schema entry is stored as a table stream
// (i.e. everything except _SummaryInformation, which lives in its own
// property-set stream but still gets _Validation rows).
func isRealTable(name string) bool {
	return name != "_SummaryInformation"
}

// catalogStreams builds the !_Tables and !_Columns streams. Rows are ordered
// by table-name string ID (and column number), matching how msi.dll stores
// its catalog.
func (db *database) catalogStreams() (tablesStream, columnsStream []byte) {
	le := binary.LittleEndian

	names := []string{}
	for _, s := range fleetdSchemas {
		if isRealTable(s.name) {
			names = append(names, s.name)
		}
	}
	sort.Slice(names, func(a, b int) bool {
		return db.pool.ids[names[a]] < db.pool.ids[names[b]]
	})

	var tb bytes.Buffer
	for _, n := range names {
		var b [2]byte
		le.PutUint16(b[:], uint16(db.pool.ids[n])) //nolint:gosec // pool IDs fit in 16 bits (enforced by table cell encoding)
		tb.Write(b[:])
	}

	// _Columns rows: (Table, Number, Name, Type), column-major, ordered by
	// (table string ID, column number).
	type colRow struct {
		tableID int
		num     int
		nameID  int
		typ     uint16
	}
	var rows []colRow
	for _, n := range names {
		t, ok := db.tables[n]
		if !ok {
			continue
		}
		for i, c := range t.schema.cols {
			rows = append(rows, colRow{db.pool.ids[n], i + 1, db.pool.ids[c.name], c.typ})
		}
	}
	sort.SliceStable(rows, func(a, b int) bool {
		if rows[a].tableID != rows[b].tableID {
			return rows[a].tableID < rows[b].tableID
		}
		return rows[a].num < rows[b].num
	})
	var cb bytes.Buffer
	writeCol := func(get func(colRow) uint16) {
		for _, r := range rows {
			var b [2]byte
			le.PutUint16(b[:], get(r))
			cb.Write(b[:])
		}
	}
	writeCol(func(r colRow) uint16 { return uint16(r.tableID) })      //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
	writeCol(func(r colRow) uint16 { return uint16(r.num) ^ 0x8000 }) //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
	writeCol(func(r colRow) uint16 { return uint16(r.nameID) })       //nolint:gosec // MSI cell encoding; IDs and biased integers are defined modulo 2^16/2^32
	writeCol(func(r colRow) uint16 { return r.typ })

	return tb.Bytes(), cb.Bytes()
}

// addValidationRows inserts the _Validation rows describing every table's
// columns, in schema order. WiX inserts these before any data rows, which is
// what gives schema strings their low pool IDs.
func (db *database) addValidationRows() error {
	// Creating the _Validation table itself interns its name and column
	// names first, matching WiX's pool layout ("_Validation", "Table",
	// "Column", ... get the lowest IDs).
	db.pool.add("_Validation")
	validation, ok := db.tables["_Validation"]
	if !ok {
		return fmt.Errorf("missing _Validation schema")
	}
	for _, c := range validation.schema.cols {
		db.pool.add(c.name)
	}
	for _, s := range fleetdSchemas {
		for _, c := range s.cols {
			var minV, maxV, keyC cell
			if c.minValue != nil {
				minV = int(*c.minValue)
			}
			if c.maxValue != nil {
				maxV = int(*c.maxValue)
			}
			if c.keyColumn != nil {
				keyC = int(*c.keyColumn)
			}
			cells := []cell{
				s.name, c.name, c.nullable, minV, maxV,
				nilIfEmpty(c.keyTable), keyC, nilIfEmpty(c.category), nilIfEmpty(c.set), nilIfEmpty(c.description),
			}
			if err := db.insert("_Validation", cells...); err != nil {
				return err
			}
		}
	}
	return nil
}

func nilIfEmpty(s string) cell {
	if s == "" {
		return nil
	}
	return s
}

// databaseStream is one named stream of the final database.
type databaseStream struct {
	Name string // logical name; "!" prefix marks a table stream
	Data []byte
}

// streams serializes the database in the stream order WiX produces:
// !_Tables, !_StringPool, !_StringData, then table streams in reverse
// schema-name order, then !_Validation and !_Columns.
func (db *database) streams() ([]databaseStream, error) {
	for _, t := range db.tables {
		db.sortRows(t)
	}

	poolStream, dataStream, err := db.pool.streams()
	if err != nil {
		return nil, err
	}
	tablesStream, columnsStream := db.catalogStreams()

	var out []databaseStream
	out = append(out,
		databaseStream{"!_Tables", tablesStream},
		databaseStream{"!_StringPool", poolStream},
		databaseStream{"!_StringData", dataStream},
	)

	// Data tables in reverse name order (observed WiX/msi.dll commit
	// order), excluding _Validation which lands at the end.
	names := []string{}
	for _, s := range fleetdSchemas {
		if isRealTable(s.name) && s.name != "_Validation" {
			names = append(names, s.name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, n := range append(names, "_Validation") {
		t, ok := db.tables[n]
		if !ok {
			return nil, fmt.Errorf("missing table %q", n)
		}
		if n == "_Validation" {
			out = append(out, databaseStream{"!_Validation", db.encodeTable(t)})
		} else {
			out = append(out, databaseStream{"!" + n, db.encodeTable(t)})
		}
	}
	out = append(out, databaseStream{"!_Columns", columnsStream})
	return out, nil
}
