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
