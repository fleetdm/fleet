package live_query

import (
	"context"
	"testing"
	"time"

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
	testLiveQueryCleanupInactive,
	testLiveQuerySetBitOnlyIfKeyExists,
	testLiveQueryResultsCounts,
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
	t.Cleanup(func() { cleanupExpiredQueriesModulo = oldModulo })

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

	assert.Eventually(t, func() bool {
		activeNames, err := redigo.Strings(conn.Do("SMEMBERS", activeQueriesKey))
		require.NoError(t, err)
		if len(activeNames) == 1 && activeNames[0] == "test" {
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)
}

func testLiveQueryOnlyExpired(t *testing.T, store fleet.LiveQueryStore) {
	oldModulo := cleanupExpiredQueriesModulo
	cleanupExpiredQueriesModulo = 1 // run the cleanup each time
	t.Cleanup(func() { cleanupExpiredQueriesModulo = oldModulo })

	// simulate a "test" live query that has expired but is still in the set
	pool := store.(*redisLiveQuery).pool
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	_, err := conn.Do("SADD", activeQueriesKey, "test")
	require.NoError(t, err)

	queries, err := store.QueriesForHost(1)
	require.NoError(t, err)
	assert.Len(t, queries, 0)

	assert.Eventually(t, func() bool {
		activeNames, err := redigo.Strings(conn.Do("SMEMBERS", activeQueriesKey))
		require.NoError(t, err)
		return len(activeNames) == 0
	}, 5*time.Second, 100*time.Millisecond)
}

func testLiveQueryCleanupInactive(t *testing.T, store fleet.LiveQueryStore) {
	ctx := context.Background()

	// get a raw Redis connection to make direct checks
	pool := store.(*redisLiveQuery).pool
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()

	// run a few live queries, making them active in Redis
	err := store.RunQuery("1", "SELECT 1", []uint{1, 2, 3})
	require.NoError(t, err)
	err = store.RunQuery("2", "SELECT 2", []uint{4})
	require.NoError(t, err)
	err = store.RunQuery("3", "SELECT 3", []uint{5, 6})
	require.NoError(t, err)
	err = store.RunQuery("4", "SELECT 4", []uint{1, 2, 5})
	require.NoError(t, err)
	err = store.RunQuery("5", "SELECT 5", []uint{2, 3, 7})
	require.NoError(t, err)

	activeNames, err := store.LoadActiveQueryNames()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"1", "2", "3", "4", "5"}, activeNames)

	// sanity-check that the queries are properly stored
	m, err := store.QueriesForHost(1)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"1": "SELECT 1", "4": "SELECT 4"}, m)

	// simulate that only campaigns 2 and 4 are still active, cleanup the rest
	err = store.CleanupInactiveQueries(ctx, []uint{1, 3, 5})
	require.NoError(t, err)

	activeNames, err = store.LoadActiveQueryNames()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"2", "4"}, activeNames)

	m, err = store.QueriesForHost(1)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"4": "SELECT 4"}, m)

	// explicitly mark campaign 4 as stopped
	err = store.StopQuery("4")
	require.NoError(t, err)

	// no more queries for host 1
	m, err = store.QueriesForHost(1)
	require.NoError(t, err)
	require.Empty(t, m)

	// only campaign 2 remains, for host 4
	m, err = store.QueriesForHost(4)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"2": "SELECT 2"}, m)

	// simulate that there are no inactive campaigns to cleanup
	err = store.CleanupInactiveQueries(ctx, nil)
	require.NoError(t, err)

	activeNames, err = store.LoadActiveQueryNames()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"2"}, activeNames)

	// simulate that all campaigns are inactive, cleanup all
	err = store.CleanupInactiveQueries(ctx, []uint{1, 2, 3, 4, 5})
	require.NoError(t, err)

	activeNames, err = store.LoadActiveQueryNames()
	require.NoError(t, err)
	require.Empty(t, activeNames)

	m, err = store.QueriesForHost(4)
	require.NoError(t, err)
	require.Empty(t, m)
}

func testLiveQuerySetBitOnlyIfKeyExists(t *testing.T, store fleet.LiveQueryStore) {
	// Create a live query campaign.
	err := store.RunQuery("test", "SELECT 1;", []uint{1})
	require.NoError(t, err)

	// Get the query for the host.
	queries, err := store.QueriesForHost(1)
	require.NoError(t, err)
	require.Equal(t,
		map[string]string{
			"test": "SELECT 1;",
		},
		queries,
	)

	// Mark query as completed by host.
	err = store.QueryCompletedByHost("test", 1)
	require.NoError(t, err)

	// Query should not be returned anymore as it was marked as completed for this host.
	queries, err = store.QueriesForHost(1)
	require.NoError(t, err)
	require.Empty(t, queries)

	// A host could be attempting to write a result for a query that was already deleted.
	err = store.QueryCompletedByHost("test-2", 1)
	require.NoError(t, err)

	// Let's test that such key was not created.

	// get a raw Redis connection to make direct checks
	pool := store.(*redisLiveQuery).pool
	conn := redis.ConfigureDoer(pool, pool.Get())
	t.Cleanup(func() {
		conn.Close()
	})

	n, err := redigo.Int(conn.Do("EXISTS", queryKeyPrefix+"{test-2}"))
	require.NoError(t, err)
	require.Zero(t, n)
}

func testLiveQueryResultsCounts(t *testing.T, store fleet.LiveQueryStore) {
	// Use many query IDs so that, in cluster mode, their keys spread across
	// multiple hash slots - exercising the split-by-slot pipelining.
	queryIDs := []uint{1, 2, 3, 4, 5, 10, 42, 100, 250, 999}

	// The query_results_count keys are not covered by the test cleanup key
	// prefix, so clear any leftover state from previous runs and after this one.
	cleanup := func() {
		for _, id := range append(append([]uint{}, queryIDs...), 123456) {
			require.NoError(t, store.DeleteQueryResultsCount(id))
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	// counts for never-incremented queries default to 0
	counts, err := store.GetQueryResultsCounts(queryIDs)
	require.NoError(t, err)
	for _, id := range queryIDs {
		require.Zero(t, counts[id])
	}

	// increment each query by a distinct amount
	increments := make(map[uint]int, len(queryIDs))
	for i, id := range queryIDs {
		increments[id] = i + 1
	}
	err = store.IncrQueryResultsCounts(increments)
	require.NoError(t, err)

	counts, err = store.GetQueryResultsCounts(queryIDs)
	require.NoError(t, err)
	for _, id := range queryIDs {
		require.Equal(t, increments[id], counts[id])
	}

	// incrementing again accumulates
	err = store.IncrQueryResultsCounts(increments)
	require.NoError(t, err)

	counts, err = store.GetQueryResultsCounts(queryIDs)
	require.NoError(t, err)
	for _, id := range queryIDs {
		require.Equal(t, 2*increments[id], counts[id])
	}

	// a query that was never incremented is explicitly populated with 0 in the
	// returned map, mixed with ones that were incremented
	counts, err = store.GetQueryResultsCounts([]uint{queryIDs[0], 123456})
	require.NoError(t, err)
	require.Equal(t, 2*increments[queryIDs[0]], counts[queryIDs[0]])
	require.Contains(t, counts, uint(123456))
	require.Zero(t, counts[123456])

	// empty inputs are no-ops
	err = store.IncrQueryResultsCounts(nil)
	require.NoError(t, err)
	counts, err = store.GetQueryResultsCounts(nil)
	require.NoError(t, err)
	require.Empty(t, counts)
}
