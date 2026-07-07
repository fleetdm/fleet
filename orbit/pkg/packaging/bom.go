package packaging

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// This file implements a minimal, pure-Go writer for the macOS BOM (Bill of
// Materials) format, replacing the external `mkbom`/`lsbom` tools (and the
// fleetdm/bomutils Docker image) previously used by xarBom.
//
// A BOM is a block store:
//
//	[ 32-byte header ][ blocks... ][ vars ][ block table (index) ]
//
// The header locates the block table (an array of (offset,length) pointers, one
// per block index) and the vars section (named -> block index). Named variables
// point at the top-level structures: BomInfo, Paths, HLIndex, VIndex, Size64.
//
// "Paths" is a B-tree whose single leaf lists (PathInfo1, File) block-index
// pairs, one per path. PathInfo1 -> PathInfo2 holds the metadata (type, mode,
// uid/gid, size, checksum); File holds the parent path id and the base name.
// Ownership is fixed to root/admin (0/80), matching the previous mkbom -u0 -g80
// behavior. Per-file checksums use the POSIX cksum (CRC-32/CKSUM) algorithm,
// exactly as Apple's mkbom records them.
//
// This writer does not reproduce Apple's exact block layout byte-for-byte (its
// block table is pre-sized with a free list); it produces a compact, valid BOM
// that lsbom and the macOS Installer read identically. The Linux build has
// always used a different (bomutils) layout, so byte-identical output was never
// a requirement -- an identical lsbom manifest is.

// bomChecksumTable is the CRC-32 table for polynomial 0x04C11DB7 (MSB-first),
// used by the POSIX cksum algorithm.
var bomChecksumTable = func() [256]uint32 {
	var t [256]uint32
	for i := range t {
		c := uint32(i) << 24
		for range 8 {
			if c&0x80000000 != 0 {
				c = (c << 1) ^ 0x04C11DB7
			} else {
				c <<= 1
			}
		}
		t[i] = c
	}
	return t
}()

// bomChecksum computes the POSIX cksum (CRC-32/CKSUM) of data: the CRC-32 over
// the data followed by the little-endian minimal-byte encoding of its length,
// finally inverted. This matches the checksum Apple's mkbom stores per file.
func bomChecksum(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = (crc << 8) ^ bomChecksumTable[byte(crc>>24)^b]
	}
	for n := len(data); n != 0; n >>= 8 {
		crc = (crc << 8) ^ bomChecksumTable[byte(crc>>24)^byte(n)]
	}
	return ^crc
}

// bomPath is one entry in the BOM path tree.
type bomPath struct {
	id       uint32
	parentID uint32 // 0 for the root "."
	name     string // base name; "." for the root
	isDir    bool
	mode     uint16 // full st_mode (type bits | permissions)
	size     uint32
	checksum uint32 // POSIX cksum of contents; 0 for directories
}

// Fixed block indices. Per-path blocks follow, starting at bomFirstPathBlock.
const (
	// Block index 0 is always the null block.
	bomInfoBlock      = 1
	bomPathsTree      = 2
	bomPathsLeaf      = 3
	bomHLIndexTree    = 4
	bomHLIndexLeaf    = 5
	bomVIndexBlock    = 6
	bomVIndexTree     = 7
	bomVIndexLeaf     = 8
	bomSize64Tree     = 9
	bomSize64Leaf     = 10
	bomFirstPathBlock = 11
)

// writeBom walks srcDir and writes a BOM describing its tree to dstPath, with
// all entries owned by root/admin (0/80).
func writeBom(srcDir, dstPath string) error {
	paths, err := collectBomPaths(srcDir)
	if err != nil {
		return fmt.Errorf("collect bom paths: %w", err)
	}

	data, err := buildBom(paths)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return fmt.Errorf("write bom: %w", err)
	}
	return nil
}

