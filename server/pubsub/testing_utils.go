package pubsub

import (
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func SetupRedisForTest(t *testing.T, cluster bool) (store *redisQueryResults, teardown func()) {
	var (
		addr       = "127.0.0.1:"
		password   = ""
		database   = 0
		useTLS     = false
		dupResults = false
		port       = "6379"
	)
	if cluster {
		port = "7001"
	}
	addr += port

	pool, err := NewRedisPool(addr, password, database, useTLS)
	require.NoError(t, err)
	store = NewRedisQueryResults(pool, dupResults)

	conn := store.pool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	require.Nil(t, err)

	teardown = func() {
		err := EachRedisNode(store.pool, func(conn redis.Conn) error {
			_, err := conn.Do("FLUSHDB")
			return err
		})
		require.NoError(t, err)
		store.pool.Close()
	}

	return store, teardown
}
