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
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func testCollectScheduledQueryStats(t *testing.T, ds *mysql.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	user := test.NewUser(t, ds, "user", "user@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team1Pack := fmt.Sprintf("team-%d", team1.ID)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	team2Pack := fmt.Sprintf("team-%d", team2.ID)

	gq1 := test.NewQueryWithSchedule(t, ds, nil, "q1", "select 1", user.ID, true, 60, false)
	gq2 := test.NewQueryWithSchedule(t, ds, nil, "q2", "select 2", user.ID, true, 60, false)
	// same name as a global report, but in team1
	tq1 := test.NewQueryWithSchedule(t, ds, &team1.ID, "q1", "select 1", user.ID, true, 60, false)
	tq3 := test.NewQueryWithSchedule(t, ds, &team1.ID, "q3", "select 3", user.ID, true, 60, false)
	t2q1 := test.NewQueryWithSchedule(t, ds, &team2.ID, "q1", "select 1", user.ID, true, 60, false)

	hostIDs := createHosts(t, ds, 3, time.Now())
	hid := func(id int) uint {
		return hostIDs[id-1]
	}
	// hosts 2 and 3 are in team1; stats are recorded with the host's fleet
	hostTeams := map[uint]*uint{hid(2): &team1.ID, hid(3): &team1.ID}

	type hostSQStats struct {
		HostID           uint `db:"host_id"`
		ScheduledQueryID uint `db:"scheduled_query_id"`
		Executions       int  `db:"executions"`
	}

	stat := func(name string, execs uint64) []fleet.ScheduledQueryStats {
		return []fleet.ScheduledQueryStats{{ScheduledQueryName: name, Executions: execs}}
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
			"single host, single global stat",
			map[uint][]fleet.PackStats{
				hid(1): {
					{PackName: "Global", QueryStats: stat(gq1.Name, 1)},
				},
			},
			[]hostSQStats{
				{hid(1), gq1.ID, 1},
			},
		},
		{
			"multi hosts, global and team stats",
			map[uint][]fleet.PackStats{
				hid(2): {
					{PackName: "Global", QueryStats: stat(gq1.Name, 2)},
					{PackName: "Global", QueryStats: stat(gq2.Name, 3)},
					{PackName: team1Pack, QueryStats: stat(tq1.Name, 4)},
				},
				hid(3): {
					{PackName: team1Pack, QueryStats: stat(tq3.Name, 5)},
				},
			},
			[]hostSQStats{
				{hid(1), gq1.ID, 1},
				{hid(2), gq1.ID, 2},
				{hid(2), gq2.ID, 3},
				{hid(2), tq1.ID, 4},
				{hid(3), tq3.ID, 5},
			},
		},
		{
			"unresolvable stats are skipped",
			map[uint][]fleet.PackStats{
				hid(1): {
					{PackName: "Global", QueryStats: stat(gq1.Name, 6)},
					{PackName: "Global", QueryStats: stat("deleted-report", 7)},
					{PackName: "Global", QueryStats: stat(tq3.Name, 8)}, // team report sent as global
					{PackName: "team-999999", QueryStats: stat(gq1.Name, 9)},
					{PackName: "team-abc", QueryStats: stat(gq1.Name, 11)},
					{PackName: "2017 pack", QueryStats: stat("sq4", 10)},
				},
				hid(3): {
					// another fleet's report, with the same name as a report in the
					// host's fleet
					{PackName: team2Pack, QueryStats: stat(t2q1.Name, 12)},
				},
			},
			[]hostSQStats{
				{hid(1), gq1.ID, 6},
				{hid(2), gq1.ID, 2},
				{hid(2), gq2.ID, 3},
				{hid(2), tq1.ID, 4},
				{hid(3), tq3.ID, 5},
			},
		},
	}

	const batchSizes = 3

	setupTest := func(t *testing.T, task *Task, data map[uint][]fleet.PackStats) collectorExecStats {
		var wantStats collectorExecStats
		for hid, stats := range data {
			err := task.RecordScheduledQueryStats(ctx, hostTeams[hid], hid, stats, time.Now())
			require.NoError(t, err)
		}
		wantStats.Keys = len(data)
		return wantStats
	}

	selectRows := func(t *testing.T) []hostSQStats {
		var rows []hostSQStats
		mysqltest.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, scheduled_query_id, executions FROM scheduled_query_stats ORDER BY 1, 2`)
		})
		return rows
	}

	for _, c := range cases {
		func() {
			t.Log("test name: ", c.name)

			task := NewTask(ds, pool, clock.C, &config.FleetConfig{
				Osquery: config.OsqueryConfig{
					EnableAsyncHostProcessing:   "true",
					AsyncHostInsertBatch:        batchSizes,
					AsyncHostUpdateBatch:        batchSizes,
					AsyncHostDeleteBatch:        batchSizes,
					AsyncHostRedisPopCount:      batchSizes,
					AsyncHostRedisScanKeysCount: 10,
				},
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

// Report IDs are resolved on the primary: a report that hasn't reached the
// replica yet must still get its stats.
func TestCollectStatsReadsPrimary(t *testing.T) {
	ds := mysqltest.CreateMySQLDSWithOptions(t, &testing_utils.DatastoreTestOptions{DummyReplica: true})
	pool := redistest.SetupRedis(t, "scheduled_query_stats_primary", false, false, false)
	ctx := t.Context()

	report, err := ds.NewQuery(ctx, &fleet.Query{
		Name:     "q1",
		Query:    "select 1",
		Saved:    true,
		Interval: 60,
		Logging:  fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	hostIDs := createHosts(t, ds, 1, time.Now())

	task := NewTask(ds, pool, clock.C, &config.FleetConfig{
		Osquery: config.OsqueryConfig{
			EnableAsyncHostProcessing:   "true",
			AsyncHostInsertBatch:        3,
			AsyncHostRedisPopCount:      3,
			AsyncHostRedisScanKeysCount: 10,
		},
	})
	err = task.RecordScheduledQueryStats(ctx, nil, hostIDs[0], []fleet.PackStats{
		{PackName: "Global", QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: report.Name, Executions: 3}}},
	}, time.Now())
	require.NoError(t, err)

	var stats collectorExecStats
	require.NoError(t, task.collectScheduledQueryStats(ctx, ds, pool, &stats))

	var executions []uint64
	mysqltest.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, tx, &executions,
			`SELECT executions FROM scheduled_query_stats WHERE host_id = ? AND scheduled_query_id = ?`, hostIDs[0], report.ID)
	})
	require.Equal(t, []uint64{3}, executions)
}

func testRecordScheduledQueryStatsSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	host := &fleet.Host{ID: 1}

	stats := []fleet.PackStats{{PackName: "p1", QueryStats: []fleet.ScheduledQueryStats{{ScheduledQueryName: "sq1"}}}}
	hashKey := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, host.ID)

	task := NewTask(ds, pool, clock.C, nil)

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
	host := &fleet.Host{ID: 1, TeamID: new(uint(7))}

	stats := []fleet.PackStats{
		{
			PackName: "Global", QueryStats: []fleet.ScheduledQueryStats{
				{ScheduledQueryName: "sq1", Executions: 1},
				{ScheduledQueryName: "sq2", Executions: 2},
			},
		},
		{
			PackName: "team-7", QueryStats: []fleet.ScheduledQueryStats{
				{ScheduledQueryName: "sq3", Executions: 3},
			},
		},
		// not queued: another fleet's pack and a 2017 pack
		{
			PackName: "team-8", QueryStats: []fleet.ScheduledQueryStats{
				{ScheduledQueryName: "sq4", Executions: 4},
			},
		},
		{
			PackName: "p1", QueryStats: []fleet.ScheduledQueryStats{
				{ScheduledQueryName: "sq5", Executions: 5},
			},
		},
	}
	hashKey := fmt.Sprintf(scheduledQueryStatsHostQueriesKey, host.ID)

	task := NewTask(ds, pool, clock.C, &config.FleetConfig{
		Osquery: config.OsqueryConfig{
			EnableAsyncHostProcessing:   "true",
			AsyncHostInsertBatch:        3,
			AsyncHostRedisPopCount:      3,
			AsyncHostRedisScanKeysCount: 10,
		},
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
	require.ElementsMatch(t, []string{"Global\x00sq1", "Global\x00sq2", "team-7\x00sq3"}, keys)
	// check that the marshalled value is correct (only need to check one of them)
	var sqStat fleet.ScheduledQueryStats
	err = json.Unmarshal([]byte(res["Global\x00sq1"]), &sqStat)
	require.NoError(t, err)
	require.Equal(t, uint64(1), sqStat.Executions, res["Global\x00sq1"])

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
			PackName: "Global", QueryStats: []fleet.ScheduledQueryStats{},
		},
	}
	err = task.RecordScheduledQueryStats(ctx, host.TeamID, host.ID, stats, now)
	require.NoError(t, err)
	require.False(t, ds.SaveHostPackStatsFuncInvoked)

	n, err := redigo.Int(conn.Do("EXISTS", hashKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
