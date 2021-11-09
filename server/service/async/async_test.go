package async

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestCollectLabelQueryExecutions(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	t.Run("standalone", func(t *testing.T) {
		defer mysql.TruncateTables(t, ds)
		pool := redistest.SetupRedis(t, false, false, false)
		testCollectLabelQueryExecutions(t, ds, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		defer mysql.TruncateTables(t, ds)
		pool := redistest.SetupRedis(t, true, true, false)
		testCollectLabelQueryExecutions(t, ds, pool)
	})
}

func testCollectLabelQueryExecutions(t *testing.T, ds *mysql.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	type labelMembership struct {
		HostID    int       `db:"host_id"`
		LabelID   uint      `db:"label_id"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	hostIDs := createHosts(t, ds, 4, time.Now().Add(-24*time.Hour))
	hid := func(id int) int {
		if id < 0 {
			return id
		}
		return int(hostIDs[id-1])
	}

	// note that cases cannot be run in isolation, each case builds on the
	// previous one's state, so they are not run as distinct sub-tests.
	cases := []struct {
		name string
		// map of host ID to label IDs to insert (true) or delete (false), a
		// negative host id is stored as an invalid redis key that should be
		// ignored.
		reported map[int]map[int]bool
		want     []labelMembership
	}{
		{"no key", nil, nil},
		{
			"report host 1 label 1",
			map[int]map[int]bool{hid(1): {1: true}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
			},
		},
		{
			"report host 1 labels 1, 2",
			map[int]map[int]bool{hid(1): {1: true, 2: true}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
				{HostID: hid(1), LabelID: 2},
			},
		},
		{
			"report host 1 labels 1, 2, 3",
			map[int]map[int]bool{1: {1: true, 2: true, 3: true}},
			[]labelMembership{
				{HostID: 1, LabelID: 1},
				{HostID: 1, LabelID: 2},
				{HostID: 1, LabelID: 3},
			},
		},
		{
			"report host 1 labels -1",
			map[int]map[int]bool{hid(1): {1: false}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 2},
				{HostID: hid(1), LabelID: 3},
			},
		},
		{
			"report host 1 labels -2, -3",
			map[int]map[int]bool{hid(1): {2: false, 3: false}},
			[]labelMembership{},
		},
		{
			"report host 1 labels 1, 2, 3, 4",
			map[int]map[int]bool{hid(1): {1: true, 2: true, 3: true, 4: true}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
				{HostID: hid(1), LabelID: 2},
				{HostID: hid(1), LabelID: 3},
				{HostID: hid(1), LabelID: 4},
			},
		},
		{
			"report host 1 labels -2, -3, -4, -5",
			map[int]map[int]bool{hid(1): {2: false, 3: false, 4: false, 5: false}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
			},
		},
		{
			"report host 1 labels 2, host 2 labels 2, 3",
			map[int]map[int]bool{hid(1): {2: true}, hid(2): {2: true, 3: true}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
				{HostID: hid(1), LabelID: 2},
				{HostID: hid(2), LabelID: 2},
				{HostID: hid(2), LabelID: 3},
			},
		},
		{
			"report host 1 labels -99, non-existing",
			map[int]map[int]bool{hid(1): {99: false}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
				{HostID: hid(1), LabelID: 2},
				{HostID: hid(2), LabelID: 2},
				{HostID: hid(2), LabelID: 3},
			},
		},
		{
			"report host -99 labels 1, ignored",
			map[int]map[int]bool{hid(-99): {1: true}},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
				{HostID: hid(1), LabelID: 2},
				{HostID: hid(2), LabelID: 2},
				{HostID: hid(2), LabelID: 3},
			},
		},
		{
			"report hosts 1, 2, 3, 4, -99 labels 1, 2, -3, 4",
			map[int]map[int]bool{
				hid(1):   {1: true, 2: true, 3: false, 4: true},
				hid(2):   {1: true, 2: true, 3: false, 4: true},
				hid(3):   {1: true, 2: true, 3: false, 4: true},
				hid(4):   {1: true, 2: true, 3: false, 4: true},
				hid(-99): {1: true, 2: true, 3: false, 4: true},
			},
			[]labelMembership{
				{HostID: hid(1), LabelID: 1},
				{HostID: hid(1), LabelID: 2},
				{HostID: hid(1), LabelID: 4},
				{HostID: hid(2), LabelID: 1},
				{HostID: hid(2), LabelID: 2},
				{HostID: hid(2), LabelID: 4},
				{HostID: hid(3), LabelID: 1},
				{HostID: hid(3), LabelID: 2},
				{HostID: hid(3), LabelID: 4},
				{HostID: hid(4), LabelID: 1},
				{HostID: hid(4), LabelID: 2},
				{HostID: hid(4), LabelID: 4},
			},
		},
	}

	const batchSizes = 3

	setupTest := func(t *testing.T, data map[int]map[int]bool) collectorExecStats {
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		// store the host memberships and prepare the expected stats
		var wantStats collectorExecStats
		for hostID, res := range data {
			if len(res) > 0 {
				key := fmt.Sprintf(labelMembershipHostKey, hostID)
				args := make(redigo.Args, 0, 1+(len(res)*2))
				args = args.Add(key)
				for lblID, ins := range res {
					score := -1
					if ins {
						score = 1
					}
					args = args.Add(score, lblID)
				}
				_, err := conn.Do("ZADD", args...)
				require.NoError(t, err)
			}
			wantStats.Keys++
			if hostID >= 0 {
				wantStats.Items += len(res)
				wantStats.RedisCmds++
				wantStats.RedisCmds += len(res) / batchSizes
			}
		}
		return wantStats
	}

	selectRows := func(t *testing.T) ([]labelMembership, map[int]time.Time) {
		var rows []labelMembership
		mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, label_id, updated_at FROM label_membership ORDER BY 1, 2`)
		})

		var hosts []struct {
			ID             int       `db:"id"`
			LabelUpdatedAt time.Time `db:"label_updated_at"`
		}
		mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &hosts, `SELECT id, label_updated_at FROM hosts`)
		})

		hostsUpdated := make(map[int]time.Time, len(hosts))
		for _, h := range hosts {
			hostsUpdated[h.ID] = h.LabelUpdatedAt
		}
		return rows, hostsUpdated
	}

	minUpdatedAt := time.Now()
	for _, c := range cases {
		func() {
			t.Log("test name: ", c.name)
			wantStats := setupTest(t, c.reported)

			// run the collection
			var stats collectorExecStats
			task := Task{
				InsertBatch:        batchSizes,
				UpdateBatch:        batchSizes,
				DeleteBatch:        batchSizes,
				RedisPopCount:      batchSizes,
				RedisScanKeysCount: 10,
			}
			err := task.collectLabelQueryExecutions(ctx, ds, pool, &stats)
			require.NoError(t, err)
			// inserts, updates and deletes are a bit tricky to track automatically,
			// just ignore them when comparing stats.
			stats.Inserts, stats.Updates, stats.Deletes = 0, 0, 0
			require.Equal(t, wantStats, stats)

			// check that the table contains the expected rows
			rows, hostsUpdated := selectRows(t)
			require.Equal(t, len(c.want), len(rows))
			for i := range c.want {
				want, got := c.want[i], rows[i]
				require.Equal(t, want.HostID, got.HostID)
				require.Equal(t, want.LabelID, got.LabelID)
				require.WithinDuration(t, minUpdatedAt, got.UpdatedAt, 10*time.Second)

				ts, ok := hostsUpdated[want.HostID]
				require.True(t, ok)
				require.WithinDuration(t, minUpdatedAt, ts, 10*time.Second)
			}
		}()
	}

	// after all cases, run one last upsert (an update) to make sure that the
	// updated at column is properly updated. First we need to ensure that this
	// runs in a distinct second, because the mysql resolution is not precise.
	time.Sleep(time.Second)

	var h1l1Before labelMembership
	beforeRows, _ := selectRows(t)
	for _, row := range beforeRows {
		if row.HostID == 1 && row.LabelID == 1 {
			h1l1Before = row
			break
		}
	}

	// update host 1, label 1, already existing
	setupTest(t, map[int]map[int]bool{1: {1: true}})
	var stats collectorExecStats
	task := Task{
		InsertBatch:        batchSizes,
		UpdateBatch:        batchSizes,
		DeleteBatch:        batchSizes,
		RedisPopCount:      batchSizes,
		RedisScanKeysCount: 10,
	}
	err := task.collectLabelQueryExecutions(ctx, ds, pool, &stats)
	require.NoError(t, err)

	var h1l1After labelMembership
	afterRows, _ := selectRows(t)
	for _, row := range afterRows {
		if row.HostID == 1 && row.LabelID == 1 {
			h1l1After = row
			break
		}
	}
	require.True(t, h1l1Before.UpdatedAt.Before(h1l1After.UpdatedAt))
}

