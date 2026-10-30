package redis_nonces_store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestRedisNoncesStore(t *testing.T) {
	for _, f := range []func(*testing.T, *RedisNoncesStore){
		testStoreConsume,
		testMarkLoginAssertionUsed,
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

func newRedisNoncesStoreForTest(t testName, pool RedisPool) *RedisNoncesStore {
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

	// Consuming the same nonce twice returns false the second time.
	err = store.Store(ctx, "once", 5*time.Second)
	require.NoError(t, err)
	ok, err = store.Consume(ctx, "once")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = store.Consume(ctx, "once")
	require.NoError(t, err)
	require.False(t, ok)
}

func testMarkLoginAssertionUsed(t *testing.T, store *RedisNoncesStore) {
	ctx := context.Background()

	firstUse, err := store.MarkLoginAssertionUsed(ctx, "foo", 5*time.Second)
	require.NoError(t, err)
	require.True(t, firstUse)

	firstUse, err = store.MarkLoginAssertionUsed(ctx, "foo", 5*time.Second)
	require.NoError(t, err)
	require.False(t, firstUse)

	// consuming a nonce named like the record, or its key, doesn't remove it
	for _, nonce := range []string{"foo", usedLoginAssertionPrefix + "foo", strings.TrimPrefix(usedLoginAssertionPrefix+"foo", prefix)} {
		ok, err := store.Consume(ctx, nonce)
		require.NoError(t, err)
		require.False(t, ok)
	}
	firstUse, err = store.MarkLoginAssertionUsed(ctx, "foo", 5*time.Second)
	require.NoError(t, err)
	require.False(t, firstUse)

	// recorded again once it expires
	firstUse, err = store.MarkLoginAssertionUsed(ctx, "bar", time.Millisecond)
	require.NoError(t, err)
	require.True(t, firstUse)
	time.Sleep(2 * time.Millisecond)
	firstUse, err = store.MarkLoginAssertionUsed(ctx, "bar", time.Millisecond)
	require.NoError(t, err)
	require.True(t, firstUse)
}
