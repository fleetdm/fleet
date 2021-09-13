package redis

import (
	"fmt"
	"io"
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
	timeout   bool
	temporary bool
	count     int // once this reaches 0, mockDial does not return an error
}

func (t *netError) Timeout() bool   { return t.timeout }
func (t *netError) Temporary() bool { return t.temporary }

type redisConn struct{}

func (redisConn) Close() error                                       { return io.EOF }
func (redisConn) Err() error                                         { return io.EOF }
func (redisConn) Do(_ string, _ ...interface{}) (interface{}, error) { return nil, io.EOF }
func (redisConn) Send(_ string, _ ...interface{}) error              { return io.EOF }
func (redisConn) Flush() error                                       { return io.EOF }
func (redisConn) Receive() (interface{}, error)                      { return nil, io.EOF }

func TestConnectRetry(t *testing.T) {
	mockDial := func(err error) func(net, addr string, opts ...redis.DialOption) (redis.Conn, error) {
		return func(net, addr string, opts ...redis.DialOption) (redis.Conn, error) {
			var ne *netError
			if errors.As(err, &ne) {
				ne.count--
				if ne.count <= 0 {
					return redisConn{}, nil
				}
			}
			return nil, err
		}
	}

	cases := []struct {
		err      error
		retries  int
		min, max time.Duration
	}{
		{io.EOF, 0, 0, 100 * time.Millisecond},                                                  // non-retryable, no retry configured
		{&netError{io.EOF, true, false, 10}, 0, 0, 100 * time.Millisecond},                      // retryable, but no retry configured
		{io.EOF, 3, 0, 100 * time.Millisecond},                                                  // non-retryable, retry configured
		{&netError{io.EOF, true, false, 10}, 2, time.Second, 3 * time.Second},                   // retryable, retry configured
		{&netError{io.EOF, false, true, 10}, 2, time.Second, 3 * time.Second},                   // retryable, retry configured
		{&netError{io.EOF, false, false, 10}, 2, 0, 100 * time.Millisecond},                     // net error, but non-retryable
		{&netError{io.EOF, true, false, 2}, 10, 100 * time.Millisecond, 500 * time.Millisecond}, // retryable, but succeeded after one
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
			// the error is returned as part of the cluster.Refresh error, hence the
			// check with Contains.
			require.Contains(t, err.Error(), io.EOF.Error())
		})
	}
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
