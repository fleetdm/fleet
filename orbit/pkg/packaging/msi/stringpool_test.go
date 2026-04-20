package msi

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringPool_AddAndLookup(t *testing.T) {
	sp := NewStringPool()

	// First string gets index 1.
	idx1 := sp.Add("hello")
	assert.Equal(t, uint16(1), idx1)

	// Same string returns same index, increments refcount.
	idx2 := sp.Add("hello")
	assert.Equal(t, uint16(1), idx2)

	// Different string gets index 2.
	idx3 := sp.Add("world")
	assert.Equal(t, uint16(2), idx3)

	// Lookup existing.
	assert.Equal(t, uint16(1), sp.Lookup("hello"))
	assert.Equal(t, uint16(2), sp.Lookup("world"))

	// Lookup non-existing.
	assert.Equal(t, uint16(0), sp.Lookup("missing"))

	assert.Equal(t, 2, sp.Count())
}

func TestStringPool_RoundTrip(t *testing.T) {
	// Encode a string pool and decode it using the same logic as pkg/file/msi.go:decodeStrings().
	sp := NewStringPool()
	sp.Add("ProductName")
	sp.Add("Fleet osquery")
	sp.Add("ProductVersion")
	sp.Add("1.0.0")

	poolData := sp.EncodePool()
	stringData := sp.EncodeData()

	// Decode using the same algorithm as pkg/file/msi.go.
	decoded, err := testDecodeStrings(bytes.NewReader(stringData), bytes.NewReader(poolData))
	require.NoError(t, err)

	require.Len(t, decoded, 4)
	assert.Equal(t, "ProductName", decoded[0])
	assert.Equal(t, "Fleet osquery", decoded[1])
	assert.Equal(t, "ProductVersion", decoded[2])
	assert.Equal(t, "1.0.0", decoded[3])
}

func TestStringPool_EmptyString(t *testing.T) {
	// Empty strings return index 0 (MSI null) and are not stored in the pool.
	sp := NewStringPool()
	idx := sp.Add("")
	assert.Equal(t, uint16(0), idx)
	assert.Equal(t, 0, sp.Count())
}

// testDecodeStrings mimics the string pool decoding from pkg/file/msi.go.
func testDecodeStrings(dataReader, poolReader io.Reader) ([]string, error) {
	type header struct {
		Codepage uint16
		Unknown  uint16
	}
	var poolHeader header
	if err := binary.Read(poolReader, binary.LittleEndian, &poolHeader); err != nil {
		return nil, err
	}

	type entry struct {
		Size     uint16
		RefCount uint16
	}
	var stringEntry entry
	var result []string
	var buf bytes.Buffer
	for {
		err := binary.Read(poolReader, binary.LittleEndian, &stringEntry)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		stringEntrySize := uint32(stringEntry.Size)
		if stringEntry.Size == 0 && stringEntry.RefCount != 0 {
			if err := binary.Read(poolReader, binary.LittleEndian, &stringEntrySize); err != nil {
				return nil, err
			}
		}
		buf.Reset()
		buf.Grow(int(stringEntrySize))
		if _, err := io.CopyN(&buf, dataReader, int64(stringEntrySize)); err != nil {
			return nil, err
		}
		result = append(result, buf.String())
	}
	return result, nil
}
