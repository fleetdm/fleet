package file

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/sassoftware/relic/v8/lib/comdoc"
)

func ExtractMSIMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	// compute its hash
	h := sha256.New()
	_, _ = io.Copy(h, tfr) // writes to a hash cannot fail

	if err := tfr.Rewind(); err != nil {
		return nil, err
	}

	c, err := comdoc.ReadFile(tfr)
	if err != nil {
		return nil, fmt.Errorf("reading msi file: %w", err)
	}
	defer c.Close()

	e, err := c.ListDir(nil)
	if err != nil {
		return nil, fmt.Errorf("listing files in msi: %w", err)
	}

	// the product name and version are stored in the Property table, but the
	// strings are interned in the _StringData table (which requires the
	// _StringPool to decode). The structure of the tables is found in the
	// _Columns table.
	targetedTables := map[string]io.Reader{
		"Table._StringData": nil,
		"Table._StringPool": nil,
		"Table._Columns":    nil,
		"Table.Property":    nil,
	}
	for _, ee := range e {
		if ee.Type != comdoc.DirStream {
			continue
		}

		name := msiDecodeName(ee.Name())
		if _, ok := targetedTables[name]; ok {
			rr, err := c.ReadStream(ee)
			if err != nil {
				return nil, fmt.Errorf("opening file stream %s: %w", name, err)
			}
			targetedTables[name] = rr
		}
	}

	// all tables must've been found
	for k, v := range targetedTables {
		if v == nil {
			return nil, fmt.Errorf("table %s not found in the .msi", k)
		}
	}

	allStrings, err := decodeStrings(targetedTables["Table._StringData"], targetedTables["Table._StringPool"])
	if err != nil {
		return nil, err
	}
	propTbl, err := decodePropertyTableColumns(targetedTables["Table._Columns"], allStrings)
	if err != nil {
		return nil, err
	}
	props, err := decodePropertyTable(targetedTables["Table.Property"], propTbl, allStrings)
	if err != nil {
		return nil, err
	}

	// MSI installer product information properties: https://learn.microsoft.com/en-us/windows/win32/msi/property-reference#product-information-properties
	return &InstallerMetadata{
		Name:       strings.TrimSpace(props["ProductName"]),
		Version:    strings.TrimSpace(props["ProductVersion"]),
		PackageIDs: []string{strings.TrimSpace(props["ProductCode"])},
		SHASum:     h.Sum(nil),
	}, nil
}

type msiTable struct {
	Name string
	Cols []msiColumn
}

type msiColumn struct {
	Number     int
	Name       string
	Attributes uint16
}

func (c msiColumn) Type() msiType {
	if c.Attributes&0x0F00 < 0x800 {
		return msiType(c.Attributes & 0xFFF)
	}
	return msiType(c.Attributes & 0xF00)
}

type msiType uint16

// column types
const (
	msiLong            msiType = 0x104
	msiShort           msiType = 0x502
	msiBinary          msiType = 0x900
	msiString          msiType = 0xD00
	msiStringLocalized msiType = 0xF00
	msiUnknown         msiType = 0
)

func decodePropertyTable(propReader io.Reader, table *msiTable, strings []string) (map[string]string, error) {
	// The Property table is a table of key-value pairs. Ensure the table has the
	// expected format, otherwise we cannot extract the information.
	if len(table.Cols) != 2 || table.Cols[0].Type() != msiString || table.Cols[1].Type() != msiStringLocalized {
		return nil, errors.New("unexpected Property table structure")
	}

	const propTableRowSize = 4 // 2 uint16s

	b, err := io.ReadAll(propReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read columns table: %w", err)
	}
	rowCount := len(b) / propTableRowSize
	propReader = bytes.NewReader(b)

	cols := [][]uint16{
		make([]uint16, 0, rowCount),
		make([]uint16, 0, rowCount),
	}
	for i := 0; i < 2; i++ {
		for j := 0; j < rowCount; j++ {
			var v uint16
			err := binary.Read(propReader, binary.LittleEndian, &v)
			if err != nil {
				return nil, fmt.Errorf("failed to read column %d: %w", i, err)
			}
			cols[i] = append(cols[i], v)
		}
	}

	kv := make(map[string]string, rowCount)
	for i := 0; i < rowCount; i++ {
		kv[strings[cols[0][i]-1]] = strings[cols[1][i]-1]
	}
	return kv, nil
}

