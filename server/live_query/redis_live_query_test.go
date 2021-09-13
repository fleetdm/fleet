package live_query

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/test"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLiveQuery(t *testing.T) {
	for _, f := range testFunctions {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				store, teardown := setupRedisLiveQuery(t, false)
				defer teardown()
				f(t, store)
			})

			t.Run("cluster", func(t *testing.T) {
				store, teardown := setupRedisLiveQuery(t, true)
				defer teardown()
				f(t, store)
			})
		})
	}
}

func TestMigrateKeys(t *testing.T) {
	startKeys := map[string]string{
		"unrelated":                           "u",
		queryKeyPrefix + "a":                  "a",
		sqlKeyPrefix + queryKeyPrefix + "a":   "sqla",
		queryKeyPrefix + "b":                  "b",
		queryKeyPrefix + "{c}":                "c",
		sqlKeyPrefix + queryKeyPrefix + "{c}": "sqlc",
	}

	endKeys := map[string]string{
		"unrelated":                           "u",
		queryKeyPrefix + "{a}":                "a",
		sqlKeyPrefix + queryKeyPrefix + "{a}": "sqla",
		queryKeyPrefix + "{b}":                "b",
		queryKeyPrefix + "{c}":                "c",
		sqlKeyPrefix + queryKeyPrefix + "{c}": "sqlc",
	}

	runTest := func(t *testing.T, store *redisLiveQuery) {
		conn := store.pool.Get()
		defer conn.Close()
		if rc, err := redisc.RetryConn(conn, 3, 100*time.Millisecond); err == nil {
			conn = rc
		}

		for k, v := range startKeys {
			_, err := conn.Do("SET", k, v)
			require.NoError(t, err)
		}

		err := store.MigrateKeys()
		require.NoError(t, err)

		got := make(map[string]string)
		err = redis.EachRedisNode(store.pool, func(conn redigo.Conn) error {
			keys, err := redigo.Strings(conn.Do("KEYS", "*"))
			if err != nil {
				return err
			}

			for _, k := range keys {
				v, err := redigo.String(conn.Do("GET", k))
				if err != nil {
					return err
				}
				got[k] = v
			}
			return nil
		})
		require.NoError(t, err)

		require.EqualValues(t, endKeys, got)
	}

	t.Run("standalone", func(t *testing.T) {
		store, teardown := setupRedisLiveQuery(t, false)
		defer teardown()
		runTest(t, store)
	})

	t.Run("cluster", func(t *testing.T) {
		store, teardown := setupRedisLiveQuery(t, true)
		defer teardown()
		runTest(t, store)
	})
}

func setupRedisLiveQuery(t *testing.T, cluster bool) (store *redisLiveQuery, teardown func()) {
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
		Server:      addr,
		Password:    password,
		Database:    database,
		UseTLS:      useTLS,
		ConnTimeout: 5 * time.Second,
		KeepAlive:   10 * time.Second,
	})
	require.NoError(t, err)
	store = NewRedisLiveQuery(pool)

	conn := store.pool.Get()
	defer conn.Close()
	_, err = conn.Do("PING")
	require.NoError(t, err)

	teardown = func() {
		err := redis.EachRedisNode(store.pool, func(conn redigo.Conn) error {
			_, err := conn.Do("FLUSHDB")
			return err
		})
		require.NoError(t, err)
		store.pool.Close()
	}

	return store, teardown
}

func TestMapBitfield(t *testing.T) {
	// empty
	assert.Equal(t, []byte{}, mapBitfield(nil))
	assert.Equal(t, []byte{}, mapBitfield([]uint{}))

	// one byte
	assert.Equal(t, []byte("\x80"), mapBitfield([]uint{0}))
	assert.Equal(t, []byte("\x40"), mapBitfield([]uint{1}))
	assert.Equal(t, []byte("\xc0"), mapBitfield([]uint{0, 1}))

	assert.Equal(t, []byte("\x08"), mapBitfield([]uint{4}))
	assert.Equal(t, []byte("\xf8"), mapBitfield([]uint{0, 1, 2, 3, 4}))
	assert.Equal(t, []byte("\xff"), mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7}))

	// two bytes
	assert.Equal(t, []byte("\x00\x80"), mapBitfield([]uint{8}))
	assert.Equal(t, []byte("\xff\x80"), mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8}))

	// more bytes
	assert.Equal(
		t,
		[]byte("\xff\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00 "),
		mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 170}),
	)
	assert.Equal(
		t,
		[]byte("\xff\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00@\x00\x00\x00\x00\x00\x00 "),
		mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 113, 170}),
	)
	assert.Equal(
		t,
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01"),
		mapBitfield([]uint{79}),
	)
}
