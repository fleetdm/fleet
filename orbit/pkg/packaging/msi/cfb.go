package msi

import (
	"encoding/binary"
	"io"
	"sort"
	"unicode/utf16"
)

const (
	cfbMagic0          = 0xE011CFD0
	cfbMagic1          = 0xE11AB1A1
	cfbVersion         = 3
	cfbRevision        = 0x003E
	cfbByteOrder       = 0xFFFE
	cfbSectorShift     = 9
	cfbMiniSectorShift = 6
	cfbSectorSize      = 512
	cfbMiniSectorSize  = 64
	cfbMinStdStreamSz  = 4096
	cfbMSATInHeader    = 109
	cfbDirEntSize      = 128
	cfbDirEntsPerSec   = cfbSectorSize / cfbDirEntSize // 4
	cfbDIFATPerSec     = cfbSectorSize/4 - 1           // 127

	secIDFree       int32 = -1
	secIDEndOfChain int32 = -2
	secIDSAT        int32 = -3
	secIDMSAT       int32 = -4

	dirRoot    = 5
	dirStream  = 2
	colorBlack = 1
	noChild    = -1
)

func putSecID(b []byte, v int32) {
	binary.LittleEndian.PutUint32(b, uint32(v)) //nolint:gosec // G115
}

type cfbStream struct {
	name string
	data []byte
}

type cfbWriter struct {
	streams []cfbStream
}

func newCFBWriter() *cfbWriter { return &cfbWriter{} }

func (cw *cfbWriter) addStream(name string, data []byte) {
	cw.streams = append(cw.streams, cfbStream{name: name, data: data})
}

