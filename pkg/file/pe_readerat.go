package file

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"golang.org/x/text/encoding/unicode"
)

// Limits for parsing untrusted PE files. Variables so tests can lower them.
var (
	// peMaxHeaderSize bounds the DOS + NT + section headers region.
	peMaxHeaderSize int64 = 1 << 20
	// peMaxResourceEntries bounds the total number of resource directory
	// entries read across all directories visited.
	peMaxResourceEntries = 4096
	// peMaxVersionInfoSize bounds a single VS_VERSIONINFO resource.
	peMaxVersionInfoSize uint32 = 1 << 20
	// peMaxVersionInfoTotal bounds the bytes read for all VS_VERSIONINFO resources.
	peMaxVersionInfoTotal int64 = 16 << 20
	// peMaxVersionInfoSteps bounds the structures walked in a single VS_VERSIONINFO.
	peMaxVersionInfoSteps = 1 << 16
)

const (
	peHeaderPrefetch      = 4096
	peResourceTypeVersion = 16
	peResourceDirSize     = 16
	peResourceEntrySize   = 8
	peResourceDataSize    = 16
	peFixedFileInfoSize   = 52
	peFixedFileInfoSig    = 0xFEEF04BD
	peVersionHeaderSize   = 6
	// Extra bytes read past a VS_VERSIONINFO resource: the reference parser
	// reads fixed-size windows (e.g. 100 bytes for a key) that may extend past
	// the end of the last structure.
	peVersionInfoSlack = 256
)

// parsePEVersionResources extracts the version-info strings of a PE file
// (.exe) via ranged reads on ra, without loading the file into memory. It
// mirrors github.com/saferwall/pe's ParseVersionResourcesForEntries: one map
// per RT_VERSION language entry (of the first resource name), containing the
// String key/value pairs of the first StringTable of each StringFileInfo.
// Malformed structures yield an error wrapping ErrInvalidInstaller.
func parsePEVersionResources(ra io.ReaderAt, size int64) ([]map[string]string, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: empty PE file", ErrInvalidInstaller)
	}
	p := &peVersionParser{ra: ra, size: size}
	if err := p.parseHeaders(); err != nil {
		return nil, err
	}
	if p.rsrcRVA == 0 {
		return nil, nil
	}
	return p.parseResources()
}

type peSection struct {
	virtualAddress, virtualSize, sizeOfRawData, pointerToRawData uint32
}

type peVersionParser struct {
	ra   io.ReaderAt
	size int64

	sections         []peSection
	fileAlignment    uint32
	sectionAlignment uint32
	rsrcRVA          uint32

	entriesRead      int
	versionBytesRead int64
}

// readAt reads exactly n bytes at off, which must lie within the file.
func (p *peVersionParser) readAt(off int64, n int64) ([]byte, error) {
	if off < 0 || n < 0 || off > p.size || n > p.size-off {
		return nil, fmt.Errorf("%w: PE read [%d,+%d) out of bounds", ErrInvalidInstaller, off, n)
	}
	buf := make([]byte, n)
	read, err := p.ra.ReadAt(buf, off)
	if read == len(buf) {
		return buf, nil
	}
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("%w: PE file truncated", ErrInvalidInstaller)
	}
	return nil, fmt.Errorf("reading PE file: %w", err)
}

// inBounds mirrors saferwall's ReadBytesAtOffset bounds check.
func (p *peVersionParser) inBounds(off, n uint32) bool {
	total := off + n
	if (total > off) != (n > 0) {
		return false
	}
	return int64(off) < p.size && int64(total) <= p.size
}

