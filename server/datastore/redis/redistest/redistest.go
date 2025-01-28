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
	return SetupRedisWithConfig(tb, cleanupKeyPrefix, cluster, redir, readReplica, redis.PoolConfig{})
}

func SetupRedisWithConfig(tb testing.TB, cleanupKeyPrefix string, cluster, redir, readReplica bool, config redis.PoolConfig) fleet.RedisPool {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		tb.Skip("set REDIS_TEST environment variable to run redis-based tests")
	}
	if cluster && (runtime.GOOS == "darwin" || runtime.GOOS == "windows") {
		tb.Skipf("docker networking limitations prevent running redis cluster tests on %s", runtime.GOOS)
	}
	if cleanupKeyPrefix == "" {
		tb.Fatal("missing cleanup key prefix: all redis tests need to specify a key prefix to delete on test cleanup, otherwise different packages' tests running concurrently might delete other tests' keys")
	}

	var (
		addr     = "127.0.0.1:"
		username = ""
		password = ""
		database = 0
		useTLS   = false
		port     = "6379"
	)
	if cluster {
		port = "7001"
	}
	addr += port

	// set the mandatory, non-configurable configs for tests
	config.Server = addr
	config.Username = username
	config.Password = password
	config.Database = database
	config.UseTLS = useTLS
	config.ClusterFollowRedirections = redir
	config.ClusterReadFromReplica = readReplica
	if config.ConnTimeout == 0 {
		config.ConnTimeout = 5 * time.Second
	}
	if config.KeepAlive == 0 {
		config.KeepAlive = 10 * time.Second
	}

	pool, err := redis.NewPool(config)
	require.NoError(tb, err)

	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	require.Nil(tb, err)

	// We run a cleanup before running the tests in case a previous
	// run failed to run the cleanup (e.g. Ctrl+C while running tests).
	cleanup(tb, pool, cleanupKeyPrefix)

	tb.Cleanup(func() {
		cleanup(tb, pool, cleanupKeyPrefix)
		pool.Close()
	})

	return pool
}

func cleanup(tb testing.TB, pool fleet.RedisPool, cleanupKeyPrefix string) {
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
}
