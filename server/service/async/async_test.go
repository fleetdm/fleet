package async

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestCollectQueryExecutions(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	oldMaxPolicy := maxRedisPolicyResultsPerHost
	maxRedisPolicyResultsPerHost = 3
	t.Cleanup(func() {
		maxRedisPolicyResultsPerHost = oldMaxPolicy
	})

	t.Run("Label", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, false, false)
			testCollectLabelQueryExecutions(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, true, true)
			testCollectLabelQueryExecutions(t, ds, pool)
		})
	})

	t.Run("Policy", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, false, false)
			testCollectPolicyQueryExecutions(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, true, true)
			testCollectPolicyQueryExecutions(t, ds, pool)
		})
	})
}

func testCollectLabelQueryExecutions(t *testing.T, ds fleet.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	type labelMembership struct {
		HostID    int       `db:"host_id"`
		LabelID   uint      `db:"label_id"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	hostIDs := createHosts(t, ds, 4, time.Now().Add(-24*time.Hour))
	t.Logf("real host IDs: %v", hostIDs)
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
		conn := pool.ConfigureDoer(pool.Get())
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
		err := ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, label_id, updated_at FROM label_membership ORDER BY 1, 2`)
		})
		require.NoError(t, err)

		var hosts []struct {
			ID             int       `db:"id"`
			LabelUpdatedAt time.Time `db:"label_updated_at"`
		}
		err = ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &hosts, `SELECT id, label_updated_at FROM hosts`)
		})
		require.NoError(t, err)

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

