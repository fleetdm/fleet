package redis_nonces_store

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestRedisNoncesStore(t *testing.T) {
	for _, f := range []func(*testing.T, *RedisNoncesStore){
		testStoreConsume,
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

func setupRedis(t testing.TB, cluster, redir bool) *RedisNoncesStore {
	pool := redistest.SetupRedis(t, t.Name(), cluster, redir, true)
	return newRedisNoncesStoreForTest(t, pool)
}

type testName interface {
	Name() string
}

func newRedisNoncesStoreForTest(t testName, pool acme.RedisPool) *RedisNoncesStore {
	return &RedisNoncesStore{
		pool:       pool,
		testPrefix: t.Name() + ":",
	}
}

func testStoreConsume(t *testing.T, store *RedisNoncesStore) {
	ctx := context.Background()

	err := store.Store(ctx, "foo", time.Millisecond)
	require.NoError(t, err)

	err = store.Store(ctx, "bar", 5*time.Second)
	require.NoError(t, err)

	ok, err := store.Consume(ctx, "bar")
	require.NoError(t, err)
	require.True(t, ok)

	time.Sleep(2 * time.Millisecond)

	ok, err = store.Consume(ctx, "foo")
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = store.Consume(ctx, "no-such")
	require.NoError(t, err)
	require.False(t, ok)
}
