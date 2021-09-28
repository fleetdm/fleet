package redis

import (
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type netError struct {
	error
	timeout      bool
	temporary    bool
	allowedCalls int // once this reaches 0, mockDial does not return an error
	countCalls   int
}

func (t *netError) Timeout() bool   { return t.timeout }
func (t *netError) Temporary() bool { return t.temporary }

var errFromConn = errors.New("SUCCESS")

type redisConn struct{}

func (redisConn) Close() error                                       { return errFromConn }
func (redisConn) Err() error                                         { return errFromConn }
func (redisConn) Do(_ string, _ ...interface{}) (interface{}, error) { return nil, errFromConn }
func (redisConn) Send(_ string, _ ...interface{}) error              { return errFromConn }
func (redisConn) Flush() error                                       { return errFromConn }
func (redisConn) Receive() (interface{}, error)                      { return nil, errFromConn }

func TestConnectRetry(t *testing.T) {
	mockDial := func(err error) func(net, addr string, opts ...redis.DialOption) (redis.Conn, error) {
		return func(net, addr string, opts ...redis.DialOption) (redis.Conn, error) {
			var ne *netError
			if errors.As(err, &ne) {
				ne.countCalls++
				if ne.allowedCalls <= 0 {
					return redisConn{}, nil
				}
				ne.allowedCalls--
			}
			return nil, err
		}
	}

	cases := []struct {
		err       error
		retries   int
		wantCalls int
		min, max  time.Duration
	}{
		// the min-max time intervals are based on the backoff default configuration as
		// used in the Dial func of the redis pool. It starts with 500ms interval,
		// multiplies by 1.5 on each attempt, and has a randomization of 0.5 that must
		// be accounted for. Example ranges of intervals are given at
		// https://github.com/fleetdm/fleet/pull/1962#issue-729635664
		// and were used to calculate the (approximate) expected range.
		{
			io.EOF, 0, 1, 0, 100 * time.Millisecond,
		}, // non-retryable, no retry configured
		{
			&netError{error: io.EOF, timeout: true, allowedCalls: 10}, 0, 1, 0, 100 * time.Millisecond,
		}, // retryable, but no retry configured
		{
			io.EOF, 3, 1, 0, 100 * time.Millisecond,
		}, // non-retryable, retry configured
		{
			&netError{error: io.EOF, timeout: true, allowedCalls: 10}, 2, 3, 625 * time.Millisecond, 3500 * time.Millisecond,
		}, // retryable, retry configured
		{
			&netError{error: io.EOF, temporary: true, allowedCalls: 10}, 2, 3, 625 * time.Millisecond, 3500 * time.Millisecond,
		}, // retryable, retry configured
		{
			&netError{error: io.EOF, allowedCalls: 10}, 2, 1, 0, 100 * time.Millisecond,
		}, // net error, but non-retryable
		{
			&netError{error: io.EOF, timeout: true, allowedCalls: 1}, 10, 2, 250 * time.Millisecond, 750 * time.Millisecond,
		}, // retryable, but succeeded after one retry
	}
	for _, c := range cases {
		t.Run(c.err.Error(), func(t *testing.T) {
			start := time.Now()
			_, err := NewRedisPool(PoolConfig{
				Server:               "127.0.0.1:12345",
				ConnectRetryAttempts: c.retries,
				testRedisDialFunc:    mockDial(c.err),
			})
			diff := time.Since(start)
			require.GreaterOrEqual(t, diff, c.min)
			require.LessOrEqual(t, diff, c.max)
			require.Error(t, err)

			wantErr := io.EOF
			var ne *netError
			if errors.As(c.err, &ne) {
				require.Equal(t, c.wantCalls, ne.countCalls)
				if ne.allowedCalls == 0 {
					wantErr = errFromConn
				}
			} else {
				require.Equal(t, c.wantCalls, 1)
			}

			// the error is returned as part of the cluster.Refresh error, hence the
			// check with Contains.
			require.Contains(t, err.Error(), wantErr.Error())
		})
	}
}

func TestRedisPoolConfigureDoer(t *testing.T) {
	const prefix = "TestRedisPoolConfigureDoer:"

	t.Run("standalone", func(t *testing.T) {
		pool := setupRedisForTest(t, false, false)

		c1 := pool.Get()
		defer c1.Close()
		c2 := pool.ConfigureDoer(pool.Get())
		defer c2.Close()

		// both conns work equally well, get nil because keys do not exist,
		// but no redirection error (this is standalone redis).
		_, err := redis.String(c1.Do("GET", prefix+"{a}"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(c1.Do("GET", prefix+"{b}"))
		require.Equal(t, redis.ErrNil, err)

		_, err = redis.String(c2.Do("GET", prefix+"{a}"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(c2.Do("GET", prefix+"{b}"))
		require.Equal(t, redis.ErrNil, err)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := setupRedisForTest(t, true, true)

		c1 := pool.Get()
		defer c1.Close()
		c2 := pool.ConfigureDoer(pool.Get())
		defer c2.Close()

		// unconfigured conn gets MOVED error on the second key
		// (it is bound to {a}, {b} is on a different node)
		_, err := redis.String(c1.Do("GET", prefix+"{a}"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(c1.Do("GET", prefix+"{b}"))
		rerr := redisc.ParseRedir(err)
		require.Error(t, rerr)
		require.Equal(t, "MOVED", rerr.Type)

		// configured conn gets the nil value, it redirected automatically
		_, err = redis.String(c2.Do("GET", prefix+"{a}"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(c2.Do("GET", prefix+"{b}"))
		require.Equal(t, redis.ErrNil, err)
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
		pool := setupRedisForTest(t, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := setupRedisForTest(t, true, false)
		runTest(t, pool)
	})
}

func TestBindConn(t *testing.T) {
	const prefix = "TestBindConn:"

	t.Run("standalone", func(t *testing.T) {
		pool := setupRedisForTest(t, false, false)

		conn := pool.Get()
		defer conn.Close()

		err := BindConn(pool, conn, prefix+"a", prefix+"b", prefix+"c")
		require.NoError(t, err)
		_, err = redis.String(conn.Do("GET", prefix+"a"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(conn.Do("GET", prefix+"b"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(conn.Do("GET", prefix+"c"))
		require.Equal(t, redis.ErrNil, err)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := setupRedisForTest(t, true, false)

		conn := pool.Get()
		defer conn.Close()

		err := BindConn(pool, conn, prefix+"a", prefix+"b", prefix+"c")
		require.Error(t, err)

		err = BindConn(pool, conn, prefix+"{z}a", prefix+"{z}b", prefix+"{z}c")
		require.NoError(t, err)

		_, err = redis.String(conn.Do("GET", prefix+"{z}a"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(conn.Do("GET", prefix+"{z}b"))
		require.Equal(t, redis.ErrNil, err)
		_, err = redis.String(conn.Do("GET", prefix+"{z}c"))
		require.Equal(t, redis.ErrNil, err)
	})
}

func setupRedisForTest(t *testing.T, cluster, redir bool) (pool fleet.RedisPool) {
	if cluster && (runtime.GOOS == "darwin" || runtime.GOOS == "windows") {
		t.Skipf("docker networking limitations prevent running redis cluster tests on %s", runtime.GOOS)
	}

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
		Server:                    addr,
		Password:                  password,
		Database:                  database,
		UseTLS:                    useTLS,
		ConnTimeout:               5 * time.Second,
		KeepAlive:                 10 * time.Second,
		ClusterFollowRedirections: redir,
	})
	require.NoError(t, err)

	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	require.Nil(t, err)

	t.Cleanup(func() {
		err := EachRedisNode(pool, func(conn redis.Conn) error {
			_, err := conn.Do("FLUSHDB")
			return err
		})
		require.NoError(t, err)
		pool.Close()
	})

	return pool
}
