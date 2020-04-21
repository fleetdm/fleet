package live_query

import (
	"fmt"
	"os"
	"testing"

	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/pubsub"
	"github.com/kolide/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLiveQuery(t *testing.T) {
	if _, ok := os.LookupEnv("REDIS_TEST"); !ok {
		t.SkipNow()
	}

	for _, f := range testFunctions {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			store, teardown := setupRedisLiveQuery(t)
			defer teardown()
			f(t, store)
		})
	}
}

var testFunctions = [...]func(*testing.T, kolide.LiveQueryStore){
	testRedisLiveQuery,
	testRedisLiveQueryNoTargets,
	testRedisLiveQueryStopQuery,
}

func setupRedisLiveQuery(t *testing.T) (store *redisLiveQuery, teardown func()) {
	var (
		addr     = "127.0.0.1:6379"
		password = ""
	)

	if a, ok := os.LookupEnv("REDIS_PORT_6379_TCP_ADDR"); ok {
		addr = fmt.Sprintf("%s:6379", a)
	}

	store = NewRedisLiveQuery(pubsub.NewRedisPool(addr, password))

	_, err := store.pool.Get().Do("PING")
	require.Nil(t, err)

	teardown = func() {
		store.pool.Get().Do("FLUSHDB")
		store.pool.Close()
	}

	return store, teardown
}

func testRedisLiveQueryNoTargets(t *testing.T, store kolide.LiveQueryStore) {
	assert.Error(t, store.RunQuery("test", "select 1", []uint{}))
}

func testRedisLiveQueryStopQuery(t *testing.T, store kolide.LiveQueryStore) {
	require.NoError(t, store.RunQuery("test", "select 1", []uint{1, 3}))
	require.NoError(t, store.RunQuery("test2", "select 2", []uint{1, 3}))
	require.NoError(t, store.StopQuery("test"))

	queries, err := store.QueriesForHost(1)
	assert.NoError(t, err)
	assert.Len(t, queries, 1)
}

func testRedisLiveQuery(t *testing.T, store kolide.LiveQueryStore) {
	queries, err := store.QueriesForHost(1)
	assert.NoError(t, err)
	assert.Len(t, queries, 0)
	queries, err = store.QueriesForHost(3)
	assert.NoError(t, err)
	assert.Len(t, queries, 0)

	assert.NoError(t, store.RunQuery("test", "select 1", []uint{1, 3}))
	assert.NoError(t, store.RunQuery("test2", "select 2", []uint{3}))
	assert.NoError(t, store.RunQuery("test3", "select 3", []uint{1}))
	assert.NoError(t, store.RunQuery("test4", "select 4", []uint{4}))

	queries, err = store.QueriesForHost(1)
	assert.NoError(t, err)
	assert.Equal(t,
		map[string]string{
			"test":  "select 1",
			"test3": "select 3",
		},
		queries,
	)
	queries, err = store.QueriesForHost(2)
	assert.NoError(t, err)
	assert.Len(t, queries, 0)
	queries, err = store.QueriesForHost(3)
	assert.NoError(t, err)
	assert.Equal(t,
		map[string]string{
			"test":  "select 1",
			"test2": "select 2",
		},
		queries,
	)

	assert.NoError(t, store.QueryCompletedByHost("test", 1))
	assert.NoError(t, store.QueryCompletedByHost("test2", 3))

	queries, err = store.QueriesForHost(1)
	assert.NoError(t, err)
	assert.Equal(t,
		map[string]string{
			"test3": "select 3",
		},
		queries,
	)
	queries, err = store.QueriesForHost(2)
	assert.NoError(t, err)
	assert.Len(t, queries, 0)
	queries, err = store.QueriesForHost(3)
	assert.NoError(t, err)
	assert.Equal(t,
		map[string]string{
			"test": "select 1",
		},
		queries,
	)
}
