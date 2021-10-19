package redis_test

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestThrottledStore(t *testing.T) {
	const prefix = "TestThrottledStore:"

	runTest := func(t *testing.T, pool fleet.RedisPool) {
		store := redis.ThrottledStore{
			Pool:      pool,
			KeyPrefix: prefix,
		}

		t.Run("GetWithTime", func(t *testing.T) {
			conn := pool.Get()
			defer conn.Close()

			// key does not exist
			v, ts, err := store.GetWithTime("a")
			require.NoError(t, err)
			require.Equal(t, v, int64(-1))
			require.WithinDuration(t, time.Now(), ts, time.Second)

			_, err = conn.Do("SET", store.KeyPrefix+"a", 1)
			require.NoError(t, err)

			// key exists
			v, ts, err = store.GetWithTime("a")
			require.NoError(t, err)
			require.Equal(t, v, int64(1))
			require.WithinDuration(t, time.Now(), ts, time.Second)
		})

		t.Run("SetIfNotExistsWithTTL", func(t *testing.T) {
			conn := pool.Get()
			defer conn.Close()

			// key does not exist
			ok, err := store.SetIfNotExistsWithTTL("{b}", 1, time.Second)
			require.NoError(t, err)
			require.True(t, ok)

			v, err := redigo.Int(conn.Do("GET", store.KeyPrefix+"{b}"))
			require.NoError(t, err)
			require.Equal(t, v, 1)

			// key exists
			ok, err = store.SetIfNotExistsWithTTL("{b}", 2, time.Second)
			require.NoError(t, err)
			require.False(t, ok)

			// value is still 1
			v, err = redigo.Int(conn.Do("GET", store.KeyPrefix+"{b}"))
			require.NoError(t, err)
			require.Equal(t, v, 1)

			// key does not exist, but ttl less than a second
			ok, err = store.SetIfNotExistsWithTTL("{b}2", 3, time.Millisecond)
			require.NoError(t, err)
			require.True(t, ok)

			v, err = redigo.Int(conn.Do("GET", store.KeyPrefix+"{b}2"))
			require.NoError(t, err)
			require.Equal(t, v, 3)
		})

		t.Run("CompareAndSwapWithTTL", func(t *testing.T) {
			conn := pool.Get()
			defer conn.Close()

			// key does not exist
			ok, err := store.CompareAndSwapWithTTL("{c}", 1, 2, time.Second)
			require.NoError(t, err)
			require.False(t, ok)

			_, err = conn.Do("SET", store.KeyPrefix+"{c}", 1)
			require.NoError(t, err)

			// key exists, but values do not match
			ok, err = store.CompareAndSwapWithTTL("{c}", 2, 3, time.Second)
			require.NoError(t, err)
			require.False(t, ok)

			// key exists, values match
			ok, err = store.CompareAndSwapWithTTL("{c}", 1, 4, time.Second)
			require.NoError(t, err)
			require.True(t, ok)

			v, err := redigo.Int(conn.Do("GET", store.KeyPrefix+"{c}"))
			require.NoError(t, err)
			require.Equal(t, v, 4)

			// key exists, ttl less than a second
			ok, err = store.CompareAndSwapWithTTL("{c}", 4, 5, time.Millisecond)
			require.NoError(t, err)
			require.True(t, ok)

			v, err = redigo.Int(conn.Do("GET", store.KeyPrefix+"{c}"))
			require.NoError(t, err)
			require.Equal(t, v, 5)
		})
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, true, false)
		runTest(t, pool)
	})

	t.Run("cluster_nofollow", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, false, false)
		runTest(t, pool)
	})
}
