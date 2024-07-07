package redis_lock

import (
	"context"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

func (r *redisLock) AcquireLock(ctx context.Context, name string, value string, expireMs uint64) (result string, err error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if expireMs == 0 {
		expireMs = defaultExpireMs
	}

	// Reference: https://redis.io/docs/latest/commands/set/
	// NX -- Only set the key if it does not already exist.
	res, err := conn.Do("SET", r.testPrefix+name, value, "NX", "PX", expireMs)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "redis acquire lock")
	}
	var ok bool
	result, ok = res.(string)
	if !ok {
		return "", nil
	}

	return result, nil
}

func (r *redisLock) ReleaseLock(ctx context.Context, name string, value string) (ok bool, err error) {
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
	res, err := conn.Do("EVAL", unlockScript, 1, r.testPrefix+name, value)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "redis release lock")
	}
	var result int64
	var castOk bool
	if result, castOk = res.(int64); !castOk {
		return false, nil
	}

	return result > 0, nil
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
	raw, err := conn.Do("SMEMBERS", r.testPrefix+key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "redis get set")
	}
	rawMembers, ok := raw.([]interface{})
	if !ok {
		return nil, ctxerr.Errorf(ctx, "redis get set: unexpected result type %T", raw)
	}
	var members []string
	for _, member := range rawMembers {
		members = append(members, fmt.Sprintf("%s", member))
	}
	return members, nil
}

func (r *redisLock) Get(ctx context.Context, name string) (*string, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	res, err := conn.Do("GET", r.testPrefix+name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "redis get")
	}

	if res == nil {
		return nil, nil
	}

	result := fmt.Sprintf("%s", res)
	return &result, nil
}