// writeTo writes a complete CFB v3 file with mini-stream support.
// Streams < 4096 bytes go to the mini-stream; larger ones use regular sectors.
func (cw *cfbWriter) writeTo(w io.Writer) error {
	sort.Slice(cw.streams, func(i, j int) bool {
		return cfbNameLess(cw.streams[i].name, cw.streams[j].name)
	})

	type allocInfo struct {
		startSector int32
		numSectors  int
		isMini      bool
	}
	allocs := make([]allocInfo, len(cw.streams))

	// 1. Allocate regular sectors for large streams (>= 4096 bytes).
	nextReg := int32(0)
	for i, s := range cw.streams {
		if len(s.data) >= cfbMinStdStreamSz {
			n := (len(s.data) + cfbSectorSize - 1) / cfbSectorSize
			allocs[i] = allocInfo{startSector: nextReg, numSectors: n}
			nextReg += int32(n) //nolint:gosec // G115
		}
	}

	// 2. Build mini-stream from small streams (0 < size < 4096).
	var miniData []byte
	nextMini := int32(0)
	for i, s := range cw.streams {
		if len(s.data) == 0 {
			allocs[i] = allocInfo{startSector: secIDEndOfChain}
			continue
		}
		if len(s.data) < cfbMinStdStreamSz {
			ms := (len(s.data) + cfbMiniSectorSize - 1) / cfbMiniSectorSize
			allocs[i] = allocInfo{startSector: nextMini, numSectors: ms, isMini: true}
			miniData = append(miniData, s.data...)
			if pad := cfbMiniSectorSize - (len(s.data) % cfbMiniSectorSize); pad < cfbMiniSectorSize {
				miniData = append(miniData, make([]byte, pad)...)
			}
			nextMini += int32(ms) //nolint:gosec // G115
		}
	}
	totalMiniSectors := int(nextMini)

	// Allocate regular sectors for mini-stream container (root's stream).
	miniContStart := secIDEndOfChain
	miniContSectors := 0
	if len(miniData) > 0 {
		miniContStart = nextReg
		miniContSectors = (len(miniData) + cfbSectorSize - 1) / cfbSectorSize
		nextReg += int32(miniContSectors) //nolint:gosec // G115
	}

	// 3. Allocate SSAT sectors.
	ssatStart := secIDEndOfChain
	ssatSectors := 0
	if totalMiniSectors > 0 {
		ssatSectors = (totalMiniSectors*4 + cfbSectorSize - 1) / cfbSectorSize
		ssatStart = nextReg
		nextReg += int32(ssatSectors) //nolint:gosec // G115
	}

	// 4. Directory sectors.
	numDirEntries := 1 + len(cw.streams)
	numDirSectors := (numDirEntries + cfbDirEntsPerSec - 1) / cfbDirEntsPerSec
	dirStart := nextReg
	nextReg += int32(numDirSectors) //nolint:gosec // G115

	// 3. FAT/DIFAT allocation (fixed-point).
	totalSectors := int(nextReg)
	numFAT, numDIFAT := 0, 0
	for {
		nf := (totalSectors + numFAT + numDIFAT + cfbSectorSize/4 - 1) / (cfbSectorSize / 4)
		nd := 0
		if nf > cfbMSATInHeader {
			nd = (nf - cfbMSATInHeader + cfbDIFATPerSec - 1) / cfbDIFATPerSec
		}
		if nf <= numFAT && nd <= numDIFAT {
			break
		}
		numFAT, numDIFAT = nf, nd
		totalSectors = int(nextReg) + numFAT + numDIFAT
	}
	fatStart := nextReg
	difatStart := fatStart + int32(numFAT) //nolint:gosec // G115

	// 6. Build FAT.
	fat := make([]int32, totalSectors)
	for i := range fat {
		fat[i] = secIDFree
	}
	// Large stream chains.
	for i, a := range allocs {
		if a.isMini || len(cw.streams[i].data) == 0 || len(cw.streams[i].data) < cfbMinStdStreamSz {
			continue
		}
		chainSectors(fat, int(a.startSector), a.numSectors)
	}
	// Mini-stream container chain.
	if miniContSectors > 0 {
		chainSectors(fat, int(miniContStart), miniContSectors)
	}
	// SSAT chain.
	if ssatSectors > 0 {
		chainSectors(fat, int(ssatStart), ssatSectors)
	}
	// Directory chain.
	chainSectors(fat, int(dirStart), numDirSectors)
	// FAT/DIFAT markers.
	for i := range numFAT {
		fat[int(fatStart)+i] = secIDSAT
	}
	for i := range numDIFAT {
		fat[int(difatStart)+i] = secIDMSAT
	}

	// 7. Build SSAT.
	var ssatBuf []byte
	if totalMiniSectors > 0 {
		ssatBuf = make([]byte, ssatSectors*cfbSectorSize)
		for i := range ssatSectors * (cfbSectorSize / 4) {
			putSecID(ssatBuf[i*4:], secIDFree)
		}
		for i, a := range allocs {
			if !a.isMini || len(cw.streams[i].data) == 0 {
				continue
			}
			for j := range a.numSectors {
				ms := int(a.startSector) + j
				if j == a.numSectors-1 {
					putSecID(ssatBuf[ms*4:], secIDEndOfChain)
				} else {
					putSecID(ssatBuf[ms*4:], int32(ms+1)) //nolint:gosec // G115
				}
			}
		}
	}

	// 8. Build directory entries.
	dirEntries := make([]cfbDirEntry, numDirEntries)
	dirEntries[0] = cfbDirEntry{
		name: "Root Entry", entryType: dirRoot, color: colorBlack,
		leftChild: noChild, rightChild: noChild, storageRoot: noChild,
		startSector: miniContStart,
		streamSize:  uint32(len(miniData)), //nolint:gosec // G115
	}
	for i, s := range cw.streams {
		dirEntries[i+1] = cfbDirEntry{
			name: s.name, entryType: dirStream, color: colorBlack,
			leftChild: noChild, rightChild: noChild, storageRoot: noChild,
			startSector: allocs[i].startSector,
			streamSize:  uint32(len(s.data)), //nolint:gosec // G115
		}
	}
	if len(cw.streams) > 0 {
		rootIdx := buildRedBlackTree(dirEntries[1:])
		dirEntries[0].storageRoot = int32(rootIdx + 1) //nolint:gosec // G115
	}
	dirEntries[0].clsid = [16]byte{
		0x84, 0x10, 0x0C, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46,
	}

	// === WRITE ===

	hdr := make([]byte, cfbSectorSize)
	binary.LittleEndian.PutUint32(hdr[0:], cfbMagic0)
	binary.LittleEndian.PutUint32(hdr[4:], cfbMagic1)
	binary.LittleEndian.PutUint16(hdr[24:], cfbRevision)
	binary.LittleEndian.PutUint16(hdr[26:], cfbVersion)
	binary.LittleEndian.PutUint16(hdr[28:], cfbByteOrder)
	binary.LittleEndian.PutUint16(hdr[30:], cfbSectorShift)
	binary.LittleEndian.PutUint16(hdr[32:], cfbMiniSectorShift)
	binary.LittleEndian.PutUint32(hdr[44:], uint32(numFAT))      //nolint:gosec // G115
	putSecID(hdr[48:], dirStart)
	binary.LittleEndian.PutUint32(hdr[56:], cfbMinStdStreamSz)   // Standard 4096 cutoff
	putSecID(hdr[60:], ssatStart)
	binary.LittleEndian.PutUint32(hdr[64:], uint32(ssatSectors)) //nolint:gosec // G115
	if numDIFAT > 0 {
		putSecID(hdr[68:], difatStart)
		binary.LittleEndian.PutUint32(hdr[72:], uint32(numDIFAT)) //nolint:gosec // G115
	} else {
		putSecID(hdr[68:], secIDEndOfChain)
	}
	for i := range cfbMSATInHeader {
		off := 76 + i*4
		if i < numFAT {
			binary.LittleEndian.PutUint32(hdr[off:], uint32(fatStart)+uint32(i)) //nolint:gosec // G115
		} else {
			putSecID(hdr[off:], secIDFree)
		}
	}
	if _, err := w.Write(hdr); err != nil {
		return err
	}

	// Large stream data.
	for i, s := range cw.streams {
		if allocs[i].isMini || len(s.data) == 0 || len(s.data) < cfbMinStdStreamSz {
			continue
		}
		if err := writePadded(w, s.data, cfbSectorSize); err != nil {
			return err
		}
	}

	// Mini-stream container data.
	if len(miniData) > 0 {
		if err := writePadded(w, miniData, cfbSectorSize); err != nil {
			return err
		}
	}

	// SSAT.
	if len(ssatBuf) > 0 {
		if _, err := w.Write(ssatBuf); err != nil {
			return err
		}
	}

	// Directory.
	dirBuf := make([]byte, numDirSectors*cfbSectorSize)
	for i, de := range dirEntries {
		de.encode(dirBuf[i*cfbDirEntSize : (i+1)*cfbDirEntSize])
	}
	for i := len(dirEntries); i < numDirSectors*cfbDirEntsPerSec; i++ {
		off := i * cfbDirEntSize
		putSecID(dirBuf[off+68:], noChild)  // leftChild
		putSecID(dirBuf[off+72:], noChild)  // rightChild
		putSecID(dirBuf[off+76:], noChild)  // storageRoot
		putSecID(dirBuf[off+116:], secIDEndOfChain) // startSector (must not be 0 = valid sector)
	}
	if _, err := w.Write(dirBuf); err != nil {
		return err
	}

	// FAT.
	fatBuf := make([]byte, numFAT*cfbSectorSize)
	for i, e := range fat {
		putSecID(fatBuf[i*4:], e)
	}
	for i := len(fat); i < numFAT*(cfbSectorSize/4); i++ {
		putSecID(fatBuf[i*4:], secIDFree)
	}
	if _, err := w.Write(fatBuf); err != nil {
		return err
	}

	// DIFAT.
	if numDIFAT > 0 {
		db := make([]byte, numDIFAT*cfbSectorSize)
		fi := cfbMSATInHeader
		for i := range numDIFAT {
			base := i * cfbSectorSize
			for j := range cfbDIFATPerSec {
				off := base + j*4
				if fi < numFAT {
					binary.LittleEndian.PutUint32(db[off:], uint32(fatStart)+uint32(fi)) //nolint:gosec // G115
					fi++
				} else {
					putSecID(db[off:], secIDFree)
				}
			}
			np := base + cfbDIFATPerSec*4
			if i < numDIFAT-1 {
				putSecID(db[np:], difatStart+int32(i+1)) //nolint:gosec // G115
			} else {
				putSecID(db[np:], secIDEndOfChain)
			}
		}
		if _, err := w.Write(db); err != nil {
			return err
		}
	}
	return nil
}

