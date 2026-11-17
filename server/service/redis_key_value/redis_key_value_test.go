package redis_key_value

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestRedisKeyValue(t *testing.T) {
	for _, f := range []func(*testing.T, *RedisKeyValue){
		testSetGet,
		testSetIfNotExists,
	} {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				kv := setupRedis(t, false, false)
				f(t, kv)
			})
			t.Run("cluster", func(t *testing.T) {
				kv := setupRedis(t, true, true)
				f(t, kv)
			})
		})
	}
}

func testSetIfNotExists(t *testing.T, kv *RedisKeyValue) {
	ctx := t.Context()

	set, err := kv.SetIfNotExists(ctx, "reserved", "first", 10*time.Second)
	require.NoError(t, err)
	require.True(t, set)
	set, err = kv.SetIfNotExists(ctx, "reserved", "second", 10*time.Second)
	require.NoError(t, err)
	require.False(t, set)
	value, err := kv.Get(ctx, "reserved")
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, "first", *value)

	// It expires like Set.
	set, err = kv.SetIfNotExists(ctx, "short", "v", 50*time.Millisecond)
	require.NoError(t, err)
	require.True(t, set)
	require.Eventually(t, func() bool {
		set, err := kv.SetIfNotExists(ctx, "short", "v", 50*time.Millisecond)
		return err == nil && set
	}, 2*time.Second, 25*time.Millisecond)
}

func setupRedis(t testing.TB, cluster, redir bool) *RedisKeyValue {
	pool := redistest.SetupRedis(t, t.Name(), cluster, redir, true)
	return newRedisKeyValueForTest(t, pool)
}

type testName interface {
	Name() string
}

func newRedisKeyValueForTest(t testName, pool fleet.RedisPool) *RedisKeyValue {
	return &RedisKeyValue{
		pool:       pool,
		testPrefix: t.Name() + ":",
	}
}

func testSetGet(t *testing.T, kv *RedisKeyValue) {
	ctx := context.Background()

	result, err := kv.Get(ctx, "foo")
	require.NoError(t, err)
	require.Nil(t, result)

	err = kv.Set(ctx, "foo", "bar", 5*time.Second)
	require.NoError(t, err)

	result, err = kv.Get(ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "bar", *result)

	err = kv.Set(ctx, "foo", "zoo", 5*time.Second)
	require.NoError(t, err)

	result, err = kv.Get(ctx, "foo")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "zoo", *result)

	err = kv.Set(ctx, "boo", "bar", 2*time.Second)
	require.NoError(t, err)
	result, err = kv.Get(ctx, "boo")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "bar", *result)

	time.Sleep(3 * time.Second)
	result, err = kv.Get(ctx, "boo")
	require.NoError(t, err)
	require.Nil(t, result)

	// Updating an item, updates the expiration time.
	err = kv.Set(ctx, "test", "foo", 2*time.Second)
	require.NoError(t, err)
	err = kv.Set(ctx, "test", "foo", 10*time.Second)
	require.NoError(t, err)
	time.Sleep(5 * time.Second)
	result, err = kv.Get(ctx, "test")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "foo", *result)
}
