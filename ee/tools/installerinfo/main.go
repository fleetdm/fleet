// Command installerinfo ...
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		log.Fatal("at least one package path must be provided")
	}

	for _, path := range args {
		if err := printInfo(path); err != nil {
			log.Fatal(err)
		}
	}
}

func printInfo(path string) error {
	var name, version string
	var outerr error

	if strings.HasSuffix(path, ".app") {
		name, version, outerr = file.GetAppInfo(path)
		if outerr != nil {
			return fmt.Errorf("extracting info: %w", outerr)
		}
		pprint(path, name, version)
		return nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file contents: %w", err)
	}

	reader := bytes.NewReader(contents)
	br := bufio.NewReader(reader)
	switch {
	case hasPrefix(br, []byte{0x78, 0x61, 0x72, 0x21}):
		name, version, outerr = file.GetXarInfo(reader, int64(len(contents)))
	case hasPrefix(br, []byte("!<arch>\ndebian")):
		name, version, outerr = file.GetMSIInfo(reader, int64(len(contents)))
	case hasPrefix(br, []byte{0xd0, 0xcf}):
		name, version, outerr = file.GetMSIInfo(reader, int64(len(contents)))
	case hasPrefix(br, []byte("MZ")):
		if blob, _ := br.Peek(0x3e); len(blob) == 0x3e {
			reloc := binary.LittleEndian.Uint16(blob[0x3c:0x3e])
			if blob, err := br.Peek(int(reloc) + 4); err == nil {
				if bytes.Equal(blob[reloc:reloc+4], []byte("PE\x00\x00")) {
					name, version, outerr = file.GetPEInfo(contents)
					break
				}
			}
		}
		fallthrough
	default:
		return errors.New("unsupported file type")
	}

	if outerr != nil {
		return fmt.Errorf("extracting info: %w", outerr)
	}

	pprint(path, name, version)

	return nil
}

func pprint(path, name, version string) {
	fmt.Printf(`
File:    %s
Name:    %s
Version: %s

`, path, name, version)

}

func hasPrefix(br *bufio.Reader, blob []byte) bool {
	d, _ := br.Peek(len(blob))
	if len(d) < len(blob) {
		return false
	}
	return bytes.Equal(d, blob)
}
