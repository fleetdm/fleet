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
	"github.com/fleetdm/fleet/v4/server/test"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func testCollectScheduledQueryStats(t *testing.T, ds *mysql.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	// create some scheduled queries
	user := test.NewUser(t, ds, "user", "user@example.com", true)
	p1 := test.NewPack(t, ds, "p1")
	p2 := test.NewPack(t, ds, "p2")
	p3 := test.NewPack(t, ds, "p3")

	q1 := test.NewQuery(t, ds, nil, "q1", "select 1", user.ID, true)
	q2 := test.NewQuery(t, ds, nil, "q2", "select 2", user.ID, true)
	q3 := test.NewQuery(t, ds, nil, "q3", "select 3", user.ID, true)
	q4 := test.NewQuery(t, ds, nil, "q4", "select 4", user.ID, true)

	sq1 := test.NewScheduledQuery(t, ds, p1.ID, q1.ID, 60, false, false, "sq1")
	sq2 := test.NewScheduledQuery(t, ds, p2.ID, q2.ID, 60, false, false, "sq2")
	sq3 := test.NewScheduledQuery(t, ds, p3.ID, q3.ID, 60, false, false, "sq3")
	sq4 := test.NewScheduledQuery(t, ds, p3.ID, q4.ID, 60, false, false, "sq4") // pack 3 has two scheduled queries

	// create some hosts that will report stats for those scheduled queries
	hostIDs := createHosts(t, ds, 4, time.Now())
	t.Logf("real host IDs: %v", hostIDs)
	hid := func(id int) uint {
		return hostIDs[id-1]
	}

	type hostSQStats struct {
		HostID           uint `db:"host_id"`
		ScheduledQueryID uint `db:"scheduled_query_id"`
		Executions       int  `db:"executions"`
	}

	// note that cases cannot be run in isolation, each case builds on the
	// previous one's state, so they are not run as distinct sub-tests. Only the
	// Executions field of the stats is used to simplify testing and assertions.
	cases := []struct {
		name      string
		hostStats map[uint][]fleet.PackStats // key is host ID
		wantRows  []hostSQStats
	}{
		{
			"no recorded stats",
			nil,
			nil,
		},
		{
			"single host, single stat",
			map[uint][]fleet.PackStats{
				hid(1): {
					{PackName: p1.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq1.Name, Executions: 1}}},
				},
			},
			[]hostSQStats{
				{hid(1), sq1.ID, 1},
			},
		},
		{
			"multi hosts, multi stats",
			map[uint][]fleet.PackStats{
				hid(2): {
					{PackName: p1.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq1.Name, Executions: 2}}},
					{PackName: p2.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq2.Name, Executions: 3}}},
				},
				hid(3): {
					{PackName: p2.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq2.Name, Executions: 4}}},
					{PackName: p3.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq3.Name, Executions: 5}}},
					{PackName: p3.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq4.Name, Executions: 6}}},
				},
			},
			[]hostSQStats{
				{hid(1), sq1.ID, 1},
				{hid(2), sq1.ID, 2},
				{hid(2), sq2.ID, 3},
				{hid(3), sq2.ID, 4},
				{hid(3), sq3.ID, 5},
				{hid(3), sq4.ID, 6},
			},
		},
		{
			"update some, create some",
			map[uint][]fleet.PackStats{
				hid(1): {
					{PackName: p1.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq1.Name, Executions: 7}}},
					{PackName: p2.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq2.Name, Executions: 8}}},
				},
				hid(3): {
					{PackName: p3.Name, QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: sq4.Name, Executions: 9}}},
				},
			},
			[]hostSQStats{
				{hid(1), sq1.ID, 7},
				{hid(1), sq2.ID, 8},
				{hid(2), sq1.ID, 2},
				{hid(2), sq2.ID, 3},
				{hid(3), sq2.ID, 4},
				{hid(3), sq3.ID, 5},
				{hid(3), sq4.ID, 9},
			},
		},
	}

	const batchSizes = 3

	setupTest := func(t *testing.T, task *Task, data map[uint][]fleet.PackStats) collectorExecStats {
		var wantStats collectorExecStats
		for hid, stats := range data {
			err := task.RecordScheduledQueryStats(ctx, nil, hid, stats, time.Now())
			require.NoError(t, err)
		}
		wantStats.Keys = len(data)
		return wantStats
	}

	selectRows := func(t *testing.T) []hostSQStats {
		var rows []hostSQStats
		mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, scheduled_query_id, executions FROM scheduled_query_stats ORDER BY 1, 2`)
		})
		return rows
	}

	for _, c := range cases {
		func() {
			t.Log("test name: ", c.name)

			task := NewTask(ds, pool, clock.C, config.OsqueryConfig{
				EnableAsyncHostProcessing:   "true",
				AsyncHostInsertBatch:        batchSizes,
				AsyncHostUpdateBatch:        batchSizes,
				AsyncHostDeleteBatch:        batchSizes,
				AsyncHostRedisPopCount:      batchSizes,
				AsyncHostRedisScanKeysCount: 10,
			})
			wantStats := setupTest(t, task, c.hostStats)

			// run the collection
			var stats collectorExecStats
			err := task.collectScheduledQueryStats(ctx, ds, pool, &stats)
			require.NoError(t, err)
			require.Equal(t, wantStats.Keys, stats.Keys)

			// check that the table contains the expected rows
			rows := selectRows(t)
			require.Equal(t, c.wantRows, rows)
		}()
	}
}

func testRecordScheduledQueryStatsSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	host := &fleet.Host{ID: 1}

	stats := []fleet.PackStats{{PackName: "p1", QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: "sq1"}}}}
	hashKey := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{})

	err := task.RecordScheduledQueryStats(ctx, host.TeamID, host.ID, stats, now)
	require.NoError(t, err)
	require.True(t, ds.SaveHostPackStatsFuncInvoked)
	ds.SaveHostPackStatsFuncInvoked = false

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", hashKey) //nolint:errcheck

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

	err := task.RecordScheduledQueryStats(ctx, host.TeamID, host.ID, stats, now)
	require.NoError(t, err)
	require.False(t, ds.SaveHostPackStatsFuncInvoked)

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", hashKey) //nolint:errcheck

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
	require.Equal(t, uint64(1), sqStat.Executions, res["p1\x00sq1"])

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
	err = task.RecordScheduledQueryStats(ctx, host.TeamID, host.ID, stats, now)
	require.NoError(t, err)
	require.False(t, ds.SaveHostPackStatsFuncInvoked)

	n, err := redigo.Int(conn.Do("EXISTS", hashKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
