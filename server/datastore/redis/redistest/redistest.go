package redistest

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

type nopRedis struct{}

func (nopRedis) Get() redigo.Conn { return nopConn{} }

func (nopRedis) Close() error { return nil }

func (nopRedis) Stats() map[string]redigo.PoolStats { return nil }

func (nopRedis) Mode() fleet.RedisMode { return fleet.RedisStandalone }

type nopConn struct{}

func (nopConn) Close() error                                       { return nil }
func (nopConn) Err() error                                         { return nil }
func (nopConn) Do(_ string, _ ...interface{}) (interface{}, error) { return nil, nil }
func (nopConn) Send(_ string, _ ...interface{}) error              { return nil }
func (nopConn) Flush() error                                       { return nil }
func (nopConn) Receive() (interface{}, error)                      { return nil, nil }

func NopRedis() fleet.RedisPool {
	return nopRedis{}
}

func SetupRedis(tb testing.TB, cleanupKeyPrefix string, cluster, redir, readReplica bool) fleet.RedisPool {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		tb.Skip("set REDIS_TEST environment variable to run redis-based tests")
	}
	if cluster && (runtime.GOOS == "darwin" || runtime.GOOS == "windows") {
		tb.Skipf("docker networking limitations prevent running redis cluster tests on %s", runtime.GOOS)
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

	pool, err := redis.NewPool(redis.PoolConfig{
		Server:                    addr,
		Password:                  password,
		Database:                  database,
		UseTLS:                    useTLS,
		ConnTimeout:               5 * time.Second,
		KeepAlive:                 10 * time.Second,
		ClusterFollowRedirections: redir,
		ClusterReadFromReplica:    readReplica,
	})
	require.NoError(tb, err)

	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	require.Nil(tb, err)

	tb.Cleanup(func() {
		if cleanupKeyPrefix != "" {
			keys, err := redis.ScanKeys(pool, cleanupKeyPrefix+"*", 1000)
			require.NoError(tb, err)
			for _, k := range keys {
				func() {
					conn := pool.Get()
					defer conn.Close()
					if _, err := conn.Do("DEL", k); err != nil {
						require.NoError(tb, err)
					}
				}()
			}
		} else {
			// NOTE: it's definitely best to avoid this - can create random failures for other
			// packages' tests when run in parallel (their keys will suddenly disappear).
			// Try to use a common prefix for you test's redis keys, or if it doesn't use any
			// key per se (e.g. just publish-listen), make it not delete anything by providing
			// an improbable prefix (e.g. zz).
			err := redis.EachNode(pool, false, func(conn redigo.Conn) error {
				_, err := conn.Do("FLUSHDB")
				return err
			})
			require.NoError(tb, err)
		}
		pool.Close()
	})

	return pool
}
