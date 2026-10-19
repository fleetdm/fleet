//go:build windows

package amcache

import (
	"bytes"
	"errors"
	"fmt"

	"www.velocidex.com/golang/regparser"
)

// hiveResult is the acquired hive. raw is set when the hive was locked and had to be read
// from the NTFS volume.
type hiveResult struct {
	data []byte
	raw  bool
}

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

// hiveDirty reports whether the hive's base block sequence numbers differ, meaning Windows
// started writing to the primary file and hasn't finished.
func hiveDirty(hive []byte) (bool, error) {
	reg, err := regparser.NewRegistry(bytes.NewReader(hive))
	if err != nil {
		return false, fmt.Errorf("amcache: base block: %w", err)
	}
	return reg.BaseBlock.Sequence1() != reg.BaseBlock.Sequence2(), nil
}
