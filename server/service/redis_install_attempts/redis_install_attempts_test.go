package redis_install_attempts

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/test"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestRedisInstallAttempts(t *testing.T) {
	for _, f := range []func(*testing.T, *redisInstallAttemptCounter){
		testRecordAndCount,
		testCountWithNothingRecorded,
		testResetAttempts,
		testRecordSetsExpiry,
		testCountsAreScopedToHostAndInstaller,
	} {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				f(t, setupRedis(t, false, false))
			})
			t.Run("cluster", func(t *testing.T) {
				f(t, setupRedis(t, true, true))
			})
		})
	}
}

func setupRedis(t testing.TB, cluster, redir bool) *redisInstallAttemptCounter {
	pool := redistest.SetupRedis(t, t.Name(), cluster, redir, true)
	return NewTest(t, pool)
}

func testRecordAndCount(t *testing.T, counter *redisInstallAttemptCounter) {
	ctx := t.Context()
	const hostID, installerID = 1, 1

	// Each attempt returns the running count.
	for expected := 1; expected <= 3; expected++ {
		count, err := counter.RecordAttempt(ctx, hostID, installerID, time.Hour)
		require.NoError(t, err)
		require.Equal(t, expected, count)
	}

	count, err := counter.CountAttempts(ctx, hostID, installerID)
	require.NoError(t, err)
	require.Equal(t, 3, count)
}

func testCountWithNothingRecorded(t *testing.T, counter *redisInstallAttemptCounter) {
	// A host that has never failed an install counts 0 rather than erroring, so the
	// limit never blocks it.
	count, err := counter.CountAttempts(t.Context(), 404, 404)
	require.NoError(t, err)
	require.Zero(t, count)
}

func testResetAttempts(t *testing.T, counter *redisInstallAttemptCounter) {
	ctx := t.Context()
	const hostID, installerID = 2, 2

	_, err := counter.RecordAttempt(ctx, hostID, installerID, time.Hour)
	require.NoError(t, err)

	require.NoError(t, counter.ResetAttempts(ctx, hostID, installerID))

	count, err := counter.CountAttempts(ctx, hostID, installerID)
	require.NoError(t, err)
	require.Zero(t, count)

	// Resetting a host with nothing recorded is not an error.
	require.NoError(t, counter.ResetAttempts(ctx, 405, 405))
}

func testRecordSetsExpiry(t *testing.T, counter *redisInstallAttemptCounter) {
	ctx := t.Context()
	const hostID, installerID = 3, 3

	// Without the expiry the count would never clear on its own.
	_, err := counter.RecordAttempt(ctx, hostID, installerID, 90*time.Minute)
	require.NoError(t, err)
	require.InDelta(t, (90 * time.Minute).Seconds(), ttlSeconds(t, counter, hostID, installerID), 60)

	// A later attempt pushes the expiry back out.
	_, err = counter.RecordAttempt(ctx, hostID, installerID, 3*time.Hour)
	require.NoError(t, err)
	require.InDelta(t, (3 * time.Hour).Seconds(), ttlSeconds(t, counter, hostID, installerID), 60)

	// An expiry under a second still has to outlive the call that set it.
	count, err := counter.RecordAttempt(ctx, hostID, installerID, 500*time.Millisecond)
	require.NoError(t, err)
	require.NotZero(t, count)

	stored, err := counter.CountAttempts(ctx, hostID, installerID)
	require.NoError(t, err)
	require.Equal(t, count, stored)
}

func testCountsAreScopedToHostAndInstaller(t *testing.T, counter *redisInstallAttemptCounter) {
	ctx := t.Context()

	for range 2 {
		_, err := counter.RecordAttempt(ctx, 10, 20, time.Hour)
		require.NoError(t, err)
	}

	// A different installer on the same host counts separately.
	count, err := counter.CountAttempts(ctx, 10, 21)
	require.NoError(t, err)
	require.Zero(t, count)

	// So does the same installer on a different host.
	count, err = counter.CountAttempts(ctx, 11, 20)
	require.NoError(t, err)
	require.Zero(t, count)

	// The host and installer that recorded the attempts keeps its own count.
	count, err = counter.CountAttempts(ctx, 10, 20)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func ttlSeconds(t *testing.T, counter *redisInstallAttemptCounter, hostID uint, softwareInstallerID uint) float64 {
	conn := redis.ConfigureDoer(counter.pool, counter.pool.Get())
	defer conn.Close()

	ttl, err := redigo.Int(conn.Do("TTL", counter.key(hostID, softwareInstallerID)))
	require.NoError(t, err)
	return float64(ttl)
}
