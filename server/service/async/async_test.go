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
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)

	oldMaxPolicy := maxRedisPolicyResultsPerHost
	maxRedisPolicyResultsPerHost = 3
	t.Cleanup(func() {
		maxRedisPolicyResultsPerHost = oldMaxPolicy
	})

	t.Run("Label", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "label_membership", false, false, false)
			testCollectLabelQueryExecutions(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "label_membership", true, true, false)
			testCollectLabelQueryExecutions(t, ds, pool)
		})
	})

	t.Run("Policy", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "policy_pass", false, false, false)
			testCollectPolicyQueryExecutions(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "policy_pass", true, true, false)
			testCollectPolicyQueryExecutions(t, ds, pool)
		})
	})

	t.Run("Host Last Seen", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "host_last_seen", false, false, false)
			testCollectHostsLastSeen(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "host_last_seen", true, true, false)
			testCollectHostsLastSeen(t, ds, pool)
		})
	})

	t.Run("Scheduled Query Stats", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "scheduled_query_stats", false, false, false)
			testCollectScheduledQueryStats(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, "scheduled_query_stats", true, true, false)
			testCollectScheduledQueryStats(t, ds, pool)
		})
	})
}

func TestRecord(t *testing.T) {
	ds := new(mock.Store)
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
		return nil
	}
	ds.AsyncBatchUpdateLabelTimestampFunc = func(ctx context.Context, ids []uint, ts time.Time) error {
		return nil
	}
	ds.RecordPolicyQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
		return nil
	}
	ds.AsyncBatchInsertPolicyMembershipFunc = func(ctx context.Context, batch []fleet.PolicyMembershipResult) error {
		return nil
	}
	ds.AsyncBatchUpdatePolicyTimestampFunc = func(ctx context.Context, ids []uint, ts time.Time) error {
		return nil
	}
	ds.SaveHostPackStatsFunc = func(ctx context.Context, teamID *uint, hid uint, stats []fleet.PackStats) error {
		return nil
	}
	ds.AsyncBatchSaveHostsScheduledQueryStatsFunc = func(ctx context.Context, batch map[uint][]fleet.ScheduledQueryStats, batchSize int) (int, error) {
		return 1, nil
	}
	ds.ScheduledQueryIDsByNameFunc = func(ctx context.Context, batchSize int, names ...[2]string) ([]uint, error) {
		return make([]uint, len(names)), nil
	}

	t.Run("Label", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "label_membership", false, false, false)
			t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
		})

		t.Run("cluster", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "label_membership", true, true, false)
			t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
		})
	})

	t.Run("Policy", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "policy_pass", false, false, false)
			t.Run("sync", func(t *testing.T) { testRecordPolicyQueryExecutionsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordPolicyQueryExecutionsAsync(t, ds, pool) })
			t.Run("sync", func(t *testing.T) { testRecordPolicyQueryExecutionsNoPoliciesSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordPolicyQueryExecutionsNoPoliciesAsync(t, ds, pool) })
		})

		t.Run("cluster", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "policy_pass", true, true, false)
			t.Run("sync", func(t *testing.T) { testRecordPolicyQueryExecutionsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordPolicyQueryExecutionsAsync(t, ds, pool) })
			t.Run("sync", func(t *testing.T) { testRecordPolicyQueryExecutionsNoPoliciesSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordPolicyQueryExecutionsNoPoliciesAsync(t, ds, pool) })
		})
	})

	t.Run("Host Last Seen", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "host_last_seen", false, false, false)
			t.Run("sync", func(t *testing.T) { testRecordHostLastSeenSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordHostLastSeenAsync(t, ds, pool) })
		})

		t.Run("cluster", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "host_last_seen", true, true, false)
			t.Run("sync", func(t *testing.T) { testRecordHostLastSeenSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordHostLastSeenAsync(t, ds, pool) })
		})
	})

	t.Run("Scheduled Query Stats", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "scheduled_query_stats", false, false, false)
			t.Run("sync", func(t *testing.T) { testRecordScheduledQueryStatsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordScheduledQueryStatsAsync(t, ds, pool) })
		})

		t.Run("cluster", func(t *testing.T) {
			pool := redistest.SetupRedis(t, "scheduled_query_stats", true, true, false)
			t.Run("sync", func(t *testing.T) { testRecordScheduledQueryStatsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordScheduledQueryStatsAsync(t, ds, pool) })
		})
	})
}

