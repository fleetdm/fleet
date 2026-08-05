package msi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"unicode/utf16"
)

// cfb.go implements a writer for the Compound File Binary (MS-CFB, also
// known as OLE structured storage) format, version 3 (512-byte sectors),
// which is the container format of an MSI. Only the features an MSI needs
// are implemented: a root storage with a flat list of streams (MSI databases
// have no sub-storages).

const (
	cfbSectorSize     = 512
	cfbMiniSectorSize = 64
	cfbMiniCutoff     = 4096

	cfbFreeSect     = 0xFFFFFFFF
	cfbEndOfChain   = 0xFFFFFFFE
	cfbFATSect      = 0xFFFFFFFD
	cfbDIFATSect    = 0xFFFFFFFC
	cfbNoStream     = 0xFFFFFFFF
	cfbDirEntrySize = 128
)

// msiRootCLSID is the COM class of an MSI database, stamped on the root
// directory entry ({000C1084-0000-0000-C000-000000000046}).
var msiRootCLSID = [16]byte{0x84, 0x10, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}

// cfbStream is a named stream to store. Name is the raw stream name
// (already encoded with the MSI name encoding where applicable). The order
// of the streams slice determines directory entry allocation order, which is
// how WiX-produced MSIs are laid out (observably: system streams first, then
// data streams in creation order).
type cfbStream struct {
	Name string
	Data []byte
}

