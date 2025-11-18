package redis

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gomodule/redigo/redis"
)

// IPBanner implements an IP banning mechanism with Redis as backend.
//
// It allows a configurable number of consecutive failures (allowedConsecutiveFailuresCount)
// on a configurable time window (allowedConsecutiveFailuresTimeWindow) and after hitting
// such threshold it will ban the IP for a configurable amout of time (banDuration).
//
// - The CheckBanned operation can be used before running a request to check whether an IP is banned.
// - The RunRequest operation is to be executed with the result of every request (success or failure).
type IPBanner struct {
	pool      fleet.RedisPool
	keyPrefix string

	// allowedConsecutiveFailuresCount is the allowed number of failed requests for an IP.
	allowedConsecutiveFailuresCount int
	// allowedConsecutiveFailuresTimeWindow is the time window to allow up to
	// allowedConsecutiveFailuresCount request failures.
	allowedConsecutiveFailuresTimeWindow time.Duration
	// banDuration is the duration an IP will be banned after hitting the threshold.
	banDuration time.Duration
}

// NewIPBanner creates an IPBanner backed by Redis.
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
// KEYS[1]: $keyPrefix::{$ip}::count (value integer)
// KEYS[2]: $keyPrefix::{$ip}::banned (value boolean)
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

// CheckBanned returns true if the IP is currently banned.
func (s *IPBanner) CheckBanned(ip string) (bool, error) {
	ip = SetNullIfEmptyIP(ip)

	// enclosing in {} to support Redis cluster.
	key := s.keyPrefix + "{" + ip + "}::banned"

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

// RunRequest will update the status of the given IP with the result of a request.
func (s *IPBanner) RunRequest(ip string, success bool) error {
	ip = SetNullIfEmptyIP(ip)

	// enclosing in {} to support Redis cluster.
	ipCountKey := s.keyPrefix + "{" + ip + "}::count"
	ipBannedKey := s.keyPrefix + "{" + ip + "}::banned"

	conn := s.pool.Get()
	defer conn.Close()

	if err := BindConn(s.pool, conn, ipBannedKey, ipCountKey); err != nil {
		return fmt.Errorf("bind conn: %w", err)
	}
	// must come after BindConn due to redisc restrictions
	conn = ConfigureDoer(s.pool, conn)

	allowedConsecutiveFailuresCount := s.allowedConsecutiveFailuresCount
	allowedConsecutiveFailuresTimeWindow := s.allowedConsecutiveFailuresTimeWindow
	banDuration := s.banDuration

	// This is just for testing purposes (no op in production).
	if isTest, testIPBannerAllowedConsecutiveFailuresCount, testIPBannerAllowedConsecutiveFailuresTimeWindow, testIPBannerBanDuration := getIPBannerTestValues(); isTest {
		allowedConsecutiveFailuresCount = testIPBannerAllowedConsecutiveFailuresCount
		allowedConsecutiveFailuresTimeWindow = testIPBannerAllowedConsecutiveFailuresTimeWindow
		banDuration = testIPBannerBanDuration
	}

	allowedConsecutiveFailuresTimeWindowTTLSeconds := max(
		// An `EX 0` will fail, make sure that we set expiry for a minimum of one second
		int(allowedConsecutiveFailuresTimeWindow.Seconds()), 1,
	)
	banTTLSeconds := max(
		// An `EX 0` will fail, make sure that we set expiry for a minimum of one second
		int(banDuration.Seconds()), 1,
	)
	script := redis.NewScript(2, updateCountScript)

	action := "0" // failure
	if success {
		action = "1" // success
	}

	if _, err := script.Do(conn,
		ipCountKey,
		ipBannedKey,

		action,
		allowedConsecutiveFailuresCount,
		allowedConsecutiveFailuresTimeWindowTTLSeconds,
		banTTLSeconds,
	); err != nil {
		return err
	}
	return nil
}

// SetNullIfEmptyIP sets the string "null" if the input IP is empty.
//
// Exported for tests.
func SetNullIfEmptyIP(ip string) string {
	if ip == "" {
		return "null"
	}
	return ip
}

var (
	testIPBannerMu sync.Mutex
	testIPBanner   bool

	testIPBannerAllowedConsecutiveFailuresCount      int
	testIPBannerAllowedConsecutiveFailuresTimeWindow time.Duration
	testIPBannerBanDuration                          time.Duration
)

func SetIPBannerTestValues(
	allowedConsecutiveFailuresCount int,
	allowedConsecutiveFailuresTimeWindow time.Duration,
	banDuration time.Duration,
) {
	testIPBannerMu.Lock()
	defer testIPBannerMu.Unlock()

	testIPBanner = true
	testIPBannerAllowedConsecutiveFailuresCount = allowedConsecutiveFailuresCount
	testIPBannerAllowedConsecutiveFailuresTimeWindow = allowedConsecutiveFailuresTimeWindow
	testIPBannerBanDuration = banDuration
}

func UnsetIPBannerTestValues() {
	testIPBannerMu.Lock()
	defer testIPBannerMu.Unlock()

	testIPBanner = false

	testIPBannerAllowedConsecutiveFailuresCount = 0
	testIPBannerAllowedConsecutiveFailuresTimeWindow = 0
	testIPBannerBanDuration = 0
}

func getIPBannerTestValues() (test bool, allowedConsecutiveFailuresCount int, allowedConsecutiveFailuresTimeWindow time.Duration, banDuration time.Duration) {
	testIPBannerMu.Lock()
	defer testIPBannerMu.Unlock()

	test = testIPBanner
	allowedConsecutiveFailuresCount = testIPBannerAllowedConsecutiveFailuresCount
	allowedConsecutiveFailuresTimeWindow = testIPBannerAllowedConsecutiveFailuresTimeWindow
	banDuration = testIPBannerBanDuration
	return
}
