package msi

import (
	"bytes"
	"crypto/md5" //nolint:gosec // MsiGetFileHash-compatible content hash, not used for security
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/klauspost/compress/flate"
)

// cab.go writes the embedded cabinet (MS-CAB) with MSZIP compression,
// replicating the observable behavior of WiX's cabbing (cabinet.dll FCI
// driven by WiX's dutil/cabcutil.cpp):
//
//   - Files are added in File-table sequence order; each CFFOLDER compresses
//     32 KiB blocks with MSZIP (deflate with the previous block as history).
//   - "Smart cabbing": a file whose size and MD5 (MsiGetFileHash) match an
//     earlier file is not stored again; its directory entry points at the
//     original's folder/offset.
//   - Folder flushes: before adding a file that has duplicates later, and
//     after adding a duplicate that is not adjacent to its original (while
//     non-duplicate files remain), the current folder is closed so that
//     msiexec can seek to those entries without re-extracting earlier files.
//     Flushes are skipped when no data was added since the last flush.

const (
	cabBlockSize    = 32768
	mszipSignature  = 0x4B43 // "CK"
	compTypeMSZIP   = 1
	cabVersionMinor = 3
	cabVersionMajor = 1
	attribArchive   = 0x20
)

// cabFile describes one file to add to the cabinet, in sequence order.
type cabFile struct {
	Key     string // CFFILE name: the File table key
	Path    string // path on disk
	Size    int64
	ModTime time.Time
	// Hash is the MsiGetFileHash-style MD5 (zeros for empty files), used
	// for duplicate detection.
	Hash [16]byte
}

// msiFileHash computes the file content hash the way MsiGetFileHash does:
// the MD5 of the content, except that empty files hash to all zeros.
func msiFileHash(path string, size int64) ([16]byte, error) {
	var zero [16]byte
	if size == 0 {
		return zero, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("read %s: %w", path, err)
	}
	return md5.Sum(data), nil //nolint:gosec
}

// dosDateTime converts a modification time to FAT date/time words (local
// time, 2-second resolution), as stored in CFFILE entries.
func dosDateTime(t time.Time) (date, timeOfDay uint16) {
	t = t.Local()
	year := max(t.Year(), 1980)
	date = uint16((year-1980)<<9 | int(t.Month())<<5 | t.Day())     //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
	timeOfDay = uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
	return date, timeOfDay
}

type cabFolderState struct {
	uncompressed bytes.Buffer // pending data of the open folder
}

type cabFileEntry struct {
	key        string
	size       uint32
	folder     int
	folderOff  uint32
	date, time uint16
}

