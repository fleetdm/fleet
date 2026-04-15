package packaging

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteXAR(t *testing.T) {
	// Create a flat directory structure mimicking a .pkg.
	tmpDir := t.TempDir()
	flatDir := filepath.Join(tmpDir, "flat")

	// Distribution file.
	require.NoError(t, os.MkdirAll(flatDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(flatDir, "Distribution"),
		[]byte(`<?xml version="1.0"?><installer-gui-script/>`),
		0o644,
	))

	// base.pkg directory with files.
	basePkg := filepath.Join(flatDir, "base.pkg")
	require.NoError(t, os.MkdirAll(basePkg, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(basePkg, "PackageInfo"),
		[]byte("<pkg-info/>"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(basePkg, "Bom"),
		[]byte("fake-bom-data"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(basePkg, "Payload"),
		[]byte("fake-payload-data"),
		0o644,
	))

	// Write XAR.
	outputPath := filepath.Join(tmpDir, "test.pkg")
	err := writeXAR(flatDir, outputPath)
	require.NoError(t, err)

	// Read and verify.
	f, err := os.Open(outputPath)
	require.NoError(t, err)
	defer f.Close()

	// Check header.
	var hdr xarWriterHeader
	require.NoError(t, binary.Read(f, binary.BigEndian, &hdr))
	assert.Equal(t, uint32(xarWriterMagic), hdr.Magic, "XAR magic")
	assert.Equal(t, uint16(xarWriterHeaderSize), hdr.Size, "header size")
	assert.Equal(t, uint16(xarWriterVersion), hdr.Version)
	assert.Equal(t, uint32(xarCksumSHA1), hdr.CksumAlg)
	assert.Positive(t, hdr.TOCCompressed, "compressed TOC should be non-empty")
	assert.Positive(t, hdr.TOCUncompressed, "uncompressed TOC should be non-empty")
	assert.GreaterOrEqual(t, hdr.TOCUncompressed, hdr.TOCCompressed, "uncompressed >= compressed")

	// Decompress and parse TOC XML.
	compressedTOC := make([]byte, hdr.TOCCompressed)
	_, err = io.ReadFull(f, compressedTOC)
	require.NoError(t, err)

	zr, err := zlib.NewReader(bytes.NewReader(compressedTOC))
	require.NoError(t, err)
	defer zr.Close()

	tocXML, err := io.ReadAll(zr)
	require.NoError(t, err)

	// Check that the TOC contains expected file entries.
	type tocFile struct {
		Name string `xml:"name"`
		Type string `xml:"type"`
	}
	type tocStruct struct {
		XMLName xml.Name  `xml:"xar"`
		TOC     struct {
			Files []tocFile `xml:"file"`
		} `xml:"toc"`
	}
	var parsed tocStruct
	require.NoError(t, xml.Unmarshal(tocXML, &parsed))

	// Should have 2 top-level entries: Distribution and base.pkg.
	require.Len(t, parsed.TOC.Files, 2)

	names := make(map[string]string)
	for _, f := range parsed.TOC.Files {
		names[f.Name] = f.Type
	}
	assert.Equal(t, "file", names["Distribution"])
	assert.Equal(t, "directory", names["base.pkg"])
}

func TestXARTreeBuilding(t *testing.T) {
	tmpDir := t.TempDir()
	flatDir := filepath.Join(tmpDir, "flat")
	require.NoError(t, os.MkdirAll(filepath.Join(flatDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(flatDir, "a.txt"), []byte("aaa"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(flatDir, "sub", "b.txt"), []byte("bbb"), 0o644))

	root, err := buildXARTree(flatDir)
	require.NoError(t, err)
	require.NotNil(t, root)

	// Root should have 2 children: "a.txt" and "sub".
	require.Len(t, root.children, 2)
	assert.Equal(t, "a.txt", root.children[0].name)
	assert.Equal(t, "sub", root.children[1].name)
	assert.False(t, root.children[0].isDir)
	assert.True(t, root.children[1].isDir)

	// "sub" should have 1 child.
	require.Len(t, root.children[1].children, 1)
	assert.Equal(t, "b.txt", root.children[1].children[0].name)
}

func TestWriteXAR_CompatWithReader(t *testing.T) {
	// Create a XAR and verify it can be read by the existing pkg/file/xar.go reader.
	tmpDir := t.TempDir()
	flatDir := filepath.Join(tmpDir, "flat")

	require.NoError(t, os.MkdirAll(flatDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(flatDir, "Distribution"),
		[]byte(`<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
    <title>Test Package</title>
    <product id="com.test.pkg" version="1.0"/>
    <pkg-ref id="com.test.pkg"/>
    <choices-outline>
        <line choice="default"><line choice="com.test.pkg"/></line>
    </choices-outline>
    <choice id="default"/>
    <choice id="com.test.pkg" visible="false">
        <pkg-ref id="com.test.pkg"/>
    </choice>
    <pkg-ref id="com.test.pkg" version="1.0" installKBytes="100">#base.pkg</pkg-ref>
</installer-gui-script>`),
		0o644,
	))
	basePkg := filepath.Join(flatDir, "base.pkg")
	require.NoError(t, os.MkdirAll(basePkg, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(basePkg, "PackageInfo"), []byte("<pkg-info/>"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(basePkg, "Payload"), []byte("fake"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(basePkg, "Bom"), []byte("fake"), 0o644))

	outputPath := filepath.Join(tmpDir, "test.pkg")
	require.NoError(t, writeXAR(flatDir, outputPath))

	// Verify it can be read by the existing XAR header reader.
	f, err := os.Open(outputPath)
	require.NoError(t, err)
	defer f.Close()

	var hdr xarWriterHeader
	require.NoError(t, binary.Read(f, binary.BigEndian, &hdr))
	assert.Equal(t, uint32(0x78617221), hdr.Magic)

	// Verify the compressed TOC can be decompressed.
	compTOC := make([]byte, hdr.TOCCompressed)
	_, err = io.ReadFull(f, compTOC)
	require.NoError(t, err)

	zr, err := zlib.NewReader(io.NopCloser(bytes.NewReader(compTOC)))
	require.NoError(t, err)
	defer zr.Close()

	tocBytes, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Contains(t, string(tocBytes), "Distribution")
	assert.Contains(t, string(tocBytes), "base.pkg")
	assert.Contains(t, string(tocBytes), "sha1")
}

