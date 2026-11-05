package file

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/blakesmith/ar"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// setLimits lowers the decompressed-member caps for the duration of a test so
// oversized fixtures stay small.
func setLimits(t *testing.T, plist, member, toc int64) {
	t.Helper()
	prevPlist, prevMember, prevTOC := maxPlistSize, maxMetadataMemberSize, maxXARTOCSize
	maxPlistSize, maxMetadataMemberSize, maxXARTOCSize = plist, member, toc
	t.Cleanup(func() { maxPlistSize, maxMetadataMemberSize, maxXARTOCSize = prevPlist, prevMember, prevTOC })
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

func ipaWithInfoPlist(t *testing.T, plist string) *fleet.TempFileReader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("Payload/Big.app/Info.plist")
	require.NoError(t, err)
	_, err = w.Write([]byte(plist))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return keepFileReader(t, "big.ipa", buf.Bytes())
}

// paddedPlist is a valid iOS Info.plist followed by an XML comment, so only a
// size check can reject it.
func paddedPlist(padding int) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>com.example.big</string>
<key>CFBundleName</key><string>Big</string>
<key>CFBundleShortVersionString</key><string>1.0</string>
<key>LSRequiresIPhoneOS</key><true/>
</dict></plist>
<!-- ` + strings.Repeat("x", padding) + ` -->`
}

func TestExtractZIPMetadataInfoPlistLimit(t *testing.T) {
	plist := paddedPlist(4096)

	setLimits(t, int64(len(plist)), maxMetadataMemberSize, maxXARTOCSize)
	meta, err := ExtractZIPMetadata(ipaWithInfoPlist(t, plist))
	require.NoError(t, err)
	require.Equal(t, "com.example.big", meta.BundleIdentifier)

	setLimits(t, int64(len(plist))-1, maxMetadataMemberSize, maxXARTOCSize)
	_, err = ExtractZIPMetadata(ipaWithInfoPlist(t, plist))
	require.ErrorIs(t, err, ErrInvalidInstaller)
}

func TestExtractZIPMetadataRejectsNestedPlist(t *testing.T) {
	// howett.net/plist recurses once per nesting level and a Go stack overflow
	// is fatal, not recoverable. A million levels (about 15 MB of XML, a few KB
	// compressed) overflows the stack, so the size cap must reject it before
	// the parser runs.
	const depth = 1_000_000
	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0">` +
		strings.Repeat("<array>", depth) + strings.Repeat("</array>", depth) + `</plist>`

	_, err := ExtractZIPMetadata(ipaWithInfoPlist(t, plist))
	require.ErrorIs(t, err, ErrInvalidInstaller)
}

func debWithControl(t *testing.T, control string) []byte {
	t.Helper()
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
	return deb.Bytes()
}

func TestExtractDebMetadataControlLimit(t *testing.T) {
	control := "Package: big\nVersion: 1.0\nDescription: " + strings.Repeat("x", 4096) + "\n"
	deb := debWithControl(t, control)

	setLimits(t, maxPlistSize, int64(len(control)), maxXARTOCSize)
	meta, err := ExtractDebMetadata(bytes.NewReader(deb))
	require.NoError(t, err)
	require.Equal(t, "big", meta.Name)

	setLimits(t, maxPlistSize, int64(len(control))-1, maxXARTOCSize)
	_, err = ExtractDebMetadata(bytes.NewReader(deb))
	require.ErrorIs(t, err, ErrInvalidInstaller)
}

// buildXARWithTOC assembles a xar with a zlib-compressed TOC followed by heap.
func buildXARWithTOC(t *testing.T, tocXML string, heap []byte) []byte {
	t.Helper()
	compressed := zlibCompress(t, []byte(tocXML))
	hdr := xarHeader{
		Magic:            xarMagic,
		HeaderSize:       xarHeaderSize,
		Version:          1,
		CompressedSize:   int64(len(compressed)),
		UncompressedSize: int64(len(tocXML)),
		HashType:         hashSHA256,
	}
	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.BigEndian, hdr))
	buf.Write(compressed)
	buf.Write(heap)
	return buf.Bytes()
}

