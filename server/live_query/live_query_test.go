package live_query

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFunctions = [...]func(*testing.T, fleet.LiveQueryStore){
	testLiveQuery,
	testLiveQueryNoTargets,
	testLiveQueryStopQuery,
	testLiveQueryExpiredQuery,
	testLiveQueryOnlyExpired,
}

func testLiveQuery(t *testing.T, store fleet.LiveQueryStore) {
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

func testLiveQueryNoTargets(t *testing.T, store fleet.LiveQueryStore) {
	assert.Error(t, store.RunQuery("test", "select 1", []uint{}))
}

func testLiveQueryStopQuery(t *testing.T, store fleet.LiveQueryStore) {
	require.NoError(t, store.RunQuery("test", "select 1", []uint{1, 3}))
	require.NoError(t, store.RunQuery("test2", "select 2", []uint{1, 3}))
	require.NoError(t, store.StopQuery("test"))

	queries, err := store.QueriesForHost(1)
	require.NoError(t, err)
	assert.Len(t, queries, 1)
}

func testLiveQueryExpiredQuery(t *testing.T, store fleet.LiveQueryStore) {
	oldModulo := cleanupExpiredQueriesModulo
	cleanupExpiredQueriesModulo = 1 // run the cleanup each time
	defer func() {
		cleanupExpiredQueriesModulo = oldModulo
	}()

	require.NoError(t, store.RunQuery("test", "select 1", []uint{1}))

	// simulate a "test2" live query that has expired but is still in the set
	pool := store.(*redisLiveQuery).pool
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	_, err := conn.Do("SADD", activeQueriesKey, "test2")
	require.NoError(t, err)

	queries, err := store.QueriesForHost(1)
	require.NoError(t, err)
	assert.Len(t, queries, 1)
	assert.Equal(t, map[string]string{"test": "select 1"}, queries)

	activeNames, err := redigo.Strings(conn.Do("SMEMBERS", activeQueriesKey))
	require.NoError(t, err)
	require.Equal(t, []string{"test"}, activeNames)
}

func testLiveQueryOnlyExpired(t *testing.T, store fleet.LiveQueryStore) {
	oldModulo := cleanupExpiredQueriesModulo
	cleanupExpiredQueriesModulo = 1 // run the cleanup each time
	defer func() {
		cleanupExpiredQueriesModulo = oldModulo
	}()

	// simulate a "test" live query that has expired but is still in the set
	pool := store.(*redisLiveQuery).pool
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	_, err := conn.Do("SADD", activeQueriesKey, "test")
	require.NoError(t, err)

	queries, err := store.QueriesForHost(1)
	require.NoError(t, err)
	assert.Len(t, queries, 0)

	activeNames, err := redigo.Strings(conn.Do("SMEMBERS", activeQueriesKey))
	require.NoError(t, err)
	require.Len(t, activeNames, 0)
}