func writePadded(w io.Writer, data []byte, boundary int) error {
	if _, err := w.Write(data); err != nil {
		return err
	}
	if pad := boundary - (len(data) % boundary); pad < boundary {
		if _, err := w.Write(make([]byte, pad)); err != nil {
			return err
		}
	}
	return nil
}

func chainSectors(fat []int32, start, count int) {
	for i := range count {
		sec := start + i
		if i == count-1 {
			fat[sec] = secIDEndOfChain
		} else {
			fat[sec] = int32(sec + 1) //nolint:gosec // G115
		}
	}
}

type cfbDirEntry struct {
	name        string
	entryType   uint8
	color       uint8
	leftChild   int32
	rightChild  int32
	storageRoot int32
	clsid       [16]byte
	startSector int32
	streamSize  uint32
}

func (de *cfbDirEntry) encode(buf []byte) {
	runes := utf16.Encode([]rune(de.name))
	for i, r := range runes {
		if i >= 32 {
			break
		}
		binary.LittleEndian.PutUint16(buf[i*2:], r)
	}
	if len(runes) < 32 {
		binary.LittleEndian.PutUint16(buf[len(runes)*2:], 0)
	}
	binary.LittleEndian.PutUint16(buf[64:], uint16((len(runes)+1)*2)) //nolint:gosec // G115
	buf[66] = de.entryType
	buf[67] = de.color
	putSecID(buf[68:], de.leftChild)
	putSecID(buf[72:], de.rightChild)
	putSecID(buf[76:], de.storageRoot)
	copy(buf[80:96], de.clsid[:])
	putSecID(buf[116:], de.startSector)
	binary.LittleEndian.PutUint32(buf[120:], de.streamSize)
}

