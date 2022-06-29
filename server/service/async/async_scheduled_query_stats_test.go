package async

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func testCollectScheduledQueryStats(t *testing.T, ds *mysql.Datastore, pool fleet.RedisPool) {
	// TODO(mna): test the collection part of the task
}

func testRecordScheduledQueryStatsSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	host := &fleet.Host{ID: 1}

	stats := []fleet.PackStats{{PackName: "p1", QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: "sq1"}}}}
	hashKey := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{})

	err := task.RecordScheduledQueryStats(ctx, host.ID, stats, now)
	require.NoError(t, err)
	require.True(t, ds.SaveHostPackStatsFuncInvoked)
	ds.SaveHostPackStatsFuncInvoked = false

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", hashKey)

	n, err := redigo.Int(conn.Do("EXISTS", hashKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("ZCARD", scheduledQueryStatsHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func testRecordScheduledQueryStatsAsync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	host := &fleet.Host{ID: 1}

	stats := []fleet.PackStats{
		{
			PackName: "p1", QueryStats: []fleet.ScheduledQueryStats{
				{ScheduledQueryName: "sq1", Executions: 1},
				{ScheduledQueryName: "sq2", Executions: 2},
			},
		},
		{
			PackName: "p2", QueryStats: []fleet.ScheduledQueryStats{
				{ScheduledQueryName: "sq3", Executions: 3},
			},
		},
	}
	hashKey := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{
		EnableAsyncHostProcessing:   "true",
		AsyncHostInsertBatch:        3,
		AsyncHostRedisPopCount:      3,
		AsyncHostRedisScanKeysCount: 10,
	})

	err := task.RecordScheduledQueryStats(ctx, host.ID, stats, now)
	require.NoError(t, err)
	require.False(t, ds.SaveHostPackStatsFuncInvoked)

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", hashKey)

	res, err := redigo.StringMap(conn.Do("HGETALL", hashKey))
	require.NoError(t, err)
	require.Equal(t, 3, len(res))
	keys := make([]string, 0, len(res))
	for k := range res {
		keys = append(keys, k)
	}
	require.ElementsMatch(t, []string{"p1\x00sq1", "p1\x00sq2", "p2\x00sq3"}, keys)
	// check that the marshalled value is correct (only need to check one of them)
	var sqStat fleet.ScheduledQueryStats
	err = json.Unmarshal([]byte(res["p1\x00sq1"]), &sqStat)
	require.NoError(t, err)
	require.Equal(t, 1, sqStat.Executions, res["p1\x00sq1"])

	count, err := redigo.Int(conn.Do("ZCARD", scheduledQueryStatsHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 1, count)
	tsActive, err := redigo.Int64(conn.Do("ZSCORE", scheduledQueryStatsHostIDsKey, host.ID))
	require.NoError(t, err)
	require.Equal(t, now.Unix(), tsActive)

	// running the collector removes the host from the active set
	var collStats collectorExecStats
	err = task.collectScheduledQueryStats(ctx, ds, pool, &collStats)
	require.NoError(t, err)
	require.Equal(t, 1, collStats.Keys)
	require.Equal(t, 3, collStats.Items) // sq1, sq2, sq3
	require.False(t, collStats.Failed)

	count, err = redigo.Int(conn.Do("ZCARD", scheduledQueryStatsHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// recording with pack but empty query stats works and records nothing
	stats = []fleet.PackStats{
		{
			PackName: "p1", QueryStats: []fleet.ScheduledQueryStats{},
		},
	}
	err = task.RecordScheduledQueryStats(ctx, host.ID, stats, now)
	require.NoError(t, err)
	require.False(t, ds.SaveHostPackStatsFuncInvoked)

	n, err := redigo.Int(conn.Do("EXISTS", hashKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
