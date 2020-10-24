package datastore

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/test"
	"github.com/stretchr/testify/require"
)

func testCarveMetadata(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	expectedCarve := &kolide.CarveMetadata{
		HostId:     h.ID,
		BlockCount: 10,
		BlockSize:  12,
		CarveSize:  123,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
	}

	expectedCarve, err := ds.NewCarve(expectedCarve)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve.ID)
	expectedCarve.MaxBlock = -1

	carve, err := ds.CarveBySessionId(expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Check for increment of max block

	err = ds.NewBlock(carve.ID, 0, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 0

	carve, err = ds.CarveBySessionId(expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Check for increment of max block

	err = ds.NewBlock(carve.ID, 1, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 1

	carve, err = ds.CarveBySessionId(expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)
}

func testCarveBlocks(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	blockCount := 25
	blockSize := 30
	carve := &kolide.CarveMetadata{
		HostId:     h.ID,
		BlockCount: blockCount,
		BlockSize:  blockSize,
		CarveSize:  blockCount * blockSize,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
	}

	carve, err := ds.NewCarve(carve)
	require.NoError(t, err)

	// Randomly generate and insert blocks
	expectedBlocks := make([][]byte, blockCount)
	for i := 0; i < blockCount; i++ {
		block := make([]byte, blockSize)
		_, err := rand.Read(block)
		require.NoError(t, err, "generate block")
		expectedBlocks[i] = block

		err = ds.NewBlock(carve.ID, i, block)
		require.NoError(t, err, "write block %v", block)
	}

	// Verify retrieved blocks match inserted blocks
	for i := 0; i < blockCount; i++ {
		data, err := ds.GetBlock(carve.ID, i)
		require.NoError(t, err, "get block %d %v", i, expectedBlocks[i])
		assert.Equal(t, expectedBlocks[i], data)
	}

}

func testCarveListCarves(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	expectedCarve := &kolide.CarveMetadata{
		HostId:     h.ID,
		BlockCount: 10,
		BlockSize:  12,
		CarveSize:  113,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
	}

	expectedCarve, err := ds.NewCarve(expectedCarve)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve.ID)
	// Add a block to this carve
	err = ds.NewBlock(expectedCarve.ID, 0, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 0

	expectedCarve2 := &kolide.CarveMetadata{
		HostId:     h.ID,
		BlockCount: 42,
		BlockSize:  13,
		CarveSize:  42 * 13,
		CarveId:    "carve_id2",
		RequestId:  "request_id2",
		SessionId:  "session_id2",
	}

	expectedCarve2, err = ds.NewCarve(expectedCarve2)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve2.ID)
	expectedCarve2.MaxBlock = -1

	carves, err := ds.ListCarves(kolide.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, []*kolide.CarveMetadata{expectedCarve, expectedCarve2}, carves)
}
