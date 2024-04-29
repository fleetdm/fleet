package file

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/sassoftware/relic/v7/lib/comdoc"
)

func ExtractMSIMetadata(r io.Reader) (name, version string, shaSum []byte, err error) {
	h := sha256.New()
	r = io.TeeReader(r, h)
	b, err := io.ReadAll(r)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to read all content: %w", err)
	}

	rr := bytes.NewReader(b)
	c, err := comdoc.ReadFile(rr)
	if err != nil {
		return "", "", nil, fmt.Errorf("reading msi file: %w", err)
	}
	defer c.Close()

	e, err := c.ListDir(nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("listing files in msi: %w", err)
	}

	var dataReader, poolReader, colReader io.Reader
	for _, ee := range e {
		if ee.Type != comdoc.DirStream {
			continue
		}

		name := msiDecodeName(ee.Name())
		fmt.Println(name, ee.Type)

		if name == "Table._StringData" || name == "Table._StringPool" || name == "Table._Columns" {
			rr, err := c.ReadStream(ee)
			if err != nil {
				return "", "", nil, fmt.Errorf("opening file stream %s: %w", name, err)
			}
			if name == "Table._StringData" {
				dataReader = rr
			} else if name == "Table._Columns" {
				colReader = rr
			} else if name == "Table._StringPool" {
				poolReader = rr
			}
		}
	}
	allStrings, err := buildStringsTable(dataReader, poolReader)
	if err != nil {
		return "", "", nil, err
	}
	tables, err := buildColumnsTable(colReader, allStrings)
	if err != nil {
		return "", "", nil, err
	}
	_ = tables

	return "", "", h.Sum(nil), nil
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

func buildColumnsTable(colReader io.Reader, strings []string) ([]msiTable, error) {
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

	indexedTables := make(map[uint16]msiTable)
	for i := 0; i < rowCount; i++ {
		tblID, colNum, colNameID, colAttr := cols[0][i], cols[1][i], cols[2][i], cols[3][i]

		tbl := indexedTables[tblID]
		tbl.Name = strings[tblID-1]
		tbl.Cols = append(tbl.Cols, msiColumn{
			Number:     int(colNum),
			Name:       strings[colNameID-1],
			Attributes: colAttr,
		})
		indexedTables[tblID] = tbl
	}

	tables := make([]msiTable, 0, len(indexedTables))
	for _, tbl := range indexedTables {
		tables = append(tables, tbl)
		fmt.Println(">>> found ", tbl)
	}
	fmt.Println(">>> found ", len(tables), ", tables")
	return tables, nil
}

func buildStringsTable(dataReader, poolReader io.Reader) ([]string, error) {
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
	for {
		err := binary.Read(poolReader, binary.LittleEndian, &stringEntry)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read pool entry: %w", err)
		}
		buf := make([]byte, stringEntry.Size)
		if _, err := io.ReadFull(dataReader, buf); err != nil {
			return nil, fmt.Errorf("failed to read string data: %w", err)
		}
		stringTable = append(stringTable, string(buf))
	}
	return stringTable, nil
}

func msiDecodeName(msiName string) string {
	out := ""
	for _, x := range msiName {
		if x >= 0x3800 && x < 0x4800 {
			x -= 0x3800
			out += string(msiDecodeRune(x&0x3f)) + string(msiDecodeRune(x>>6))
		} else if x >= 0x4800 && x < 0x4840 {
			x -= 0x4800
			out += string(msiDecodeRune(x))
		} else if x == 0x4840 {
			out += "Table."
		} else {
			out += string(x)
		}
	}
	return out
}

func msiDecodeRune(x rune) rune {
	if x < 10 {
		return x + '0'
	} else if x < 10+26 {
		return x - 10 + 'A'
	} else if x < 10+26+26 {
		return x - 10 - 26 + 'a'
	} else if x == 10+26+26 {
		return '.'
	} else {
		return '_'
	}
}
