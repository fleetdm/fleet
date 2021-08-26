package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocks(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	owner1, err := server.GenerateRandomText(64)
	require.NoError(t, err)
	owner2, err := server.GenerateRandomText(64)
	require.NoError(t, err)

	// get first lock
	locked, err := ds.Lock("test", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)

	// renew current lock
	locked, err = ds.Lock("test", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)

	// owner2 tries to get the lock but fails
	locked, err = ds.Lock("test", owner2, 1*time.Minute)
	require.NoError(t, err)
	assert.False(t, locked)

	// owner2 gets a new lock that expires quickly
	locked, err = ds.Lock("test-expired", owner2, 1*time.Second)
	require.NoError(t, err)
	assert.True(t, locked)

	time.Sleep(3 * time.Second)

	// owner1 gets the same lock because it's now expired
	locked, err = ds.Lock("test-expired", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)

	// unlocking clears the lock
	locked, err = ds.Lock("test", owner1, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)
	err = ds.Unlock("test", owner1)
	require.NoError(t, err)

	// owner2 tries to get the lock but fails
	locked, err = ds.Lock("test", owner2, 1*time.Minute)
	require.NoError(t, err)
	assert.True(t, locked)
}
