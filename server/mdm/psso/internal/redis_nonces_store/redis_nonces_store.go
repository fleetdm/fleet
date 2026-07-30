// Package redis_nonces_store is the Redis-backed implementation of
// fleet.PSSONonceStore. It stores short-lived nonces issued by the PSSO
// /nonce endpoint and consumed by the registration and token flows.
//
// Shape mirrors server/mdm/acme/internal/redis_nonces_store so the two
// stores can be operated identically.
package redis_nonces_store

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// DefaultNonceExpiration is the recommended TTL for a PSSO nonce. Apple's
// extension typically uses the nonce within seconds of receiving it, so 5
// minutes is generous slack.
const DefaultNonceExpiration = 5 * time.Minute

// prefix avoids collisions with other Redis key domains (live queries,
// calendar locks, ACME nonces).
const prefix = "pssononce:"

// RedisPool is duplicated here (rather than imported from the parent psso
// package) to avoid an import cycle: psso/providers.go imports this internal
// package to expose its public constructor, so we can't import back the
// other way. Structural typing means any value matching this shape (incl.
// the parent psso.RedisPool and fleet.RedisPool) is accepted.
type RedisPool interface {
	Get() redigo.Conn
}

// RedisNoncesStore is the Redis-backed store for PSSO nonces.
type RedisNoncesStore struct {
	pool       RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

// New creates a new RedisNoncesStore.
func New(pool RedisPool) *RedisNoncesStore {
	return &RedisNoncesStore{pool: pool}
}

// Store persists nonce with a TTL of expireTime. The value is the nonce
// itself; only key presence matters.
func (r *RedisNoncesStore) Store(ctx context.Context, nonce string, expireTime time.Duration) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := redigo.String(conn.Do("SET", r.testPrefix+prefix+nonce, nonce, "PX", expireTime.Milliseconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "redis failed to set psso nonce")
	}
	return nil
}

// Consume removes nonce from Redis and reports whether it was present. A
// nonce can be consumed at most once; subsequent attempts return false.
func (r *RedisNoncesStore) Consume(ctx context.Context, nonce string) (ok bool, err error) {
	if nonce == "" {
		return false, nil
	}

	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	n, err := redigo.Int(conn.Do("DEL", r.testPrefix+prefix+nonce))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "redis failed to delete psso nonce")
	}
	return n > 0, nil
}
