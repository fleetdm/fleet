// Package mysqlredis wraps a mysql Datastore to support adding redis-based
// operations around the standard mysql Datastore operations. An example is to
// keep a count of active hosts so that a limit can be applied.
package mysqlredis

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/sync/singleflight"
)

// Datastore is the mysqlredis datastore type - it wraps the fleet.Datastore
// interface to keep track of enrolled hosts and extends it to implement the
// fleet.EnrollHostLimiter interface which indicates when the limit is
// reached.
type Datastore struct {
	fleet.Datastore
	pool fleet.RedisPool

	// options
	enforceHostLimit int // <= 0 means do not enforce

	// host lookup cache for LoadHostByNodeKey and LoadHostByOrbitNodeKey,
	// configured via WithHostCache. When hostCacheEnabled is false, all cache
	// helpers short-circuit without touching Redis. See host_cache.go.
	hostCacheEnabled bool
	hostCacheTTL     time.Duration
	hostCacheSF      singleflight.Group
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

// WithHostCache enables the Redis-backed cache for LoadHostByNodeKey and
// LoadHostByOrbitNodeKey lookups. ttl is the base TTL; actual per-entry TTL is
// jittered by ±10% to avoid synchronized expiry across a fleet. A ttl of zero
// or negative disables the cache (same effect as not calling this option).
func WithHostCache(ttl time.Duration) Option {
	return func(o *Datastore) {
		if ttl <= 0 {
			return
		}
		o.hostCacheEnabled = true
		o.hostCacheTTL = ttl
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
