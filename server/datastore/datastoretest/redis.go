package datastoretest

import (
	"runtime"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redis2 "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func SetupRedisForTest(t *testing.T, cluster, redir bool) (pool fleet.RedisPool) {
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

	pool, err := redis.NewRedisPool(redis.PoolConfig{
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
		err := redis.EachRedisNode(pool, func(conn redis2.Conn) error {
			_, err := conn.Do("FLUSHDB")
			return err
		})
		require.NoError(t, err)
		pool.Close()
	})

	return pool
}
