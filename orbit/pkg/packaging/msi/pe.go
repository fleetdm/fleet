package msi

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"strconv"
)

// peVersionInfo mirrors what MsiGetFileVersion returns for a file: the
// binary file version from VS_FIXEDFILEINFO formatted as "a.b.c.d", and the
// comma-separated decimal language IDs from \VarFileInfo\Translation.
// Both are empty when the file is not a PE image or carries no version
// resource — such files are "unversioned" in the File table and get a row in
// MsiFileHash instead.
type peVersionInfo struct {
	Version  string
	Language string
}

const (
	peResourceDirEntry = 2  // IMAGE_DIRECTORY_ENTRY_RESOURCE
	rtVersion          = 16 // RT_VERSION
)

// peFileVersion extracts version information from a PE file, returning the
// zero value (with no error) when the file is not a PE image or has no
// version resource.
func peFileVersion(path string) (peVersionInfo, error) {
	f, err := pe.Open(path)
	if err != nil {
		return peVersionInfo{}, nil // not a PE image: unversioned
	}
	defer f.Close()

	var rsrcVA, rsrcSize uint32
	switch oh := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		if len(oh.DataDirectory) > peResourceDirEntry {
			rsrcVA = oh.DataDirectory[peResourceDirEntry].VirtualAddress
			rsrcSize = oh.DataDirectory[peResourceDirEntry].Size
		}
	case *pe.OptionalHeader64:
		if len(oh.DataDirectory) > peResourceDirEntry {
			rsrcVA = oh.DataDirectory[peResourceDirEntry].VirtualAddress
			rsrcSize = oh.DataDirectory[peResourceDirEntry].Size
		}
	}
	if rsrcVA == 0 || rsrcSize == 0 {
		return peVersionInfo{}, nil
	}

	var section *pe.Section
	for _, s := range f.Sections {
		if rsrcVA >= s.VirtualAddress && rsrcVA < s.VirtualAddress+s.VirtualSize {
			section = s
			break
		}
	}
	if section == nil {
		return peVersionInfo{}, nil
	}
	data, err := section.Data()
	if err != nil {
		return peVersionInfo{}, fmt.Errorf("read resource section: %w", err)
	}
	// Offsets inside the resource directory are relative to the start of the
	// resource data (the section that contains rsrcVA).
	base := rsrcVA - section.VirtualAddress
	if base != 0 {
		if uint32(len(data)) < base { //nolint:gosec // resource offsets bounded by section size checks
			return peVersionInfo{}, nil
		}
		data = data[base:]
	}

	versionData := findVersionResource(data)
	if versionData == nil {
		return peVersionInfo{}, nil
	}
	// The resource data entry holds the RVA of the version blob.
	if len(versionData) < 8 {
		return peVersionInfo{}, nil
	}
	blobRVA := binary.LittleEndian.Uint32(versionData[0:4])
	blobSize := binary.LittleEndian.Uint32(versionData[4:8])
	if blobRVA < section.VirtualAddress || uint64(blobRVA)+uint64(blobSize) > uint64(section.VirtualAddress)+uint64(len(data))+uint64(base) {
		return peVersionInfo{}, nil
	}
	sectionData, err := section.Data()
	if err != nil {
		return peVersionInfo{}, fmt.Errorf("read resource section: %w", err)
	}
	off := blobRVA - section.VirtualAddress
	if uint64(off)+uint64(blobSize) > uint64(len(sectionData)) {
		return peVersionInfo{}, nil
	}
	return parseVersionInfoBlob(sectionData[off : off+blobSize]), nil
}

// findVersionResource walks the three-level resource directory tree
// (type / name / language) and returns the IMAGE_RESOURCE_DATA_ENTRY bytes
// for the first RT_VERSION resource, or nil.
func findVersionResource(rsrc []byte) []byte {
	entryOffset := findDirEntry(rsrc, 0, rtVersion)
	if entryOffset == 0 {
		return nil
	}
	// name level: first entry
	entryOffset = firstDirEntry(rsrc, entryOffset)
	if entryOffset == 0 {
		return nil
	}
	// language level: first entry
	entryOffset = firstDirEntry(rsrc, entryOffset)
	if entryOffset == 0 {
		return nil
	}
	if uint64(entryOffset)+8 > uint64(len(rsrc)) {
		return nil
	}
	return rsrc[entryOffset:]
}