// parseHeaders reads the DOS, NT and section headers in (usually) a single
// read and parses them with debug/pe.
func (p *peVersionParser) parseHeaders() error {
	hdr, err := p.readAt(0, min(p.size, peHeaderPrefetch))
	if err != nil {
		return err
	}
	ensure := func(end int64) error {
		if end <= int64(len(hdr)) {
			return nil
		}
		if end > peMaxHeaderSize {
			return fmt.Errorf("%w: PE headers exceed %d bytes", ErrInvalidInstaller, peMaxHeaderSize)
		}
		more, err := p.readAt(int64(len(hdr)), end-int64(len(hdr)))
		if err != nil {
			return err
		}
		hdr = append(hdr, more...)
		return nil
	}

	if len(hdr) < 0x40 || hdr[0] != 'M' || hdr[1] != 'Z' {
		return fmt.Errorf("%w: missing PE DOS header", ErrInvalidInstaller)
	}
	coff := int64(binary.LittleEndian.Uint32(hdr[0x3c:])) + 4
	if err := ensure(coff + 20); err != nil {
		return err
	}
	numSections := int64(binary.LittleEndian.Uint16(hdr[coff+2:]))
	sizeOfOptHeader := int64(binary.LittleEndian.Uint16(hdr[coff+16:]))
	sectionsStart := coff + 20 + sizeOfOptHeader
	// debug/pe always reads a 96-byte DOS header.
	end := max(sectionsStart+40*numSections, min(96, p.size))
	if err := ensure(end); err != nil {
		return err
	}

	// Parse a sanitized in-memory copy so debug/pe neither performs
	// unbounded small reads (COFF symbols, relocations) nor rejects fields
	// irrelevant here (machine type, long section names).
	hdr = hdr[:end:end]
	binary.LittleEndian.PutUint16(hdr[coff:], 0)    // Machine
	binary.LittleEndian.PutUint32(hdr[coff+8:], 0)  // PointerToSymbolTable
	binary.LittleEndian.PutUint32(hdr[coff+12:], 0) // NumberOfSymbols
	for i := range numSections {
		sh := hdr[sectionsStart+40*i:]
		if sh[0] == '/' {
			clear(sh[:8]) // Name
		}
		binary.LittleEndian.PutUint32(sh[24:], 0) // PointerToRelocations
		binary.LittleEndian.PutUint16(sh[32:], 0) // NumberOfRelocations
	}

	f, err := pe.NewFile(bytes.NewReader(hdr))
	if err != nil {
		return fmt.Errorf("%w: parsing PE headers: %w", ErrInvalidInstaller, err)
	}
	var dirs []pe.DataDirectory
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		p.fileAlignment, p.sectionAlignment = oh.FileAlignment, oh.SectionAlignment
		dirs = oh.DataDirectory[:min(oh.NumberOfRvaAndSizes, uint32(len(oh.DataDirectory)))]
	case *pe.OptionalHeader64:
		p.fileAlignment, p.sectionAlignment = oh.FileAlignment, oh.SectionAlignment
		dirs = oh.DataDirectory[:min(oh.NumberOfRvaAndSizes, uint32(len(oh.DataDirectory)))]
	default:
		return fmt.Errorf("%w: missing PE optional header", ErrInvalidInstaller)
	}
	if len(dirs) > pe.IMAGE_DIRECTORY_ENTRY_RESOURCE {
		p.rsrcRVA = dirs[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE].VirtualAddress
	}
	p.sections = make([]peSection, 0, len(f.Sections))
	for _, s := range f.Sections {
		p.sections = append(p.sections, peSection{
			virtualAddress:   s.VirtualAddress,
			virtualSize:      s.VirtualSize,
			sizeOfRawData:    s.Size,
			pointerToRawData: s.Offset,
		})
	}
	return nil
}

// The RVA-to-offset mapping below mirrors saferwall's GetOffsetFromRva so that
// both parsers agree on (possibly unusual) section layouts.

func (p *peVersionParser) adjustFileAlignment(va uint32) uint32 {
	if p.fileAlignment < 0x200 {
		return va
	}
	return (va / 0x200) * 0x200
}

func (p *peVersionParser) adjustSectionAlignment(va uint32) uint32 {
	sa := p.sectionAlignment
	if sa < 0x1000 {
		sa = p.fileAlignment
	}
	if sa != 0 && va%sa != 0 {
		return sa * (va / sa)
	}
	return va
}

func (p *peVersionParser) sectionContains(i int, rva uint32) bool {
	s := p.sections[i]
	fileSize := uint32(min(p.size, math.MaxUint32))
	var size uint32
	if fileSize-p.adjustFileAlignment(s.pointerToRawData) < s.sizeOfRawData {
		size = s.virtualSize
	} else {
		size = max(s.sizeOfRawData, s.virtualSize)
	}
	vaAdj := p.adjustSectionAlignment(s.virtualAddress)
	var next uint32
	if i < len(p.sections)-1 {
		next = p.sections[i+1].virtualAddress
	}
	if next != 0 && next > s.virtualAddress && vaAdj+size > next {
		size = next - vaAdj
	}
	return vaAdj <= rva && rva < vaAdj+size
}

