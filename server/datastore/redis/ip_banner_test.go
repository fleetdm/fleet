package redis_test

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestIPBanner(t *testing.T) {
	const prefix = "TestIPBanner::"

	runTest := func(t *testing.T, pool fleet.RedisPool) {
		basicTest := func(t *testing.T, ip string, otherIP string) {
			conn := pool.Get()
			t.Cleanup(func() {
				conn.Close()
			})
			conn2 := pool.Get()
			t.Cleanup(func() {
				conn2.Close()
			})

			ipKey := redis.SetNullIfEmptyIP(ip)
			countKey := prefix + "{" + ipKey + "}::count"
			countKey2 := prefix + "{" + otherIP + "}::count"

			// To support cluster mode.
			err := redis.BindConn(pool, conn, countKey)
			require.NoError(t, err)
			// Must come after BindConn due to redisc restrictions.
			conn = redis.ConfigureDoer(pool, conn)
			require.NoError(t, err)
			// To support cluster mode.
			err = redis.BindConn(pool, conn2, countKey2)
			require.NoError(t, err)
			// Must come after BindConn due to redisc restrictions.
			conn2 = redis.ConfigureDoer(pool, conn2)
			require.NoError(t, err)

			allowedConsecutiveFailuresCount := 5
			allowedConsecutiveFailuresTimeWindow := 10 * time.Second
			banDuration := 5 * time.Second

			ipBan := redis.NewIPBanner(pool, prefix, allowedConsecutiveFailuresCount, allowedConsecutiveFailuresTimeWindow, banDuration)

			// Initially the IP should not be banned.
			banned, err := ipBan.CheckBanned(ip)
			require.NoError(t, err)
			require.False(t, banned)

			// Running a successful request initially should not create any entries.
			err = ipBan.RunRequest(ip, true)
			require.NoError(t, err)
			_, err = redigo.Int(conn.Do("GET", countKey))
			require.ErrorIs(t, err, redigo.ErrNil)

			// Running one failure request decrements the counter (but still not banned).
			err = ipBan.RunRequest(ip, false)
			require.NoError(t, err)
			currentAllowedConsecutiveFailures := allowedConsecutiveFailuresCount - 1
			v, err := redigo.Int(conn.Do("GET", countKey))
			require.NoError(t, err)
			require.Equal(t, 1, v)
			banned, err = ipBan.CheckBanned(ip)
			require.NoError(t, err)
			require.False(t, banned)

			// Run all but one consecutive failing requests, still not banned.
			for range currentAllowedConsecutiveFailures - 1 {
				err = ipBan.RunRequest(ip, false)
				require.NoError(t, err)
				banned, err = ipBan.CheckBanned(ip)
				require.NoError(t, err)
				require.False(t, banned)
			}

			// Run the last remaining consecutive failing request, should be banned now.
			err = ipBan.RunRequest(ip, false)
			require.NoError(t, err)
			banned, err = ipBan.CheckBanned(ip)
			require.NoError(t, err)
			require.True(t, banned)
			// Check count has been reset.
			_, err = redigo.Int(conn.Do("GET", countKey))
			require.ErrorIs(t, err, redigo.ErrNil)

			// Sleep for the duration of the ban (and a bit more).
			time.Sleep(5*time.Second + 100*time.Millisecond)
			// Should not be banned now.
			banned, err = ipBan.CheckBanned(ip)
			require.NoError(t, err)
			require.False(t, banned)

			// Run all but one consecutive failing requests, still not banned.
			currentAllowedConsecutiveFailures = allowedConsecutiveFailuresCount
			for range currentAllowedConsecutiveFailures - 1 {
				err = ipBan.RunRequest(ip, false)
				require.NoError(t, err)
				banned, err = ipBan.CheckBanned(ip)
				require.NoError(t, err)
				require.False(t, banned)
			}
			// Run a successful request, should clear the count.
			err = ipBan.RunRequest(ip, true)
			require.NoError(t, err)
			// Check count has been reset.
			_, err = redigo.Int(conn.Do("GET", countKey))
			require.ErrorIs(t, err, redigo.ErrNil)
			// Confirm an extra failing request does not ban.
			err = ipBan.RunRequest(ip, false)
			require.NoError(t, err)
			banned, err = ipBan.CheckBanned(ip)
			require.NoError(t, err)
			require.False(t, banned)

			// Run all but one consecutive failing requests, still not banned.
			currentAllowedConsecutiveFailures = allowedConsecutiveFailuresCount
			for range currentAllowedConsecutiveFailures - 1 {
				err = ipBan.RunRequest(otherIP, false)
				require.NoError(t, err)
				banned, err = ipBan.CheckBanned(otherIP)
				require.NoError(t, err)
				require.False(t, banned)
			}
			// Wait for the time window to be over, which should clear the counts.
			time.Sleep(allowedConsecutiveFailuresTimeWindow + 100*time.Millisecond)
			// Check count has been reset.
			_, err = redigo.Int(conn2.Do("GET", countKey2))
			require.ErrorIs(t, err, redigo.ErrNil)
			// Confirm an extra failing request does not ban.
			err = ipBan.RunRequest(otherIP, false)
			require.NoError(t, err)
			banned, err = ipBan.CheckBanned(otherIP)
			require.NoError(t, err)
			require.False(t, banned)
		}

		// Test basic functionality.
		t.Run("basic", func(t *testing.T) {
			t.Parallel()

			basicTest(t, "127.0.0.1", "192.168.0.1")
		})

		// Test with empty IP (when Fleet cannot extract the IP from the request).
		// All these requests would endup on the same "bucket".
		t.Run("basic-empty", func(t *testing.T) {
			t.Parallel()

			basicTest(t, "", "192.168.0.2")
		})

		// Test that the banning/counts of different IPs are isolated.
		t.Run("two-ips", func(t *testing.T) {
			t.Parallel()

			conn := pool.Get()
			t.Cleanup(func() {
				conn.Close()
			})

			allowedConsecutiveFailuresCount := 5
			allowedConsecutiveFailuresTimeWindow := 1 * time.Minute
			banDuration := 5 * time.Second

			ipBan := redis.NewIPBanner(pool, prefix, allowedConsecutiveFailuresCount, allowedConsecutiveFailuresTimeWindow, banDuration)

			ip1 := "127.0.0.2"
			ip2 := "::1"

			// ip1 makes a failing request.
			err := ipBan.RunRequest(ip1, false)
			require.NoError(t, err)
			banned, err := ipBan.CheckBanned(ip1)
			require.NoError(t, err)
			require.False(t, banned)
			v, err := redigo.Int(conn.Do("GET", prefix+"{"+ip1+"}::count"))
			require.NoError(t, err)
			require.Equal(t, 1, v)

			// ip2 is not affected.
			banned, err = ipBan.CheckBanned(ip2)
			require.NoError(t, err)
			require.False(t, banned)
			_, err = redigo.Int(conn.Do("GET", prefix+"{"+ip2+"}::count"))
			require.ErrorIs(t, err, redigo.ErrNil)
		})
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, true, true, false)
		runTest(t, pool)
	})

	t.Run("cluster_nofollow", func(t *testing.T) {
		pool := redistest.SetupRedis(t, prefix, true, false, false)
		runTest(t, pool)
	})
}
