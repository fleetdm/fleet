package pubsub

import (
	"fmt"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/require"
)

func TestEachRedisNode(t *testing.T) {
	const prefix = "TestEachRedisNode:"

	runTest := func(t *testing.T, store *redisQueryResults) {
		conn := store.Pool().Get()
		defer conn.Close()
		if rc, err := redisc.RetryConn(conn, 3, 100*time.Millisecond); err == nil {
			conn = rc
		}

		for i := 0; i < 10; i++ {
			_, err := conn.Do("SET", fmt.Sprintf("%s%d", prefix, i), i)
			require.NoError(t, err)
		}

		var keys []string
		err := EachRedisNode(store.Pool(), func(conn redis.Conn) error {
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
		store, teardown := SetupRedisForTest(t, false)
		defer teardown()
		runTest(t, store)
	})

	t.Run("cluster", func(t *testing.T) {
		store, teardown := SetupRedisForTest(t, true)
		defer teardown()
		runTest(t, store)
	})
}
