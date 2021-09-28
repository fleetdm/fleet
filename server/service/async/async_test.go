package async

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestRecordLabelQueryExecutions(t *testing.T) {
	ds := new(mock.Store)
	ds.RecordLabelQueryExecutionsFunc = func(ctx context.Context, host *fleet.Host, results map[uint]*bool, ts time.Time) error {
		return nil
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redis.SetupRedis(t, false, false)
		t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redis.SetupRedis(t, true, false)
		t.Run("sync", func(t *testing.T) { testRecordLabelQueryExecutionsSync(t, ds, pool) })
		t.Run("async", func(t *testing.T) { testRecordLabelQueryExecutionsAsync(t, ds, pool) })
	})
}

func testRecordLabelQueryExecutionsSync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	host := &fleet.Host{
		ID:       1,
		Platform: "linux",
	}
	now := time.Now()
	var yes, no = true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keySet, keyTs := fmt.Sprintf(labelMembershipHostKey, host.ID), fmt.Sprintf(labelMembershipReportedKey, host.ID)

	task := Task{
		Datastore:    ds,
		Pool:         pool,
		AsyncEnabled: false,
	}
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
}

func testRecordLabelQueryExecutionsAsync(t *testing.T, ds *mock.Store, pool fleet.RedisPool) {
	ctx := context.Background()
	host := &fleet.Host{
		ID:       1,
		Platform: "linux",
	}
	now := time.Now()
	var yes, no = true, false
	results := map[uint]*bool{1: &yes, 2: &yes, 3: &no, 4: nil}
	keySet, keyTs := fmt.Sprintf(labelMembershipHostKey, host.ID), fmt.Sprintf(labelMembershipReportedKey, host.ID)

	task := Task{
		Datastore:    ds,
		Pool:         pool,
		AsyncEnabled: true,
	}
	err := task.RecordLabelQueryExecutions(ctx, host, results, now)
	require.NoError(t, err)
	require.False(t, ds.RecordLabelQueryExecutionsFuncInvoked)
	ds.RecordLabelQueryExecutionsFuncInvoked = false

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
}
