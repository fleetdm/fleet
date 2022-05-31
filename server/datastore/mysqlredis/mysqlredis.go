// Package mysqlredis wraps a mysql Datastore to support adding redis-based
// operations around the standard mysql Datastore operations. An example is to
// keep a count of active hosts so that a limit can be applied.
package mysqlredis

import "github.com/fleetdm/fleet/v4/server/fleet"

type datastore struct {
	fleet.Datastore
	pool fleet.RedisPool

	// options
	enforceHostLimit int // <= 0 means do not enforce
}

type Option func(*datastore)

// WithEnforcedHostLimit enables enforcing the host limit count of the current
// license.
func WithEnforcedHostLimit(limit int) Option {
	return func(o *datastore) {
		o.enforceHostLimit = limit
	}
}

// New creates a Datastore that wraps ds and uses pool to execute redis-based
// operations.
func New(ds fleet.Datastore, pool fleet.RedisPool, opts ...Option) fleet.Datastore {
	newDS := &datastore{Datastore: ds, pool: pool}
	for _, opt := range opts {
		opt(newDS)
	}
	return newDS
}
