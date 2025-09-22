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

	allowedConsecutiveFailuresCount      int
	allowedConsecutiveFailuresTimeWindow time.Duration
	banDuration                          time.Duration
}

func NewIPBanner(
	pool fleet.RedisPool,
	keyPrefix string,
	allowedConsecutiveFailuresCount int,
	allowedConsecutiveFailuresTimeWindow time.Duration,
	banDuration time.Duration,
) *IPBanner {
	return &IPBanner{
		pool:      pool,
		keyPrefix: keyPrefix,

		allowedConsecutiveFailuresCount:      allowedConsecutiveFailuresCount,
		allowedConsecutiveFailuresTimeWindow: allowedConsecutiveFailuresTimeWindow,
		banDuration:                          banDuration,
	}
}

// updateCountScript is the Redis script to run on every request.
//
// KEYS[1]: $keyPrefix::$ip::count (value integer)
// KEYS[2]: $keyPrefix::$ip::banned (value boolean)
// ARGV[1]: "0" for failure, "1" for success
// ARGV[2]: threshold of consecutive failures (e.g. "1000")
// ARGV[3]: counter TTL in seconds (window for consecutive failures, e.g. "60")
// ARGV[4]: ban duration in seconds, (e.g. "60")
//
// Scenarios and operations:
// - Consecutive successes: 1 DEL
// - A failure after a success: 1 INCR + 1 EXPIRE
// - A failure after a failure (below allowed consecutive failures): 1 INCR
// - A failure after hitting allowed consecutive failures: 1 INCR + 1 SET + 1 DEL
// - Success after a failure: 1 DEL
const updateCountScript = `
local threshold = tonumber(ARGV[2])
local counter_ttl = tonumber(ARGV[3])
local ban_ttl = tonumber(ARGV[4])

if ARGV[1] == "0" then
  -- failure: increment consecutive-failure counter
  local current = redis.call("INCR", KEYS[1])
  if current == 1 then
  	redis.call("EXPIRE", KEYS[1], counter_ttl)
  elseif current >= threshold then
    -- set ban key with expiry
    redis.call("SET", KEYS[2], 1, "EX", ban_ttl)
    -- reset counter (delete it)
    redis.call("DEL", KEYS[1])
  end
else
  -- success: reset consecutive-failure counter
  redis.call("DEL", KEYS[1])
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

	allowedConsecutiveFailuresTimeWindowTTLSeconds := max(
		int(s.allowedConsecutiveFailuresTimeWindow.Seconds()),
		// An `EX 0` will fail, make sure that we set expiry for a minimum of one second
		1,
	)
	banTTLSeconds := max(
		int(s.banDuration.Seconds()),
		// An `EX 0` will fail, make sure that we set expiry for a minimum of one second
		1,
	)
	script := redis.NewScript(2, updateCountScript)

	action := "0"
	if success {
		action = "1"
	}

	if _, err := script.Do(conn,
		ipCountKey,
		ipBannedKey,

		action,
		s.allowedConsecutiveFailuresCount,
		allowedConsecutiveFailuresTimeWindowTTLSeconds,
		banTTLSeconds,
	); err != nil {
		return err
	}
	return nil
}
