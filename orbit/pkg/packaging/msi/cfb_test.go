package msi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"unicode/utf16"

	"github.com/sassoftware/relic/v8/lib/comdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCFBWriter_Basic(t *testing.T) {
	cw := newCFBWriter()
	cw.addStream("TestStream", []byte("hello world"))
	cw.addStream("AnotherStream", []byte("test data"))

	var buf bytes.Buffer
	err := cw.writeTo(&buf)
	require.NoError(t, err)

	// Verify the output starts with CFB magic.
	data := buf.Bytes()
	require.Greater(t, len(data), 512)
	assert.Equal(t, byte(0xD0), data[0])
	assert.Equal(t, byte(0xCF), data[1])
	assert.Equal(t, byte(0x11), data[2])
	assert.Equal(t, byte(0xE0), data[3])
	assert.Equal(t, byte(0xA1), data[4])
	assert.Equal(t, byte(0xB1), data[5])
	assert.Equal(t, byte(0x1A), data[6])
	assert.Equal(t, byte(0xE1), data[7])
}

func TestCFBWriter_ReadBack(t *testing.T) {
	// Write a CFB and read it back with comdoc to verify structural validity.
	cw := newCFBWriter()
	cw.addStream("Stream1", []byte("content one"))
	cw.addStream("Stream2", []byte("content two with more data to test"))

	var buf bytes.Buffer
	require.NoError(t, cw.writeTo(&buf))

	// Read back with comdoc.
	reader := bytes.NewReader(buf.Bytes())
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	// List directory entries.
	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	names := make(map[string]struct{})
	for _, e := range entries {
		if e.Type == comdoc.DirStream {
			names[e.Name()] = struct{}{}
		}
	}

	_, hasStream1 := names["Stream1"]
	assert.True(t, hasStream1, "Stream1 should exist")
	_, hasStream2 := names["Stream2"]
	assert.True(t, hasStream2, "Stream2 should exist")

	// Read stream contents.
	for _, e := range entries {
		if e.Name() == "Stream1" {
			r, err := doc.ReadStream(e)
			require.NoError(t, err)
			var content bytes.Buffer
			_, err = content.ReadFrom(r)
			require.NoError(t, err)
			assert.Equal(t, "content one", content.String())
		}
	}
}

func TestCFBWriter_Empty(t *testing.T) {
	// A CFB with no streams should still produce a valid file with the magic header.
	cw := newCFBWriter()

	var buf bytes.Buffer
	require.NoError(t, cw.writeTo(&buf))

	data := buf.Bytes()
	require.Greater(t, len(data), 512)
	// Check magic.
	assert.Equal(t, byte(0xD0), data[0])
	assert.Equal(t, byte(0xCF), data[1])
}

func TestCFBWriter_LargeStream(t *testing.T) {
	// Test with a stream larger than one sector (512 bytes).
	cw := newCFBWriter()
	largeData := make([]byte, 2000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	cw.addStream("LargeStream", largeData)

	var buf bytes.Buffer
	require.NoError(t, cw.writeTo(&buf))

	reader := bytes.NewReader(buf.Bytes())
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	for _, e := range entries {
		if e.Name() == "LargeStream" {
			r, err := doc.ReadStream(e)
			require.NoError(t, err)
			var content bytes.Buffer
			_, err = content.ReadFrom(r)
			require.NoError(t, err)
			assert.Equal(t, largeData, content.Bytes())
		}
	}
}

func TestCFBWriter_DIFAT(t *testing.T) {
	// When the file is large enough to need >109 FAT sectors, DIFAT
	// continuation sectors are required. 109 FAT sectors cover
	// 109 * 128 = 13,952 sectors = ~6.8 MB of data. Exceed that threshold.
	cw := newCFBWriter()
	// ~8 MB stream → needs ~15,700 data sectors → ~123 FAT sectors → triggers DIFAT.
	largeData := make([]byte, 8*1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 251) // prime to avoid trivial patterns
	}
	cw.addStream("BigStream", largeData)

	var buf bytes.Buffer
	require.NoError(t, cw.writeTo(&buf))

	// Read back with comdoc to verify structural validity.
	reader := bytes.NewReader(buf.Bytes())
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	found := false
	for _, e := range entries {
		if e.Name() == "BigStream" {
			found = true
			r, err := doc.ReadStream(e)
			require.NoError(t, err)
			var content bytes.Buffer
			_, err = content.ReadFrom(r)
			require.NoError(t, err)
			assert.Equal(t, largeData, content.Bytes())
		}
	}
	assert.True(t, found, "BigStream should exist")
}

