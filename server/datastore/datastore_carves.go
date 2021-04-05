package datastore

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mockCreatedAt time.Time = time.Now().UTC().Truncate(time.Second)

func testCarveMetadata(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	expectedCarve := &kolide.CarveMetadata{
		HostId:     h.ID,
		Name:       "foobar",
		BlockCount: 10,
		BlockSize:  12,
		CarveSize:  123,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
		CreatedAt:  mockCreatedAt,
	}

	expectedCarve, err := ds.NewCarve(expectedCarve)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve.ID)
	expectedCarve.MaxBlock = -1

	carve, err := ds.CarveBySessionId(expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	carve, err = ds.Carve(expectedCarve.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Check for increment of max block

	err = ds.NewBlock(carve, 0, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 0

	carve, err = ds.CarveBySessionId(expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	carve, err = ds.Carve(expectedCarve.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Check for increment of max block

	err = ds.NewBlock(carve, 1, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 1

	carve, err = ds.CarveBySessionId(expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Get by name also
	carve, err = ds.CarveByName(expectedCarve.Name)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)
}

func testCarveBlocks(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	blockCount := int64(25)
	blockSize := int64(30)
	carve := &kolide.CarveMetadata{
		HostId:     h.ID,
		Name:       "foobar",
		BlockCount: blockCount,
		BlockSize:  blockSize,
		CarveSize:  blockCount * blockSize,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
		CreatedAt:  mockCreatedAt,
	}

	carve, err := ds.NewCarve(carve)
	require.NoError(t, err)

	// Randomly generate and insert blocks
	expectedBlocks := make([][]byte, blockCount)
	for i := int64(0); i < blockCount; i++ {
		block := make([]byte, blockSize)
		_, err := rand.Read(block)
		require.NoError(t, err, "generate block")
		expectedBlocks[i] = block

		err = ds.NewBlock(carve, i, block)
		require.NoError(t, err, "write block %v", block)
	}

	// Verify retrieved blocks match inserted blocks
	for i := int64(0); i < blockCount; i++ {
		data, err := ds.GetBlock(carve, i)
		require.NoError(t, err, "get block %d %v", i, expectedBlocks[i])
		assert.Equal(t, expectedBlocks[i], data)
	}

}

func testCarveCleanupCarves(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	blockCount := int64(25)
	blockSize := int64(30)
	carve := &kolide.CarveMetadata{
		HostId:     h.ID,
		Name:       "foobar",
		BlockCount: blockCount,
		BlockSize:  blockSize,
		CarveSize:  blockCount * blockSize,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
		CreatedAt:  mockCreatedAt,
	}

	carve, err := ds.NewCarve(carve)
	require.NoError(t, err)

	// Randomly generate and insert blocks
	expectedBlocks := make([][]byte, blockCount)
	for i := int64(0); i < blockCount; i++ {
		block := make([]byte, blockSize)
		_, err := rand.Read(block)
		require.NoError(t, err, "generate block")
		expectedBlocks[i] = block

		err = ds.NewBlock(carve, i, block)
		require.NoError(t, err, "write block %v", block)
	}

	expired, err := ds.CleanupCarves(time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, expired)

	_, err = ds.GetBlock(carve, 0)
	require.NoError(t, err)

	expired, err = ds.CleanupCarves(time.Now().Add(24 * time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, expired)

	// Should no longer be able to get data
	_, err = ds.GetBlock(carve, 0)
	require.Error(t, err, "data should be expired")

	carve, err = ds.Carve(carve.ID)
	require.NoError(t, err)
	assert.True(t, carve.Expired)
}

func testCarveListCarves(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	expectedCarve := &kolide.CarveMetadata{
		HostId:     h.ID,
		Name:       "foobar",
		BlockCount: 10,
		BlockSize:  12,
		CarveSize:  113,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
		CreatedAt:  mockCreatedAt,
		MaxBlock:   -1,
	}

	expectedCarve, err := ds.NewCarve(expectedCarve)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve.ID)
	// Add a block to this carve
	err = ds.NewBlock(expectedCarve, 0, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 0

	expectedCarve2 := &kolide.CarveMetadata{
		HostId:     h.ID,
		Name:       "foobar2",
		BlockCount: 42,
		BlockSize:  13,
		CarveSize:  42 * 13,
		CarveId:    "carve_id2",
		RequestId:  "request_id2",
		SessionId:  "session_id2",
		CreatedAt:  mockCreatedAt,
	}

	expectedCarve2, err = ds.NewCarve(expectedCarve2)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve2.ID)
	expectedCarve2.MaxBlock = -1

	carves, err := ds.ListCarves(kolide.CarveListOptions{Expired: true})
	require.NoError(t, err)
	assert.Equal(t, []*kolide.CarveMetadata{expectedCarve, expectedCarve2}, carves)

	// Expire the carves
	_, err = ds.CleanupCarves(time.Now().Add(24 * time.Hour))
	require.NoError(t, err)

	carves, err = ds.ListCarves(kolide.CarveListOptions{Expired: false})
	require.NoError(t, err)
	assert.Empty(t, carves)

	carves, err = ds.ListCarves(kolide.CarveListOptions{Expired: true})
	require.NoError(t, err)
	assert.Len(t, carves, 2)
}

func testCarveUpdateCarve(t *testing.T, ds kolide.Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	actualCount := int64(10)
	carve := &kolide.CarveMetadata{
		HostId:     h.ID,
		Name:       "foobar",
		BlockCount: actualCount,
		BlockSize:  20,
		CarveSize:  actualCount * 20,
		CarveId:    "carve_id",
		RequestId:  "request_id",
		SessionId:  "session_id",
		CreatedAt:  mockCreatedAt,
	}

	carve, err := ds.NewCarve(carve)
	require.NoError(t, err)

	carve.Expired = true
	carve.MaxBlock = 10
	carve.BlockCount = 15 // it should not get updated
	err = ds.UpdateCarve(carve)
	require.NoError(t, err)

	carve.BlockCount = actualCount
	dbCarve, err := ds.Carve(carve.ID)
	require.NoError(t, err)
	assert.Equal(t, carve, dbCarve)
}
