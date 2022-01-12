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
			pool := redistest.SetupRedis(t, false, false, false)
			testCollectLabelQueryExecutions(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, true, true, false)
			testCollectLabelQueryExecutions(t, ds, pool)
		})
	})

	t.Run("Policy", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, false, false, false)
			testCollectPolicyQueryExecutions(t, ds, pool)
		})

		t.Run("cluster", func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			pool := redistest.SetupRedis(t, true, true, false)
			testCollectPolicyQueryExecutions(t, ds, pool)
		})
	})
}

func TestRecordQueryExecutions(t *testing.T) {
	ds := new(mock.Store)
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time, deferred bool) error {
		return nil
	}
	ds.AsyncBatchUpdateLabelTimestampFunc = func(ctx context.Context, ids []uint, ts time.Time) error {
		return nil
	}

	t.Run("Label", func(t *testing.T) {
		t.Run("standalone", func(t *testing.T) {
			pool := redistest.SetupRedis(t, false, false, false)
			t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
		})

		t.Run("cluster", func(t *testing.T) {
			pool := redistest.SetupRedis(t, true, true, false)
			t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
			t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
		})
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
