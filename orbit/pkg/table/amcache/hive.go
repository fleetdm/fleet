//go:build windows

package amcache

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"www.velocidex.com/golang/regparser"
)

// hiveResult is the acquired hive. raw is set when the hive was locked and had to be read
// from the NTFS volume; replayed is set when the transaction logs were applied.
type hiveResult struct {
	data     []byte
	raw      bool
	replayed bool
}

// Caps on every allocation driven by a file-declared size. orbit runs as SYSTEM and a local
// user can influence what the appraiser writes.
const (
	maxHiveBytes = 256 << 20
	maxLogBytes  = 64 << 20
)

var errHiveNotFound = errors.New("amcache: hive not found")

// hiveDirty reports whether the hive still needs its transaction logs applied. A bad base
// block checksum also counts as dirty in the regf spec, but on a live hive that means a torn
// read that the logs cannot repair, so it is not checked.
func hiveDirty(hive []byte) (bool, error) {
	reg, err := regparser.NewRegistry(bytes.NewReader(hive))
	if err != nil {
		return false, fmt.Errorf("amcache: base block: %w", err)
	}
	return reg.BaseBlock.Sequence1() != reg.BaseBlock.Sequence2(), nil
}

// replayLogs applies the dirty pages in Amcache.hve.LOG1/.LOG2 to a copy of the hive. It
// doesn't use regparser's own recovery helper, which writes to os.TempDir(), prints to stdout
// and loops forever on a zero-size log entry.
func replayLogs(hive []byte, logs ...[]byte) ([]byte, error) {
	if len(hive) > maxHiveBytes {
		return nil, fmt.Errorf("amcache: hive is %d bytes, over the %d byte cap", len(hive), maxHiveBytes)
	}
	for i, l := range logs {
		if len(l) > maxLogBytes {
			return nil, fmt.Errorf("amcache: log %d is %d bytes, over the %d byte cap", i, len(l), maxLogBytes)
		}
	}
	if len(hive) < 0x1000 {
		return nil, fmt.Errorf("amcache: hive is %d bytes, shorter than the 4096 byte base block", len(hive))
	}

	out := append([]byte(nil), hive...)

	base, err := regparser.NewRegistry(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("amcache: base block: %w", err)
	}
	baseSeq2 := base.BaseBlock.Sequence2()

	type applicable struct {
		buf []byte
		seq uint32
	}
	var ready []applicable
	for _, l := range logs {
		if len(l) == 0 {
			continue
		}
		lr, err := regparser.NewRegistry(bytes.NewReader(l))
		if err != nil {
			continue
		}
		// Types 1 and 2 are the pre-Windows 8.1 log formats, which have no HvLE entries.
		if t := lr.BaseBlock.Type(); t == 1 || t == 2 {
			continue
		}
		// >= rather than ==: the logs are circular and keep running while the primary file stays
		// dirty, so a live hive is routinely many sequences behind its logs.
		if lr.BaseBlock.Sequence1() < baseSeq2 {
			continue
		}
		ready = append(ready, applicable{buf: l, seq: lr.BaseBlock.Sequence1()})
	}
	if len(ready) > 2 {
		return nil, fmt.Errorf("amcache: %d applicable logs, a primary hive has at most 2", len(ready))
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i].seq < ready[j].seq })

	profile := regparser.NewRegistryProfile()
	var (
		applied  bool
		seq      uint32
		hbins    uint32
		flags    uint32
		expected uint32
	)
	for i, lg := range ready {
		r := bytes.NewReader(lg.buf)
		// The sequence chain continues across logs instead of restarting at each log's base, so
		// an overlapping second log can't put an older page image on top of a newer one. Jumping
		// forward to a later log's base is allowed because the circular logs recycle entries.
		if i == 0 || lg.seq > expected {
			expected = lg.seq
		}
		for off := int64(0x200); off < int64(len(lg.buf)); {
			le := &regparser.HIVE_LOG_ENTRY{Reader: r, Offset: off, Profile: profile}
			if le.Signature() != 0x454C7648 { // "HvLE"
				break
			}
			size := int64(le.LogEntrySize())
			if size < 40 || size%512 != 0 || off+size > int64(len(lg.buf)) {
				break
			}
			// regparser preallocates DirtyPagesCount() slots, so bound the count first. Each ref
			// is 8 bytes plus at least one 4096-byte page in the same entry.
			count := int64(le.DirtyPagesCount())
			if 40+count*(8+4096) > size {
				break
			}
			if le.HiveBinsDataSize()%4096 != 0 {
				break
			}
			if le.SequenceNumber() != expected {
				break
			}

			// Validate every page before applying any: a half-applied entry would leave a hive
			// that mixes two generations but still passes the dirty check after the header
			// rewrite below. A torn tail is normal for a log read from under the appraiser, so
			// stop at the last valid entry instead of failing.
			pages := le.GetDirtyPages()
			torn := false
			for _, p := range pages {
				if p.PageSize == 0 || p.PageSize%4096 != 0 || p.DataOffset+int64(p.PageSize) > off+size {
					torn = true
					break
				}
			}
			if torn {
				break
			}

			for _, p := range pages {
				data, err := p.Data()
				if err != nil {
					return nil, fmt.Errorf("amcache: dirty page at %#x: %w", p.DataOffset, err)
				}
				dst := int64(0x1000) + int64(p.PageOffset)
				if dst+int64(p.PageSize) > maxHiveBytes {
					return nil, fmt.Errorf("amcache: dirty page at %#x would grow the hive past the %d byte cap", dst, maxHiveBytes)
				}
				if need := dst + int64(p.PageSize); need > int64(len(out)) {
					out = append(out, make([]byte, need-int64(len(out)))...)
				}
				copy(out[dst:], data)
			}

			seq, hbins, flags = le.SequenceNumber(), le.HiveBinsDataSize(), le.Flags()
			applied = true
			expected++
			off += size
		}
	}

	if applied {
		// hbins comes from the log and bounds the hive bin walk, so it must match the bytes
		// actually recovered.
		if int64(hbins) > int64(len(out))-0x1000 {
			return nil, fmt.Errorf("amcache: replayed hive declares %d bytes of hive bins data but holds %d",
				hbins, int64(len(out))-0x1000)
		}
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], seq)
		copy(out[4:8], b[:])
		copy(out[8:12], b[:])
		binary.LittleEndian.PutUint32(b[:], hbins)
		copy(out[0x28:0x2c], b[:])
		binary.LittleEndian.PutUint32(b[:], flags)
		copy(out[0x90:0x94], b[:])
		binary.LittleEndian.PutUint32(b[:], baseBlockChecksum(out))
		copy(out[0x1fc:0x200], b[:])
	}

	return out, nil
}

// baseBlockChecksum is the XOR of the little-endian uint32s in the first 0x1fc bytes, with 0
// and 0xFFFFFFFF reserved.
func baseBlockChecksum(buf []byte) uint32 {
	var c uint32
	for i := 0; i < 0x1fc; i += 4 {
		c ^= binary.LittleEndian.Uint32(buf[i : i+4])
	}
	switch c {
	case 0:
		return 1
	case 0xFFFFFFFF:
		return 0xFFFFFFFE
	}
	return c
}
