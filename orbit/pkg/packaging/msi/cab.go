package msi

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

// CAB file format constants.
const (
	cabSignature    = 0x4643534D // "MSCF" as little-endian uint32
	cabVersionMajor = 1
	cabVersionMinor = 3
	cabHeaderSize   = 36
	cabFolderSize   = 8
	cabCompressNone = 0
	cabBlockSize    = 32768 // Max uncompressed bytes per CFDATA block.

	cabAttrUTF8 = 0x80 // File name is UTF-8 encoded.
)

// CabFile represents a file to include in a CAB archive.
type CabFile struct {
	Name    string // Path within the CAB (backslash separators for MSI).
	Data    []byte
	ModTime time.Time
}

// WriteCab creates a CAB archive with MS-Zip compression and writes it to w.
// All files are placed in a single folder. Returns the number of bytes written.
func WriteCab(w io.Writer, files []CabFile) (int64, error) {
	// We build the entire CAB in memory since we need to know sizes up front
	// for the header. MSI CABs are typically a few tens of MB at most.

	// 1. Build CFFILE entries.
	var cffilesBuf bytes.Buffer
	uncompressedOffset := uint32(0)
	for _, f := range files {
		writeCFFile(&cffilesBuf, f, uncompressedOffset)
		uncompressedOffset += uint32(len(f.Data)) //nolint:gosec // G115
	}

	// 2. Build CFDATA blocks (uncompressed). We concatenate every file's data
	// and split into 32KB blocks, each wrapped in a CFDATA header. Going
	// uncompressed avoids MS-ZIP encoder compatibility issues: Go's stdlib
	// flate produces RFC 1951-compliant DEFLATE output that libarchive accepts,
	// but libmspack (used by cabextract) and Windows CABINET.DLL reject the
	// same streams with a "decompression error" — they expect MS-ZIP's
	// single-Huffman-block-per-stream layout that Go's flate does not emit.
	// Fleet's installers are ~70MB uncompressed and the files are already
	// compressed binaries (EXEs, PEMs), so going to stored blocks roughly
	// doubles the CAB size but keeps the installer simple and reliable.
	var allData bytes.Buffer
	for _, f := range files {
		allData.Write(f.Data)
	}
	dataBlocks := storeBlocks(allData.Bytes())

	var cfdataBuf bytes.Buffer
	for _, block := range dataBlocks {
		cfdataBuf.Write(block)
	}

	// 3. Calculate offsets.
	coffFiles := uint32(cabHeaderSize + cabFolderSize)
	coffCabStart := coffFiles + uint32(cffilesBuf.Len()) //nolint:gosec // G115
	cbCabinet := coffCabStart + uint32(cfdataBuf.Len())  //nolint:gosec // G115

	// 4. Write everything.
	var buf bytes.Buffer

	// CFHEADER
	writeCFHeader(&buf, cbCabinet, coffFiles, uint16(len(files)), uint16(len(dataBlocks))) //nolint:gosec // G115

	// CFFOLDER
	writeCFFolder(&buf, coffCabStart, uint16(len(dataBlocks))) //nolint:gosec // G115

	// CFFILE entries
	buf.Write(cffilesBuf.Bytes())

	// CFDATA blocks
	buf.Write(cfdataBuf.Bytes())

	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

func writeCFHeader(w *bytes.Buffer, cbCabinet, coffFiles uint32, cFiles, _ uint16) {
	binary.Write(w, binary.LittleEndian, uint32(cabSignature)) //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint32(0))            //nolint:errcheck // reserved1
	binary.Write(w, binary.LittleEndian, cbCabinet)            //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint32(0))            //nolint:errcheck // reserved2
	binary.Write(w, binary.LittleEndian, coffFiles)            //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint32(0))            //nolint:errcheck // reserved3
	w.WriteByte(cabVersionMinor)                               //nolint:errcheck
	w.WriteByte(cabVersionMajor)                               //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint16(1))            //nolint:errcheck // cFolders = 1
	binary.Write(w, binary.LittleEndian, cFiles)               //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint16(0))            //nolint:errcheck // flags = 0
	binary.Write(w, binary.LittleEndian, uint16(0))            //nolint:errcheck // setID
	binary.Write(w, binary.LittleEndian, uint16(0))            //nolint:errcheck // iCabinet
}

func writeCFFolder(w *bytes.Buffer, coffCabStart uint32, cCFData uint16) {
	binary.Write(w, binary.LittleEndian, coffCabStart)            //nolint:errcheck
	binary.Write(w, binary.LittleEndian, cCFData)                 //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint16(cabCompressNone)) //nolint:errcheck
}

func writeCFFile(w *bytes.Buffer, f CabFile, uoffFolderStart uint32) {
	binary.Write(w, binary.LittleEndian, uint32(len(f.Data))) //nolint:errcheck,gosec
	binary.Write(w, binary.LittleEndian, uoffFolderStart)     //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint16(0))           //nolint:errcheck // iFolder = 0

	date, dosTime := dosDateTime(f.ModTime)
	binary.Write(w, binary.LittleEndian, date)    //nolint:errcheck
	binary.Write(w, binary.LittleEndian, dosTime)  //nolint:errcheck
	binary.Write(w, binary.LittleEndian, uint16(cabAttrUTF8|0x20)) //nolint:errcheck // UTF-8 + Archive

	// Null-terminated filename.
	w.WriteString(f.Name) //nolint:errcheck
	w.WriteByte(0)        //nolint:errcheck
}

// dosDateTime converts a time.Time to DOS date and time uint16 values.
func dosDateTime(t time.Time) (date, dosTime uint16) {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()

	if year < 1980 {
		year = 1980
	}
	date = uint16((year-1980)<<9 | int(month)<<5 | day) //nolint:gosec // G115
	dosTime = uint16(hour<<11 | min<<5 | sec/2)         //nolint:gosec // G115
	return date, dosTime
}

// storeBlocks chunks data into uncompressed CFDATA blocks (TYPE_NONE). Each
// block is a plain 32KB (or shorter, for the final block) slice of the input
// with a CFDATA header (checksum, cbData, cbUncomp). Always produces at least
// one block so that a folder with zero files still has a trailing CFDATA.
func storeBlocks(data []byte) [][]byte {
	var blocks [][]byte
	for offset := 0; offset < len(data) || len(blocks) == 0; {
		end := min(offset+cabBlockSize, len(data))
		chunk := data[offset:end]
		cb := uint16(len(chunk)) //nolint:gosec // G115: max 32768

		csum := cabChecksum(chunk, cb, cb)

		var block bytes.Buffer
		binary.Write(&block, binary.LittleEndian, csum) //nolint:errcheck
		binary.Write(&block, binary.LittleEndian, cb)   //nolint:errcheck // cbData = cbUncomp (no compression)
		binary.Write(&block, binary.LittleEndian, cb)   //nolint:errcheck
		block.Write(chunk)

		blocks = append(blocks, block.Bytes())
		offset = end
	}
	return blocks
}

// cabChecksum computes the CAB checksum for a CFDATA block.
// The checksum is an XOR of the payload in 4-byte LE chunks,
// seeded with cbData | (cbUncomp << 16).
func cabChecksum(payload []byte, cbData, cbUncomp uint16) uint32 {
	// Seed: fold cbData and cbUncomp into the checksum.
	csum := uint32(cbData) | (uint32(cbUncomp) << 16)

	// XOR 4-byte little-endian chunks.
	i := 0
	for i+4 <= len(payload) {
		csum ^= binary.LittleEndian.Uint32(payload[i : i+4])
		i += 4
	}

	// Handle remainder (0-3 bytes). Per MS-CAB spec / libmspack, bytes are
	// placed at DESCENDING shift positions: data[0] at bit 16 (not bit 0).
	//   3 bytes: d[0]<<16 | d[1]<<8 | d[2]
	//   2 bytes: d[0]<<8  | d[1]
	//   1 byte:  d[0]
	var remainder uint32
	switch len(payload) - i {
	case 3:
		remainder |= uint32(payload[i]) << 16
		remainder |= uint32(payload[i+1]) << 8
		remainder |= uint32(payload[i+2])
	case 2:
		remainder |= uint32(payload[i]) << 8
		remainder |= uint32(payload[i+1])
	case 1:
		remainder |= uint32(payload[i])
	}
	csum ^= remainder

	return csum
}