// collectBomPaths walks srcDir depth-first (children sorted by name), returning
// path entries with sequential ids assigned in that order. The root directory
// itself is recorded as ".".
func collectBomPaths(srcDir string) ([]*bomPath, error) {
	info, err := os.Stat(srcDir)
	if err != nil {
		return nil, err
	}

	var (
		out    []*bomPath
		nextID uint32 = 1
	)
	root := &bomPath{id: nextID, parentID: 0, name: ".", isDir: true, mode: bomUnixMode(info)}
	nextID++
	out = append(out, root)

	var walk func(dir string, parentID uint32) error
	walk = func(dir string, parentID uint32) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

		for _, de := range entries {
			// The fleetd payload is built only from files the packaging code
			// writes directly and from TUF target tarballs unpacked by
			// extractTarGz, which rejects any tar entry that is not a regular
			// file or directory. The orbit "current" symlink is created by the
			// postinstall script at install time, not shipped in the payload.
			// So a symlink (or other special file) should never appear here;
			// fail loudly rather than emit a malformed BOM entry (e.g. a type=3
			// symlink with no link target).
			if !de.IsDir() && !de.Type().IsRegular() {
				return fmt.Errorf("unsupported file type %s for %q", de.Type(), filepath.Join(dir, de.Name()))
			}

			fi, err := de.Info()
			if err != nil {
				return err
			}
			p := &bomPath{id: nextID, parentID: parentID, name: de.Name(), isDir: de.IsDir(), mode: bomUnixMode(fi)}
			nextID++

			if !de.IsDir() {
				contents, err := os.ReadFile(filepath.Join(dir, de.Name()))
				if err != nil {
					return err
				}
				p.size = uint32(len(contents)) //nolint:gosec // fleetd payload files are well under 4GB
				p.checksum = bomChecksum(contents)
			}
			out = append(out, p)

			if de.IsDir() {
				if err := walk(filepath.Join(dir, de.Name()), p.id); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(srcDir, root.id); err != nil {
		return nil, err
	}
	return out, nil
}

// bomUnixMode returns the full st_mode (file-type bits OR-ed with permission
// bits) for a file, as stored in a BOM.
func bomUnixMode(info os.FileInfo) uint16 {
	perm := uint16(info.Mode().Perm()) //nolint:gosec // permission bits fit in uint16
	// collectBomPaths rejects anything that isn't a regular file or directory,
	// so only those two types reach here.
	if info.IsDir() {
		return 0o040000 | perm
	}
	return 0o100000 | perm
}

// buildBom assembles the full BOM byte stream for the given path entries.
func buildBom(paths []*bomPath) ([]byte, error) {
	n := len(paths)
	blocks := make([][]byte, bomFirstPathBlock+3*n)

	// Per-path blocks, laid out as [PathInfo2, File, PathInfo1] per path (the
	// same ordering Apple's mkbom uses).
	type leafEntry struct {
		parentID       uint32
		name           string
		pi1Idx, fileID uint32
	}
	entries := make([]leafEntry, 0, n)
	for k, p := range paths {
		pi2Idx := bomFirstPathBlock + 3*k
		fileIdx := pi2Idx + 1
		pi1Idx := pi2Idx + 2

		blocks[pi2Idx] = buildBomPathInfo2(p)
		blocks[fileIdx] = buildBomFile(p)
		blocks[pi1Idx] = buildBomPathInfo1(p.id, uint32(pi2Idx)) //nolint:gosec // block index fits uint32

		entries = append(entries, leafEntry{p.parentID, p.name, uint32(pi1Idx), uint32(fileIdx)}) //nolint:gosec // block indices fit uint32
	}

	// The Paths leaf is a B-tree node keyed by (parent id, name): lsbom and the
	// Installer traverse it in key order, so the pairs must be sorted that way.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].parentID != entries[j].parentID {
			return entries[i].parentID < entries[j].parentID
		}
		return entries[i].name < entries[j].name
	})
	leafPairs := make([][2]uint32, len(entries))
	for i, e := range entries {
		leafPairs[i] = [2]uint32{e.pi1Idx, e.fileID}
	}

	// Fixed structures.
	blocks[bomInfoBlock] = buildBomInfo(uint32(n) + 1)                 //nolint:gosec // path count fits uint32
	blocks[bomPathsTree] = buildBomTree(bomPathsLeaf, uint32(n), 4096) //nolint:gosec // path count fits uint32
	blocks[bomPathsLeaf] = buildBomLeaf(leafPairs)
	blocks[bomHLIndexTree] = buildBomTree(bomHLIndexLeaf, 0, 4096)
	blocks[bomHLIndexLeaf] = buildBomLeaf(nil)
	blocks[bomVIndexBlock] = buildBomVIndex(bomVIndexTree)
	blocks[bomVIndexTree] = buildBomTree(bomVIndexLeaf, 0, 128)
	blocks[bomVIndexLeaf] = buildBomLeaf(nil)
	blocks[bomSize64Tree] = buildBomTree(bomSize64Leaf, 0, 4096)
	blocks[bomSize64Leaf] = buildBomLeaf(nil)

	// Lay out block data after the 32-byte header, recording each block's
	// address. The null block (index 0) has address 0 and length 0.
	addrs := make([]uint32, len(blocks))
	var body bytes.Buffer
	cursor := uint32(32)
	for i := 1; i < len(blocks); i++ {
		addrs[i] = cursor
		body.Write(blocks[i])
		cursor += uint32(len(blocks[i])) //nolint:gosec // block sizes are small
	}

	// Vars section, then the block table (index).
	vars := buildBomVars()
	varsOffset := cursor
	cursor += uint32(len(vars)) //nolint:gosec // vars section is tiny

	index := buildBomIndex(blocks, addrs)
	indexOffset := cursor

	var out bytes.Buffer
	out.WriteString("BOMStore")
	be := binary.BigEndian
	writeU32 := func(v uint32) { _ = binary.Write(&out, be, v) }
	writeU32(1)                       // version
	writeU32(uint32(len(blocks) - 1)) //nolint:gosec // number of non-null blocks
	writeU32(indexOffset)             // indexOffset
	writeU32(uint32(len(index)))      //nolint:gosec // indexLength
	writeU32(varsOffset)              // varsOffset
	writeU32(uint32(len(vars)))       //nolint:gosec // varsLength
	out.Write(body.Bytes())
	out.Write(vars)
	out.Write(index)
	return out.Bytes(), nil
}