func TestCFBWriter_DIFATMultipleStreams(t *testing.T) {
	// Trigger DIFAT with multiple streams totaling well over the threshold.
	cw := newCFBWriter()
	streamCount := 10
	streamSize := 1024 * 1024 // 1 MB each → 10 MB total
	streams := make([][]byte, streamCount)
	for i := range streamCount {
		data := make([]byte, streamSize)
		for j := range data {
			data[j] = byte((i + j) % 256)
		}
		streams[i] = data
		cw.addStream(fmt.Sprintf("Stream%02d", i), data)
	}

	var buf bytes.Buffer
	require.NoError(t, cw.writeTo(&buf))

	reader := bytes.NewReader(buf.Bytes())
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	names := make(map[string]struct{})
	for _, e := range entries {
		if e.Type == comdoc.DirStream {
			names[e.Name()] = struct{}{}
		}
	}
	for i := range streamCount {
		name := fmt.Sprintf("Stream%02d", i)
		_, ok := names[name]
		assert.True(t, ok, "%s should exist", name)
	}
}

// TestCFBWriter_BSTLookup verifies that every stream in a full MSI can be
// found via BST lookup (as Windows does), not just DFS walk (as comdoc does).
func TestCFBWriter_BSTLookup(t *testing.T) {
	// Build an MSI with all 18 tables + system streams.
	tmpDir := t.TempDir()
	rootDir := tmpDir + "/root"
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(rootDir+"/test.txt", []byte("test"), 0o644))

	opts := MSIOptions{
		ProductName:    "Test", ProductVersion: "1.0.0",
		Manufacturer: "Test", UpgradeCode: "{00000000-0000-0000-0000-000000000000}",
		Architecture: "amd64", OrbitChannel: "stable",
		OsquerydChannel: "stable", DesktopChannel: "stable",
	}
	var buf bytes.Buffer
	require.NoError(t, WriteMSI(&buf, rootDir, opts))
	data := buf.Bytes()

	// Parse the CFB header to find directory start.
	dirStartSec := int32(binary.LittleEndian.Uint32(data[48:52]))

	// Read FAT to follow sector chains.
	numFAT := int(binary.LittleEndian.Uint32(data[44:48]))
	fatSectors := make([]int32, 0, numFAT)
	for i := range numFAT {
		if i < 109 {
			fatSectors = append(fatSectors, int32(binary.LittleEndian.Uint32(data[76+i*4:])))
		}
	}

	// Build FAT array.
	fat := make([]int32, 0)
	for _, fs := range fatSectors {
		off := int(fs+1) * 512
		for j := range 128 {
			fat = append(fat, int32(binary.LittleEndian.Uint32(data[off+j*4:])))
		}
	}

	// Read directory chain.
	var dirData []byte
	for sec := dirStartSec; sec >= 0; sec = fat[sec] {
		off := int(sec+1) * 512
		dirData = append(dirData, data[off:off+512]...)
	}

	// Parse directory entries.
	type rawDirEntry struct {
		nameRunes [32]uint16
		nameLen   uint16
		entType   uint8
		color     uint8
		left      int32
		right     int32
		root      int32
	}

	numEntries := len(dirData) / 128
	entries := make([]rawDirEntry, numEntries)
	for i := range numEntries {
		base := i * 128
		var e rawDirEntry
		for j := range 32 {
			e.nameRunes[j] = binary.LittleEndian.Uint16(dirData[base+j*2:])
		}
		e.nameLen = binary.LittleEndian.Uint16(dirData[base+64:])
		e.entType = dirData[base+66]
		e.color = dirData[base+67]
		e.left = int32(binary.LittleEndian.Uint32(dirData[base+68:]))
		e.right = int32(binary.LittleEndian.Uint32(dirData[base+72:]))
		e.root = int32(binary.LittleEndian.Uint32(dirData[base+76:]))
		entries[i] = e
	}

	getName := func(e rawDirEntry) string {
		if e.nameLen == 0 { return "" }
		nChars := int(e.nameLen)/2 - 1 // exclude null terminator
		runes := make([]uint16, nChars)
		copy(runes, e.nameRunes[:nChars])
		return string(utf16.Decode(runes))
	}

	// cfbCompare implements the CFB spec comparison: length first, then case-insensitive.
	cfbCompare := func(a, b []uint16) int {
		if len(a) != len(b) {
			if len(a) < len(b) { return -1 }
			return 1
		}
		for i := range a {
			au, bu := a[i], b[i]
			if au >= 'a' && au <= 'z' { au -= 0x20 }
			if bu >= 'a' && bu <= 'z' { bu -= 0x20 }
			if au != bu {
				if au < bu { return -1 }
				return 1
			}
		}
		return 0
	}

	// BST lookup: start at root's storageRoot, follow left/right.
	bstLookup := func(targetName string) (int, bool) {
		target := utf16.Encode([]rune(targetName))
		rootEntry := entries[0]
		cur := rootEntry.root
		depth := 0
		for cur >= 0 && cur < int32(numEntries) && depth < 100 {
			e := entries[cur]
			nChars := int(e.nameLen)/2 - 1
			entName := make([]uint16, nChars)
			copy(entName, e.nameRunes[:nChars])

			cmp := cfbCompare(target, entName)
			if cmp == 0 {
				return int(cur), true
			}
			if cmp < 0 {
				cur = e.left
			} else {
				cur = e.right
			}
			depth++
		}
		return -1, false
	}

	// Collect all stream names via DFS (what comdoc does).
	var dfsNames []string
	var dfs func(idx int32)
	dfs = func(idx int32) {
		if idx < 0 || idx >= int32(numEntries) { return }
		e := entries[idx]
		if e.entType == 2 { // DirStream
			dfsNames = append(dfsNames, getName(e))
		}
		dfs(e.left)
		dfs(e.right)
	}
	dfs(entries[0].root)

	t.Logf("Found %d streams via DFS", len(dfsNames))

	// Now verify each stream can be found via BST lookup.
	for _, name := range dfsNames {
		idx, found := bstLookup(name)
		if !found {
			t.Errorf("BST lookup FAILED for stream %q (decoded: %s)", name, msiDecodeName(name))
		} else {
			t.Logf("BST lookup OK for %q → dirEntry[%d] (decoded: %s)", name, idx, msiDecodeName(name))
		}
	}
}

func TestCFBNameLess(t *testing.T) {
	// CFB name comparison: shorter UTF-16 names first, then case-insensitive.
	assert.True(t, cfbNameLess("A", "AB"))
	assert.False(t, cfbNameLess("AB", "A"))
	assert.True(t, cfbNameLess("abc", "def"))
	assert.False(t, cfbNameLess("def", "abc"))
	// Same length, case-insensitive.
	assert.True(t, cfbNameLess("abc", "DEF"))

	// MSI-encoded names have fewer UTF-16 code units than UTF-8 bytes.
	// Encoded "Property" table = "\u4840\u3CA3\u4062\u45B1\u4492" (5 UTF-16 code units, 15 UTF-8 bytes).
	msiProp := msiEncodeName("Property", true)
	// "orbit.cab" = 9 UTF-16 code units, 9 UTF-8 bytes.
	// In UTF-16 comparison: msiProp (5 units) < "orbit.cab" (9 units).
	assert.True(t, cfbNameLess(msiProp, "orbit.cab"), "MSI-encoded name (5 UTF-16 units) should sort before orbit.cab (9 units)")
	assert.False(t, cfbNameLess("orbit.cab", msiProp))
}
