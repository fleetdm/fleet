package redis_lock

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

// This package implements a distributed lock using Redis. The lock can be used
// to prevent multiple Fleet servers from accessing a shared resource.

const (
	defaultExpireMs = 60 * 1000
)

type redisLock struct {
	pool       fleet.RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

func NewLock(pool fleet.RedisPool) fleet.Lock {
	lock := &redisLock{
		pool: pool,
	}
	return fleet.Lock(lock)
}

func (r *redisLock) SetIfNotExist(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if expireMs == 0 {
		expireMs = defaultExpireMs
	}

	// Reference: https://redis.io/docs/latest/commands/set/
	// NX -- Only set the key if it does not already exist.
	result, err := redigo.String(conn.Do("SET", r.testPrefix+key, value, "NX", "PX", expireMs))
	if err != nil && !errors.Is(err, redigo.ErrNil) {
		return false, ctxerr.Wrap(ctx, err, "redis acquire lock")
	}
	return result != "", nil
}

func (r *redisLock) ReleaseLock(ctx context.Context, key string, value string) (ok bool, err error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	const unlockScript = `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	// Reference: https://redis.io/docs/latest/commands/set/
	// Only release the lock if the value matches.
	res, err := redigo.Int64(conn.Do("EVAL", unlockScript, 1, r.testPrefix+key, value))
	if err != nil && !errors.Is(err, redigo.ErrNil) {
		return false, ctxerr.Wrap(ctx, err, "redis release lock")
	}
	return res > 0, nil
}

func (r *redisLock) AddToSet(ctx context.Context, key string, value string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// Reference: https://redis.io/docs/latest/commands/sadd/
	_, err := conn.Do("SADD", r.testPrefix+key, value)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "redis add to set")
	}
	return nil
}

func (r *redisLock) RemoveFromSet(ctx context.Context, key string, value string) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// Reference: https://redis.io/docs/latest/commands/srem/
	_, err := conn.Do("SREM", r.testPrefix+key, value)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "redis add to set")
	}
	return nil
}

func (r *redisLock) GetSet(ctx context.Context, key string) ([]string, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// Reference: https://redis.io/docs/latest/commands/smembers/
	members, err := redigo.Strings(conn.Do("SMEMBERS", r.testPrefix+key))
	if err != nil && !errors.Is(err, redigo.ErrNil) {
		return nil, ctxerr.Wrap(ctx, err, "redis get set members")
	}
	return members, nil
}

func (r *redisLock) Get(ctx context.Context, key string) (*string, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	res, err := redigo.String(conn.Do("GET", r.testPrefix+key))
	if errors.Is(err, redigo.ErrNil) {
		return nil, nil
	}
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "redis get")
	}
	return &res, nil
}

func (r *redisLock) GetAndDelete(ctx context.Context, key string) (*string, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// Note: In Redis 6.2.0, this can be accomplished with a single command: GETDEL.

	res, err := redigo.String(conn.Do("GET", r.testPrefix+key))
	if errors.Is(err, redigo.ErrNil) {
		return nil, nil
	}
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "redis GET")
	}

	_, err = conn.Do("DEL", r.testPrefix+key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "redis DEL")
	}

	return &res, nil
}