// buildBomPathInfo2 renders the metadata block for a path (35 bytes for files,
// 31 for directories). Ownership is fixed to uid 0 / gid 80 (root/admin).
func buildBomPathInfo2(p *bomPath) []byte {
	var b bytes.Buffer
	be := binary.BigEndian
	typ := byte(1) // regular file
	if p.isDir {
		typ = 2 // directory
	}
	b.WriteByte(typ)
	b.WriteByte(1)                      // unknown0 (always 1)
	_ = binary.Write(&b, be, uint16(3)) // architecture
	_ = binary.Write(&b, be, p.mode)
	_ = binary.Write(&b, be, uint32(0))  // uid = root
	_ = binary.Write(&b, be, uint32(80)) // gid = admin
	_ = binary.Write(&b, be, uint32(0))  // mtime (0, matching mkbom -i)
	_ = binary.Write(&b, be, p.size)
	b.WriteByte(1) // unknown1 (always 1)
	_ = binary.Write(&b, be, p.checksum)
	_ = binary.Write(&b, be, uint32(0)) // linkNameLength (no symlink targets in our payload)
	if !p.isDir {
		_ = binary.Write(&b, be, uint32(0)) // trailing reserved word present only on files
	}
	return b.Bytes()
}

// buildBomFile renders a File block: parent path id followed by the NUL-
// terminated base name.
func buildBomFile(p *bomPath) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, p.parentID)
	b.WriteString(p.name)
	b.WriteByte(0)
	return b.Bytes()
}

// buildBomPathInfo1 renders a PathInfo1 block: the path id and the block index
// of its PathInfo2.
func buildBomPathInfo1(id, pathInfo2Block uint32) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, id)
	_ = binary.Write(&b, binary.BigEndian, pathInfo2Block)
	return b.Bytes()
}

// buildBomTree renders a "tree" block pointing at its (single) child leaf.
func buildBomTree(childBlock, pathCount, blockSize uint32) []byte {
	var b bytes.Buffer
	be := binary.BigEndian
	b.WriteString("tree")
	_ = binary.Write(&b, be, uint32(1)) // version
	_ = binary.Write(&b, be, childBlock)
	_ = binary.Write(&b, be, blockSize)
	_ = binary.Write(&b, be, pathCount)
	b.WriteByte(0) // unknown
	return b.Bytes()
}

// buildBomLeaf renders a B-tree leaf listing (index0, index1) pairs.
func buildBomLeaf(pairs [][2]uint32) []byte {
	var b bytes.Buffer
	be := binary.BigEndian
	_ = binary.Write(&b, be, uint16(1))          // isLeaf
	_ = binary.Write(&b, be, uint16(len(pairs))) //nolint:gosec // pair count fits uint16
	_ = binary.Write(&b, be, uint32(0))          // forward
	_ = binary.Write(&b, be, uint32(0))          // backward
	for _, pr := range pairs {
		_ = binary.Write(&b, be, pr[0])
		_ = binary.Write(&b, be, pr[1])
	}
	return b.Bytes()
}

// buildBomVIndex renders the VIndex wrapper: {version, tree block index, flag}.
func buildBomVIndex(treeBlock uint32) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(1))
	_ = binary.Write(&b, binary.BigEndian, treeBlock)
	b.WriteByte(0)
	return b.Bytes()
}

// buildBomInfo renders the BomInfo block.
func buildBomInfo(numPaths uint32) []byte {
	var b bytes.Buffer
	be := binary.BigEndian
	_ = binary.Write(&b, be, uint32(1)) // version
	_ = binary.Write(&b, be, numPaths)
	_ = binary.Write(&b, be, uint32(0)) // numberOfInfoEntries
	return b.Bytes()
}

// buildBomVars renders the named variables pointing at the top-level blocks.
func buildBomVars() []byte {
	vars := []struct {
		name  string
		block uint32
	}{
		{"BomInfo", bomInfoBlock},
		{"Paths", bomPathsTree},
		{"HLIndex", bomHLIndexTree},
		{"VIndex", bomVIndexBlock},
		{"Size64", bomSize64Tree},
	}
	var b bytes.Buffer
	be := binary.BigEndian
	_ = binary.Write(&b, be, uint32(len(vars))) //nolint:gosec // small fixed count
	for _, v := range vars {
		_ = binary.Write(&b, be, v.block)
		b.WriteByte(byte(len(v.name))) //nolint:gosec // var names are short constants
		b.WriteString(v.name)
	}
	return b.Bytes()
}

// buildBomIndex renders the block table: a pointer (offset, length) per block
// index, followed by an empty free list.
func buildBomIndex(blocks [][]byte, addrs []uint32) []byte {
	var b bytes.Buffer
	be := binary.BigEndian
	_ = binary.Write(&b, be, uint32(len(blocks))) //nolint:gosec // block count fits uint32
	for i := range blocks {
		_ = binary.Write(&b, be, addrs[i])
		_ = binary.Write(&b, be, uint32(len(blocks[i]))) //nolint:gosec // block sizes are small
	}
	_ = binary.Write(&b, be, uint32(0)) // free-list count
	return b.Bytes()
}
