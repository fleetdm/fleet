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

func (r *redisLock) Increment(ctx context.Context, name string, expireMs uint64) (int64, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	res, err := conn.Do("INCR", r.testPrefix+name)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "redis increment")
	}

	var result int64
	var ok bool
	if result, ok = res.(int64); !ok {
		return 0, ctxerr.Errorf(ctx, "redis increment: unexpected result type %T", res)
	}

	// A result of 1 indicates that the key was created. So we must also add an expiration to it.
	if result == 1 {
		if expireMs == 0 {
			expireMs = defaultExpireMs
		}
		// Reference: https://redis.io/docs/latest/commands/pexpire/
		expireResult, err := conn.Do("PEXPIRE", r.testPrefix+name, expireMs)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "redis increment expire")
		}
		if expireResult != int64(1) {
			return 0, ctxerr.Errorf(ctx, "redis increment expire: unexpected result %v", expireResult)
		}
	}

	return result, nil
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