func TestRecordLabelQueryExecutions(t *testing.T) {
	ds := new(mock.Store)
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
		return nil
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false, false)
		t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, false, false)
		t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
	})
}

func testRecordLabelQueryExecutionsSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)
	host := &fleet.Host{
		ID:             1,
		Platform:       "linux",
		LabelUpdatedAt: lastYear,
	}

	var yes, no = true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keySet, keyTs := fmt.Sprintf(labelMembershipHostKey, host.ID), fmt.Sprintf(labelMembershipReportedKey, host.ID)

	task := Task{
		Datastore:    ds,
		Pool:         pool,
		AsyncEnabled: false,
	}

	labelReportedAt := task.GetHostLabelReportedAt(ctx, host)
	require.True(t, labelReportedAt.Equal(lastYear))

	err := task.RecordLabelQueryExecutions(ctx, host, results, now)
	require.NoError(t, err)
	require.True(t, ds.RecordLabelQueryExecutionsFuncInvoked)
	ds.RecordLabelQueryExecutionsFuncInvoked = false

	conn := pool.Get()
	defer conn.Close()
	defer conn.Do("DEL", keySet, keyTs)

	n, err := redigo.Int(conn.Do("EXISTS", keySet))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("EXISTS", keyTs))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	labelReportedAt = task.GetHostLabelReportedAt(ctx, host)
	require.True(t, labelReportedAt.Equal(now))
}