func TestActiveHostIDsSet(t *testing.T) {
	const zkey = "testActiveHostIDsSet"

	runTest := func(t *testing.T, pool fleet.RedisPool) {
		activeHosts, err := loadActiveHostIDs(pool, zkey, 10)
		require.NoError(t, err)
		require.Len(t, activeHosts, 0)

		// add a few hosts with a timestamp that increases by a second for each
		// note that host IDs will be 1..10 (t[0] == host 1, t[1] == host 2, etc.)
		tpurgeNone := time.Now()
		ts := make([]int64, 10)
		for i := range ts {
			if i > 0 {
				ts[i] = time.Unix(ts[i-1], 0).Add(time.Second).Unix()
			} else {
				ts[i] = tpurgeNone.Add(time.Second).Unix()
			}

			// none ever get deleted, all are after tpurgeNone
			n, err := storePurgeActiveHostID(pool, zkey, uint(i+1), time.Unix(ts[i], 0), tpurgeNone) //nolint:gosec // dismiss G115
			require.NoError(t, err)
			require.Equal(t, 0, n)
		}

		activeHosts, err = loadActiveHostIDs(pool, zkey, 10)
		require.NoError(t, err)
		require.Len(t, activeHosts, len(ts))
		for i, host := range activeHosts {
			require.Equal(t, ts[i], host.LastReported)
		}

		// store a new one but now use t[1] as purge date - will remove two
		ts2 := ts
		ts2 = append(ts2, time.Unix(ts[len(ts)-1], 0).Add(time.Second).Unix())
		n, err := storePurgeActiveHostID(pool, zkey, uint(len(ts2)), time.Unix(ts2[len(ts2)-1], 0), time.Unix(ts2[1], 0))
		require.NoError(t, err)
		require.Equal(t, 2, n)

		// report t[3] and t[5] (hosts 4 and 6) as processed
		batch := []hostIDLastReported{
			{HostID: 4, LastReported: ts2[3]},
			{HostID: 6, LastReported: ts2[5]},
		}
		n, err = removeProcessedHostIDs(pool, zkey, batch)
		require.NoError(t, err)
		require.Equal(t, 2, n)

		// update t[6] of host 7, as if it had reported new data since the load
		newT6 := time.Unix(ts2[len(ts2)-1], 0).Add(time.Second)
		n, err = storePurgeActiveHostID(pool, zkey, 7, newT6, tpurgeNone)
		require.NoError(t, err)
		require.Equal(t, 0, n)

		// report t[6] and t[7] (hosts 7 and 8) as processed, but only host 8
		// will get deleted, because the timestamp of host 7 has changed (we pass
		// its old timestamp, to simluate that it changed since loading the
		// information)
		batch = []hostIDLastReported{
			{HostID: 7, LastReported: ts2[6]},
			{HostID: 8, LastReported: ts2[7]},
		}
		n, err = removeProcessedHostIDs(pool, zkey, batch)
		require.NoError(t, err)
		require.Equal(t, 1, n)

		// check the remaining active hosts (only 6 remain)
		activeHosts, err = loadActiveHostIDs(pool, zkey, 10)
		require.NoError(t, err)
		require.Len(t, activeHosts, 6)
		want := []hostIDLastReported{
			{HostID: 3, LastReported: ts2[2]},
			{HostID: 5, LastReported: ts2[4]},
			{HostID: 7, LastReported: newT6.Unix()},
			{HostID: 9, LastReported: ts2[8]},
			{HostID: 10, LastReported: ts2[9]},
			{HostID: 11, LastReported: ts2[10]},
		}
		require.ElementsMatch(t, want, activeHosts)
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, zkey, false, false, false)
		t.Run("sync", func(t *testing.T) { runTest(t, pool) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, zkey, true, true, false)
		t.Run("sync", func(t *testing.T) { runTest(t, pool) })
	})
}

func createHosts(t *testing.T, ds fleet.Datastore, count int, ts time.Time) []uint {
	ids := make([]uint, count)
	for i := 0; i < count; i++ {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: ts,
			LabelUpdatedAt:  ts,
			PolicyUpdatedAt: ts,
			SeenTime:        ts,
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(t, err)
		ids[i] = host.ID
	}
	return ids
}
