// Package msi implements a pure-Go MSI (Windows Installer) file writer.
package msi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MSI stream name encoding
//
// MSI table streams use a custom base-64 encoding to pack table names into
// UTF-16 code points, doubling the effective name space within the CFB 31-char
// limit. This is the inverse of pkg/file/msi.go:msiDecodeName().

// msiEncodeRune maps a byte to its MSI base-64 index (0-63).
// Returns -1 for characters not in the MSI alphabet.
func msiEncodeRune(r byte) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'A' && r <= 'Z':
		return int(r-'A') + 10
	case r >= 'a' && r <= 'z':
		return int(r-'a') + 36
	case r == '.':
		return 62
	case r == '_':
		return 63
	}
	return -1
}

// msiEncodeName encodes an MSI stream name. If isTable is true, the "Table."
// prefix (encoded as U+4840) is prepended.
func msiEncodeName(name string, isTable bool) string {
	var out []rune
	if isTable {
		out = append(out, 0x4840)
	}

	i := 0
	for i < len(name) {
		v1 := msiEncodeRune(name[i])
		if v1 < 0 {
			// Character not in MSI alphabet — pass through as literal UTF-16.
			out = append(out, rune(name[i]))
			i++
			continue
		}

		if i+1 < len(name) {
			v2 := msiEncodeRune(name[i+1])
			if v2 >= 0 {
				// Pack two characters into one code point.
				out = append(out, rune(0x3800+(v2<<6)+v1)) //nolint:gosec // G115: MSI encoding value fits in int32 rune
				i += 2
				continue
			}
		}

		// Single character encoding.
		out = append(out, rune(0x4800+v1)) //nolint:gosec // G115: MSI encoding value fits in int32 rune
		i++
	}

	return string(out)
}

// Summary Information property set encoding
//
// The \x05SummaryInformation stream uses the OLE Property Set format,
// a binary format distinct from MSI table encoding.

// Summary Information property IDs.
const (
	pidCodepage   = 1
	pidTitle      = 2
	pidSubject    = 3
	pidAuthor     = 4
	pidKeywords   = 5
	pidComments   = 6
	pidTemplate   = 7
	pidLastAuthor = 8
	pidRevNumber  = 9
	pidPageCount  = 14
	pidWordCount  = 15
	pidSecurity   = 19
)

// OLE property types (VT_*).
const (
	vtI2    = 2  // 16-bit signed int
	vtI4    = 3  // 32-bit signed int
	vtLPSTR = 30 // Null-terminated ANSI string
)

// summaryInfoFMTID is the FMTID for the Summary Information property set.
// {F29F85E0-4FF9-1068-AB91-08002B27B3D9}
var summaryInfoFMTID = [16]byte{
	0xE0, 0x85, 0x9F, 0xF2, 0xF9, 0x4F, 0x68, 0x10,
	0xAB, 0x91, 0x08, 0x00, 0x2B, 0x27, 0xB3, 0xD9,
}

// SummaryInfo holds the properties for the MSI Summary Information stream.
type SummaryInfo struct {
	Title      string // "Installation Database"
	Subject    string // Product name
	Author     string // Manufacturer
	Keywords   string
	Comments   string
	Template   string // "x64;1033" or "Arm64;1033"
	LastAuthor string
	RevNumber  string // Package code GUID, e.g. "{GUID}"
	PageCount  int32  // InstallerVersion (e.g. 500)
	WordCount  int32  // Source type (2 = compressed short filenames)
	Security   int32  // 2 = read-only recommended
	Codepage   int16  // 1252
}

// NewSummaryInfo creates a SummaryInfo with default values for an MSI package.
func NewSummaryInfo(productName, manufacturer, arch, version string) SummaryInfo {
	template := "x64;1033"
	if strings.EqualFold(arch, "arm64") {
		template = "Arm64;1033"
	}

	packageCode := fmt.Sprintf("{%s}", strings.ToUpper(uuid.New().String()))

	return SummaryInfo{
		Title:      "Installation Database",
		Subject:    productName,
		Author:     manufacturer,
		Keywords:   productName,
		Comments:   fmt.Sprintf("This installer database contains the logic and data required to install %s.", productName),
		Template:   template,
		LastAuthor: manufacturer,
		RevNumber:  packageCode,
		PageCount:  500,
		WordCount:  2, // compressed, long filenames
		Security:   2, // read-only recommended
		Codepage:   1252,
	}
}

