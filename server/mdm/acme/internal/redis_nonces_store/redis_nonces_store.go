package redis_nonces_store

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	redigo "github.com/gomodule/redigo/redis"
)

const DefaultNonceExpiration = 1 * time.Hour

// RedisNoncesStore is a store for ACME nonces, implemented using Redis.
type RedisNoncesStore struct {
	pool       acme.RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

// New creates a new RedisNoncesStore store.
func New(pool acme.RedisPool) *RedisNoncesStore {
	return &RedisNoncesStore{pool: pool}
}

// prefix is used to not collide with other key domains (like live queries or calendar locks).
const prefix = "acmenonce:"

// Store creates the key with the given nonce.
// Argument expireTime is used to set the expiration of the item.
func (r *RedisNoncesStore) Store(ctx context.Context, nonce string, expireTime time.Duration) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// the value of the key is not really important, just that the key exists or not (indicates
	// that the nonce is valid or not), so we set the value to be the same as the nonce.
	if _, err := redigo.String(conn.Do("SET", r.testPrefix+prefix+nonce, nonce, "PX", expireTime.Milliseconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "redis failed to set")
	}
	return nil
}

// Consume validates and consumes the nonce, ensuring it does exist, and then removing
// it from the store so it can't be used again.
func (r *RedisNoncesStore) Consume(ctx context.Context, nonce string) (ok bool, err error) {
	// fast path if the nonce is missing
	if nonce == "" {
		return false, nil
	}

	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	n, err := redigo.Int(conn.Do("DEL", r.testPrefix+prefix+nonce))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "redis failed to delete")
	}
	return n > 0, nil
}
