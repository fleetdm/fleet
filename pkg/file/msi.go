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

	var dataReader, poolReader io.Reader
	for _, ee := range e {
		if ee.Type != comdoc.DirStream {
			continue
		}

		name := msiDecodeName(ee.Name())
		fmt.Println(name, ee.Type)

		if name == "Table._StringData" || name == "Table._StringPool" {
			rr, err := c.ReadStream(ee)
			if err != nil {
				return "", "", nil, fmt.Errorf("opening file stream %s: %w", name, err)
			}
			if name == "Table._StringData" {
				dataReader = rr
			} else {
				poolReader = rr
			}
		}
	}
	allStrings, err := buildStringsTable(dataReader, poolReader)
	if err != nil {
		return "", "", nil, err
	}
	_ = allStrings

	return "", "", h.Sum(nil), nil
}

func buildStringsTable(dataReader, poolReader io.Reader) (map[string]int, error) {
	type entry struct {
		Size     uint16
		RefCount uint16
	}
	var stringEntry entry
	stringTable := make(map[string]int)
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
		if stringEntry.RefCount > 0 {
			stringTable[string(buf)] = int(stringEntry.RefCount)
			fmt.Println(">>> found: ", string(buf), stringEntry.RefCount)
		}
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
