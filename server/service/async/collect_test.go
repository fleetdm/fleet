package async

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestCollector(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		var (
			countErr     int
			countHandler int
			simulKeys    = 0
			simulItems   = 100
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		coll := collector{
			name:         "test",
			pool:         pool,
			execInterval: 10 * time.Millisecond,
			jitterPct:    10,
			lockTimeout:  time.Second,
			handler: func(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool, stats *collectorExecStats) error {
				countHandler++
				simulKeys++
				simulItems--
				stats.Items = simulItems
				stats.Keys = simulKeys
				return nil
			},
			errHandler: func(name string, err error) {
				countErr++
			},
		}

		done := make(chan bool)
		go func() {
			coll.Start(ctx)
			close(done)
		}()

		<-time.After(100 * time.Millisecond)
		cancel()
		<-done

		// running at each 10ms ±10% for 100ms, min 9, max 11 but stay on the
		// safe side, especially for min.
		require.GreaterOrEqual(t, countHandler, 5)
		require.LessOrEqual(t, countHandler, 12)

		stats := coll.ReadStats()
		require.Equal(t, countHandler, stats.ExecCount)
		require.Equal(t, 0, stats.FailuresCount)
		require.Equal(t, 0, countErr)
		require.Equal(t, simulKeys, stats.MaxExecKeys)
		require.Equal(t, simulKeys, stats.LastExecKeys)
		require.Equal(t, 1, stats.MinExecKeys)
		require.Equal(t, simulItems, stats.MinExecItems)
		require.Equal(t, 99, stats.MaxExecItems)
		require.Equal(t, simulItems, stats.LastExecItems)
		require.Greater(t, stats.MinExecDuration, time.Duration(0))
		require.Greater(t, stats.MaxExecDuration, time.Duration(0))
		require.Greater(t, stats.LastExecDuration, time.Duration(0))
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, false, false)
		runTest(t, pool)
	})
}
