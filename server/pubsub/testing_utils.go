package pubsub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func SetupRedisForTest(t *testing.T) (store *redisQueryResults, teardown func()) {
	var (
		addr       = "127.0.0.1:6379"
		password   = ""
		database   = 0
		useTLS     = false
		dupResults = false
	)

	pool, err := NewRedisPool(addr, password, database, useTLS)
	require.NoError(t, err)
	store = NewRedisQueryResults(pool, dupResults)

	_, err = store.pool.Get().Do("PING")
	require.Nil(t, err)

	teardown = func() {
		store.pool.Close()
	}

	return store, teardown
}