func (p *peVersionParser) offsetFromRVA(rva uint32) uint32 {
	for i, s := range p.sections {
		if p.sectionContains(i, rva) {
			return rva - p.adjustSectionAlignment(s.virtualAddress) + p.adjustFileAlignment(s.pointerToRawData)
		}
	}
	if int64(rva) < p.size {
		return rva
	}
	return math.MaxUint32
}

type peResourceEntry struct {
	name         uint32
	offsetToData uint32
	// visitedLen is the length of the visited directories list as seen by
	// this entry's subtree.
	visitedLen int
}

func (e peResourceEntry) isDir() bool    { return e.offsetToData&0x80000000 != 0 }
func (e peResourceEntry) offset() uint32 { return e.offsetToData & 0x7FFFFFFF }

// readResourceDir returns the entries of the resource directory at rva,
// stopping (like saferwall) at the first all-zero or unreadable entry, or at an
// entry pointing to an already visited directory (a loop). visited holds the
// directories visited so far; it is extended with the subdirectories found.
func (p *peVersionParser) readResourceDir(rva uint32, visited []uint32) ([]peResourceEntry, []uint32, error) {
	visited = visited[:len(visited):len(visited)]
	off := p.offsetFromRVA(rva)
	if !p.inBounds(off, peResourceDirSize) {
		return nil, nil, fmt.Errorf("%w: PE resource directory out of bounds", ErrInvalidInstaller)
	}
	hdr, err := p.readAt(int64(off), peResourceDirSize)
	if err != nil {
		return nil, nil, err
	}
	n := int(binary.LittleEndian.Uint16(hdr[12:])) + int(binary.LittleEndian.Uint16(hdr[14:]))
	p.entriesRead += n
	if p.entriesRead > peMaxResourceEntries {
		return nil, nil, fmt.Errorf("%w: too many PE resource directory entries", ErrInvalidInstaller)
	}
	if n == 0 {
		return nil, visited, nil
	}

	entriesOff := p.offsetFromRVA(rva + peResourceDirSize)
	if int64(entriesOff) >= p.size {
		return nil, visited, nil
	}
	avail := min(int64(n)*peResourceEntrySize, (p.size-int64(entriesOff))/peResourceEntrySize*peResourceEntrySize)
	raw, err := p.readAt(int64(entriesOff), avail)
	if err != nil {
		return nil, nil, err
	}

	entries := make([]peResourceEntry, 0, len(raw)/peResourceEntrySize)
	for i := 0; i+peResourceEntrySize <= len(raw); i += peResourceEntrySize {
		e := peResourceEntry{
			name:         binary.LittleEndian.Uint32(raw[i:]),
			offsetToData: binary.LittleEndian.Uint32(raw[i+4:]),
		}
		if e.name == 0 && e.offsetToData == 0 {
			break
		}
		if e.name&0x80000000 != 0 {
			// saferwall stops when the name's length prefix is unreadable.
			nameOff := p.offsetFromRVA(p.rsrcRVA + e.name&0x7FFFFFFF)
			if int64(nameOff) > p.size-2 {
				break
			}
		}
		if e.isDir() {
			dir := p.rsrcRVA + e.offset()
			if containsUint32(visited, dir) {
				break
			}
			visited = append(visited, dir)
		}
		e.visitedLen = len(visited)
		entries = append(entries, e)
	}
	return entries, visited, nil
}

func containsUint32(s []uint32, v uint32) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func (p *peVersionParser) parseResources() ([]map[string]string, error) {
	types, visited, err := p.readResourceDir(p.rsrcRVA, []uint32{p.rsrcRVA})
	if err != nil {
		return nil, err
	}

	var all []map[string]string
	for _, t := range types {
		if t.name != peResourceTypeVersion {
			continue
		}
		if !t.isDir() {
			return nil, fmt.Errorf("%w: PE version resource is not a directory", ErrInvalidInstaller)
		}
		names, nameVisited, err := p.readResourceDir(p.rsrcRVA+t.offset(), visited[:t.visitedLen])
		if err != nil {
			return nil, err
		}
		// Like saferwall, only the first resource name is considered.
		if len(names) == 0 || !names[0].isDir() {
			return nil, fmt.Errorf("%w: invalid PE version resource directory", ErrInvalidInstaller)
		}
		langs, _, err := p.readResourceDir(p.rsrcRVA+names[0].offset(), nameVisited[:names[0].visitedLen])
		if err != nil {
			return nil, err
		}
		for _, l := range langs {
			if l.isDir() {
				return nil, fmt.Errorf("%w: invalid PE version resource language entry", ErrInvalidInstaller)
			}
			vers := make(map[string]string)
			all = append(all, vers)
			if err := p.parseVersionDataEntry(p.rsrcRVA+l.offset(), vers); err != nil {
				return nil, err
			}
		}
	}
	return all, nil
}

