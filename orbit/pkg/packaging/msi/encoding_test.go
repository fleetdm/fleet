package msi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMSIEncodeRune(t *testing.T) {
	// Verify the encoding table matches the decoding in pkg/file/msi.go.
	assert.Equal(t, 0, msiEncodeRune('0'))
	assert.Equal(t, 9, msiEncodeRune('9'))
	assert.Equal(t, 10, msiEncodeRune('A'))
	assert.Equal(t, 35, msiEncodeRune('Z'))
	assert.Equal(t, 36, msiEncodeRune('a'))
	assert.Equal(t, 61, msiEncodeRune('z'))
	assert.Equal(t, 62, msiEncodeRune('.'))
	assert.Equal(t, 63, msiEncodeRune('_'))
	assert.Equal(t, -1, msiEncodeRune(' '))
}

// msiDecodeName is a copy of the decoder from pkg/file/msi.go for round-trip testing.
func msiDecodeName(msiName string) string {
	var out strings.Builder
	for _, x := range msiName {
		switch {
		case x >= 0x3800 && x < 0x4800:
			x -= 0x3800
			out.WriteRune(msiDecodeRune(x & 0x3f))
			out.WriteRune(msiDecodeRune(x >> 6))
		case x >= 0x4800 && x < 0x4840:
			x -= 0x4800
			out.WriteRune(msiDecodeRune(x))
		case x == 0x4840:
			out.WriteString("Table.")
		default:
			out.WriteRune(x)
		}
	}
	return out.String()
}

func msiDecodeRune(x rune) rune {
	if x < 10 {
		return x + '0'
	} else if x < 10+26 {
		return x - 10 + 'A'
	} else if x < 10+26+26 {
		return x - 10 - 26 + 'a'
	} else if x == 10+26+26 {
		return '.'
	}
	return '_'
}

func TestMSIEncodeName_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		isTable bool
	}{
		{"Property", true},
		{"_StringData", false},
		{"_StringPool", false},
		{"_Columns", false},
		{"_Tables", false},
		{"File", true},
		{"Component", true},
		{"Directory", true},
		{"Feature", true},
		{"Media", true},
		{"ServiceInstall", true},
		{"CustomAction", true},
		{"Registry", true},
		{"InstallExecuteSequence", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := msiEncodeName(tc.name, tc.isTable)
			decoded := msiDecodeName(encoded)

			expected := tc.name
			if tc.isTable {
				expected = "Table." + tc.name
			}
			assert.Equal(t, expected, decoded, "round-trip failed for %q", tc.name)
		})
	}
}

func TestMSIEncodeName_SpecialStreams(t *testing.T) {
	// Non-table streams (like _StringData) should not have the Table. prefix.
	encoded := msiEncodeName("_StringData", false)
	decoded := msiDecodeName(encoded)
	assert.Equal(t, "_StringData", decoded)
}

func TestSummaryInfoEncode(t *testing.T) {
	si := NewSummaryInfo("Fleet osquery", "Fleet Device Management", "x64", "1.0.0")
	data := si.Encode()

	require.NotEmpty(t, data)
	// Check OLE property set header.
	assert.Equal(t, byte(0xFE), data[0]) // Byte order low
	assert.Equal(t, byte(0xFF), data[1]) // Byte order high
	// Check that the data is reasonably sized.
	assert.Greater(t, len(data), 100)
}