// Encode produces the binary content for the \x05SummaryInformation stream.
func (s SummaryInfo) Encode() []byte {
	// Build the property entries.
	type prop struct {
		id    uint32
		vtype uint16
		value any // string or int32 or int16
	}
	props := []prop{
		{pidCodepage, vtI2, s.Codepage},
		{pidTitle, vtLPSTR, s.Title},
		{pidSubject, vtLPSTR, s.Subject},
		{pidAuthor, vtLPSTR, s.Author},
		{pidKeywords, vtLPSTR, s.Keywords},
		{pidTemplate, vtLPSTR, s.Template},
		{pidLastAuthor, vtLPSTR, s.LastAuthor},
		{pidRevNumber, vtLPSTR, s.RevNumber},
		{pidPageCount, vtI4, s.PageCount},
		{pidWordCount, vtI4, s.WordCount},
		{pidSecurity, vtI4, s.Security},
	}
	if s.Comments != "" {
		props = append(props, prop{pidComments, vtLPSTR, s.Comments})
	}

	// Encode property values.
	type encodedProp struct {
		id   uint32
		data []byte // VT type (4 bytes) + value
	}
	var encoded []encodedProp
	for _, p := range props {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, uint32(p.vtype)) //nolint:errcheck
		switch v := p.value.(type) {
		case string:
			// VT_LPSTR: uint32 length (including null) + null-terminated string, padded to 4 bytes.
			str := v + "\x00"
			binary.Write(&buf, binary.LittleEndian, uint32(len(str))) //nolint:errcheck,gosec // G115: string length fits in uint32
			buf.WriteString(str)
			// Pad to 4-byte boundary.
			for buf.Len()%4 != 0 {
				buf.WriteByte(0)
			}
		case int32:
			binary.Write(&buf, binary.LittleEndian, v) //nolint:errcheck
		case int16:
			binary.Write(&buf, binary.LittleEndian, v) //nolint:errcheck
			// Pad VT_I2 to 4-byte alignment (type=4 + value=2 + pad=2 = 8 bytes).
			binary.Write(&buf, binary.LittleEndian, int16(0)) //nolint:errcheck
		}
		encoded = append(encoded, encodedProp{id: p.id, data: buf.Bytes()})
	}

	// Build the section data.
	// Section layout:
	//   uint32 sectionSize
	//   uint32 propertyCount
	//   [propertyCount] { uint32 propID, uint32 offset }
	//   [propertyCount] { property data }

	propCount := uint32(len(encoded)) //nolint:gosec // G115: property count fits in uint32
	idOffsetArraySize := propCount * 8 // 4 bytes ID + 4 bytes offset
	headerSize := uint32(8)            // sectionSize + propertyCount

	// Calculate offsets for each property's data.
	dataStart := headerSize + idOffsetArraySize
	offsets := make([]uint32, len(encoded))
	cur := dataStart
	for i, ep := range encoded {
		offsets[i] = cur
		cur += uint32(len(ep.data)) //nolint:gosec // G115: property data length fits in uint32
	}
	sectionSize := cur

	var section bytes.Buffer
	binary.Write(&section, binary.LittleEndian, sectionSize) //nolint:errcheck
	binary.Write(&section, binary.LittleEndian, propCount)   //nolint:errcheck
	for i, ep := range encoded {
		binary.Write(&section, binary.LittleEndian, ep.id)  //nolint:errcheck
		binary.Write(&section, binary.LittleEndian, offsets[i]) //nolint:errcheck
	}
	for _, ep := range encoded {
		section.Write(ep.data)
	}

	// Build the property set header.
	// Header:
	//   uint16 byteOrder = 0xFFFE
	//   uint16 version = 0
	//   uint32 systemIdentifier (OS version, we use 0)
	//   [16]byte CLSID (all zeros)
	//   uint32 numSections = 1
	//   [16]byte FMTID
	//   uint32 sectionOffset

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint16(0xFFFE)) //nolint:errcheck // byte order
	binary.Write(&out, binary.LittleEndian, uint16(0))      //nolint:errcheck // version
	binary.Write(&out, binary.LittleEndian, uint32(0x00020006)) //nolint:errcheck // OS: Win32, Windows NT
	out.Write(make([]byte, 16))                              // CLSID (zeros)
	binary.Write(&out, binary.LittleEndian, uint32(1))      //nolint:errcheck // numSections
	out.Write(summaryInfoFMTID[:])                           // FMTID
	sectionOffset := uint32(out.Len() + 4)                   //nolint:gosec // G115: section offset fits in uint32 // +4 for this offset field itself
	binary.Write(&out, binary.LittleEndian, sectionOffset)   //nolint:errcheck
	out.Write(section.Bytes())

	return out.Bytes()
}

// GenerateGUID returns a new uppercase GUID string in the format {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}.
func GenerateGUID() string {
	return fmt.Sprintf("{%s}", strings.ToUpper(uuid.New().String()))
}

// MSITimestamp returns the current time formatted for MSI usage.
func MSITimestamp() time.Time {
	return time.Now().UTC()
}