// writeCab builds the cabinet from files (already in sequence order) and
// returns its bytes.
func writeCab(files []cabFile) ([]byte, error) {
	type placed struct {
		folder    int
		folderOff uint32
		size      int64
		hash      [16]byte
		hasDupes  bool
	}

	// First pass: find duplicates (same size + hash as an earlier file) so
	// the flush-before rule can be applied while streaming.
	originals := make([]int, len(files)) // -1 if the file is an original
	for i := range files {
		originals[i] = -1
		for j := range i {
			if originals[j] == -1 && files[j].Size == files[i].Size && files[j].Hash == files[i].Hash {
				originals[i] = j
				break
			}
		}
	}
	hasDupes := make([]bool, len(files))
	for _, orig := range originals {
		if orig >= 0 {
			hasDupes[orig] = true
		}
	}

	folders := []*cabFolderState{{}}
	var entries []cabFileEntry
	place := make([]placed, len(files))
	currentIdx := 0
	current := folders[0]
	bytesSinceFlush := int64(0)

	flush := func() {
		folders = append(folders, &cabFolderState{})
		currentIdx = len(folders) - 1
		current = folders[currentIdx]
		bytesSinceFlush = 0
	}

	remainingNonDup := 0
	for i := range files {
		if originals[i] == -1 {
			remainingNonDup++
		}
	}

	for i, f := range files {
		date, tm := dosDateTime(f.ModTime)
		if orig := originals[i]; orig >= 0 {
			// Duplicate: point at the original's data.
			entries = append(entries, cabFileEntry{
				key: f.Key, size: uint32(f.Size), //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
				folder: place[orig].folder, folderOff: place[orig].folderOff,
				date: date, time: tm,
			})
			// Flush after a duplicate that is not immediately after its
			// original or another duplicate, when non-duplicates remain.
			adjacentToOriginal := i-1 == orig
			adjacentToDup := i > 0 && originals[i-1] >= 0
			if !adjacentToOriginal && !adjacentToDup && remainingNonDup > 0 && bytesSinceFlush > 0 {
				flush()
			}
			continue
		}

		remainingNonDup--
		if hasDupes[i] && bytesSinceFlush > 0 {
			flush()
		}
		off := uint32(current.uncompressed.Len()) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
		if f.Size > 0 {
			data, err := os.ReadFile(f.Path)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", f.Path, err)
			}
			if int64(len(data)) != f.Size {
				return nil, fmt.Errorf("file %s changed size during packaging", f.Path)
			}
			current.uncompressed.Write(data)
		}
		place[i] = placed{folder: currentIdx, folderOff: off, size: f.Size, hash: f.Hash, hasDupes: hasDupes[i]}
		entries = append(entries, cabFileEntry{
			key: f.Key, size: uint32(f.Size), folder: currentIdx, folderOff: off, date: date, time: tm, //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
		})
		bytesSinceFlush += f.Size
	}

	// Drop trailing empty folders (possible when the last real file was
	// followed only by duplicates after a flush).
	usedFolders := 0
	folderRemap := make([]int, len(folders))
	for i := range folders {
		if folders[i].uncompressed.Len() > 0 || folderIsReferenced(entries, i) {
			folderRemap[i] = usedFolders
			usedFolders++
		} else {
			folderRemap[i] = -1
		}
	}
	compact := make([]*cabFolderState, 0, usedFolders)
	for i := range folders {
		if folderRemap[i] >= 0 {
			compact = append(compact, folders[i])
		}
	}
	for i := range entries {
		entries[i].folder = folderRemap[entries[i].folder]
	}

	// Compress each folder into MSZIP CFDATA blocks.
	type folderOut struct {
		blocks []byte
		nData  uint16
	}
	outs := make([]folderOut, len(compact))
	for i, f := range compact {
		blocks, n, err := mszipCompress(f.uncompressed.Bytes())
		if err != nil {
			return nil, err
		}
		outs[i] = folderOut{blocks: blocks, nData: n}
	}

	// Assemble.
	le := binary.LittleEndian
	var fileArea bytes.Buffer
	for _, e := range entries {
		var fb [16]byte
		le.PutUint32(fb[0:], e.size)
		le.PutUint32(fb[4:], e.folderOff)
		le.PutUint16(fb[8:], uint16(e.folder)) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
		le.PutUint16(fb[10:], e.date)
		le.PutUint16(fb[12:], e.time)
		le.PutUint16(fb[14:], attribArchive)
		fileArea.Write(fb[:])
		fileArea.WriteString(e.key)
		fileArea.WriteByte(0)
	}

	headerSize := 36
	folderAreaSize := 8 * len(compact)
	coffFiles := headerSize + folderAreaSize
	dataStart := coffFiles + fileArea.Len()

	var out bytes.Buffer
	out.WriteString("MSCF")
	var h [32]byte
	totalSize := dataStart
	for _, fo := range outs {
		totalSize += len(fo.blocks)
	}
	le.PutUint32(h[0:], 0)                  // reserved1
	le.PutUint32(h[4:], uint32(totalSize))  //nolint:gosec // cbCabinet; CAB format field bounded by format limits
	le.PutUint32(h[8:], 0)                  // reserved2
	le.PutUint32(h[12:], uint32(coffFiles)) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
	le.PutUint32(h[16:], 0)                 // reserved3
	h[20] = cabVersionMinor
	h[21] = cabVersionMajor
	le.PutUint16(h[22:], uint16(len(compact))) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
	le.PutUint16(h[24:], uint16(len(entries))) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
	le.PutUint16(h[26:], 0)                    // flags
	le.PutUint16(h[28:], 0)                    // setID
	le.PutUint16(h[30:], 0)                    // iCabinet
	out.Write(h[:])

	off := dataStart
	for _, fo := range outs {
		var fb [8]byte
		le.PutUint32(fb[0:], uint32(off)) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
		le.PutUint16(fb[4:], fo.nData)
		le.PutUint16(fb[6:], compTypeMSZIP)
		out.Write(fb[:])
		off += len(fo.blocks)
	}
	out.Write(fileArea.Bytes())
	for _, fo := range outs {
		out.Write(fo.blocks)
	}

	return out.Bytes(), nil
}

