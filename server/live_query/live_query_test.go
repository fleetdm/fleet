package live_query

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFunctions = [...]func(*testing.T, fleet.LiveQueryStore){
	testLiveQuery,
	testLiveQueryNoTargets,
	testLiveQueryStopQuery,
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