func decodePropertyTableColumns(colReader io.Reader, strings []string) (*msiTable, error) {
	const colTableRowSize = 8 // 4 uint16s

	// Columns table has 4 columns:
	// - table name id (1-based index in strings array)
	// - col number
	// - col name id (1-based index in strings array)
	// - col attributes (type)
	//
	// But to make things interesting, those are stored per column, so all first
	// columns are stored for all rows, then all second columns for all rows,
	// etc.

	b, err := io.ReadAll(colReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read columns table: %w", err)
	}
	rowCount := len(b) / colTableRowSize
	colReader = bytes.NewReader(b)

	cols := [][]uint16{
		make([]uint16, 0, rowCount),
		make([]uint16, 0, rowCount),
		make([]uint16, 0, rowCount),
		make([]uint16, 0, rowCount),
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < rowCount; j++ {
			var v uint16
			err := binary.Read(colReader, binary.LittleEndian, &v)
			if err != nil {
				return nil, fmt.Errorf("failed to read column %d: %w", i, err)
			}
			cols[i] = append(cols[i], v)
		}
	}

	var tbl msiTable
	for i := 0; i < rowCount; i++ {
		tblID, colNum, colNameID, colAttr := cols[0][i], cols[1][i], cols[2][i], cols[3][i]

		tableName := strings[tblID-1]
		if tableName == "Property" {
			tbl.Name = tableName
			tbl.Cols = append(tbl.Cols, msiColumn{
				Number:     int(colNum),
				Name:       strings[colNameID-1],
				Attributes: colAttr,
			})
		}
	}
	if tbl.Name == "" {
		return nil, errors.New("Property table not found in columns table")
	}
	return &tbl, nil
}

func decodeStrings(dataReader, poolReader io.Reader) ([]string, error) {
	type header struct {
		Codepage uint16
		Unknown  uint16
	}
	var poolHeader header
	// pool data starts with 2 uint16 for the codepage and an unknown value
	err := binary.Read(poolReader, binary.LittleEndian, &poolHeader)
	if err != nil {
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("failed to read pool header: %w", err)
	}

	type entry struct {
		Size     uint16
		RefCount uint16
	}
	var stringEntry entry
	var stringTable []string
	var buf bytes.Buffer
	for {
		err := binary.Read(poolReader, binary.LittleEndian, &stringEntry)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read pool entry: %w", err)
		}
		stringEntrySize := int(stringEntry.Size)

		// For string pool entries too long for the size to fit in a single uint16, entry size is 8 bytes instead of 4,
		// with the first two bytes as zeroes, the next two are the two most-significant bytes of the size, shifted
		// 17 (?!?) bits to the left, the following two are the less-significant bits of the size, and the last two are
		// the reference count. Verified with the OpenVPN Connect v3 installer, which has a large string blob for
		// licenses where a 17-bit shift captures the length properly.
		if stringEntry.Size == 0 && stringEntry.RefCount != 0 {
			stringEntrySize = int(stringEntry.RefCount) << 17
			err := binary.Read(poolReader, binary.LittleEndian, &stringEntry)
			if err != nil {
				return nil, fmt.Errorf("failed to read large string pool entry: %w", err)
			}
			stringEntrySize += int(stringEntry.Size)
		}

		buf.Reset()
		buf.Grow(stringEntrySize)
		_, err = io.CopyN(&buf, dataReader, int64(stringEntrySize))
		if err != nil {
			return nil, fmt.Errorf("failed to read string data: %w", err)
		}
		stringTable = append(stringTable, buf.String())
	}
	return stringTable, nil
}

func msiDecodeName(msiName string) string {
	out := ""
	for _, x := range msiName {
		switch {
		case x >= 0x3800 && x < 0x4800:
			x -= 0x3800
			out += string(msiDecodeRune(x&0x3f)) + string(msiDecodeRune(x>>6))
		case x >= 0x4800 && x < 0x4840:
			x -= 0x4800
			out += string(msiDecodeRune(x))
		case x == 0x4840:
			out += "Table."
		default:
			out += string(x)
		}
	}
	return out
}

func msiDecodeRune(x rune) rune {
	if x < 10 { //nolint:gocritic // ignore ifElseChain
		return x + '0'
	} else if x < 10+26 {
		return x - 10 + 'A'
	} else if x < 10+26+26 {
		return x - 10 - 26 + 'a'
	} else if x == 10+26+26 {
		return '.'
	}

	return '_'
}
