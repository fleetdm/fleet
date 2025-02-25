package redis_lock

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLock(t *testing.T) {
	for _, f := range []func(*testing.T, fleet.Lock){
		testRedisAcquireLock,
		testRedisSet,
	} {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				lock := setupRedis(t, false, false)
				f(t, lock)
			})
			t.Run("cluster", func(t *testing.T) {
				lock := setupRedis(t, true, true)
				f(t, lock)
			})
		})
	}
}

func setupRedis(t testing.TB, cluster, redir bool) fleet.Lock {
	pool := redistest.SetupRedis(t, t.Name(), cluster, redir, true)
	return NewLockTest(t, pool)
}

type TestName interface {
	Name() string
}

// NewFailingTest creates a redis policy set for failing policies to be used
// only in tests.
func NewLockTest(t TestName, pool fleet.RedisPool) fleet.Lock {
	lock := &redisLock{
		pool:       pool,
		testPrefix: t.Name() + ":",
	}
	return fleet.Lock(lock)
}

func testRedisAcquireLock(t *testing.T, lock fleet.Lock) {
	ctx := context.Background()
	result, err := lock.SetIfNotExist(ctx, "test", "1", 0)
	require.NoError(t, err)
	assert.True(t, result)

	// Try to acquire the same lock
	result, err = lock.SetIfNotExist(ctx, "test", "1", 0)
	assert.NoError(t, err)
	assert.False(t, result)

	// Try to release the lock with a wrong value
	ok, err := lock.ReleaseLock(ctx, "test", "2")
	require.NoError(t, err)
	assert.False(t, ok)

	// Try to release the lock with the wrong key
	ok, err = lock.ReleaseLock(ctx, "bad", "1")
	require.NoError(t, err)
	assert.False(t, ok)

	// Try to release the lock with the correct key/value
	ok, err = lock.ReleaseLock(ctx, "test", "1")
	require.NoError(t, err)
	assert.True(t, ok)

	// Acquire the lock again
	result, err = lock.SetIfNotExist(ctx, "test", "1", 0)
	require.NoError(t, err)
	assert.True(t, result)

	// Get lock
	getResult, err := lock.Get(ctx, "test")
	assert.NoError(t, err)
	require.NotNil(t, getResult)
	assert.Equal(t, "1", *getResult)

	// Try to set lock with expiration
	var expire uint64 = 10
	result, err = lock.SetIfNotExist(ctx, "testE", "1", expire)
	require.NoError(t, err)
	assert.True(t, result)

	// Try to acquire the same lock after waiting
	duration := time.Duration(expire+1) * time.Millisecond //nolint:gosec // dismiss G115
	time.Sleep(duration)
	result, err = lock.SetIfNotExist(ctx, "testE", "1", 0)
	require.NoError(t, err)
	assert.True(t, result)

	// Get non-existent key
	getResult, err = lock.Get(ctx, "testNonExistent")
	assert.NoError(t, err)
	assert.Nil(t, getResult)

	// Get and delete non-existent key
	getResult, err = lock.GetAndDelete(ctx, "testNonExistent")
	assert.NoError(t, err)
	assert.Nil(t, getResult)

	// Set a new item
	result, err = lock.SetIfNotExist(ctx, "test2", "2", 0)
	require.NoError(t, err)
	assert.True(t, result)

	// Get and delete the item
	getResult, err = lock.GetAndDelete(ctx, "test2")
	assert.NoError(t, err)
	require.NotNil(t, getResult)
	assert.Equal(t, "2", *getResult)

	// Item was deleted, so we can't get it again
	getResult, err = lock.Get(ctx, "test2")
	assert.NoError(t, err)
	assert.Nil(t, getResult)
}

func testRedisSet(t *testing.T, lock fleet.Lock) {
	ctx := context.Background()

	// Get a non-existent set
	result, err := lock.GetSet(ctx, "missingSet")
	assert.NoError(t, err)
	assert.Empty(t, result)

	// Add to a set
	values := []string{"foo", "bar"}
	err = lock.AddToSet(ctx, "testSet", values[0])
	assert.NoError(t, err)
	err = lock.AddToSet(ctx, "testSet", values[1])
	assert.NoError(t, err)

	// Get the set
	result, err = lock.GetSet(ctx, "testSet")
	assert.NoError(t, err)
	assert.ElementsMatch(t, values, result)

	// Remove from set
	err = lock.RemoveFromSet(ctx, "testSet", values[0])
	assert.NoError(t, err)

	// Get the set
	result, err = lock.GetSet(ctx, "testSet")
	assert.NoError(t, err)
	assert.Equal(t, []string{values[1]}, result)

	// Remove from set
	err = lock.RemoveFromSet(ctx, "testSet", values[1])
	assert.NoError(t, err)

	// Get the set
	result, err = lock.GetSet(ctx, "testSet")
	assert.NoError(t, err)
	assert.Empty(t, result)
}
