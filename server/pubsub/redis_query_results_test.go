package pubsub

import (
	"fmt"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestEachRedisNodeStandalone(t *testing.T) {
	prefix := "TestEachRedisNodeStandalone:"
	store, teardown := SetupRedisForTest(t)
	defer teardown()

	conn := store.Pool().Get()
	defer conn.Close()

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
