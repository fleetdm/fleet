package redis

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/require"
)

func TestEachRedisNode(t *testing.T) {
	const prefix = "TestEachRedisNode:"

	runTest := func(t *testing.T, pool fleet.RedisPool) {
		conn := pool.Get()
		defer conn.Close()
		if rc, err := redisc.RetryConn(conn, 3, 100*time.Millisecond); err == nil {
			conn = rc
		}

		for i := 0; i < 10; i++ {
			_, err := conn.Do("SET", fmt.Sprintf("%s%d", prefix, i), i)
			require.NoError(t, err)
		}

		var keys []string
		err := EachRedisNode(pool, func(conn redis.Conn) error {
			var cursor int
			for {
				res, err := redis.Values(conn.Do("SCAN", cursor, "MATCH", prefix+"*"))
				if err != nil {
					return err
				}
				var curKeys []string
				if _, err = redis.Scan(res, &cursor, &curKeys); err != nil {
					return err
				}
				keys = append(keys, curKeys...)
				if cursor == 0 {
					return nil
				}
			}
		})
		require.NoError(t, err)
		require.Len(t, keys, 10)
	}

	t.Run("standalone", func(t *testing.T) {
		pool, teardown := setupRedisForTest(t, false)
		defer teardown()
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool, teardown := setupRedisForTest(t, true)
		defer teardown()
		runTest(t, pool)
	})
}

func setupRedisForTest(t *testing.T, cluster bool) (pool fleet.RedisPool, teardown func()) {
	var (
		addr     = "127.0.0.1:"
		password = ""
		database = 0
		useTLS   = false
		port     = "6379"
	)
	if cluster {
		port = "7001"
	}
	addr += port

	pool, err := NewRedisPool(PoolConfig{
		Server:      addr,
		Password:    password,
		Database:    database,
		UseTLS:      useTLS,
		ConnTimeout: 5 * time.Second,
		KeepAlive:   10 * time.Second,
	})
	require.NoError(t, err)

	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	require.Nil(t, err)

	teardown = func() {
		err := EachRedisNode(pool, func(conn redis.Conn) error {
			_, err := conn.Do("FLUSHDB")
			return err
		})
		require.NoError(t, err)
		pool.Close()
	}

	return pool, teardown
}