// buildRedBlackTree builds a balanced BST over sorted entries and colors it
// as a valid red-black tree. The CFB spec (MS-CFB 2.6.4) requires a valid RB
// tree — property 5 (equal black-height on all paths) is violated by an
// all-black balanced BST when leaves are at different depths.
//
// Coloring strategy: compute max depth. Nodes at max_depth are RED if they're
// leaves (this preserves black-height when a sibling subtree is shallower).
// All other nodes are BLACK.
func buildRedBlackTree(entries []cfbDirEntry) int {
	if len(entries) == 0 {
		return -1
	}
	// Build the balanced BST structure first.
	var build func(lo, hi, depth int, maxDepth *int) int
	build = func(lo, hi, depth int, maxDepth *int) int {
		if lo > hi {
			return -1
		}
		if depth > *maxDepth {
			*maxDepth = depth
		}
		mid := (lo + hi) / 2
		left := build(lo, mid-1, depth+1, maxDepth)
		right := build(mid+1, hi, depth+1, maxDepth)
		entries[mid].color = colorBlack
		if left >= 0 {
			entries[mid].leftChild = int32(left + 1) //nolint:gosec // G115
		}
		if right >= 0 {
			entries[mid].rightChild = int32(right + 1) //nolint:gosec // G115
		}
		return mid
	}
	maxDepth := 0
	rootIdx := build(0, len(entries)-1, 0, &maxDepth)

	// Second pass: color deepest-level leaves RED to satisfy RB property 5.
	var colorize func(idx int32, depth int)
	colorize = func(idx int32, depth int) {
		if idx < 0 {
			return
		}
		e := &entries[idx]
		isLeaf := e.leftChild < 0 && e.rightChild < 0
		if isLeaf && depth == maxDepth {
			e.color = 0 // red
		}
		if e.leftChild >= 0 {
			colorize(e.leftChild-1, depth+1)
		}
		if e.rightChild >= 0 {
			colorize(e.rightChild-1, depth+1)
		}
	}
	colorize(int32(rootIdx), 0) //nolint:gosec // G115
	return rootIdx
}

func cfbNameLess(a, b string) bool {
	au := utf16.Encode([]rune(a))
	bu := utf16.Encode([]rune(b))
	if len(au) != len(bu) {
		return len(au) < len(bu)
	}
	for i := range au {
		a16 := toUpperUTF16(au[i])
		b16 := toUpperUTF16(bu[i])
		if a16 != b16 {
			return a16 < b16
		}
	}
	return false
}

func toUpperUTF16(c uint16) uint16 {
	if c >= 'a' && c <= 'z' {
		return c - 0x20
	}
	return c
}