func testCollectPolicyQueryExecutions(t *testing.T, ds fleet.Datastore, pool fleet.RedisPool) {
	ctx := context.Background()

	type policyMembership struct {
		HostID    int       `db:"host_id"`
		PolicyID  int       `db:"policy_id"`
		Passes    bool      `db:"passes"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	hostIDs := createHosts(t, ds, 4, time.Now().Add(-24*time.Hour))
	policyIDs := createPolicies(t, ds, 4)
	t.Logf("real host IDs: %v", hostIDs)
	t.Logf("real policy IDs: %v", policyIDs)
	hid := func(id int) int {
		if id < 0 || id >= len(hostIDs) {
			return id
		}
		return int(hostIDs[id-1])
	}
	pid := func(id int) int {
		if id < 0 || id >= len(policyIDs) {
			return id
		}
		return int(policyIDs[id-1])
	}

	// note that cases cannot be run in isolation, each case builds on the
	// previous one's state, so they are not run as distinct sub-tests.
	cases := []struct {
		name string
		// map of host ID to policy IDs to insert with passes set to the bool.
		// negative host id is stored as an invalid redis key that should be
		// ignored.
		reported map[int]map[int]bool
		want     []policyMembership
	}{
		{"no key", nil, nil},
		{
			"report host 1 policy 1",
			map[int]map[int]bool{hid(1): {pid(1): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
			},
		},
		{
			"report host 1 policies 1, 2",
			map[int]map[int]bool{hid(1): {pid(1): true, pid(2): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
			},
		},
		{
			"report host 1 policies 1, 2, 3",
			map[int]map[int]bool{hid(1): {pid(1): true, pid(2): true, pid(3): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
			},
		},
		{
			"report host 1 policy -1",
			map[int]map[int]bool{hid(1): {pid(1): false}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
			},
		},
		{
			"report host 1 policies -2, -3",
			map[int]map[int]bool{hid(1): {pid(2): false, pid(3): false}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
			},
		},
		{
			"report host 1 policies 1, 2, 3, 4",
			map[int]map[int]bool{hid(1): {pid(1): true, pid(2): true, pid(3): true, pid(4): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(4), Passes: true},
			},
		},
		{
			"report host 1 policies -2, -3, -4, 1",
			map[int]map[int]bool{hid(1): {pid(2): false, pid(3): false, pid(4): false, pid(1): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(4), Passes: true},
				{HostID: hid(1), PolicyID: pid(4), Passes: false},
			},
		},
		{
			"report host 1 policy 2, host 2 policies 2, 3",
			map[int]map[int]bool{hid(1): {pid(2): true}, hid(2): {pid(2): true, pid(3): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(4), Passes: true},
				{HostID: hid(1), PolicyID: pid(4), Passes: false},
				{HostID: hid(2), PolicyID: pid(2), Passes: true},
				{HostID: hid(2), PolicyID: pid(3), Passes: true},
			},
		},
		{
			"report host -99 policy 1, ignored",
			map[int]map[int]bool{hid(-99): {pid(1): true}},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(4), Passes: true},
				{HostID: hid(1), PolicyID: pid(4), Passes: false},
				{HostID: hid(2), PolicyID: pid(2), Passes: true},
				{HostID: hid(2), PolicyID: pid(3), Passes: true},
			},
		},
		{
			"report hosts 1, 2, 3, 4, -99 policies 1, 2, -3, 4",
			map[int]map[int]bool{
				hid(1):   {pid(1): true, pid(2): true, pid(3): false, pid(4): true},
				hid(2):   {pid(1): true, pid(2): true, pid(3): false, pid(4): true},
				hid(3):   {pid(1): true, pid(2): true, pid(3): false, pid(4): true},
				hid(4):   {pid(1): true, pid(2): true, pid(3): false, pid(4): true},
				hid(-99): {pid(1): true, pid(2): true, pid(3): false, pid(4): true},
			},
			[]policyMembership{
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: false},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(1), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: false},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(2), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: true},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(3), Passes: false},
				{HostID: hid(1), PolicyID: pid(4), Passes: true},
				{HostID: hid(1), PolicyID: pid(4), Passes: false},
				{HostID: hid(1), PolicyID: pid(4), Passes: true},
				{HostID: hid(2), PolicyID: pid(1), Passes: true},
				{HostID: hid(2), PolicyID: pid(2), Passes: true},
				{HostID: hid(2), PolicyID: pid(2), Passes: true},
				{HostID: hid(2), PolicyID: pid(3), Passes: true},
				{HostID: hid(2), PolicyID: pid(3), Passes: false},
				{HostID: hid(2), PolicyID: pid(4), Passes: true},
				{HostID: hid(3), PolicyID: pid(1), Passes: true},
				{HostID: hid(3), PolicyID: pid(2), Passes: true},
				{HostID: hid(3), PolicyID: pid(3), Passes: false},
				{HostID: hid(3), PolicyID: pid(4), Passes: true},
				{HostID: hid(4), PolicyID: pid(1), Passes: true},
				{HostID: hid(4), PolicyID: pid(2), Passes: true},
				{HostID: hid(4), PolicyID: pid(3), Passes: false},
				{HostID: hid(4), PolicyID: pid(4), Passes: true},
			},
		},
	}

	const batchSizes = 3

	setupTest := func(t *testing.T, data map[int]map[int]bool) collectorExecStats {
		conn := pool.ConfigureDoer(pool.Get())
		defer conn.Close()

		// store the host memberships and prepare the expected stats
		var wantStats collectorExecStats
		for hostID, res := range data {
			if len(res) > 0 {
				key := fmt.Sprintf(policyPassHostKey, hostID)
				args := make(redigo.Args, 0, 1+(len(res)))
				args = args.Add(key)
				for polID, pass := range res {
					score := -1
					if pass {
						score = 1
					}
					args = args.Add(fmt.Sprintf("%d=%d", polID, score))
				}
				_, err := conn.Do("LPUSH", args...)
				require.NoError(t, err)
			}
			wantStats.Keys++
			if hostID >= 0 {
				wantStats.RedisCmds++
				wantStats.Items += len(res)
			}
		}
		return wantStats
	}

	selectRows := func(t *testing.T) ([]policyMembership, map[int]time.Time) {
		var rows []policyMembership
		err := ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &rows, `SELECT host_id, policy_id, passes, updated_at
        FROM policy_membership_history
        ORDER BY host_id, policy_id, id`)
		})
		require.NoError(t, err)

		var hosts []struct {
			ID              int       `db:"id"`
			PolicyUpdatedAt time.Time `db:"policy_updated_at"`
		}
		err = ds.AdhocRetryTx(ctx, func(tx sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, tx, &hosts, `SELECT id, policy_updated_at FROM hosts`)
		})
		require.NoError(t, err)

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
			task := Task{
				InsertBatch:        batchSizes,
				UpdateBatch:        batchSizes,
				DeleteBatch:        batchSizes,
				RedisPopCount:      batchSizes,
				RedisScanKeysCount: 10,
			}
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
}

func TestRecordPolicyQueryExecutions(t *testing.T) {
	ds := new(mock.Store)
	ds.RecordPolicyQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
		return nil
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false)
		t.Run("sync", func(t *testing.T) { testRecordPolicyQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordPolicyQueryExecutionsAsync(t, ds, pool) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, true)
		t.Run("sync", func(t *testing.T) { testRecordPolicyQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordPolicyQueryExecutionsAsync(t, ds, pool) })
	})
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

	var yes, no = true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keyList, keyTs := fmt.Sprintf(policyPassHostKey, host.ID), fmt.Sprintf(policyPassReportedKey, host.ID)

	task := Task{
		Datastore:    ds,
		Pool:         pool,
		AsyncEnabled: false,
	}

	policyReportedAt := task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(lastYear))

	err := task.RecordPolicyQueryExecutions(ctx, host, results, now)
	require.NoError(t, err)
	require.True(t, ds.RecordPolicyQueryExecutionsFuncInvoked)
	ds.RecordPolicyQueryExecutionsFuncInvoked = false

	conn := pool.Get()
	defer conn.Close()
	defer conn.Do("DEL", keyList, keyTs)

	n, err := redigo.Int(conn.Do("EXISTS", keyList))
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = redigo.Int(conn.Do("EXISTS", keyTs))
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
	var yes, no = true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keyList, keyTs := fmt.Sprintf(policyPassHostKey, host.ID), fmt.Sprintf(policyPassReportedKey, host.ID)

	task := Task{
		Datastore:    ds,
		Pool:         pool,
		AsyncEnabled: true,
	}

	policyReportedAt := task.GetHostPolicyReportedAt(ctx, host)
	require.True(t, policyReportedAt.Equal(lastYear))

	err := task.RecordPolicyQueryExecutions(ctx, host, results, now)
	require.NoError(t, err)
	require.False(t, ds.RecordPolicyQueryExecutionsFuncInvoked)

	conn := pool.Get()
	defer conn.Close()
	defer conn.Do("DEL", keyList, keyTs)

	res, err := redigo.Strings(conn.Do("LRANGE", keyList, 0, -1))
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.ElementsMatch(t, []string{"1=1", "2=1", "3=-1", "4=-1"}, res)

	ts, err := redigo.Int64(conn.Do("GET", keyTs))
	require.NoError(t, err)
	require.Equal(t, now.Unix(), ts)

	policyReportedAt = task.GetHostPolicyReportedAt(ctx, host)
	// because we transition via unix epoch (seconds), not exactly equal
	require.WithinDuration(t, now, policyReportedAt, time.Second)
	// host's PolicyUpdatedAt field hasn't been updated yet, because the label
	// results are in redis, not in mysql yet.
	require.True(t, host.PolicyUpdatedAt.Equal(lastYear))
}

func TestRecordLabelQueryExecutions(t *testing.T) {
	ds := new(mock.Store)
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
		return nil
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false)
		t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, true)
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

func createPolicies(t *testing.T, ds fleet.Datastore, count int) []uint {
	ctx := context.Background()

	ids := make([]uint, count)
	err := ds.AdhocRetryTx(context.Background(), func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO queries (name, description, query) VALUES (?, ?, ?)`,
			t.Name(),
			t.Name(),
			"SELECT 1",
		)
		if err != nil {
			return err
		}
		qid, _ := res.LastInsertId()
		for i := 0; i < count; i++ {
			res, err := tx.ExecContext(ctx, `INSERT INTO policies (query_id) VALUES (?)`, qid)
			if err != nil {
				return err
			}
			pid, _ := res.LastInsertId()
			ids[i] = uint(pid)
		}
		return nil
	})
	require.NoError(t, err)
	return ids
}
