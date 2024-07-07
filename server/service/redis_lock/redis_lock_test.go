package redis_lock

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRedisLock(t *testing.T) {
	for _, f := range []func(*testing.T, fleet.Lock){
		testRedisAcquireLock,
		testRedisIncrement,
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

			t.Run("cluster-no-redir", func(t *testing.T) {
				lock := setupRedis(t, true, false)
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
	result, err := lock.AcquireLock(ctx, "test", "1", 0)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Try to acquire the same lock
	result, err = lock.AcquireLock(ctx, "test", "1", 0)
	assert.NoError(t, err)
	assert.Equal(t, "", result)

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
	result, err = lock.AcquireLock(ctx, "test", "1", 0)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Try to set lock with expiration
	var expire uint64 = 10
	result, err = lock.AcquireLock(ctx, "testE", "1", expire)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Try to acquire the same lock after waiting
	duration := time.Duration(expire+1) * time.Millisecond
	time.Sleep(duration)
	result, err = lock.AcquireLock(ctx, "testE", "1", 0)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)
}

func testRedisIncrement(t *testing.T, lock fleet.Lock) {
	ctx := context.Background()

	// Increment non-existent key
	var expire uint64 = 10
	num, err := lock.Increment(ctx, "newKey", expire)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), num)

	// Try to acquire the same lock after waiting
	duration := time.Duration(expire+1) * time.Millisecond
	time.Sleep(duration)
	result, err := lock.AcquireLock(ctx, "newKey", "1", 0)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Get non-existent key
	getResult, err := lock.Get(ctx, "test")
	assert.NoError(t, err)
	assert.Nil(t, getResult)

	// Save non-integer value
	result, err = lock.AcquireLock(ctx, "test", "foo", 0)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)

	// Increment non-integer value
	_, err = lock.Increment(ctx, "test", 0)
	assert.Error(t, err)

	// Save an integer value
	result, err = lock.AcquireLock(ctx, "testI", "1", 0)
	require.NoError(t, err)
	assert.Equal(t, "OK", result)

	num, err = lock.Increment(ctx, "testI", 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), num)

	getResult, err = lock.Get(ctx, "testI")
	assert.NoError(t, err)
	require.NotNil(t, getResult)
	assert.Equal(t, "2", *getResult)

}
