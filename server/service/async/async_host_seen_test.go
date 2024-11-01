package async

import (
	"context"
	"database/sql"
	"math"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func testCollectHostsLastSeen(t *testing.T, ds *mysql.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	type hostLastSeen struct {
		HostID   int          `db:"host_id"`
		SeenTime sql.NullTime `db:"seen_time"`
	}

	mockTime := clock.NewMockClock()
	startTime := mockTime.Now()

	hostIDs := createHosts(t, ds, 4, startTime)
	t.Logf("real host IDs: %v", hostIDs)
	hid := func(id int) int {
		return int(hostIDs[id-1]) //nolint:gosec // dismiss G115
	}

	// note that cases cannot be run in isolation, each case builds on the
	// previous one's state, so they are not run as distinct sub-tests. An
	// hour is added to mockTime after each test case.
	cases := []struct {
		name    string
		hostIDs []int
		want    []hostLastSeen
	}{
		{
			"no key",
			nil,
			[]hostLastSeen{
				// createHosts (called above) stores the initial host seen time
				{HostID: hid(1), SeenTime: sql.NullTime{Time: startTime}},
				{HostID: hid(2), SeenTime: sql.NullTime{Time: startTime}},
				{HostID: hid(3), SeenTime: sql.NullTime{Time: startTime}},
				{HostID: hid(4), SeenTime: sql.NullTime{Time: startTime}},
			},
		},
		{
			"report host 1",
			[]int{hid(1)},
			[]hostLastSeen{
				{HostID: hid(1), SeenTime: sql.NullTime{Time: startTime.Add(time.Hour)}},
				{HostID: hid(2), SeenTime: sql.NullTime{Time: startTime}},
				{HostID: hid(3), SeenTime: sql.NullTime{Time: startTime}},
				{HostID: hid(4), SeenTime: sql.NullTime{Time: startTime}},
			},
		},
		{
			"report hosts 2, 3",
			[]int{hid(2), hid(3)},
			[]hostLastSeen{
				{HostID: hid(1), SeenTime: sql.NullTime{Time: startTime.Add(time.Hour)}},
				{HostID: hid(2), SeenTime: sql.NullTime{Time: startTime.Add(2 * time.Hour)}},
				{HostID: hid(3), SeenTime: sql.NullTime{Time: startTime.Add(2 * time.Hour)}},
				{HostID: hid(4), SeenTime: sql.NullTime{Time: startTime}},
			},
		},
		{
			"report hosts 1, 2, 3, 4",
			[]int{hid(1), hid(2), hid(3), hid(4)},
			[]hostLastSeen{
				{HostID: hid(1), SeenTime: sql.NullTime{Time: startTime.Add(3 * time.Hour)}},
				{HostID: hid(2), SeenTime: sql.NullTime{Time: startTime.Add(3 * time.Hour)}},
				{HostID: hid(3), SeenTime: sql.NullTime{Time: startTime.Add(3 * time.Hour)}},
				{HostID: hid(4), SeenTime: sql.NullTime{Time: startTime.Add(3 * time.Hour)}},
			},
		},
		{
			"report hosts 2, 3, 4",
			[]int{hid(2), hid(3), hid(4)},
			[]hostLastSeen{
				{HostID: hid(1), SeenTime: sql.NullTime{Time: startTime.Add(3 * time.Hour)}},
				{HostID: hid(2), SeenTime: sql.NullTime{Time: startTime.Add(4 * time.Hour)}},
				{HostID: hid(3), SeenTime: sql.NullTime{Time: startTime.Add(4 * time.Hour)}},
				{HostID: hid(4), SeenTime: sql.NullTime{Time: startTime.Add(4 * time.Hour)}},
			},
		},
		{
			"report no new hosts",
			[]int{},
			[]hostLastSeen{
				{HostID: hid(1), SeenTime: sql.NullTime{Time: startTime.Add(3 * time.Hour)}},
				{HostID: hid(2), SeenTime: sql.NullTime{Time: startTime.Add(4 * time.Hour)}},
				{HostID: hid(3), SeenTime: sql.NullTime{Time: startTime.Add(4 * time.Hour)}},
				{HostID: hid(4), SeenTime: sql.NullTime{Time: startTime.Add(4 * time.Hour)}},
			},
		},
	}

	const batchSizes = 3

	setupTest := func(t *testing.T, ids []int) collectorExecStats {
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		// store the host memberships and prepare the expected stats
		var wantStats collectorExecStats
		wantStats.Keys = 2
		wantStats.RedisCmds = 1

		if len(ids) > 0 {
			args := redigo.Args{hostSeenRecordedHostIDsKey}
			args = args.AddFlat(ids)
			_, err := conn.Do("SADD", args...)
			require.NoError(t, err)

			wantStats.Items = len(ids)
			wantStats.Inserts = int(math.Ceil(float64(len(ids)) / float64(batchSizes)))
		}
		return wantStats
	}

	selectRows := func(t *testing.T) []hostLastSeen {
		var rows []hostLastSeen
		mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, seen_time FROM host_seen_times ORDER BY 1`)
		})
		return rows
	}

	for _, c := range cases {
		func() {
			t.Log("test name: ", c.name)
			wantStats := setupTest(t, c.hostIDs)

			// run the collection
			var stats collectorExecStats
			task := NewTask(nil, nil, mockTime, config.OsqueryConfig{
				AsyncHostInsertBatch:        batchSizes,
				AsyncHostUpdateBatch:        batchSizes,
				AsyncHostDeleteBatch:        batchSizes,
				AsyncHostRedisPopCount:      batchSizes,
				AsyncHostRedisScanKeysCount: 10,
			})
			err := task.collectHostsLastSeen(ctx, ds, pool, &stats)
			require.NoError(t, err)
			require.Equal(t, wantStats, stats)

			// check that the table contains the expected rows
			rows := selectRows(t)
			require.Equal(t, len(c.want), len(rows))
			for i := range c.want {
				want, got := c.want[i], rows[i]
				require.Equal(t, want.HostID, got.HostID)
				require.WithinDuration(t, want.SeenTime.Time, got.SeenTime.Time, time.Second)
			}
			mockTime.AddTime(time.Hour)
		}()
	}
}

func testRecordHostLastSeenSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()

	var calledWithHostIDs []uint
	ds.MarkHostsSeenFunc = func(ctx context.Context, hostIDs []uint, ts time.Time) error {
		calledWithHostIDs = append(calledWithHostIDs, hostIDs...)
		return nil
	}

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{})
	err := task.RecordHostLastSeen(ctx, 1)
	require.NoError(t, err)
	err = task.RecordHostLastSeen(ctx, 2)
	require.NoError(t, err)
	err = task.RecordHostLastSeen(ctx, 3)
	require.NoError(t, err)
	require.False(t, ds.MarkHostsSeenFuncInvoked)

	err = task.FlushHostsLastSeen(ctx, time.Now())
	require.NoError(t, err)
	require.True(t, ds.MarkHostsSeenFuncInvoked)
	require.ElementsMatch(t, []uint{1, 2, 3}, calledWithHostIDs)
	ds.MarkHostsSeenFuncInvoked = false

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", hostSeenRecordedHostIDsKey, hostSeenProcessingHostIDsKey) //nolint:errcheck

	n, err := redigo.Int(conn.Do("EXISTS", hostSeenRecordedHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("EXISTS", hostSeenProcessingHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func testRecordHostLastSeenAsync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()

	var calledWithHostIDs []uint
	ds.MarkHostsSeenFunc = func(ctx context.Context, hostIDs []uint, ts time.Time) error {
		calledWithHostIDs = append(calledWithHostIDs, hostIDs...)
		return nil
	}

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{
		EnableAsyncHostProcessing:   "true",
		AsyncHostInsertBatch:        2,
		AsyncHostRedisScanKeysCount: 10,
	})

	err := task.RecordHostLastSeen(ctx, 1)
	require.NoError(t, err)
	err = task.RecordHostLastSeen(ctx, 2)
	require.NoError(t, err)
	err = task.RecordHostLastSeen(ctx, 3)
	require.NoError(t, err)
	require.False(t, ds.MarkHostsSeenFuncInvoked)

	err = task.FlushHostsLastSeen(ctx, time.Now())
	require.NoError(t, err)
	require.False(t, ds.MarkHostsSeenFuncInvoked)

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", hostSeenRecordedHostIDsKey, hostSeenProcessingHostIDsKey) //nolint:errcheck

	n, err := redigo.Int(conn.Do("SCARD", hostSeenRecordedHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 3, n)

	n, err = redigo.Int(conn.Do("EXISTS", hostSeenProcessingHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// running the collector removes the recorded host IDs key
	var stats collectorExecStats
	err = task.collectHostsLastSeen(ctx, ds, pool, &stats)
	require.NoError(t, err)
	require.Equal(t, 2, stats.Keys)
	require.Equal(t, 3, stats.Items)
	require.False(t, stats.Failed)
	require.True(t, ds.MarkHostsSeenFuncInvoked)
	require.ElementsMatch(t, []uint{1, 2, 3}, calledWithHostIDs)
	ds.MarkHostsSeenFuncInvoked = false

	n, err = redigo.Int(conn.Do("EXISTS", hostSeenRecordedHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("EXISTS", hostSeenProcessingHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
