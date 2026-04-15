package packaging

import (
	"encoding/binary"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// BOM file format constants.
const (
	bomMagic     = "BOMStore"
	bomVersion   = 1
	bomHeaderLen = 512

	bomTypeFile = 1
	bomTypeDir  = 2
	bomTypeLink = 3

	bomTreeMagic    = "tree"
	bomTreeVersion  = 1
	bomBlockSize    = 4096
	bomMaxLeafItems = 256
)

// bomNode is an in-memory tree node for BOM construction.
type bomNode struct {
	name     string
	children []*bomNode // sorted by name

	entryType uint8
	mode      uint16
	uid       uint32
	gid       uint32
	modtime   uint32
	size      uint32
	checksum  uint32
	linkTarget string
}

// writeBOM creates a BOM file at bomPath from the directory tree rooted at
// rootDir. All entries are assigned the given uid and gid. This is a pure Go
// replacement for the mkbom command.
func writeBOM(rootDir, bomPath string, uid, gid uint32) error {
	root, err := buildBOMTree(rootDir, uid, gid)
	if err != nil {
		return err
	}

	f, err := os.Create(bomPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return encodeBOM(f, root)
}

// buildBOMTree walks rootDir and builds an in-memory tree matching bomutils'
// structure: a virtual root with "." as its only child.
func buildBOMTree(rootDir string, uid, gid uint32) (*bomNode, error) {
	// Collect all entries by walking the filesystem.
	type rawEntry struct {
		relPath string // e.g. ".", "opt", "opt/orbit"
		node    bomNode
	}
	var entries []rawEntry

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden files (matching bomutils behavior of skipping d_name[0]=='.')
		// except for the root "." itself.
		if path != rootDir {
			base := filepath.Base(path)
			if len(base) > 0 && base[0] == '.' {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		// rel is "." for root, "a", "a/b", etc. for descendants.
		// filepath.Dir works correctly with these: Dir("a/b") = "a", Dir("a") = ".".

		info, err := d.Info()
		if err != nil {
			return err
		}

		name := filepath.Base(rel)
		if rel == "." {
			name = "."
		}
		n := bomNode{
			name:    name,
			uid:     uid,
			gid:     gid,
			modtime: uint32(info.ModTime().Unix()), //nolint:gosec // G115: modtime fits in uint32 until 2106
		}

		mode := info.Mode()
		switch {
		case mode.IsDir():
			n.entryType = bomTypeDir
			n.mode = uint16(mode.Perm()) | 0o40000 //nolint:gosec // G115: Perm() returns 9 bits, fits in uint16
		case mode&fs.ModeSymlink != 0:
			n.entryType = bomTypeLink
			n.mode = uint16(mode.Perm()) | 0o120000 //nolint:gosec // G115: Perm() returns 9 bits
			target, linkErr := os.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			n.linkTarget = target
			n.size = uint32(len(target)) //nolint:gosec // G115: symlink targets are short
			n.checksum = posixCksum([]byte(target))
		default:
			n.entryType = bomTypeFile
			n.mode = uint16(mode.Perm()) | 0o100000 //nolint:gosec // G115: Perm() returns 9 bits
			n.size = uint32(info.Size())             //nolint:gosec // G115: BOM format uses 32-bit sizes
			cksum, cksumErr := posixCksumFile(path)
			if cksumErr != nil {
				return cksumErr
			}
			n.checksum = cksum
		}

		entries = append(entries, rawEntry{relPath: rel, node: n})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build tree from flat entries. The virtual root has "." as its child.
	nodeMap := make(map[string]*bomNode, len(entries))
	virtualRoot := &bomNode{name: ""}

	for i := range entries {
		e := &entries[i]
		nodeCopy := e.node
		nodeMap[e.relPath] = &nodeCopy
	}

	// Sort paths so parents are created before children.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relPath < entries[j].relPath
	})

	for _, e := range entries {
		n := nodeMap[e.relPath]
		if e.relPath == "." {
			virtualRoot.children = append(virtualRoot.children, n)
			continue
		}
		parentPath := filepath.Dir(e.relPath)
		if parentPath == "" {
			parentPath = "."
		}
		parent := nodeMap[parentPath]
		if parent != nil {
			parent.children = append(parent.children, n)
		}
	}

	// Sort children at each level (std::map in bomutils is sorted by key).
	var sortChildren func(n *bomNode)
	sortChildren = func(n *bomNode) {
		sort.Slice(n.children, func(i, j int) bool {
			return n.children[i].name < n.children[j].name
		})
		for _, c := range n.children {
			sortChildren(c)
		}
	}
	sortChildren(virtualRoot)

	return virtualRoot, nil
}

// bomStorage manages the block-based storage for BOM file construction.
type bomStorage struct {
	blocks [][]byte // blocks[0] is always nil (null entry)
}

func newBOMStorage() *bomStorage {
	return &bomStorage{
		blocks: [][]byte{nil},
	}
}

func (s *bomStorage) addBlock(data []byte) uint32 {
	id := uint32(len(s.blocks)) //nolint:gosec // G115: block count is bounded by BOM entry count
	block := make([]byte, len(data))
	copy(block, data)
	s.blocks = append(s.blocks, block)
	return id
}

// encodeBOM writes a complete BOM file. virtualRoot is the virtual root whose
// children are the actual filesystem entries (starting with ".").
func encodeBOM(w io.Writer, virtualRoot *bomNode) error {
	st := newBOMStorage()

	// BFS traversal of the tree to collect entries in bomutils order.
	type bfsItem struct {
		node     *bomNode
		parentID uint32 // 0 for children of virtualRoot
	}
	queue := []bfsItem{{node: virtualRoot, parentID: 0}}
	type bomEntry struct {
		node     *bomNode
		id       uint32
		parentID uint32
	}
	var entries []bomEntry
	nextID := uint32(1)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		for _, child := range item.node.children {
			id := nextID
			nextID++
			entries = append(entries, bomEntry{node: child, id: id, parentID: item.parentID})
			queue = append(queue, bfsItem{node: child, parentID: id})
		}
	}

	num := uint32(len(entries)) //nolint:gosec // G115: entry count bounded by directory tree size

	// --- BomInfo ---
	bomInfoSize := 12
	if num > 0 {
		bomInfoSize += 16
	}
	bomInfo := make([]byte, bomInfoSize)
	binary.BigEndian.PutUint32(bomInfo[0:], 1)    // version
	binary.BigEndian.PutUint32(bomInfo[4:], num+1) // numberOfPaths
	if num > 0 {
		binary.BigEndian.PutUint32(bomInfo[8:], 1) // numberOfInfoEntries
		// BOMInfoEntry: 16 bytes of zeros
	}

	// --- Build path entries and leaf nodes ---
	numLeaves := (num + bomMaxLeafItems - 1) / bomMaxLeafItems
	if numLeaves == 0 {
		numLeaves = 1
	}

	leaves := make([][]leafEntry, numLeaves)

	for i, e := range entries {
		n := e.node

		// BOMPathInfo2
		linkNameLen := uint32(0)
		if n.entryType == bomTypeLink {
			linkNameLen = uint32(len(n.linkTarget)) + 1 //nolint:gosec // G115: symlink targets are short
		}
		pi2 := make([]byte, 31+linkNameLen)
		pi2[0] = n.entryType
		pi2[1] = 1 // unknown0
		binary.BigEndian.PutUint16(pi2[2:], 3) // architecture
		binary.BigEndian.PutUint16(pi2[4:], n.mode)
		binary.BigEndian.PutUint32(pi2[6:], n.uid)
		binary.BigEndian.PutUint32(pi2[10:], n.gid)
		binary.BigEndian.PutUint32(pi2[14:], n.modtime)
		binary.BigEndian.PutUint32(pi2[18:], n.size)
		pi2[22] = 1 // unknown1
		binary.BigEndian.PutUint32(pi2[23:], n.checksum)
		binary.BigEndian.PutUint32(pi2[27:], linkNameLen)
		if linkNameLen > 0 {
			copy(pi2[31:], n.linkTarget)
		}
		pi2Idx := st.addBlock(pi2)

		// BOMPathInfo1
		pi1 := make([]byte, 8)
		binary.BigEndian.PutUint32(pi1[0:], e.id)
		binary.BigEndian.PutUint32(pi1[4:], pi2Idx)
		pi1Idx := st.addBlock(pi1)

		// BOMFile
		bf := make([]byte, 4+len(n.name)+1)
		binary.BigEndian.PutUint32(bf[0:], e.parentID)
		copy(bf[4:], n.name)
		bfIdx := st.addBlock(bf)

		leafIdx := uint32(i) / bomMaxLeafItems
		leaves[leafIdx] = append(leaves[leafIdx], leafEntry{pi1Idx, bfIdx})
	}

	// --- Build BOMPaths B-tree ---
	var treeChildIdx uint32

	if numLeaves <= 1 {
		// Single leaf: tree root points directly to it.
		treeChildIdx = st.addBlock(serializeBOMLeaf(leaves[0], 0, 0))
	} else {
		// Multiple leaves with a branch root.
		leafBlockIDs := make([]uint32, numLeaves)

		// Add all leaves (forward links patched after).
		for li := uint32(0); li < numLeaves; li++ {
			backward := uint32(0)
			if li > 0 {
				backward = leafBlockIDs[li-1]
			}
			leafBlockIDs[li] = st.addBlock(serializeBOMLeaf(leaves[li], 0, backward))
		}

		// Patch forward links.
		for li := uint32(0); li < numLeaves-1; li++ {
			binary.BigEndian.PutUint32(st.blocks[leafBlockIDs[li]][4:8], leafBlockIDs[li+1])
		}
		// Patch backward links (leaf 0 has backward=0 which is correct).
		for li := uint32(1); li < numLeaves; li++ {
			binary.BigEndian.PutUint32(st.blocks[leafBlockIDs[li]][8:12], leafBlockIDs[li-1])
		}

		// Branch root: indices point to (leafBlockID, lastFileBlockIdx).
		branchSize := 12 + int(numLeaves)*8
		branch := make([]byte, branchSize)
		// isLeaf=0, count=numLeaves, forward=0, backward=0
		binary.BigEndian.PutUint16(branch[2:], uint16(numLeaves))
		for li := uint32(0); li < numLeaves; li++ {
			off := 12 + li*8
			binary.BigEndian.PutUint32(branch[off:], leafBlockIDs[li])
			// Last file block index from this leaf.
			lastEntry := leaves[li][len(leaves[li])-1]
			binary.BigEndian.PutUint32(branch[off+4:], lastEntry.fileBlockIdx)
		}
		treeChildIdx = st.addBlock(branch)
	}

	// Paths BOMTree (21 bytes).
	pathsTree := makeBOMTree(treeChildIdx, num, bomBlockSize)

	// --- Empty trees for HLIndex, VIndex, Size64 ---
	emptyLeaf := serializeBOMLeaf(nil, 0, 0)

	hlLeafIdx := st.addBlock(emptyLeaf)
	hlTree := makeBOMTree(hlLeafIdx, 0, bomBlockSize)

	viLeafIdx := st.addBlock(emptyLeaf)
	viInnerTree := makeBOMTree(viLeafIdx, 0, 128)
	viInnerTreeIdx := st.addBlock(viInnerTree)
	vindex := make([]byte, 13)
	binary.BigEndian.PutUint32(vindex[0:], 1)
	binary.BigEndian.PutUint32(vindex[4:], viInnerTreeIdx)

	s64LeafIdx := st.addBlock(emptyLeaf)
	s64Tree := makeBOMTree(s64LeafIdx, 0, bomBlockSize)

	// --- Build vars section ---
	type bomVarDef struct {
		name string
		data []byte
	}
	varDefs := []bomVarDef{
		{"BomInfo", bomInfo},
		{"Paths", pathsTree},
		{"HLIndex", hlTree},
		{"VIndex", vindex},
		{"Size64", s64Tree},
	}

	varBlockIDs := make([]uint32, len(varDefs))
	for i, v := range varDefs {
		varBlockIDs[i] = st.addBlock(v.data)
	}

	varsSize := 4
	for _, v := range varDefs {
		varsSize += 4 + 1 + len(v.name)
	}
	varsData := make([]byte, varsSize)
	binary.BigEndian.PutUint32(varsData[0:], uint32(len(varDefs))) //nolint:gosec // G115: 5 fixed vars
	off := 4
	for i, v := range varDefs {
		binary.BigEndian.PutUint32(varsData[off:], varBlockIDs[i])
		off += 4
		varsData[off] = byte(len(v.name)) //nolint:gosec // G115: var names are short (<20 chars)
		off++
		copy(varsData[off:], v.name)
		off += len(v.name)
	}

	// --- Compute layout ---
	entryDataSize := 0
	for _, b := range st.blocks[1:] {
		entryDataSize += len(b)
	}
	blockTableSize := 4 + len(st.blocks)*8
	freeListSize := 4 + 2*8

	// --- Write file ---
	// Header
	header := make([]byte, bomHeaderLen)
	copy(header[0:8], bomMagic)
	binary.BigEndian.PutUint32(header[8:], bomVersion)
	binary.BigEndian.PutUint32(header[12:], uint32(len(st.blocks)-1))                      //nolint:gosec // G115: bounded by entry count
	binary.BigEndian.PutUint32(header[16:], uint32(bomHeaderLen+varsSize+entryDataSize)) //nolint:gosec // G115: file offset
	binary.BigEndian.PutUint32(header[20:], uint32(blockTableSize+freeListSize))         //nolint:gosec // G115: index size
	binary.BigEndian.PutUint32(header[24:], uint32(bomHeaderLen))                        // varsOffset
	binary.BigEndian.PutUint32(header[28:], uint32(varsSize))                            // varsLength
	if _, err := w.Write(header); err != nil {
		return err
	}

	// Vars
	if _, err := w.Write(varsData); err != nil {
		return err
	}

	// Block data
	blockOffsets := make([]uint32, len(st.blocks))
	cumOff := uint32(0)
	for i := 1; i < len(st.blocks); i++ {
		blockOffsets[i] = cumOff
		if _, err := w.Write(st.blocks[i]); err != nil {
			return err
		}
		cumOff += uint32(len(st.blocks[i])) //nolint:gosec // G115: block size bounded by single entry
	}

	// Block table
	bt := make([]byte, blockTableSize)
	binary.BigEndian.PutUint32(bt[0:], uint32(len(st.blocks))) //nolint:gosec // G115: bounded by entry count
	for i := range st.blocks {
		o := 4 + i*8
		if i == 0 {
			// Null entry.
		} else {
			binary.BigEndian.PutUint32(bt[o:], uint32(bomHeaderLen)+uint32(varsSize)+blockOffsets[i])
			binary.BigEndian.PutUint32(bt[o+4:], uint32(len(st.blocks[i]))) //nolint:gosec // G115: block size
		}
	}
	if _, err := w.Write(bt); err != nil {
		return err
	}

	// Free list
	fl := make([]byte, freeListSize) // count=0 + 2 empty slots, all zeros
	if _, err := w.Write(fl); err != nil {
		return err
	}

	return nil
}

// serializeBOMLeaf creates a serialized BOMPaths leaf node.
func serializeBOMLeaf(entries []leafEntry, forward, backward uint32) []byte {
	size := 12 + len(entries)*8
	buf := make([]byte, size)
	binary.BigEndian.PutUint16(buf[0:], 1) // isLeaf
	binary.BigEndian.PutUint16(buf[2:], uint16(len(entries))) //nolint:gosec // G115: max 256 entries per leaf
	binary.BigEndian.PutUint32(buf[4:], forward)
	binary.BigEndian.PutUint32(buf[8:], backward)
	for i, e := range entries {
		off := 12 + i*8
		binary.BigEndian.PutUint32(buf[off:], e.pi1BlockIdx)
		binary.BigEndian.PutUint32(buf[off+4:], e.fileBlockIdx)
	}
	return buf
}

// makeBOMTree creates a serialized BOMTree (21 bytes).
func makeBOMTree(childBlockIdx, pathCount, blockSize uint32) []byte {
	buf := make([]byte, 21)
	copy(buf[0:4], bomTreeMagic)
	binary.BigEndian.PutUint32(buf[4:], bomTreeVersion)
	binary.BigEndian.PutUint32(buf[8:], childBlockIdx)
	binary.BigEndian.PutUint32(buf[12:], blockSize)
	binary.BigEndian.PutUint32(buf[16:], pathCount)
	buf[20] = 0 // unknown3
	return buf
}

// leafEntry pairs path info and file block indices for a leaf node.
type leafEntry struct {
	pi1BlockIdx  uint32
	fileBlockIdx uint32
}

// posixCksum computes the POSIX cksum CRC-32 of data.
// Uses polynomial 0x04C11DB7 in unreflected (MSB-first) form,
// initial value 0, file-length folding, final XOR 0xFFFFFFFF.
func posixCksum(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = posixCRC32Table[b^byte(crc>>24)] ^ (crc << 8)
	}
	length := uint64(len(data))
	for length > 0 {
		crc = posixCRC32Table[byte(length)^byte(crc>>24)] ^ (crc << 8)
		length >>= 8
	}
	return crc ^ 0xFFFFFFFF
}

// posixCksumFile computes the POSIX cksum CRC-32 of a file.
func posixCksumFile(path string) (uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	fileLen := uint64(info.Size()) //nolint:gosec // G115: file sizes are non-negative

	var crc uint32
	buf := make([]byte, 512*1024)
	for {
		n, readErr := f.Read(buf)
		for i := range n {
			crc = posixCRC32Table[buf[i]^byte(crc>>24)] ^ (crc << 8)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return 0, readErr
		}
	}

	for fileLen > 0 {
		crc = posixCRC32Table[byte(fileLen)^byte(crc>>24)] ^ (crc << 8)
		fileLen >>= 8
	}
	return crc ^ 0xFFFFFFFF, nil
}

// posixCRC32Table is the CRC-32 lookup table for polynomial 0x04C11DB7.
var posixCRC32Table = func() [256]uint32 {
	var table [256]uint32
	const poly = uint32(0x04C11DB7)
	for i := range 256 {
		crc := uint32(i) << 24 //nolint:gosec // G115: i is [0,255]
		for range 8 {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		table[i] = crc
	}
	return table
}()
