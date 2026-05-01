package chart

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostIDsToBlob(t *testing.T) {
	t.Run("nil for empty input", func(t *testing.T) {
		assert.Nil(t, HostIDsToBlob(nil))
		assert.Nil(t, HostIDsToBlob([]uint{}))
	})

	t.Run("single host", func(t *testing.T) {
		blob := HostIDsToBlob([]uint{0})
		require.Len(t, blob, 1)
		assert.Equal(t, byte(0x01), blob[0])
	})

	t.Run("host ID 7", func(t *testing.T) {
		blob := HostIDsToBlob([]uint{7})
		require.Len(t, blob, 1)
		assert.Equal(t, byte(0x80), blob[0])
	})

	t.Run("host ID 8 starts second byte", func(t *testing.T) {
		blob := HostIDsToBlob([]uint{8})
		require.Len(t, blob, 2)
		assert.Equal(t, byte(0x00), blob[0])
		assert.Equal(t, byte(0x01), blob[1])
	})

	t.Run("multiple hosts", func(t *testing.T) {
		blob := HostIDsToBlob([]uint{0, 1, 8, 16})
		require.Len(t, blob, 3)
		assert.Equal(t, byte(0x03), blob[0]) // bits 0,1
		assert.Equal(t, byte(0x01), blob[1]) // bit 8
		assert.Equal(t, byte(0x01), blob[2]) // bit 16
	})

	t.Run("large host ID", func(t *testing.T) {
		blob := HostIDsToBlob([]uint{1000})
		require.Len(t, blob, 126) // 1000/8+1
		assert.Equal(t, byte(0x01), blob[125])
	})
}

func TestBlobPopcount(t *testing.T) {
	assert.Equal(t, 0, BlobPopcount(nil))
	assert.Equal(t, 0, BlobPopcount([]byte{}))
	assert.Equal(t, 1, BlobPopcount([]byte{0x01}))
	assert.Equal(t, 8, BlobPopcount([]byte{0xFF}))
	assert.Equal(t, 3, BlobPopcount([]byte{0x07}))

	// Multi-byte
	assert.Equal(t, 4, BlobPopcount([]byte{0x0F, 0x00}))
	assert.Equal(t, 16, BlobPopcount([]byte{0xFF, 0xFF}))

	// Exercises the uint64 fast path (>= 8 bytes)
	blob := make([]byte, 16)
	blob[0] = 0xFF  // 8 bits
	blob[15] = 0x01 // 1 bit
	assert.Equal(t, 9, BlobPopcount(blob))
}

func TestBlobAND(t *testing.T) {
	assert.Nil(t, BlobAND([]byte{}, []byte{}))
	assert.Nil(t, BlobAND([]byte{0xFF}, []byte{}))

	result := BlobAND([]byte{0xFF, 0x0F}, []byte{0x0F, 0xFF})
	assert.Equal(t, []byte{0x0F, 0x0F}, result)

	// Different lengths: result is min length
	result = BlobAND([]byte{0xFF, 0xFF, 0xFF}, []byte{0x0F})
	assert.Equal(t, []byte{0x0F}, result)
}

func TestBlobOR(t *testing.T) {
	assert.Nil(t, BlobOR(nil, nil))

	// One nil
	result := BlobOR([]byte{0x0F}, nil)
	assert.Equal(t, []byte{0x0F}, result)

	result = BlobOR([]byte{0xF0, 0x00}, []byte{0x0F, 0xFF})
	assert.Equal(t, []byte{0xFF, 0xFF}, result)

	// Different lengths: result is max length
	result = BlobOR([]byte{0x01}, []byte{0x02, 0xFF})
	assert.Equal(t, []byte{0x03, 0xFF}, result)
}

func TestRoundTrip(t *testing.T) {
	ids := []uint{1, 5, 10, 42, 100, 255}
	blob := HostIDsToBlob(ids)
	assert.Equal(t, len(ids), BlobPopcount(blob))

	// Filter to only even IDs
	filterIDs := []uint{10, 42, 100}
	filterBlob := HostIDsToBlob(filterIDs)
	filtered := BlobAND(blob, filterBlob)
	assert.Equal(t, 3, BlobPopcount(filtered))
}