func folderIsReferenced(entries []cabFileEntry, folder int) bool {
	for _, e := range entries {
		if e.folder == folder {
			return true
		}
	}
	return false
}

// mszipCompress splits data into 32 KiB blocks and deflate-compresses each
// into a CFDATA record. Per MSZIP, each block is an independent deflate
// stream prefixed with "CK", but the LZ77 window carries over: block N may
// reference data from block N-1, supplied to the encoder as a preset
// dictionary.
//
// klauspost/compress/flate is used instead of the standard library because
// stdlib flate emits single-code distance Huffman tables as incomplete codes
// (RFC 1951 permits this and zlib accepts it), which Windows' cabinet.dll
// FDI and libmspack both reject, making the MSI fail with error 1335.
func mszipCompress(data []byte) ([]byte, uint16, error) {
	var out bytes.Buffer
	var nBlocks int
	le := binary.LittleEndian
	for start := 0; start < len(data); start += cabBlockSize {
		end := min(start+cabBlockSize, len(data))
		block := data[start:end]
		var dict []byte
		if start > 0 {
			dict = data[max(start-cabBlockSize, 0):start]
		}

		var comp bytes.Buffer
		comp.WriteByte('C')
		comp.WriteByte('K')
		fw, err := flate.NewWriterDict(&comp, flate.BestCompression, dict)
		if err != nil {
			return nil, 0, fmt.Errorf("create deflate writer: %w", err)
		}
		if _, err := fw.Write(block); err != nil {
			return nil, 0, fmt.Errorf("deflate: %w", err)
		}
		if err := fw.Close(); err != nil {
			return nil, 0, fmt.Errorf("deflate close: %w", err)
		}
		// MSZIP caps a CFDATA record at 32768+12 bytes; when deflate expands
		// incompressible data past that, store the block uncompressed
		// (a single stored deflate block is 5 bytes of overhead).
		if comp.Len() > cabBlockSize+12 {
			comp.Reset()
			comp.WriteByte('C')
			comp.WriteByte('K')
			comp.WriteByte(0x01) // BFINAL=1, BTYPE=00 (stored)
			var lens [4]byte
			le.PutUint16(lens[0:], uint16(len(block)))  //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
			le.PutUint16(lens[2:], ^uint16(len(block))) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
			comp.Write(lens[:])
			comp.Write(block)
		}

		var dh [8]byte
		le.PutUint16(dh[4:], uint16(comp.Len())) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
		le.PutUint16(dh[6:], uint16(len(block))) //nolint:gosec // CAB format field; bounded by format limits validated elsewhere
		csum := cabChecksum(comp.Bytes(), cabChecksum(dh[4:8], 0))
		le.PutUint32(dh[0:], csum)
		out.Write(dh[:])
		out.Write(comp.Bytes())
		nBlocks++
	}
	if nBlocks > 0xFFFF {
		return nil, 0, fmt.Errorf("too many CFDATA blocks: %d", nBlocks)
	}
	return out.Bytes(), uint16(nBlocks), nil
}

// cabChecksum implements the CSUMCompute checksum from MS-CAB: a running
// 32-bit XOR over little-endian dwords, with trailing bytes folded in
// big-endian order.
func cabChecksum(data []byte, seed uint32) uint32 {
	csum := seed
	i := 0
	for ; i+4 <= len(data); i += 4 {
		csum ^= binary.LittleEndian.Uint32(data[i:])
	}
	var ul uint32
	switch len(data) - i {
	case 3:
		ul |= uint32(data[i]) << 16
		ul |= uint32(data[i+1]) << 8
		ul |= uint32(data[i+2])
	case 2:
		ul |= uint32(data[i]) << 8
		ul |= uint32(data[i+1])
	case 1:
		ul |= uint32(data[i])
	}
	return csum ^ ul
}
