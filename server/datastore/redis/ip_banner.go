package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
)

type IPBanner struct {
	pool      fleet.RedisPool
	keyPrefix string

	allowedConsecutiveFailures int
	banDuration                time.Duration
}

func NewIPBanner(pool fleet.RedisPool, keyPrefix string, allowedConsecutiveFailures int, banDuration time.Duration) *IPBanner {
	return &IPBanner{
		pool:                       pool,
		keyPrefix:                  keyPrefix,
		allowedConsecutiveFailures: allowedConsecutiveFailures,
		banDuration:                banDuration,
	}
}

// KEYS[1]: $keyPrefix::$ip::count (value integer)
// KEYS[2]: $keyPrefix::$ip::banned (value boolean)
// ARGV[1]: boolean: true if request succeeded
// %[1]d: allowed consecutive failures
// %[2]d: ban duration in seconds
//
// Scenarios and operations:
// - Consecutive successes: 1 read.
// - A failure after a success: 1 read + 1 write
// - A failure after a failure (below allowed consecutive failures): 1 read + 1 write
// - A failure after hitting allowed consecutive failures: 1 read + 2 writes
// - Success after a failure: 1 read + 1 write.
const updateCountScript = `
local count = redis.call('GET', KEYS[1])

if not count then
  count = %[1]d
else
  count = tonumber(count)
end

if tonumber(ARGV[1]) == 0 then
  -- failure, decrease count
  count = count - 1
  if count == 0 then
    -- mark IP as banned
	redis.call('SET', KEYS[2], 1, 'EX', %[2]d)
	-- reset count
	count = %[1]d
  end
  redis.call('SET', KEYS[1], count)
else
  -- success sets to allowed consecutive failures again
  if count < %[1]d then
    redis.call('SET', KEYS[1], %[1]d)
  end
end
`

func (s *IPBanner) CheckBanned(ip string) (bool, error) {
	key := s.keyPrefix + ip + "::banned"

	conn := s.pool.Get()
	defer conn.Close()

	// To support cluster mode.
	if err := BindConn(s.pool, conn, key); err != nil {
		return false, err
	}
	// Must come after BindConn due to redisc restrictions.
	conn = ConfigureDoer(s.pool, conn)

	if _, err := redis.String(conn.Do("GET", key)); err != nil {
		if errors.Is(err, redis.ErrNil) {
			return false, nil
		}
		return false, fmt.Errorf("redis failed to get: %w", err)
	}
	return true, nil
}

func (s *IPBanner) RunRequest(ip string, success bool) error {
	ipCountKey := s.keyPrefix + ip + "::count"
	ipBannedKey := s.keyPrefix + ip + "::banned"

	conn := s.pool.Get()
	defer conn.Close()
	if err := BindConn(s.pool, conn, ipBannedKey, ipCountKey); err != nil {
		return fmt.Errorf("bind conn: %w", err)
	}
	// must come after BindConn due to redisc restrictions
	conn = ConfigureDoer(s.pool, conn)

	ttlSeconds := max(
		int(s.banDuration.Seconds()),
		// An `EX 0` will fail, make sure that we set expiry for a minimum of one second
		1,
	)
	script := redis.NewScript(2, fmt.Sprintf(updateCountScript, s.allowedConsecutiveFailures, ttlSeconds))

	v := "0"
	if success {
		v = "1"
	}

	if _, err := script.Do(conn, ipCountKey, ipBannedKey, v); err != nil {
		return err
	}
	return nil
}
