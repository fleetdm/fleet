package async

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func testCollectPolicyQueryExecutions(t *testing.T, ds *mysql.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	type policyMembership struct {
		HostID    int          `db:"host_id"`
		PolicyID  int          `db:"policy_id"`
		Passes    sql.NullBool `db:"passes"`
		UpdatedAt time.Time    `db:"updated_at"`
	}

	hostIDs := createHosts(t, ds, 4, time.Now().Add(-24*time.Hour))
	policyIDs := createPolicies(t, ds, 4)
	t.Logf("real host IDs: %v", hostIDs)
	t.Logf("real policy IDs: %v", policyIDs)
	hid := func(id int) int {
		return int(hostIDs[id-1]) //nolint:gosec // dismiss G115
	}
	pid := func(id int) int {
		if id < 0 || id >= len(policyIDs) {
			return id
		}
		return int(policyIDs[id-1]) //nolint:gosec // dismiss G115
	}

	nbTrue := sql.NullBool{Valid: true, Bool: true}
	nbFalse := sql.NullBool{Valid: true, Bool: false}
	nbNull := sql.NullBool{Valid: false}

	// note that cases cannot be run in isolation, each case builds on the
	// previous one's state, so they are not run as distinct sub-tests.
	cases := []struct {
		name string
		// map of host ID to policy IDs to insert with passes set to the bool.
		reported map[int]map[int]*bool
		want     []policyMembership
	}{
		{"no key", nil, nil},
		{
			"report host 1 policy 1",
			map[int]map[int]*bool{hid(1): {pid(1): ptr.Bool(true)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
			},
		},
		{
			"report host 1 policies 1, 2",
			map[int]map[int]*bool{hid(1): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbTrue},
			},
		},
		{
			"report host 1 policies 1, 2, 3",
			map[int]map[int]*bool{hid(1): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true), pid(3): ptr.Bool(true)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbTrue},
			},
		},
		{
			"report host 1 policy -1",
			map[int]map[int]*bool{hid(1): {pid(1): ptr.Bool(false)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbTrue},
			},
		},
		{
			"report host 1 policies -2, -3",
			map[int]map[int]*bool{hid(1): {pid(2): ptr.Bool(false), pid(3): ptr.Bool(false)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbFalse},
			},
		},
		{
			"report host 1 policies 1, 2, (3), 4",
			map[int]map[int]*bool{hid(1): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true), pid(3): nil, pid(4): ptr.Bool(true)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbNull},
				{HostID: hid(1), PolicyID: pid(4), Passes: nbTrue},
			},
		},
		{
			"report host 1 policies -2, -3, -4, 1",
			map[int]map[int]*bool{hid(1): {pid(2): ptr.Bool(false), pid(3): ptr.Bool(false), pid(4): ptr.Bool(false), pid(1): ptr.Bool(true)}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(4), Passes: nbFalse},
			},
		},
		{
			"report host 1 policy 2, host 2 policies 2, 3",
			map[int]map[int]*bool{hid(1): {pid(2): ptr.Bool(true)}, hid(2): {pid(2): ptr.Bool(true), pid(3): nil}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(4), Passes: nbFalse},
				{HostID: hid(2), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(2), PolicyID: pid(3), Passes: nbNull},
			},
		},
		{
			"report hosts 1, 2, 3, 4 policies 1, 2, -3, (4)",
			map[int]map[int]*bool{
				hid(1): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true), pid(3): ptr.Bool(false), pid(4): nil},
				hid(2): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true), pid(3): ptr.Bool(false), pid(4): nil},
				hid(3): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true), pid(3): ptr.Bool(false), pid(4): nil},
				hid(4): {pid(1): ptr.Bool(true), pid(2): ptr.Bool(true), pid(3): ptr.Bool(false), pid(4): nil},
			},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(1), PolicyID: pid(3), Passes: nbFalse},
				{HostID: hid(1), PolicyID: pid(4), Passes: nbNull},
				{HostID: hid(2), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(2), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(2), PolicyID: pid(3), Passes: nbFalse},
				{HostID: hid(2), PolicyID: pid(4), Passes: nbNull},
				{HostID: hid(3), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(3), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(3), PolicyID: pid(3), Passes: nbFalse},
				{HostID: hid(3), PolicyID: pid(4), Passes: nbNull},
				{HostID: hid(4), PolicyID: pid(1), Passes: nbTrue},
				{HostID: hid(4), PolicyID: pid(2), Passes: nbTrue},
				{HostID: hid(4), PolicyID: pid(3), Passes: nbFalse},
				{HostID: hid(4), PolicyID: pid(4), Passes: nbNull},
			},
		},
	}

	const batchSizes = 3

	setupTest := func(t *testing.T, data map[int]map[int]*bool) collectorExecStats {
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		// store the host memberships and prepare the expected stats
		var wantStats collectorExecStats
		for hostID, res := range data {
			if len(res) > 0 {
				key := fmt.Sprintf(policyPassHostKey, hostID)
				args := make(redigo.Args, 0, 1+(len(res)))
				args = args.Add(key)
				for polID, pass := range res {
					score := 0
					if pass != nil {
						if *pass {
							score = 1
						} else {
							score = -1
						}
					}
					args = args.Add(fmt.Sprintf("%d=%d", polID, score))
				}
				_, err := conn.Do("LPUSH", args...)
				require.NoError(t, err)
				_, err = conn.Do("ZADD", policyPassHostIDsKey, time.Now().Unix(), hostID)
				require.NoError(t, err)
			}
			cnt, err := redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
			require.NoError(t, err)
			wantStats.Keys = cnt
			wantStats.RedisCmds++
			wantStats.Items += len(res)
		}
		return wantStats
	}

	selectRows := func(t *testing.T) ([]policyMembership, map[int]time.Time) {
		var rows []policyMembership
		mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, policy_id, passes, updated_at
        FROM policy_membership
        ORDER BY host_id, policy_id`)
		})

		var hosts []struct {
			ID              int       `db:"id"`
			PolicyUpdatedAt time.Time `db:"policy_updated_at"`
		}
		mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &hosts, `SELECT id, policy_updated_at FROM hosts`)
		})

		hostsUpdated := make(map[int]time.Time, len(hosts))
		for _, h := range hosts {
			hostsUpdated[h.ID] = h.PolicyUpdatedAt
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
			task := NewTask(nil, nil, clock.C, config.OsqueryConfig{
				AsyncHostInsertBatch:        batchSizes,
				AsyncHostUpdateBatch:        batchSizes,
				AsyncHostDeleteBatch:        batchSizes,
				AsyncHostRedisPopCount:      batchSizes,
				AsyncHostRedisScanKeysCount: 10,
			})
			err := task.collectPolicyQueryExecutions(ctx, ds, pool, &stats)
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
				require.Equal(t, want.HostID, got.HostID, "[%d] host id", i)
				require.Equal(t, want.PolicyID, got.PolicyID, "[%d] policy id", i)
				require.Equal(t, want.Passes, got.Passes, "[%d] passes", i)
				require.WithinDuration(t, minUpdatedAt, got.UpdatedAt, 10*time.Second, "[%d] membership updated at", i)

				ts, ok := hostsUpdated[want.HostID]
				require.True(t, ok)
				require.WithinDuration(t, minUpdatedAt, ts, 10*time.Second, "[%d] host updated at", i)
			}
		}()
	}

	// after all cases, run one last upsert (an update) to make sure that the
	// updated at column is properly updated. First we need to ensure that this
	// runs in a distinct second, because the mysql resolution is not precise.
	time.Sleep(time.Second)

	var h1p1Before policyMembership
	beforeRows, _ := selectRows(t)
	for _, row := range beforeRows {
		if row.HostID == 1 && row.PolicyID == 1 {
			h1p1Before = row
			break
		}
	}

	// update host 1, policy 1, already existing
	setupTest(t, map[int]map[int]*bool{1: {1: nil}})
	var stats collectorExecStats
	task := NewTask(nil, nil, clock.C, config.OsqueryConfig{
		AsyncHostInsertBatch:        batchSizes,
		AsyncHostUpdateBatch:        batchSizes,
		AsyncHostDeleteBatch:        batchSizes,
		AsyncHostRedisPopCount:      batchSizes,
		AsyncHostRedisScanKeysCount: 10,
	})
	err := task.collectPolicyQueryExecutions(ctx, ds, pool, &stats)
	require.NoError(t, err)

	var h1p1After policyMembership
	afterRows, _ := selectRows(t)
	for _, row := range afterRows {
		if row.HostID == 1 && row.PolicyID == 1 {
			h1p1After = row
			break
		}
	}
	require.True(t, h1p1Before.UpdatedAt.Before(h1p1After.UpdatedAt))
}

func testRecordPolicyQueryExecutionsSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)
	host := &fleet.Host{
		ID:              1,
		Platform:        "linux",
		PolicyUpdatedAt: lastYear,
	}

	yes, no := true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keyList, keyTs := fmt.Sprintf(policyPassHostKey, host.ID), fmt.Sprintf(policyPassReportedKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{})

	policyReportedAt := task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(lastYear))

	err := task.RecordPolicyQueryExecutions(ctx, host, results, now, false)
	require.NoError(t, err)
	require.True(t, ds.RecordPolicyQueryExecutionsFuncInvoked)
	ds.RecordPolicyQueryExecutionsFuncInvoked = false

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", keyList, keyTs) //nolint:errcheck

	n, err := redigo.Int(conn.Do("EXISTS", keyList))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("EXISTS", keyTs))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	policyReportedAt = task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(now))
}

func testRecordPolicyQueryExecutionsAsync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)
	host := &fleet.Host{
		ID:              1,
		Platform:        "linux",
		PolicyUpdatedAt: lastYear,
	}
	yes, no := true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keyList, keyTs := fmt.Sprintf(policyPassHostKey, host.ID), fmt.Sprintf(policyPassReportedKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{
		EnableAsyncHostProcessing:   "true",
		AsyncHostInsertBatch:        3,
		AsyncHostUpdateBatch:        3,
		AsyncHostDeleteBatch:        3,
		AsyncHostRedisPopCount:      3,
		AsyncHostRedisScanKeysCount: 10,
	})

	policyReportedAt := task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(lastYear))

	err := task.RecordPolicyQueryExecutions(ctx, host, results, now, false)
	require.NoError(t, err)
	require.False(t, ds.RecordPolicyQueryExecutionsFuncInvoked)

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", keyList, keyTs) //nolint:errcheck

	res, err := redigo.Strings(conn.Do("LRANGE", keyList, 0, -1))
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.ElementsMatch(t, []string{"1=1", "2=1", "3=-1", "4=0"}, res)

	ts, err := redigo.Int64(conn.Do("GET", keyTs))
	require.NoError(t, err)
	require.Equal(t, now.Unix(), ts)

	count, err := redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 1, count)
	tsActive, err := redigo.Int64(conn.Do("ZSCORE", policyPassHostIDsKey, host.ID))
	require.NoError(t, err)
	require.Equal(t, tsActive, ts)

	policyReportedAt = task.GetHostPolicyReportedAt(ctx, host)
	// because we transition via unix epoch (seconds), not exactly equal
	require.WithinDuration(t, now, policyReportedAt, time.Second)
	// host's PolicyUpdatedAt field hasn't been updated yet, because the label
	// results are in redis, not in mysql yet.
	require.True(t, host.PolicyUpdatedAt.Equal(lastYear))

	// running the collector removes the host from the active set
	var stats collectorExecStats
	err = task.collectPolicyQueryExecutions(ctx, ds, pool, &stats)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Keys)
	require.Equal(t, 4, stats.Items)
	require.False(t, stats.Failed)

	count, err = redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testRecordPolicyQueryExecutionsNoPoliciesSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)
	host := &fleet.Host{
		ID:              1,
		Platform:        "linux",
		PolicyUpdatedAt: lastYear,
	}

	var emptyResults map[uint]*bool
	keyList, keyTs := fmt.Sprintf(policyPassHostKey, host.ID), fmt.Sprintf(policyPassReportedKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{})

	policyReportedAt := task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(lastYear))

	err := task.RecordPolicyQueryExecutions(ctx, host, emptyResults, now, false)
	require.NoError(t, err)
	require.True(t, ds.RecordPolicyQueryExecutionsFuncInvoked)
	ds.RecordPolicyQueryExecutionsFuncInvoked = false

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()

	n, err := redigo.Int(conn.Do("EXISTS", keyList))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("EXISTS", keyTs))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	policyReportedAt = task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(now))
}

func testRecordPolicyQueryExecutionsNoPoliciesAsync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	now := time.Now()
	lastYear := now.Add(-365 * 24 * time.Hour)
	host := &fleet.Host{
		ID:              1,
		Platform:        "linux",
		PolicyUpdatedAt: lastYear,
	}
	var emptyResults map[uint]*bool
	keyList, keyTs := fmt.Sprintf(policyPassHostKey, host.ID), fmt.Sprintf(policyPassReportedKey, host.ID)

	task := NewTask(ds, pool, clock.C, config.OsqueryConfig{
		EnableAsyncHostProcessing:   "true",
		AsyncHostInsertBatch:        3,
		AsyncHostUpdateBatch:        3,
		AsyncHostDeleteBatch:        3,
		AsyncHostRedisPopCount:      3,
		AsyncHostRedisScanKeysCount: 10,
	})

	policyReportedAt := task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(lastYear))

	err := task.RecordPolicyQueryExecutions(ctx, host, emptyResults, now, false)
	require.NoError(t, err)
	require.False(t, ds.RecordPolicyQueryExecutionsFuncInvoked)

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()
	defer conn.Do("DEL", keyTs) //nolint:errcheck

	n, err := redigo.Int(conn.Do("EXISTS", keyList))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	ts, err := redigo.Int64(conn.Do("GET", keyTs))
	require.NoError(t, err)
	require.Equal(t, now.Unix(), ts)

	count, err := redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 1, count)
	tsActive, err := redigo.Int64(conn.Do("ZSCORE", policyPassHostIDsKey, host.ID))
	require.NoError(t, err)
	require.Equal(t, tsActive, ts)

	policyReportedAt = task.GetHostPolicyReportedAt(ctx, host)
	// because we transition via unix epoch (seconds), not exactly equal
	require.WithinDuration(t, now, policyReportedAt, time.Second)
	// host's PolicyUpdatedAt field hasn't been updated yet, because the policy
	// results are in redis, not in mysql yet.
	require.True(t, host.PolicyUpdatedAt.Equal(lastYear))

	// running the collector removes the host from the active set
	var stats collectorExecStats
	err = task.collectPolicyQueryExecutions(ctx, ds, pool, &stats)
	require.NoError(t, err)
	require.Equal(t, 1, stats.Keys)
	require.Equal(t, 0, stats.Items)
	require.False(t, stats.Failed)

	count, err = redigo.Int(conn.Do("ZCARD", policyPassHostIDsKey))
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func createPolicies(t *testing.T, ds *mysql.Datastore, count int) []uint {
	ctx := context.Background()

	ids := make([]uint, count)
	mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		for i := 0; i < count; i++ {
			res, err := tx.ExecContext(
				ctx, `INSERT INTO policies (name, description, query, checksum) VALUES (?, ?, ?, ?)`,
				fmt.Sprintf("%s-%d", t.Name(), i), t.Name(), "SELECT 1", strconv.Itoa(i),
			)
			if err != nil {
				return err
			}
			pid, _ := res.LastInsertId()
			ids[i] = uint(pid) //nolint:gosec // dismiss G115
		}
		return nil
	})
	return ids
}
