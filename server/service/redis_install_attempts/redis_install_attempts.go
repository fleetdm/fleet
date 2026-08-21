// Package redis_install_attempts counts failed software install attempts per host
// and installer, so Fleet can stop retrying an install that never succeeds.
package redis_install_attempts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

type redisInstallAttemptCounter struct {
	pool       fleet.RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

var _ fleet.SoftwareInstallAttemptCounter = (*redisInstallAttemptCounter)(nil)

// New creates a new install attempt counter.
func New(pool fleet.RedisPool) *redisInstallAttemptCounter {
	return &redisInstallAttemptCounter{pool: pool}
}

// prefix is used to not collide with other key domains (like live queries or calendar locks).
const prefix = "install_attempts:"

// KEYS[1]: $prefix$hostID:$softwareInstallerID (value integer)
// ARGV[1]: key TTL in seconds
//
// The TTL is refreshed on every failure, so the count clears once a host has gone
// a full window without failing this installer.
const recordAttemptScript = `
local count = redis.call("INCR", KEYS[1])
redis.call("EXPIRE", KEYS[1], ARGV[1])
return count
`

func (r *redisInstallAttemptCounter) key(hostID uint, softwareInstallerID uint) string {
	return fmt.Sprintf("%s%s%d:%d", r.testPrefix, prefix, hostID, softwareInstallerID)
}

func (r *redisInstallAttemptCounter) RecordAttempt(ctx context.Context, hostID uint, softwareInstallerID uint,
	window time.Duration,
) (int, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	count, err := redigo.Int(conn.Do("EVAL", recordAttemptScript, 1,
		r.key(hostID, softwareInstallerID),
		int(window.Seconds()),
	))
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "redis record software install attempt")
	}
	return count, nil
}

func (r *redisInstallAttemptCounter) CountAttempts(ctx context.Context, hostID uint, softwareInstallerID uint) (int, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	count, err := redigo.Int(conn.Do("GET", r.key(hostID, softwareInstallerID)))
	if errors.Is(err, redigo.ErrNil) {
		return 0, nil
	}
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "redis count software install attempts")
	}
	return count, nil
}

func (r *redisInstallAttemptCounter) ResetAttempts(ctx context.Context, hostID uint, softwareInstallerID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("DEL", r.key(hostID, softwareInstallerID)); err != nil {
		return ctxerr.Wrap(ctx, err, "redis reset software install attempts")
	}
	return nil
}
