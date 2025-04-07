// Package redis_key_value implements a most basic SET & GET key/value store
// where both the key and the value are strings.
package redis_key_value

import (
	"context"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// RedisKeyValue is a basic key/value store with SET and GET operations
// Items are removed via expiration (defined in the SET operation).
type RedisKeyValue struct {
	pool       fleet.RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

// New creates a new RedisKeyValue store.
func New(pool fleet.RedisPool) *RedisKeyValue {
	return &RedisKeyValue{pool: pool}
}

// prefix is used to not collide with other key domains (like live queries or calendar locks).
const prefix = "key_value_"

// Set creates or overrides the given key with the given value.
// Argument expireTime is used to set the expiration of the item
// (when updating, the expiration of the item is updated).
func (r *RedisKeyValue) Set(ctx context.Context, key string, value string, expireTime time.Duration) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := redigo.String(conn.Do("SET", r.testPrefix+prefix+key, value, "PX", expireTime.Milliseconds())); err != nil {
		return ctxerr.Wrap(ctx, err, "redis failed to set")
	}
	return nil
}

// Get returns the value for a given key.
// It returns (nil, nil) if the key doesn't exist.
func (r *RedisKeyValue) Get(ctx context.Context, key string) (*string, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	res, err := redigo.String(conn.Do("GET", r.testPrefix+prefix+key))
	if errors.Is(err, redigo.ErrNil) {
		return nil, nil
	}
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "redis failed to get")
	}
	return &res, nil
}