// findDirEntry scans the resource directory at dirOffset for an entry with
// the given integer ID and returns the offset of the subdirectory or data
// entry it points to (with the high "is directory" bit stripped), or 0.
func findDirEntry(rsrc []byte, dirOffset, id uint32) uint32 {
	if uint64(dirOffset)+16 > uint64(len(rsrc)) {
		return 0
	}
	nNamed := binary.LittleEndian.Uint16(rsrc[dirOffset+12 : dirOffset+14])
	nID := binary.LittleEndian.Uint16(rsrc[dirOffset+14 : dirOffset+16])
	entries := dirOffset + 16
	for i := uint32(0); i < uint32(nNamed)+uint32(nID); i++ {
		off := entries + i*8
		if uint64(off)+8 > uint64(len(rsrc)) {
			return 0
		}
		name := binary.LittleEndian.Uint32(rsrc[off : off+4])
		target := binary.LittleEndian.Uint32(rsrc[off+4 : off+8])
		if name == id {
			return target &^ 0x80000000
		}
	}
	return 0
}

// firstDirEntry returns the target offset of the first entry of the resource
// directory at dirOffset (high bit stripped), or 0.
func firstDirEntry(rsrc []byte, dirOffset uint32) uint32 {
	if uint64(dirOffset)+16 > uint64(len(rsrc)) {
		return 0
	}
	nNamed := binary.LittleEndian.Uint16(rsrc[dirOffset+12 : dirOffset+14])
	nID := binary.LittleEndian.Uint16(rsrc[dirOffset+14 : dirOffset+16])
	if nNamed == 0 && nID == 0 {
		return 0
	}
	off := dirOffset + 16
	if uint64(off)+8 > uint64(len(rsrc)) {
		return 0
	}
	return binary.LittleEndian.Uint32(rsrc[off+4:off+8]) &^ 0x80000000
}

// parseVersionInfoBlob extracts the fixed file version and the first
// translation language from a VS_VERSIONINFO blob.
func parseVersionInfoBlob(blob []byte) peVersionInfo {
	var info peVersionInfo
	// VS_VERSIONINFO: wLength, wValueLength, wType, szKey "VS_VERSION_INFO",
	// padding, then VS_FIXEDFILEINFO (signature 0xFEEF04BD).
	fixedOff := -1
	for i := 0; i+4 <= len(blob) && i < 128; i += 4 {
		if binary.LittleEndian.Uint32(blob[i:i+4]) == 0xFEEF04BD {
			fixedOff = i
			break
		}
	}
	if fixedOff >= 0 && fixedOff+24 <= len(blob) {
		ms := binary.LittleEndian.Uint32(blob[fixedOff+8 : fixedOff+12])
		ls := binary.LittleEndian.Uint32(blob[fixedOff+12 : fixedOff+16])
		info.Version = fmt.Sprintf("%d.%d.%d.%d", ms>>16, ms&0xffff, ls>>16, ls&0xffff)
	}

	// Language: find the VarFileInfo\Translation value. Rather than fully
	// walking the nested block structure, locate the UTF-16LE "Translation"
	// key; its DWORD values follow at the next 4-byte boundary.
	key := utf16LEBytes("Translation")
	idx := indexBytes(blob, key)
	if idx >= 0 {
		// skip key and its NUL terminator, then align to 4 bytes
		v := idx + len(key) + 2
		v = (v + 3) &^ 3
		// MsiGetFileVersion reports the first translation's language ID.
		if v+4 <= len(blob) {
			if lang := binary.LittleEndian.Uint16(blob[v : v+2]); lang != 0 {
				info.Language = strconv.Itoa(int(lang))
			}
		}
	}
	return info
}

func indexBytes(haystack, needle []byte) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return i
		}
	}
	return -1
}
