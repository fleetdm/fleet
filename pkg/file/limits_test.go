package file

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blakesmith/ar"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// setMemberLimit lowers the decompressed-member cap for the duration of a test
// so oversized fixtures stay small.
func setMemberLimit(t *testing.T, n int64) {
	t.Helper()
	prev := maxMetadataMemberSize
	maxMetadataMemberSize = n
	t.Cleanup(func() { maxMetadataMemberSize = prev })
}

func keepFileReader(t *testing.T, name string, contents []byte) *fleet.TempFileReader {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, contents, 0o600))
	tfr, err := fleet.NewKeepFileReader(path)
	require.NoError(t, err)
	t.Cleanup(func() { tfr.Close() })
	return tfr
}

func zlibCompress(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, err := zw.Write(b)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestExtractZIPMetadataRejectsOversizedInfoPlist(t *testing.T) {
	setMemberLimit(t, 1024)

	// A plist padded past the cap with a comment: still valid XML, so only the
	// size check can reject it.
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>com.example.big</string>
<key>CFBundleName</key><string>Big</string>
<key>CFBundleShortVersionString</key><string>1.0</string>
<key>LSRequiresIPhoneOS</key><true/>
</dict></plist>
<!-- ` + strings.Repeat("x", 4096) + ` -->`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("Payload/Big.app/Info.plist")
	require.NoError(t, err)
	_, err = w.Write([]byte(plist))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	tfr := keepFileReader(t, "big.ipa", buf.Bytes())
	_, err = ExtractZIPMetadata(tfr)
	require.ErrorIs(t, err, errMemberTooLarge)
}

func TestExtractDebMetadataRejectsOversizedControl(t *testing.T) {
	setMemberLimit(t, 1024)

	control := "Package: big\nVersion: 1.0\nDescription: " + strings.Repeat("x", 4096) + "\n"

	var tarBuf bytes.Buffer
	gz := gzip.NewWriter(&tarBuf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "./control", Mode: 0o644, Size: int64(len(control))}))
	_, err := tw.Write([]byte(control))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	var deb bytes.Buffer
	aw := ar.NewWriter(&deb)
	require.NoError(t, aw.WriteGlobalHeader())
	for _, m := range []struct {
		name string
		body []byte
	}{
		{"debian-binary", []byte("2.0\n")},
		{"control.tar.gz", tarBuf.Bytes()},
	} {
		require.NoError(t, aw.WriteHeader(&ar.Header{Name: m.name, Mode: 0o644, Size: int64(len(m.body))}))
		_, err := aw.Write(m.body)
		require.NoError(t, err)
	}

	_, err = ExtractDebMetadata(bytes.NewReader(deb.Bytes()))
	require.ErrorIs(t, err, errMemberTooLarge)
}

func TestReadCompressedFileRejectsOversizedMember(t *testing.T) {
	setMemberLimit(t, 1024)

	// Highly compressible: a few hundred compressed bytes expand past the cap.
	heap := zlibCompress(t, bytes.Repeat([]byte("A"), 64*1024))
	f := &xmlFile{Name: "Distribution", Data: &xmlFileData{Offset: 0, Length: int64(len(heap))}}
	f.Data.Encoding.Style = "application/x-gzip"

	_, err := readCompressedFile(bytes.NewReader(heap), 0, int64(len(heap)), f)
	require.ErrorIs(t, err, errMemberTooLarge)
}

func TestReadCompressedFileWithoutDataReturnsError(t *testing.T) {
	// A TOC <file> entry with no <data> child must fail cleanly rather than
	// dereference a nil pointer.
	f := &xmlFile{Name: "Distribution"}
	_, err := readCompressedFile(bytes.NewReader(nil), 0, 0, f)
	require.Error(t, err)
}

func TestDecodeXARTOCDataRejectsOversizedTOC(t *testing.T) {
	setMemberLimit(t, 1024)

	tocXML := `<?xml version="1.0"?><xar><toc>` + strings.Repeat(`<file><name>a</name></file>`, 2000) + `</toc></xar>`
	compressed := zlibCompress(t, []byte(tocXML))
	hdr := xarHeader{Magic: xarMagic, HeaderSize: xarHeaderSize, Version: 1, CompressedSize: int64(len(compressed)), HashType: hashSHA256}

	_, err := decodeXARTOCData(bytes.NewReader(compressed), hdr)
	require.ErrorIs(t, err, errMemberTooLarge)

	_, err = parseTOC(bytes.NewReader(compressed), hdrHash(t, hdr))
	require.ErrorIs(t, err, errMemberTooLarge)
}

func hdrHash(t *testing.T, hdr xarHeader) crypto.Hash {
	t.Helper()
	var raw bytes.Buffer
	require.NoError(t, binary.Write(&raw, binary.BigEndian, hdr))
	_, h, err := parseHeader(&raw)
	require.NoError(t, err)
	return h
}

func TestDecodePropertyTableRejectsOutOfRangeStringIndex(t *testing.T) {
	strs := []string{"ProductName", "Fleet"}
	tbl := &msiTable{Name: "Property", Cols: []msiColumn{{Attributes: uint16(msiString)}, {Attributes: uint16(msiStringLocalized)}}}

	for _, idx := range []uint16{0, 99} {
		// Two rows, column-major: keys then values. Index 1-based into strs.
		var b bytes.Buffer
		for _, v := range []uint16{1, idx, 2, 2} {
			require.NoError(t, binary.Write(&b, binary.LittleEndian, v))
		}
		_, err := decodePropertyTable(&b, tbl, strs)
		require.Error(t, err, "index %d", idx)
	}
}

func TestDecodePropertyTableColumnsRejectsOutOfRangeStringIndex(t *testing.T) {
	strs := []string{"Property", "Value"}

	for _, idx := range []uint16{0, 99} {
		// One row, column-major: table name id, col number, col name id, attrs.
		var b bytes.Buffer
		for _, v := range []uint16{idx, 1, 1, uint16(msiString)} {
			require.NoError(t, binary.Write(&b, binary.LittleEndian, v))
		}
		_, err := decodePropertyTableColumns(&b, strs)
		require.Error(t, err, "index %d", idx)
	}
}

func TestExtractInstallerMetadataReturnsErrorOnParserPanic(t *testing.T) {
	// 68-byte ar archive whose member header carries a mode field of "00"
	// followed by spaces, which makes the ar reader slice out of range.
	var deb bytes.Buffer
	deb.WriteString("!<arch>\n")
	deb.WriteString("debian-binary   ") // name, 16
	deb.WriteString("0           ")     // mtime, 12
	deb.WriteString("0     ")           // uid, 6
	deb.WriteString("0     ")           // gid, 6
	deb.WriteString("00      ")         // mode, 8: the trigger
	deb.WriteString("0         ")       // size, 10
	deb.WriteString("`\n")              // terminator, 2
	require.Len(t, deb.Bytes(), 68)

	tfr := keepFileReader(t, "panic.deb", deb.Bytes())
	require.NotPanics(t, func() {
		_, err := ExtractInstallerMetadata(tfr)
		require.Error(t, err)
	})
}