// xarWithMember builds a xar whose single top-level entry, named name, holds
// content compressed with zlib in the heap.
func xarWithMember(t *testing.T, name, content string) []byte {
	t.Helper()
	heap := zlibCompress(t, []byte(content))
	toc := fmt.Sprintf(`<?xml version="1.0"?><xar><toc><file id="1"><name>%s</name>`+
		`<data><length>%d</length><offset>0</offset><size>%d</size>`+
		`<encoding style="application/x-gzip"/></data></file></toc></xar>`, name, len(heap), len(content))
	return buildXARWithTOC(t, toc, heap)
}

func TestExtractXARMetadataMemberLimit(t *testing.T) {
	const distribution = `<installer-gui-script><title>Big</title>` +
		`<pkg-ref id="com.example.big" version="1.0"/></installer-gui-script>`
	const packageInfo = `<pkg-info identifier="com.example.big" version="1.0"/>`

	for _, tc := range []struct{ name, content string }{
		{"Distribution", distribution + "<!-- " + strings.Repeat("x", 4096) + " -->"},
		{"PackageInfo", packageInfo + "<!-- " + strings.Repeat("x", 4096) + " -->"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := keepFileReader(t, "big.pkg", xarWithMember(t, tc.name, tc.content))

			setLimits(t, maxPlistSize, int64(len(tc.content)), maxXARTOCSize)
			meta, err := ExtractXARMetadata(pkg)
			require.NoError(t, err)
			require.Equal(t, "com.example.big", meta.PackageIDs[0])

			setLimits(t, maxPlistSize, int64(len(tc.content))-1, maxXARTOCSize)
			_, err = ExtractXARMetadata(pkg)
			require.ErrorIs(t, err, ErrInvalidInstaller)
		})
	}
}

// xml.Unmarshal turns each empty element into a struct many times its size, so
// the default caps have to stay low enough to bound that.
func TestDefaultCapsStayLow(t *testing.T) {
	require.LessOrEqual(t, maxMetadataMemberSize, int64(2<<20))
	require.LessOrEqual(t, maxXARTOCSize, int64(4<<20))
}

// An empty <file/> allocates a full struct per 7 bytes of TOC.
func TestXARTOCDefaultCapRejectsEmptyFiles(t *testing.T) {
	const elem = "<file/>"
	toc := `<?xml version="1.0"?><xar><toc>` + strings.Repeat(elem, int(maxXARTOCSize)/len(elem)+1) + `</toc></xar>`
	xar := buildXARWithTOC(t, toc, nil)

	_, err := ExtractXARMetadata(keepFileReader(t, "big.pkg", xar))
	require.ErrorIs(t, err, ErrInvalidInstaller)
}

func TestXARTOCLimit(t *testing.T) {
	toc := `<?xml version="1.0"?><xar><toc>` + strings.Repeat(`<file><name>a</name></file>`, 2000) + `</toc></xar>`
	xar := buildXARWithTOC(t, toc, nil)

	setLimits(t, maxPlistSize, maxMetadataMemberSize, int64(len(toc))-1)

	_, err := ExtractXARMetadata(keepFileReader(t, "big.pkg", xar))
	require.ErrorIs(t, err, ErrInvalidInstaller)

	_, err = XARHasDistribution(bytes.NewReader(xar))
	require.ErrorIs(t, err, ErrInvalidInstaller)

	// the signature check decompresses the TOC separately
	hdr, h, err := parseHeader(bytes.NewReader(xar))
	require.NoError(t, err)
	_, err = parseTOC(bytes.NewReader(xar[xarHeaderSize:xarHeaderSize+hdr.CompressedSize]), h)
	require.ErrorIs(t, err, ErrInvalidInstaller)

	setLimits(t, maxPlistSize, maxMetadataMemberSize, int64(len(toc)))
	_, err = parseTOC(bytes.NewReader(xar[xarHeaderSize:xarHeaderSize+hdr.CompressedSize]), h)
	require.NoError(t, err)
}
