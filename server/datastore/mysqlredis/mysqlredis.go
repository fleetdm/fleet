// Package mysqlredis wraps a mysql Datastore to support adding redis-based
// operations around the standard mysql Datastore operations. An example is to
// keep a count of active hosts so that a limit can be applied.
package mysqlredis

import "github.com/fleetdm/fleet/v4/server/fleet"

type datastore struct {
	fleet.Datastore
	pool fleet.RedisPool
}

// New creates a Datastore that wraps ds and uses pool to execute redis-based
// operations.
func New(ds fleet.Datastore, pool fleet.RedisPool) fleet.Datastore {
	return &datastore{Datastore: ds, pool: pool}
}
