//go:build windows

package amcache

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"www.velocidex.com/golang/regparser"
)

// hiveResult is the acquired hive. after is the base block read again from the source once
// data was complete. raw is set when the hive was locked and had to be read from the NTFS volume.
type hiveResult struct {
	data  []byte
	after []byte
	raw   bool
}

// baseBlockSize is the regf header that holds the sequence numbers.
const baseBlockSize = 0x1000

// maxHiveBytes caps the read. orbit runs as SYSTEM and a local user can influence what the
// appraiser writes.
const maxHiveBytes = 256 << 20

var (
	errHiveNotFound = errors.New("amcache: hive not found")
	// The transaction logs aren't replayed: dirty hives are rare (Windows is mid-write, or
	// crashed mid-write) and a wrong replay yields plausible but incorrect rows. Once Windows
	// finishes writing, the hive's mtime/size change invalidates the cached error.
	errHiveDirty = errors.New("amcache: hive is being written by Windows; try again shortly")
)

// hiveDirty reports whether Windows wrote to the hive while it was being read. Windows bumps
// Sequence1 before writing any page and sets Sequence2 to match once done, so the copy is dirty
// if its two numbers differ, or if the base block read afterwards no longer matches the copy's:
// a write that started after the header was read leaves a clean-looking header over pages from
// two generations.
func hiveDirty(res hiveResult) (bool, error) {
	reg, err := regparser.NewRegistry(bytes.NewReader(res.data))
	if err != nil {
		return false, fmt.Errorf("amcache: base block: %w", err)
	}
	seq1, seq2 := reg.BaseBlock.Sequence1(), reg.BaseBlock.Sequence2()
	if seq1 != seq2 {
		return true, nil
	}
	if len(res.after) < 12 || !bytes.HasPrefix(res.after, []byte("regf")) {
		return false, errors.New("amcache: base block read after the hive is not a regf header")
	}
	return binary.LittleEndian.Uint32(res.after[4:]) != seq1 || binary.LittleEndian.Uint32(res.after[8:]) != seq2, nil
}