// writeCFB writes a version 3 compound file with the given streams under the
// root storage.
func writeCFB(w io.Writer, streams []cfbStream) error {
	// Directory entries: root + one per stream, in creation order.
	type dirEntry struct {
		name               [32]uint16 // UTF-16, NUL-terminated
		nameLen            uint16     // bytes including terminator
		objectType         byte
		color              byte
		left, right, child uint32
		clsid              [16]byte
		startSector        uint32
		size               uint64
	}

	entries := make([]dirEntry, 1+len(streams))
	root := &entries[0]
	setName := func(e *dirEntry, name string) error {
		codes := utf16.Encode([]rune(name))
		if len(codes) > 31 {
			return fmt.Errorf("stream name too long: %q", name)
		}
		copy(e.name[:], codes)
		e.nameLen = uint16(2 * (len(codes) + 1)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
		return nil
	}
	if err := setName(root, "Root Entry"); err != nil {
		return err
	}
	root.objectType = 5 // root storage
	root.clsid = msiRootCLSID
	root.left, root.right, root.child = cfbNoStream, cfbNoStream, cfbNoStream

	for i, s := range streams {
		e := &entries[i+1]
		if err := setName(e, s.Name); err != nil {
			return err
		}
		e.objectType = 2 // stream
		e.left, e.right, e.child = cfbNoStream, cfbNoStream, cfbNoStream
		e.size = uint64(len(s.Data))
	}

	// Build the red-black name tree for the root storage's children.
	// CFB orders names by length first, then by uppercased UTF-16 code
	// units. We build a balanced BST from the sorted list and color nodes
	// red on the deepest level so black-height is uniform.
	order := make([]int, len(streams)) // entry indices (1-based into entries)
	for i := range order {
		order[i] = i + 1
	}
	nameKey := func(idx int) []uint16 {
		e := &entries[idx]
		n := int(e.nameLen)/2 - 1
		key := make([]uint16, n)
		for i := range n {
			c := e.name[i]
			// uppercase the UTF-16 code unit as CFB comparison demands
			if c >= 'a' && c <= 'z' {
				c -= 'a' - 'A'
			}
			key[i] = c
		}
		return key
	}
	sort.SliceStable(order, func(a, b int) bool {
		ka, kb := nameKey(order[a]), nameKey(order[b])
		if len(ka) != len(kb) {
			return len(ka) < len(kb)
		}
		for i := range ka {
			if ka[i] != kb[i] {
				return ka[i] < kb[i]
			}
		}
		return false
	})

	// depth of a complete tree holding n nodes
	fullDepth := 0
	for n := len(order); n > 1; n >>= 1 {
		fullDepth++
	}
	var build func(lo, hi, depth int) uint32
	build = func(lo, hi, depth int) uint32 {
		if lo >= hi {
			return cfbNoStream
		}
		mid := (lo + hi) / 2
		idx := order[mid]
		e := &entries[idx]
		e.left = build(lo, mid, depth+1)
		e.right = build(mid+1, hi, depth+1)
		if depth == fullDepth {
			e.color = 0 // red on the deepest (possibly incomplete) level
		} else {
			e.color = 1 // black
		}
		return uint32(idx) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
	}
	root.child = build(0, len(order), 0)
	root.color = 1

	// Partition stream data into the mini stream and regular sectors.
	// The mini stream itself is stored in regular sectors and owned by the
	// root entry.
	var miniData bytes.Buffer
	var miniFAT []uint32
	for i, s := range streams {
		e := &entries[i+1]
		if len(s.Data) == 0 {
			e.startSector = cfbEndOfChain
			continue
		}
		if len(s.Data) < cfbMiniCutoff {
			nsec := (len(s.Data) + cfbMiniSectorSize - 1) / cfbMiniSectorSize
			e.startSector = uint32(len(miniFAT)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
			for j := 0; j < nsec-1; j++ {
				miniFAT = append(miniFAT, uint32(len(miniFAT))+1) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
			}
			miniFAT = append(miniFAT, cfbEndOfChain)
			miniData.Write(s.Data)
			if pad := (cfbMiniSectorSize - miniData.Len()%cfbMiniSectorSize) % cfbMiniSectorSize; pad > 0 {
				miniData.Write(make([]byte, pad))
			}
		}
	}
	root.size = uint64(miniData.Len()) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size

	// Compute the regular-sector layout. Sector order:
	//   [FAT sectors][DIFAT sectors][directory][miniFAT][mini stream][streams >= 4KB]
	// Sizes of FAT/DIFAT depend on the total sector count, so solve
	// iteratively.
	sectorsFor := func(n int) int { return (n + cfbSectorSize - 1) / cfbSectorSize }

	numDirSectors := sectorsFor((1 + len(streams)) * cfbDirEntrySize)
	numMiniFATSectors := sectorsFor(len(miniFAT) * 4)
	numMiniSectors := sectorsFor(miniData.Len())
	numStreamSectors := 0
	for _, s := range streams {
		if len(s.Data) >= cfbMiniCutoff {
			numStreamSectors += sectorsFor(len(s.Data))
		}
	}
	payloadSectors := numDirSectors + numMiniFATSectors + numMiniSectors + numStreamSectors

	numFAT, numDIFAT := 0, 0
	for {
		total := payloadSectors + numFAT + numDIFAT
		needFAT := sectorsFor(total * 4)
		needDIFAT := 0
		if needFAT > 109 {
			// each DIFAT sector holds 127 FAT sector references
			needDIFAT = (needFAT - 109 + 126) / 127
		}
		if needFAT == numFAT && needDIFAT == numDIFAT {
			break
		}
		numFAT, numDIFAT = needFAT, needDIFAT
	}
	totalSectors := payloadSectors + numFAT + numDIFAT

	fat := make([]uint32, numFAT*(cfbSectorSize/4))
	for i := range fat {
		fat[i] = cfbFreeSect
	}
	next := 0
	alloc := func(n int, mark uint32) int {
		start := next
		for range n {
			fat[next] = mark
			next++
		}
		return start
	}
	chain := func(n int) int {
		start := next
		for i := range n {
			if i == n-1 {
				fat[next] = cfbEndOfChain
			} else {
				fat[next] = uint32(next + 1) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
			}
			next++
		}
		return start
	}

	fatStart := alloc(numFAT, cfbFATSect)
	difatStart := 0
	if numDIFAT > 0 {
		difatStart = alloc(numDIFAT, cfbDIFATSect)
	}
	dirStart := chain(numDirSectors)
	miniFATStart := uint32(cfbEndOfChain)
	if numMiniFATSectors > 0 {
		miniFATStart = uint32(chain(numMiniFATSectors)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
	}
	if numMiniSectors > 0 {
		root.startSector = uint32(chain(numMiniSectors)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
	} else {
		root.startSector = cfbEndOfChain
	}
	for i, s := range streams {
		if len(s.Data) >= cfbMiniCutoff {
			entries[i+1].startSector = uint32(chain(sectorsFor(len(s.Data)))) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
		}
	}
	if next != totalSectors {
		return fmt.Errorf("internal error: cfb sector layout mismatch (%d != %d)", next, totalSectors)
	}

	// Header.
	hdr := make([]byte, cfbSectorSize)
	copy(hdr, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1})
	le := binary.LittleEndian
	le.PutUint16(hdr[24:], 0x003E) // minor version
	le.PutUint16(hdr[26:], 0x0003) // major version 3
	le.PutUint16(hdr[28:], 0xFFFE) // byte order
	le.PutUint16(hdr[30:], 9)      // sector shift (512)
	le.PutUint16(hdr[32:], 6)      // mini sector shift (64)
	// number of directory sectors: always 0 for v3
	le.PutUint32(hdr[44:], uint32(numFAT))
	le.PutUint32(hdr[48:], uint32(dirStart)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
	le.PutUint32(hdr[56:], cfbMiniCutoff)
	le.PutUint32(hdr[60:], uint32(miniFATStart))
	le.PutUint32(hdr[64:], uint32(numMiniFATSectors)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
	if numDIFAT > 0 {
		le.PutUint32(hdr[68:], uint32(difatStart))
	} else {
		le.PutUint32(hdr[68:], cfbEndOfChain)
	}
	le.PutUint32(hdr[72:], uint32(numDIFAT))
	for i := range 109 {
		if i < numFAT {
			le.PutUint32(hdr[76+4*i:], uint32(fatStart+i)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
		} else {
			le.PutUint32(hdr[76+4*i:], cfbFreeSect)
		}
	}
	if _, err := w.Write(hdr); err != nil {
		return err
	}

	buf := make([]byte, cfbSectorSize)

	// FAT sectors.
	for i := range numFAT {
		for j := range cfbSectorSize / 4 {
			le.PutUint32(buf[4*j:], fat[i*(cfbSectorSize/4)+j])
		}
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}

	// DIFAT sectors.
	for i := range numDIFAT {
		for j := range 127 {
			idx := 109 + i*127 + j
			if idx < numFAT {
				le.PutUint32(buf[4*j:], uint32(fatStart+idx)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
			} else {
				le.PutUint32(buf[4*j:], cfbFreeSect)
			}
		}
		if i == numDIFAT-1 {
			le.PutUint32(buf[508:], cfbEndOfChain)
		} else {
			le.PutUint32(buf[508:], uint32(difatStart+i+1))
		}
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}

	// Directory sectors.
	var dirBuf bytes.Buffer
	for i := range entries {
		e := &entries[i]
		var eb [cfbDirEntrySize]byte
		for j := range 32 {
			le.PutUint16(eb[2*j:], e.name[j])
		}
		le.PutUint16(eb[64:], e.nameLen)
		eb[66] = e.objectType
		eb[67] = e.color
		le.PutUint32(eb[68:], e.left)
		le.PutUint32(eb[72:], e.right)
		le.PutUint32(eb[76:], e.child)
		copy(eb[80:96], e.clsid[:])
		le.PutUint32(eb[116:], e.startSector)
		le.PutUint64(eb[120:], e.size)
		dirBuf.Write(eb[:])
	}
	// Pad with free ("unallocated") entries whose sibling pointers are
	// cfbNoStream.
	for dirBuf.Len()%cfbSectorSize != 0 {
		var eb [cfbDirEntrySize]byte
		le.PutUint32(eb[68:], cfbNoStream)
		le.PutUint32(eb[72:], cfbNoStream)
		le.PutUint32(eb[76:], cfbNoStream)
		dirBuf.Write(eb[:])
	}
	if _, err := w.Write(dirBuf.Bytes()); err != nil {
		return err
	}

	// MiniFAT sectors.
	if numMiniFATSectors > 0 {
		mfBuf := make([]byte, numMiniFATSectors*cfbSectorSize)
		for i := range mfBuf {
			mfBuf[i] = 0xFF
		}
		for i, v := range miniFAT {
			le.PutUint32(mfBuf[4*i:], v)
		}
		if _, err := w.Write(mfBuf); err != nil {
			return err
		}
	}

	// Mini stream data.
	if miniData.Len() > 0 {
		pad := (cfbSectorSize - miniData.Len()%cfbSectorSize) % cfbSectorSize
		miniData.Write(make([]byte, pad))
		if _, err := w.Write(miniData.Bytes()); err != nil {
			return err
		}
	}

	// Large stream data.
	for _, s := range streams {
		if len(s.Data) < cfbMiniCutoff {
			continue
		}
		if _, err := w.Write(s.Data); err != nil {
			return err
		}
		if pad := (cfbSectorSize - len(s.Data)%cfbSectorSize) % cfbSectorSize; pad > 0 {
			if _, err := w.Write(make([]byte, pad)); err != nil {
				return err
			}
		}
	}

	return nil
}

// encodeStreamName encodes a logical MSI stream name (e.g. "!_StringPool" or
// "Binary.WixCA") into the packed representation stored in the CFB
// directory. A leading '!' becomes the 0x4840 table prefix.
func encodeStreamName(name string) string {
	idx := func(b byte) int {
		switch {
		case b >= '0' && b <= '9':
			return int(b - '0')
		case b >= 'A' && b <= 'Z':
			return int(b-'A') + 10
		case b >= 'a' && b <= 'z':
			return int(b-'a') + 36
		case b == '.':
			return 62
		case b == '_':
			return 63
		}
		return -1
	}
	var out []rune
	i := 0
	if len(name) > 0 && name[0] == '!' {
		out = append(out, rune(0x4840))
		i = 1
	}
	for i < len(name) {
		c1 := idx(name[i])
		if c1 < 0 {
			out = append(out, rune(name[i]))
			i++
			continue
		}
		if i+1 < len(name) {
			if c2 := idx(name[i+1]); c2 >= 0 {
				out = append(out, rune(0x3800+c1+(c2<<6))) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
				i += 2
				continue
			}
		}
		out = append(out, rune(0x4800+c1)) //nolint:gosec // CFB format field; sector/stream counts bounded by MSI size
		i++
	}
	return string(out)
}