func (p *peVersionParser) parseVersionDataEntry(rva uint32, vers map[string]string) error {
	off := p.offsetFromRVA(rva)
	if !p.inBounds(off, peResourceDataSize) {
		return fmt.Errorf("%w: PE version resource data entry out of bounds", ErrInvalidInstaller)
	}
	raw, err := p.readAt(int64(off), peResourceDataSize)
	if err != nil {
		return err
	}
	dataRVA := binary.LittleEndian.Uint32(raw[0:])
	dataSize := binary.LittleEndian.Uint32(raw[4:])
	if dataSize > peMaxVersionInfoSize {
		return fmt.Errorf("%w: PE version resource too large (%d bytes)", ErrInvalidInstaller, dataSize)
	}
	dataOff := p.offsetFromRVA(dataRVA)
	if !p.inBounds(dataOff, dataSize) {
		return fmt.Errorf("%w: PE version resource out of bounds", ErrInvalidInstaller)
	}
	n := min(int64(dataSize)+peVersionInfoSlack, p.size-int64(dataOff))
	p.versionBytesRead += n
	if p.versionBytesRead > peMaxVersionInfoTotal {
		return fmt.Errorf("%w: PE version resources too large", ErrInvalidInstaller)
	}
	data, err := p.readAt(int64(dataOff), n)
	if err != nil {
		return err
	}
	b := &peVersionBlob{data: data, start: dataOff, fileSize: p.size, base: dataRVA}
	return b.parse(vers)
}

// peVersionBlob holds a VS_VERSIONINFO resource. Offsets are absolute file
// offsets and the arithmetic (including its 32/16-bit wraparound and
// alignment quirks) mirrors saferwall's version.go for parity.
type peVersionBlob struct {
	data     []byte
	start    uint32
	fileSize int64
	base     uint32 // RVA of the resource, used for DWORD alignment
}

var errPEVersionRead = fmt.Errorf("%w: PE version resource read out of bounds", ErrInvalidInstaller)

// read returns up to n bytes at absolute offset off. Like saferwall, the
// range must lie within the file; it is truncated to the bytes buffered.
func (b *peVersionBlob) read(off, n uint32) ([]byte, error) {
	total := off + n
	if (total > off) != (n > 0) || int64(off) >= b.fileSize || int64(total) > b.fileSize || off < b.start {
		return nil, errPEVersionRead
	}
	rel := int64(off - b.start)
	if rel > int64(len(b.data)) {
		return nil, nil
	}
	end := min(rel+int64(n), int64(len(b.data)))
	return b.data[rel:end], nil
}

// readHeader reads the Length and ValueLength fields of a version structure.
func (b *peVersionBlob) readHeader(off uint32) (length, valueLength uint16, err error) {
	h, err := b.read(off, peVersionHeaderSize)
	if err != nil {
		return 0, 0, err
	}
	if len(h) < peVersionHeaderSize {
		return 0, 0, errPEVersionRead
	}
	return binary.LittleEndian.Uint16(h), binary.LittleEndian.Uint16(h[2:]), nil
}

func (b *peVersionBlob) readString(off, n uint32) (string, error) {
	raw, err := b.read(off, n)
	if err != nil {
		return "", err
	}
	return decodeUTF16String(raw)
}

func alignDword(offset, base uint32) uint32 {
	return ((offset + base + 3) & 0xfffffffc) - (base & 0xfffffffc)
}

