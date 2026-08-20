// Package redis_install_attempts records software install attempts per host and
// installer in a Redis sorted set, so Fleet can cap how many times a policy
// automation retries an install that never succeeds.
package redis_install_attempts

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

type redisInstallAttempts struct {
	pool       fleet.RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

func New(pool fleet.RedisPool) fleet.SoftwareInstallAttemptStore {
	return &redisInstallAttempts{pool: pool}
}

// prefix is used to not collide with other key domains (like live queries or calendar locks).
const prefix = "install_attempts_"

// Members are execution IDs scored by attempt time, so ZADD of an execution
// already present updates its score instead of counting it twice. Trimming by
// score before counting gives a rolling window rather than a count that only
// clears when the key expires. The expiration is a backstop that reclaims keys
// for hosts that stop attempting installs.
const recordAttemptScript = `
	redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
	redis.call('ZADD', KEYS[1], ARGV[2], ARGV[3])
	redis.call('PEXPIRE', KEYS[1], ARGV[4])
	return redis.call('ZCARD', KEYS[1])
`

func (r *redisInstallAttempts) key(hostID uint, softwareInstallerID uint) string {
	return fmt.Sprintf("%s%s%d:%d", r.testPrefix, prefix, hostID, softwareInstallerID)
}

func (r *redisInstallAttempts) RecordAttempt(ctx context.Context, hostID uint, softwareInstallerID uint, executionID string,
	now time.Time, window time.Duration,
) (int, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	count, err := redigo.Int(conn.Do("EVAL", recordAttemptScript, 1,
		r.key(hostID, softwareInstallerID),
		now.Add(-window).UnixMilli(),
		now.UnixMilli(),
		executionID,
		window.Milliseconds(),
	))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "redis record software install attempt")
	}
	return count, nil
}

func (r *redisInstallAttempts) CountAttempts(ctx context.Context, hostID uint, softwareInstallerID uint, now time.Time,
	window time.Duration,
) (int, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// ZCOUNT rather than trimming then ZCARD, so counting never writes and never
	// creates the key.
	count, err := redigo.Int(conn.Do("ZCOUNT", r.key(hostID, softwareInstallerID), now.Add(-window).UnixMilli(), "+inf"))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "redis count software install attempts")
	}
	return count, nil
}

func (r *redisInstallAttempts) ClearAttempts(ctx context.Context, hostID uint, softwareInstallerID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("DEL", r.key(hostID, softwareInstallerID)); err != nil {
		return ctxerr.Wrap(ctx, err, "redis clear software install attempts")
	}
	return nil
}
