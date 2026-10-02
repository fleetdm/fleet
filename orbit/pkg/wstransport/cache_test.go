package wstransport

import (
	"testing"

	"github.com/fleetdm/fleet/v4/client"
	"github.com/stretchr/testify/assert"
)

func TestQueryCacheTakeClears(t *testing.T) {
	cache := NewQueryCache()

	// Empty cache: nothing to take.
	queries, discovery, accelerate := cache.Take()
	assert.Nil(t, queries)
	assert.Nil(t, discovery)
	assert.Zero(t, accelerate)

	cache.Set(&client.DistributedReadResponse{
		Queries:    map[string]string{"q1": "SELECT 1"},
		Discovery:  map[string]string{"q1": "SELECT 1"},
		Accelerate: 10,
	})

	queries, discovery, accelerate = cache.Take()
	assert.Equal(t, map[string]string{"q1": "SELECT 1"}, queries)
	assert.Equal(t, map[string]string{"q1": "SELECT 1"}, discovery)
	assert.Equal(t, uint(10), accelerate)

	// A second Take returns nothing: each batch is delivered exactly once.
	queries, _, accelerate = cache.Take()
	assert.Nil(t, queries)
	assert.Zero(t, accelerate)
}

func TestQueryCacheSetMerges(t *testing.T) {
	cache := NewQueryCache()

	cache.Set(&client.DistributedReadResponse{
		Queries:   map[string]string{"q1": "SELECT 1"},
		Discovery: map[string]string{"q1": "SELECT 1"},
	})
	// A second read before osquery polls must not lose q1.
	cache.Set(&client.DistributedReadResponse{
		Queries:    map[string]string{"q2": "SELECT 2"},
		Accelerate: 5,
	})

	queries, discovery, accelerate := cache.Take()
	assert.Equal(t, map[string]string{"q1": "SELECT 1", "q2": "SELECT 2"}, queries)
	assert.Equal(t, map[string]string{"q1": "SELECT 1"}, discovery)
	assert.Equal(t, uint(5), accelerate)

	// Later values overwrite same-name queries.
	cache.Set(&client.DistributedReadResponse{Queries: map[string]string{"q3": "SELECT 3"}})
	cache.Set(&client.DistributedReadResponse{Queries: map[string]string{"q3": "SELECT 33"}})
	queries, _, _ = cache.Take()
	assert.Equal(t, map[string]string{"q3": "SELECT 33"}, queries)
}