func testRecordLabelQueryExecutionsAsync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)
	host := &fleet.Host{
		ID:             1,
		Platform:       "linux",
		LabelUpdatedAt: lastYear,
	}
	var yes, no = true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keySet, keyTs := fmt.Sprintf(labelMembershipHostKey, host.ID), fmt.Sprintf(labelMembershipReportedKey, host.ID)

	task := Task{
		Datastore:    ds,
		Pool:         pool,
		AsyncEnabled: true,
	}

	labelReportedAt := task.GetHostLabelReportedAt(ctx, host)
	require.True(t, labelReportedAt.Equal(lastYear))

	err := task.RecordLabelQueryExecutions(ctx, host, results, now)
	require.NoError(t, err)
	require.False(t, ds.RecordLabelQueryExecutionsFuncInvoked)

	conn := pool.Get()
	defer conn.Close()
	defer conn.Do("DEL", keySet, keyTs)

	res, err := redigo.IntMap(conn.Do("ZPOPMIN", keySet, 10))
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.Equal(t, map[string]int{"1": 1, "2": 1, "3": -1, "4": -1}, res)

	ts, err := redigo.Int64(conn.Do("GET", keyTs))
	require.NoError(t, err)
	require.Equal(t, now.Unix(), ts)

	labelReportedAt = task.GetHostLabelReportedAt(ctx, host)
	// because we transition via unix epoch (seconds), not exactly equal
	require.WithinDuration(t, now, labelReportedAt, time.Second)
	// host's LabelUpdatedAt field hasn't been updated yet, because the label
	// results are in redis, not in mysql yet.
	require.True(t, host.LabelUpdatedAt.Equal(lastYear))
}

func createHosts(t *testing.T, ds fleet.Datastore, count int, ts time.Time) []uint {
	ids := make([]uint, count)
	for i := 0; i < count; i++ {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: ts,
			LabelUpdatedAt:  ts,
			PolicyUpdatedAt: ts,
			SeenTime:        ts,
			OsqueryHostID:   fmt.Sprintf("%s%d", t.Name(), i),
			NodeKey:         fmt.Sprintf("%s%d", t.Name(), i),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(t, err)
		ids[i] = host.ID
	}
	return ids
}
