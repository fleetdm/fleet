package msi

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCab_Basic(t *testing.T) {
	files := []CabFile{
		{Name: "hello.txt", Data: []byte("hello world"), ModTime: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
		{Name: "test.dat", Data: []byte("test data content"), ModTime: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)},
	}

	var buf bytes.Buffer
	n, err := WriteCab(&buf, files)
	require.NoError(t, err)
	assert.Equal(t, int64(buf.Len()), n)

	data := buf.Bytes()
	// Check MSCF magic.
	assert.Equal(t, byte('M'), data[0])
	assert.Equal(t, byte('S'), data[1])
	assert.Equal(t, byte('C'), data[2])
	assert.Equal(t, byte('F'), data[3])

	// Check cabinet size matches actual size.
	cbCabinet := binary.LittleEndian.Uint32(data[8:12])
	assert.Equal(t, uint32(buf.Len()), cbCabinet) //nolint:gosec // G115

	// Check version.
	assert.Equal(t, byte(3), data[0x18]) // versionMinor
	assert.Equal(t, byte(1), data[0x19]) // versionMajor

	// Check file count.
	cFiles := binary.LittleEndian.Uint16(data[0x1C:0x1E])
	assert.Equal(t, uint16(2), cFiles)

	// Check folder count.
	cFolders := binary.LittleEndian.Uint16(data[0x1A:0x1C])
	assert.Equal(t, uint16(1), cFolders)
}

func TestWriteCab_EmptyFile(t *testing.T) {
	files := []CabFile{
		{Name: "empty.txt", Data: []byte{}, ModTime: time.Now()},
	}

	var buf bytes.Buffer
	_, err := WriteCab(&buf, files)
	require.NoError(t, err)

	// Should still have valid header.
	data := buf.Bytes()
	assert.Equal(t, byte('M'), data[0])
	assert.Equal(t, byte('S'), data[1])
}

func TestWriteCab_LargeFile(t *testing.T) {
	// Test with data larger than one 32KB block to verify multi-block compression.
	largeData := make([]byte, 50000)
	for i := range largeData {
		largeData[i] = byte(i % 251) // Use a prime to avoid trivial patterns.
	}

	files := []CabFile{
		{Name: "large.bin", Data: largeData, ModTime: time.Now()},
	}

	var buf bytes.Buffer
	_, err := WriteCab(&buf, files)
	require.NoError(t, err)

	data := buf.Bytes()
	// Verify compression type is NONE (stored).
	// CFFOLDER starts at offset 36 (after CFHEADER).
	compressType := binary.LittleEndian.Uint16(data[36+4+2 : 36+4+2+2])
	assert.Equal(t, uint16(cabCompressNone), compressType)
}

func TestDosDateTime(t *testing.T) {
	tm := time.Date(2024, 3, 15, 14, 30, 42, 0, time.UTC)
	date, dosTime := dosDateTime(tm)

	// Date: ((2024-1980) << 9) | (3 << 5) | 15 = (44 << 9) | (3 << 5) | 15
	expectedDate := uint16((44 << 9) | (3 << 5) | 15)
	assert.Equal(t, expectedDate, date)

	// Time: (14 << 11) | (30 << 5) | (42/2) = (14 << 11) | (30 << 5) | 21
	expectedTime := uint16((14 << 11) | (30 << 5) | 21)
	assert.Equal(t, expectedTime, dosTime)
}

func TestCabChecksum(t *testing.T) {
	// Basic checksum test: verify it produces a non-zero result for non-empty data.
	payload := []byte("hello world test data")
	csum := cabChecksum(payload, uint16(len(payload)), 20) //nolint:gosec // G115
	assert.NotEqual(t, uint32(0), csum)

	// Verify consistency: same input → same output.
	csum2 := cabChecksum(payload, uint16(len(payload)), 20) //nolint:gosec // G115
	assert.Equal(t, csum, csum2)
}
