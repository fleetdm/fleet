package msi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// summary.go writes the "\x05SummaryInformation" OLE property-set stream.
// The property set matches what WiX emits for the fleetd installer: codepage
// 1252, title/subject/author/keywords/comments from the Package element,
// the platform;language template, a fresh package code, create/save times,
// installer version 500, source flags 2 (compressed), and security 2
// (read-only recommended).

// vt (variant type) codes used in the property set.
const (
	vtI2       = 2
	vtI4       = 3
	vtLPSTR    = 30
	vtFILETIME = 64
)

var summaryFMTID = [16]byte{0xe0, 0x85, 0x9f, 0xf2, 0xf9, 0x4f, 0x68, 0x10, 0xab, 0x91, 0x08, 0x00, 0x2b, 0x27, 0xb3, 0xd9}

type summaryProperty struct {
	id  uint32
	typ uint32
	i   int32     // vtI2 / vtI4
	s   string    // vtLPSTR
	t   time.Time // vtFILETIME
}

// buildSummaryInfo builds the summary-information stream. template is e.g.
// "Arm64;1033" or "x64;1033"; packageCode is a fresh braced GUID.
func buildSummaryInfo(template, packageCode string, now time.Time) []byte {
	// The creating-application string matches the WiX toolset that the
	// fleetdm/wix image produced so Go-built and WiX-built installers
	// compare byte-for-byte.
	props := []summaryProperty{
		{id: 1, typ: vtI2, i: 1252},
		{id: 2, typ: vtLPSTR, s: "Installation Database"},
		{id: 3, typ: vtLPSTR, s: "Fleet osquery"},
		{id: 4, typ: vtLPSTR, s: "Fleet Device Management (fleetdm.com)"},
		{id: 5, typ: vtLPSTR, s: "Fleet osquery"},
		{id: 6, typ: vtLPSTR, s: "This installer database contains the logic and data required to install Fleet osquery."},
		{id: 7, typ: vtLPSTR, s: template},
		{id: 9, typ: vtLPSTR, s: packageCode},
		{id: 12, typ: vtFILETIME, t: now},
		{id: 13, typ: vtFILETIME, t: now},
		{id: 14, typ: vtI4, i: 500}, // InstallerVersion
		{id: 15, typ: vtI4, i: 2},   // word count: compressed source
		{id: 18, typ: vtLPSTR, s: "Windows Installer XML Toolset ()"},
		{id: 19, typ: vtI4, i: 2}, // security: read-only recommended
	}

	le := binary.LittleEndian

	// Serialize property values to compute offsets.
	var values bytes.Buffer
	offsets := make([]uint32, len(props))
	// Offsets are relative to the section start: size + count (8 bytes)
	// followed by the id/offset pairs.
	headerLen := 8 + 8*len(props)
	for i, p := range props {
		offsets[i] = uint32(headerLen + values.Len()) //nolint:gosec // OLE property set field; values bounded by the format
		var tb [4]byte
		le.PutUint32(tb[:], p.typ)
		values.Write(tb[:])
		switch p.typ {
		case vtI2:
			var vb [4]byte                    // padded to 4 bytes
			le.PutUint16(vb[:2], uint16(p.i)) //nolint:gosec // OLE property set field; values bounded by the format
			values.Write(vb[:])
		case vtI4:
			var vb [4]byte
			le.PutUint32(vb[:], uint32(p.i)) //nolint:gosec // OLE property set field; values bounded by the format
			values.Write(vb[:])
		case vtLPSTR:
			var lb [4]byte
			le.PutUint32(lb[:], uint32(len(p.s)+1)) //nolint:gosec // OLE property set field; values bounded by the format
			values.Write(lb[:])
			values.WriteString(p.s)
			values.WriteByte(0)
			for values.Len()%4 != 0 {
				values.WriteByte(0)
			}
		case vtFILETIME:
			var vb [8]byte
			le.PutUint64(vb[:], uint64(filetime(p.t))) //nolint:gosec // OLE property set field; values bounded by the format
			values.Write(vb[:])
		}
	}
	sectionSize := uint32(headerLen + values.Len()) //nolint:gosec // OLE property set field; values bounded by the format

	var out bytes.Buffer
	var hdr [28]byte
	le.PutUint16(hdr[0:], 0xFFFE)     // byte order
	le.PutUint16(hdr[2:], 0)          // format
	le.PutUint16(hdr[4:], 5)          // OS version
	le.PutUint16(hdr[6:], 2)          // OS: Win32
	copy(hdr[8:24], make([]byte, 16)) // CLSID: zero
	le.PutUint32(hdr[24:], 1)         // one section
	out.Write(hdr[:])
	out.Write(summaryFMTID[:])
	var so [4]byte
	le.PutUint32(so[:], 48) // section offset: 28 header + 20 fmtid/offset
	out.Write(so[:])

	var sh [8]byte
	le.PutUint32(sh[0:], sectionSize)
	le.PutUint32(sh[4:], uint32(len(props))) //nolint:gosec // OLE property set field; values bounded by the format
	out.Write(sh[:])
	for i, p := range props {
		var pb [8]byte
		le.PutUint32(pb[0:], p.id)
		le.PutUint32(pb[4:], offsets[i])
		out.Write(pb[:])
	}
	out.Write(values.Bytes())
	return out.Bytes()
}

// filetime converts a time to a Windows FILETIME (100ns intervals since
// 1601-01-01 UTC).
func filetime(t time.Time) int64 {
	const epochDiff = 116444736000000000 // 100ns units between 1601 and 1970
	return t.UnixNano()/100 + epochDiff
}

// summaryTemplate returns the summary-information template property for an
// architecture.
func summaryTemplate(arch string) (string, error) {
	switch arch {
	case "amd64":
		return "x64;1033", nil
	case "arm64":
		return "Arm64;1033", nil
	}
	return "", fmt.Errorf("unsupported MSI architecture %q", arch)
}
