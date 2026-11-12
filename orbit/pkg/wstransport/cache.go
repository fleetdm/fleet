package wstransport

import (
	"maps"
	"sync"

	"github.com/fleetdm/fleet/v4/client"
)

// QueryCache holds distributed queries fetched from the Fleet server (via
// HTTP distributed/read) until osquery picks them up on its next local
// distributed poll (a thrift call to orbit's distributed plugin).
type QueryCache struct {
	mu         sync.Mutex
	queries    map[string]string
	discovery  map[string]string
	accelerate uint
}

func NewQueryCache() *QueryCache {
	return &QueryCache{}
}

// Set merges resp into the cache. Merging (rather than replacing) matters: a
// second distributed/read may complete before osquery's next local poll, and
// its queries must not clobber ones still waiting to be picked up.
func (c *QueryCache) Set(resp *client.DistributedReadResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.queries == nil {
		c.queries = make(map[string]string)
		c.discovery = make(map[string]string)
	}
	maps.Copy(c.queries, resp.Queries)
	maps.Copy(c.discovery, resp.Discovery)
	c.accelerate = max(c.accelerate, resp.Accelerate)
}

// Take returns the cached queries and clears the cache. Clearing is
// load-bearing: the server's distributed/read is consume-on-read for live
// queries, so osquery must receive each cached batch exactly once — leaving
// queries in the cache would make osquery re-run them on every 10s local poll.
func (c *QueryCache) Take() (queries, discovery map[string]string, accelerate uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	queries, discovery, accelerate = c.queries, c.discovery, c.accelerate
	c.queries, c.discovery, c.accelerate = nil, nil, 0
	return queries, discovery, accelerate
}
