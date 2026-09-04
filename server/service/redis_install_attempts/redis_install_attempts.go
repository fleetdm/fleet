// Package redis_install_attempts counts failed software install attempts per host
// and installer, so Fleet can stop retrying an install that never succeeds.
package redis_install_attempts

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

type TestNamer interface {
	Name() string
}

// NewTest creates an install attempt counter to be used only in tests.
func NewTest(t TestNamer, pool fleet.RedisPool) *redisInstallAttemptCounter {
	return &redisInstallAttemptCounter{
		pool:       pool,
		testPrefix: t.Name() + ":",
	}
}

// prefix is used to not collide with other key domains (like live queries or calendar locks).
const prefix = "install:attempts:"

// KEYS[1]: $prefix$softwareInstallerID:$hostID (value integer)
// ARGV[1]: key TTL in milliseconds
//
// One script keeps the INCR and the PEXPIRE together, so a key can never be left
// without a TTL.
const recordAttemptScript = `
local count = redis.call("INCR", KEYS[1])
redis.call("PEXPIRE", KEYS[1], ARGV[1])
return count
`

// The installer comes first so every host for one installer shares a key prefix.
func (r *redisInstallAttemptCounter) key(hostID uint, softwareInstallerID uint) string {
	return fmt.Sprintf("%s%s%d:%d", r.testPrefix, prefix, softwareInstallerID, hostID)
}

func (r *redisInstallAttemptCounter) RecordAttempt(ctx context.Context, hostID uint, softwareInstallerID uint, expireIn time.Duration) (int, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	count, err := redigo.Int(conn.Do("EVAL", recordAttemptScript, 1,
		r.key(hostID, softwareInstallerID),
		expireIn.Milliseconds(),
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

// A MATCH pattern does not narrow the scan, redis walks the whole keyspace either way,
// so this scans once for every installer and picks the matching keys apart in Go.
func (r *redisInstallAttemptCounter) ResetInstallerAttempts(ctx context.Context, softwareInstallerIDs []uint) error {
	if len(softwareInstallerIDs) == 0 {
		return nil
	}

	base := r.testPrefix + prefix
	keys, err := redis.ScanKeys(r.pool, base+"*", 1000)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "redis scan software install attempt keys")
	}

	installerPrefixes := make([]string, 0, len(softwareInstallerIDs))
	for _, softwareInstallerID := range softwareInstallerIDs {
		installerPrefixes = append(installerPrefixes, fmt.Sprintf("%s%d:", base, softwareInstallerID))
	}

	var matched []string
	for _, key := range keys {
		for _, installerPrefix := range installerPrefixes {
			if strings.HasPrefix(key, installerPrefix) {
				matched = append(matched, key)
				break
			}
		}
	}
	if len(matched) == 0 {
		return nil
	}

	for _, keysInSlot := range redis.SplitKeysBySlot(r.pool, matched...) {
		conn := redis.ConfigureDoer(r.pool, r.pool.Get())
		_, err := conn.Do("DEL", redigo.Args{}.AddFlat(keysInSlot)...)
		conn.Close()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "redis reset software install attempts for installer")
		}
	}
	return nil
}
