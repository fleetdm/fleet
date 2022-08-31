// Package mysqlredis wraps a mysql Datastore to support adding redis-based
// operations around the standard mysql Datastore operations. An example is to
// keep a count of active hosts so that a limit can be applied.
package mysqlredis

import "github.com/fleetdm/fleet/v4/server/fleet"

// Datastore is the mysqlredis datastore type - it wraps the fleet.Datastore
// interface to keep track of enrolled hosts and extends it to implement the
// fleet.EnrollHostLimiter interface which indicates when the limit is
// reached.
type Datastore struct {
	fleet.Datastore
	pool fleet.RedisPool

	// options
	enforceHostLimit int // <= 0 means do not enforce
}

// Option is an option that can be passed to New to configure the datastore.
type Option func(*Datastore)

// WithEnforcedHostLimit enables enforcing the host limit count of the current
// license.
func WithEnforcedHostLimit(limit int) Option {
	return func(o *Datastore) {
		o.enforceHostLimit = limit
	}
}

// New creates a Datastore that wraps ds and uses pool to execute redis-based
// operations.
func New(ds fleet.Datastore, pool fleet.RedisPool, opts ...Option) *Datastore {
	newDS := &Datastore{Datastore: ds, pool: pool}
	for _, opt := range opts {
		opt(newDS)
	}
	return newDS
}
