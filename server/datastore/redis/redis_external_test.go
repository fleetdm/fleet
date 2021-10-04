package redis_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/require"
)

func TestRedisPoolConfigureDoer(t *testing.T) {
	const prefix = "TestRedisPoolConfigureDoer:"

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false)

		c1 := pool.Get()
		defer c1.Close()
		c2 := pool.ConfigureDoer(pool.Get())
		defer c2.Close()

		// both conns work equally well, get nil because keys do not exist,
		// but no redirection error (this is standalone redis).
		_, err := redigo.String(c1.Do("GET", prefix+"{a}"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(c1.Do("GET", prefix+"{b}"))
		require.Equal(t, redigo.ErrNil, err)

		_, err = redigo.String(c2.Do("GET", prefix+"{a}"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(c2.Do("GET", prefix+"{b}"))
		require.Equal(t, redigo.ErrNil, err)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, true)

		c1 := pool.Get()
		defer c1.Close()
		c2 := pool.ConfigureDoer(pool.Get())
		defer c2.Close()

		// unconfigured conn gets MOVED error on the second key
		// (it is bound to {a}, {b} is on a different node)
		_, err := redigo.String(c1.Do("GET", prefix+"{a}"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(c1.Do("GET", prefix+"{b}"))
		rerr := redisc.ParseRedir(err)
		require.Error(t, rerr)
		require.Equal(t, "MOVED", rerr.Type)

		// configured conn gets the nil value, it redirected automatically
		_, err = redigo.String(c2.Do("GET", prefix+"{a}"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(c2.Do("GET", prefix+"{b}"))
		require.Equal(t, redigo.ErrNil, err)
	})
}

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
		err := redis.EachRedisNode(pool, func(conn redigo.Conn) error {
			var cursor int
			for {
				res, err := redigo.Values(conn.Do("SCAN", cursor, "MATCH", prefix+"*"))
				if err != nil {
					return err
				}
				var curKeys []string
				if _, err = redigo.Scan(res, &cursor, &curKeys); err != nil {
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
		pool := redistest.SetupRedis(t, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, false)
		runTest(t, pool)
	})
}

func TestBindConn(t *testing.T) {
	const prefix = "TestBindConn:"

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false)

		conn := pool.Get()
		defer conn.Close()

		err := redis.BindConn(pool, conn, prefix+"a", prefix+"b", prefix+"c")
		require.NoError(t, err)
		_, err = redigo.String(conn.Do("GET", prefix+"a"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(conn.Do("GET", prefix+"b"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(conn.Do("GET", prefix+"c"))
		require.Equal(t, redigo.ErrNil, err)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, false)

		conn := pool.Get()
		defer conn.Close()

		err := redis.BindConn(pool, conn, prefix+"a", prefix+"b", prefix+"c")
		require.Error(t, err)

		err = redis.BindConn(pool, conn, prefix+"{z}a", prefix+"{z}b", prefix+"{z}c")
		require.NoError(t, err)

		_, err = redigo.String(conn.Do("GET", prefix+"{z}a"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(conn.Do("GET", prefix+"{z}b"))
		require.Equal(t, redigo.ErrNil, err)
		_, err = redigo.String(conn.Do("GET", prefix+"{z}c"))
		require.Equal(t, redigo.ErrNil, err)
	})
}
