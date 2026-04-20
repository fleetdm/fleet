package msi

import (
	"bytes"
	"encoding/binary"
)

// StringPool is a deduplicated string pool for MSI database tables.
// String indices are 1-based (0 means null/empty in table cells).
// This is the encoding inverse of pkg/file/msi.go:decodeStrings().
type StringPool struct {
	strings []string        // 0-indexed storage; MSI index = Go index + 1
	index   map[string]int  // string → 0-based index for dedup
	refs    []uint16        // reference counts parallel to strings
}

// NewStringPool creates an empty string pool.
func NewStringPool() *StringPool {
	return &StringPool{
		index: make(map[string]int),
	}
}

// Add adds a string to the pool and returns its 1-based MSI index.
// Every call increments the reference count (matching msibuild behavior where
// refcount = total number of uint16 references across all MSI streams).
// Empty strings return 0 (MSI null).
func (sp *StringPool) Add(s string) uint16 {
	if s == "" {
		return 0
	}
	if idx, ok := sp.index[s]; ok {
		sp.refs[idx]++
		return uint16(idx + 1) //nolint:gosec // G115: pool size bounded by table rows
	}
	idx := len(sp.strings)
	sp.strings = append(sp.strings, s)
	sp.refs = append(sp.refs, 1) // First reference → refcount = 1
	sp.index[s] = idx
	return uint16(idx + 1) //nolint:gosec // G115: pool size bounded by table rows
}

// AddRef is an alias for Add — kept for backward compatibility.
// Add now increments refcount on every call, matching msibuild behavior.
func (sp *StringPool) AddRef(s string) uint16 {
	return sp.Add(s)
}

// Lookup returns the 1-based MSI index for a string, or 0 if not found.
func (sp *StringPool) Lookup(s string) uint16 {
	if idx, ok := sp.index[s]; ok {
		return uint16(idx + 1) //nolint:gosec // G115: pool size bounded
	}
	return 0
}

// Count returns the number of strings in the pool.
func (sp *StringPool) Count() int {
	return len(sp.strings)
}

// EncodePool produces the _StringPool stream bytes.
// Format: uint16 codepage + uint16 unknown (0) + per-string (uint16 size, uint16 refcount).
// Refcounts are tracked by Add() — every call increments, matching msibuild behavior.
func (sp *StringPool) EncodePool() []byte {
	var buf bytes.Buffer
	// Header: codepage (1252 = Windows Latin-1) + pool type (0 = standard).
	// Windows Installer requires an explicit codepage for string decoding;
	// codepage 0 ("neutral") works on Linux with libmsi but the Windows engine
	// may fail to resolve string pool entries without a valid codepage.
	binary.Write(&buf, binary.LittleEndian, uint16(1252)) //nolint:errcheck
	binary.Write(&buf, binary.LittleEndian, uint16(0))    //nolint:errcheck

	for i, s := range sp.strings {
		strLen := len(s)
		if strLen <= 0xFFFF {
			binary.Write(&buf, binary.LittleEndian, uint16(strLen))  //nolint:errcheck,gosec
			binary.Write(&buf, binary.LittleEndian, sp.refs[i])     //nolint:errcheck
		} else {
			// Long string: size=0, refcount, then uint32 actual size
			binary.Write(&buf, binary.LittleEndian, uint16(0))      //nolint:errcheck
			binary.Write(&buf, binary.LittleEndian, sp.refs[i])     //nolint:errcheck
			binary.Write(&buf, binary.LittleEndian, uint32(strLen)) //nolint:errcheck,gosec
		}
	}

	return buf.Bytes()
}

// EncodeData produces the _StringData stream bytes.
// This is simply all strings concatenated in order (Windows-1252 encoded).
func (sp *StringPool) EncodeData() []byte {
	var buf bytes.Buffer
	for _, s := range sp.strings {
		buf.WriteString(s)
	}
	return buf.Bytes()
}