func (b *peVersionBlob) parse(vers map[string]string) error {
	const (
		vsVersionInfoKey  = "VS_VERSION_INFO"
		stringFileInfoKey = "StringFileInfo"
	)

	verLength, verValueLength, err := b.readHeader(b.start)
	if err != nil {
		return err
	}
	key, err := b.readString(b.start+peVersionHeaderSize, uint32(verValueLength))
	if err != nil {
		return fmt.Errorf("%w: decoding VS_VERSION_INFO key: %w", ErrInvalidInstaller, err)
	}
	if key != vsVersionInfoKey {
		return fmt.Errorf("%w: invalid VS_VERSION_INFO block", ErrInvalidInstaller)
	}

	ffOff := alignDword(b.start+peVersionHeaderSize+uint32(2*len(vsVersionInfoKey))+1, b.base)
	ff, err := b.read(ffOff, peFixedFileInfoSize)
	if err != nil {
		return err
	}
	if len(ff) < peFixedFileInfoSize || binary.LittleEndian.Uint32(ff) != peFixedFileInfoSig {
		return fmt.Errorf("%w: invalid VS_FIXEDFILEINFO signature", ErrInvalidInstaller)
	}

	steps := 0
	step := func() error {
		steps++
		if steps > peMaxVersionInfoSteps {
			return fmt.Errorf("%w: PE version resource too complex", ErrInvalidInstaller)
		}
		return nil
	}

	// Offsets below are relative to the start of the resource.
	offset := alignDword(peVersionHeaderSize+uint32(2*len(vsVersionInfoKey))+1+peFixedFileInfoSize, b.base)
	for {
		if err := step(); err != nil {
			return err
		}
		sfiOff := alignDword(b.start+offset, b.base)
		sfiLength, _, err := b.readHeader(sfiOff)
		if err != nil {
			break
		}
		name, err := b.readString(sfiOff+peVersionHeaderSize, uint32(2*len(stringFileInfoKey))+1)
		if err != nil || sfiLength == 0 {
			break
		}

		if name == stringFileInfoKey {
			// saferwall only ever parses the first StringTable of a
			// StringFileInfo (and loops forever on a zero-length one).
			tableOffset := offset + peVersionHeaderSize + uint32(2*len(stringFileInfoKey)) + 1
			if err := b.parseStringTable(tableOffset, vers, step); err != nil {
				return err
			}
		}

		offset += uint32(sfiLength)
		if offset >= uint32(verLength) {
			break
		}
	}
	return nil
}

func (b *peVersionBlob) parseStringTable(tableOffset uint32, vers map[string]string, step func() error) error {
	const langIDSize = 8*2 + 1
	tOff := alignDword(b.start+tableOffset, b.base)
	tableLength, _, err := b.readHeader(tOff)
	if err != nil {
		return nil
	}
	langID, err := b.readString(tOff+peVersionHeaderSize, langIDSize)
	if err != nil || len(langID) != langIDSize/2 {
		return nil
	}

	end := tableOffset + uint32(tableLength)
	stringOffset := alignDword(tableOffset+peVersionHeaderSize+langIDSize, b.base)
	for stringOffset < end {
		if err := step(); err != nil {
			return err
		}
		k, v, l, err := b.parseString(stringOffset)
		if err != nil {
			break
		}
		vers[k] = v
		if l == 0 {
			stringOffset = end
		} else {
			stringOffset += uint32(l)
		}
	}
	return nil
}

func (b *peVersionBlob) parseString(rva uint32) (key, value string, length uint16, err error) {
	const maxKeySize = 100
	unaligned := b.start + rva
	off := alignDword(unaligned, b.base)
	padding := uint16(off - unaligned)

	sLength, sValueLength, err := b.readHeader(off)
	if err != nil {
		return "", "", 0, err
	}
	key, err = b.readString(off+peVersionHeaderSize, maxKeySize)
	if err != nil {
		return "", "", 0, err
	}
	valueOff := alignDword(uint32(2*(len(key)+1))+off+peVersionHeaderSize, b.base)
	value, err = b.readString(valueOff, uint32(2*(sValueLength+1)))
	if err != nil {
		return "", "", 0, err
	}
	// The caller advances from the unaligned offset, so include the padding.
	return key, value, sLength + padding, nil
}

// decodeUTF16String mirrors saferwall's DecodeUTF16String: it decodes up to
// the first pair of NUL bytes (not necessarily code-unit aligned).
func decodeUTF16String(b []byte) (string, error) {
	n := bytes.Index(b, []byte{0, 0})
	if n == 0 {
		return "", nil
	}
	s, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(b[0 : n+1])
	if err != nil {
		return "", err
	}
	return string(s), nil
}
