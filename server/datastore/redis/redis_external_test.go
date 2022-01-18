package redis_test

import (
	"fmt"
	"net"
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
		pool := redistest.SetupRedis(t, prefix, false, false, false)

		c1 := pool.Get()
		defer c1.Close()
		c2 := redis.ConfigureDoer(pool, pool.Get())
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
		pool := redistest.SetupRedis(t, prefix, true, true, false)

		c1 := pool.Get()
		defer c1.Close()
		c2 := redis.ConfigureDoer(pool, pool.Get())
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

func TestEachNode(t *testing.T) {
	const prefix = "TestEachNode:"

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
		err := redis.EachNode(pool, false, func(conn redigo.Conn) error {
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
		pool := redistest.SetupRedis(t, prefix, false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, true, false, false)
		runTest(t, pool)
	})
}

func TestBindConn(t *testing.T) {
	const prefix = "TestBindConn:"

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, false, false, false)

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
		pool := redistest.SetupRedis(t, prefix, true, false, false)

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

func TestPublishHasListeners(t *testing.T) {
	const prefix = "TestPublishHasListeners:"

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, false, false, false)

		pconn := pool.Get()
		defer pconn.Close()
		sconn := pool.Get()
		defer sconn.Close()

		ok, err := redis.PublishHasListeners(pool, pconn, prefix+"a", "A")
		require.NoError(t, err)
		require.False(t, ok)

		psc := redigo.PubSubConn{Conn: sconn}
		require.NoError(t, psc.Subscribe(prefix+"a"))

		start := time.Now()
		var loopOk bool
	loop1:
		for time.Since(start) < 2*time.Second {
			msg := psc.Receive()
			switch msg := msg.(type) {
			case redigo.Subscription:
				require.Equal(t, msg.Count, 1)
				loopOk = true
				break loop1
			}
		}
		require.True(t, loopOk, "timed out")

		ok, err = redis.PublishHasListeners(pool, pconn, prefix+"a", "B")
		require.NoError(t, err)
		require.True(t, ok)

		start = time.Now()
		loopOk = false
	loop2:
		for time.Since(start) < 2*time.Second {
			msg := psc.Receive()
			switch msg := msg.(type) {
			case redigo.Message:
				require.Equal(t, "B", string(msg.Data))
				loopOk = true
				break loop2
			}
		}
		require.True(t, loopOk, "timed out")
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, true, false, false)

		pconn := pool.Get()
		defer pconn.Close()
		sconn := pool.Get()
		defer sconn.Close()

		ok, err := redis.PublishHasListeners(pool, pconn, prefix+"{a}", "A")
		require.NoError(t, err)
		require.False(t, ok)

		// one listener on a different node
		redis.BindConn(pool, sconn, "b")
		psc := redigo.PubSubConn{Conn: sconn}
		require.NoError(t, psc.Subscribe(prefix+"{a}"))

		// a standard PUBLISH returns 0
		n, err := redigo.Int(pconn.Do("PUBLISH", prefix+"{a}", "B"))
		require.NoError(t, err)
		require.Equal(t, 0, n)

		// but this returns true
		ok, err = redis.PublishHasListeners(pool, pconn, prefix+"{a}", "C")
		require.NoError(t, err)
		require.True(t, ok)

		start := time.Now()
		want := "B"
		var loopOk bool
	loop:
		for time.Since(start) < 2*time.Second {
			msg := psc.Receive()
			switch msg := msg.(type) {
			case redigo.Message:
				require.Equal(t, want, string(msg.Data))
				if want == "C" {
					loopOk = true
					break loop
				}
				want = "C"
			}
		}
		require.True(t, loopOk, "timed out")
	})
}

func TestReadOnlyConn(t *testing.T) {
	const prefix = "TestReadOnlyConn:"

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, false, false, true)
		conn := redis.ReadOnlyConn(pool, pool.Get())
		defer conn.Close()

		_, err := conn.Do("SET", prefix+"a", 1)
		require.NoError(t, err)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, true, false, true)
		conn := redis.ReadOnlyConn(pool, pool.Get())
		defer conn.Close()

		_, err := conn.Do("SET", prefix+"a", 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "MOVED")
	})
}

func TestRedisMode(t *testing.T) {
	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, "zz", false, false, false)
		require.Equal(t, pool.Mode(), fleet.RedisStandalone)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, "zz", true, false, false)
		require.Equal(t, pool.Mode(), fleet.RedisCluster)
	})
}

func rawTCPServer(t *testing.T, handler func(c net.Conn, done <-chan struct{})) (addr string) {
	// start a server on localhost, on a random free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "net.Listen")
	_, port, _ := net.SplitHostPort(l.Addr().String())
	done, exited := make(chan struct{}), make(chan struct{})

	go func() {
		defer close(exited)
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			handler(conn, done)
		}
	}()

	t.Cleanup(func() {
		require.NoError(t, l.Close())
		close(done)
		select {
		case <-exited:
		case <-time.After(time.Second):
			t.Fatalf("timed out stopping TCP server")
		}
	})

	return "127.0.0.1:" + port
}

func TestReadTimeout(t *testing.T) {
	var count int
	addr := rawTCPServer(t, func(c net.Conn, done <-chan struct{}) {
		count++
		if count == 1 {
			// the CLUSTER REFRESH request, return error so that it is not seen as a
			// cluster setup
			fmt.Fprint(c, "-ERR unknown command `CLUSTER`\r\n")
			return
		}
		select {
		case <-done:
			return
		case <-time.After(10 * time.Second):
			fmt.Fprint(c, "+OK\r\n") // the "simple string OK result" in redis protocol
		}
	})

	pool, err := redis.NewPool(redis.PoolConfig{
		Server:       addr,
		ConnTimeout:  2 * time.Second,
		KeepAlive:    2 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	require.NoError(t, err)

	start := time.Now()
	conn := pool.Get()
	_, err = redigo.String(conn.Do("PING"))
	require.Less(t, time.Since(start), 2*time.Second)
	require.Error(t, err)
	require.Contains(t, err.Error(), "i/o timeout")
}
