package mysql

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"math"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func slowStats(t *testing.T, ds *Datastore, id uint, percentile int, table column string) float64 {
	rows, err := ds.writer.Queryx(
		fmt.Sprintf(`SELECT d.%s / d.executions FROM scheduled_query_stats d WHERE d.scheduled_query_id=? ORDER BY (d.%s / d.executions) ASC`, column, column),
		id,
	)
	require.NoError(t, err)
	defer rows.Close()

	var vals []float64

	for rows.Next() {
		var val float64
		err := rows.Scan(&val)
		require.NoError(t, err)
		vals = append(vals, val)
	}

	return vals[int(math.Floor(float64(len(vals))*float64(percentile)/100.0))]
}

func TestAggregatedStats(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	prefixi := func(prefix string, i int) string {
		return fmt.Sprintf("%s%d", prefix, i)
	}
	prefixij := func(prefix string, i, j int) string {
		return fmt.Sprintf("%s%d-%d", prefix, i, j)
	}

	hostCount := 10 //10000
	var hosts []*fleet.Host
	for i := 0; i < hostCount; i++ {
		host, err := ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			SeenTime:        time.Now(),
			PolicyUpdatedAt: time.Now(),
			NodeKey:         prefixi("", i),
			UUID:            prefixi("", i),
			Hostname:        prefixi("foo.local.", i),
			PrimaryIP:       prefixi("192.168.1.", i),
			PrimaryMac:      prefixi("", i),
			OsqueryHostID:   prefixi("", i),
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		hosts = append(hosts, host)
	}

	packCount := 1         //20
	maxQueriesPerPack := 1 //40
	var packsAndSchedQueries []struct {
		pack   *fleet.Pack
		squery *fleet.ScheduledQuery
	}
	for i := 0; i < packCount; i++ {
		pack := test.NewPack(t, ds, prefixi("pack-", i))
		queriesPerPack := rand.Intn(maxQueriesPerPack) + 1
		for j := 0; j < queriesPerPack; j++ {
			query := test.NewQuery(t, ds, prefixij("query-", i, j), "select * from time", 0, true)
			squery := test.NewScheduledQuery(t, ds, pack.ID, query.ID, 30, true, true, prefixij("sched-", i, j))
			packsAndSchedQueries = append(packsAndSchedQueries, struct {
				pack   *fleet.Pack
				squery *fleet.ScheduledQuery
			}{pack: pack, squery: squery})
		}
	}

	statsCount := hostCount * 2
	for i := 0; i < statsCount; i++ {
		randomSelection := packsAndSchedQueries[rand.Intn(len(packsAndSchedQueries))]
		randomHost := hosts[rand.Intn(len(hosts))]
		stats := []fleet.ScheduledQueryStats{
			{
				ScheduledQueryName: randomSelection.squery.Name,
				ScheduledQueryID:   randomSelection.squery.ID,
				QueryName:          randomSelection.squery.Name,
				PackName:           randomSelection.pack.Name,
				PackID:             randomSelection.pack.ID,
				AverageMemory:      rand.Intn(100000) + 1000,
				Denylisted:         false,
				Executions:         rand.Intn(1000),
				Interval:           30,
				OutputSize:         rand.Intn(100000),
				SystemTime:         rand.Intn(10000) + 100,
				UserTime:           rand.Intn(10000) + 100,
				WallTime:           rand.Intn(10000) + 100,
				LastExecuted:       time.Unix(time.Now().Unix()-int64(rand.Intn(500000)), 0).UTC(),
			},
		}
		randomHost.PackStats = []fleet.PackStats{
			{
				PackName:   randomSelection.pack.Name,
				QueryStats: stats,
			},
		}
		require.NoError(t, ds.SaveHost(context.Background(), randomHost))
	}

	require.NoError(t, ds.UpdateScheduledQueryAggregatedStats(context.Background()))

	var stats []struct {
		ID            uint    `db:"id"`
		SystemTimeP50 float64 `db:"system_time_p50"`
		SystemTimeP95 float64 `db:"system_time_p95"`
		UserTimeP50   float64 `db:"user_time_p50"`
		UserTimeP95   float64 `db:"user_time_p95"`
	}
	require.NoError(t,
		ds.writer.Select(&stats,
			`
select 
       id, 
       JSON_EXTRACT(json_value, "$.user_time_p50") as user_time_p50, 
       JSON_EXTRACT(json_value, "$.user_time_p95") as user_time_p95,
       JSON_EXTRACT(json_value, "$.system_time_p50") as system_time_p50, 
       JSON_EXTRACT(json_value, "$.system_time_p95") as system_time_p95 
from aggregated_stats where type="scheduled_query"`))

	require.True(t, len(stats) > 0)
	for _, stat := range stats {
		slowp50 := slowStats(t, ds, stat.ID, 50, "user_time")
		assert.Equal(t, slowp50, stat.UserTimeP50)
		slowp95 := slowStats(t, ds, stat.ID, 95, "user_time")
		assert.Equal(t, slowp95, stat.UserTimeP95)
		slowp50 = slowStats(t, ds, stat.ID, 50, "system_time")
		assert.Equal(t, slowp50, stat.SystemTimeP50)
		slowp95 = slowStats(t, ds, stat.ID, 95, "system_time")
		assert.Equal(t, slowp95, stat.SystemTimeP95)
	}
}
