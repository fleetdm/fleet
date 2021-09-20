package mysql

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mockCreatedAt = time.Now().UTC().Truncate(time.Second)

func TestCarves(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Metadata", testCarvesMetadata},
		{"Blocks", testCarvesBlocks},
		{"Cleanup", testCarvesCleanup},
		{"List", testCarvesList},
		{"Update", testCarvesUpdate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testCarvesMetadata(t *testing.T, ds *Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	expectedCarve := &fleet.CarveMetadata{
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

	expectedCarve, err := ds.NewCarve(context.Background(), expectedCarve)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve.ID)
	expectedCarve.MaxBlock = -1

	carve, err := ds.CarveBySessionId(context.Background(), expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	carve, err = ds.Carve(context.Background(), expectedCarve.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Check for increment of max block

	err = ds.NewBlock(context.Background(), carve, 0, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 0

	carve, err = ds.CarveBySessionId(context.Background(), expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	carve, err = ds.Carve(context.Background(), expectedCarve.ID)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Check for increment of max block

	err = ds.NewBlock(context.Background(), carve, 1, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 1

	carve, err = ds.CarveBySessionId(context.Background(), expectedCarve.SessionId)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)

	// Get by name also
	carve, err = ds.CarveByName(context.Background(), expectedCarve.Name)
	require.NoError(t, err)
	assert.Equal(t, expectedCarve, carve)
}

func testCarvesBlocks(t *testing.T, ds *Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	blockCount := int64(25)
	blockSize := int64(30)
	carve := &fleet.CarveMetadata{
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

	carve, err := ds.NewCarve(context.Background(), carve)
	require.NoError(t, err)

	// Randomly generate and insert blocks
	expectedBlocks := make([][]byte, blockCount)
	for i := int64(0); i < blockCount; i++ {
		block := make([]byte, blockSize)
		_, err := rand.Read(block)
		require.NoError(t, err, "generate block")
		expectedBlocks[i] = block

		err = ds.NewBlock(context.Background(), carve, i, block)
		require.NoError(t, err, "write block %v", block)
	}

	// Verify retrieved blocks match inserted blocks
	for i := int64(0); i < blockCount; i++ {
		data, err := ds.GetBlock(context.Background(), carve, i)
		require.NoError(t, err, "get block %d %v", i, expectedBlocks[i])
		assert.Equal(t, expectedBlocks[i], data)
	}
}

func testCarvesCleanup(t *testing.T, ds *Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	blockCount := int64(25)
	blockSize := int64(30)
	carve := &fleet.CarveMetadata{
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

	carve, err := ds.NewCarve(context.Background(), carve)
	require.NoError(t, err)

	// Randomly generate and insert blocks
	expectedBlocks := make([][]byte, blockCount)
	for i := int64(0); i < blockCount; i++ {
		block := make([]byte, blockSize)
		_, err := rand.Read(block)
		require.NoError(t, err, "generate block")
		expectedBlocks[i] = block

		err = ds.NewBlock(context.Background(), carve, i, block)
		require.NoError(t, err, "write block %v", block)
	}

	expired, err := ds.CleanupCarves(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, expired)

	_, err = ds.GetBlock(context.Background(), carve, 0)
	require.NoError(t, err)

	expired, err = ds.CleanupCarves(context.Background(), time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, expired)

	// Should no longer be able to get data
	_, err = ds.GetBlock(context.Background(), carve, 0)
	require.Error(t, err, "data should be expired")

	carve, err = ds.Carve(context.Background(), carve.ID)
	require.NoError(t, err)
	assert.True(t, carve.Expired)
}

func testCarvesList(t *testing.T, ds *Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	expectedCarve := &fleet.CarveMetadata{
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

	expectedCarve, err := ds.NewCarve(context.Background(), expectedCarve)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve.ID)
	// Add a block to this carve
	err = ds.NewBlock(context.Background(), expectedCarve, 0, nil)
	require.NoError(t, err)
	expectedCarve.MaxBlock = 0

	expectedCarve2 := &fleet.CarveMetadata{
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

	expectedCarve2, err = ds.NewCarve(context.Background(), expectedCarve2)
	require.NoError(t, err)
	assert.NotEqual(t, 0, expectedCarve2.ID)
	expectedCarve2.MaxBlock = -1

	carves, err := ds.ListCarves(context.Background(), fleet.CarveListOptions{Expired: true})
	require.NoError(t, err)
	assert.Equal(t, []*fleet.CarveMetadata{expectedCarve, expectedCarve2}, carves)

	// Expire the carves
	_, err = ds.CleanupCarves(context.Background(), time.Now().Add(24*time.Hour))
	require.NoError(t, err)

	carves, err = ds.ListCarves(context.Background(), fleet.CarveListOptions{Expired: false})
	require.NoError(t, err)
	assert.Empty(t, carves)

	carves, err = ds.ListCarves(context.Background(), fleet.CarveListOptions{Expired: true})
	require.NoError(t, err)
	assert.Len(t, carves, 2)
}

func testCarvesUpdate(t *testing.T, ds *Datastore) {
	h := test.NewHost(t, ds, "foo.local", "192.168.1.10", "1", "1", time.Now())

	actualCount := int64(10)
	carve := &fleet.CarveMetadata{
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

	carve, err := ds.NewCarve(context.Background(), carve)
	require.NoError(t, err)

	carve.Expired = true
	carve.MaxBlock = 10
	carve.BlockCount = 15 // it should not get updated
	err = ds.UpdateCarve(context.Background(), carve)
	require.NoError(t, err)

	carve.BlockCount = actualCount
	dbCarve, err := ds.Carve(context.Background(), carve.ID)
	require.NoError(t, err)
	assert.Equal(t, carve, dbCarve)
}
