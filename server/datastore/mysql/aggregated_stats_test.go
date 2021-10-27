package mysql

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"math"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func slowStats(t *testing.T, ds *Datastore, id uint, percentile int, table, column string) float64 {
	scheduledQueriesSQL := fmt.Sprintf(`SELECT d.%s / d.executions FROM scheduled_query_stats d WHERE d.scheduled_query_id=? ORDER BY (d.%s / d.executions) ASC`, column, column)
	queriesSQL := fmt.Sprintf(`SELECT d.%s / d.executions FROM scheduled_query_stats d JOIN scheduled_queries sq ON (sq.id=d.scheduled_query_id) WHERE sq.query_id=? ORDER BY (d.%s / d.executions) ASC`, column, column)
	queryToRun := scheduledQueriesSQL
	if table == "queries" {
		queryToRun = queriesSQL
	}
	rows, err := ds.writer.Queryx(queryToRun, id)
	require.NoError(t, err)
	defer rows.Close()

	var vals []float64

	for rows.Next() {
		var val float64
		err := rows.Scan(&val)
		require.NoError(t, err)
		vals = append(vals, val)
	}

	if len(vals) == 0 {
		return 0.0
	}

	index := int(math.Floor(float64(len(vals)) * float64(percentile) / 100.0))
	return vals[index]
}

func TestAggregatedStats(t *testing.T) {
	ds := CreateMySQLDS(t)

	var args []interface{}

	batchSize := 4000
	hostCount := 10           // 2000
	scheduledQueryCount := 20 // 400
	queryCount := 30          // 1000

	start := time.Now()
	for i := 0; i < queryCount; i++ {
		_, err := ds.writer.Exec(`INSERT INTO queries(name, query, description) VALUES (?,?,?)`, fmt.Sprint(i), fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	for i := 0; i < scheduledQueryCount; i++ {
		_, err := ds.writer.Exec(`INSERT INTO scheduled_queries(query_id,name,query_name) VALUES (?,?,?)`, rand.Intn(queryCount)+1, fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	insertScheduledQuerySQL := `INSERT IGNORE INTO scheduled_query_stats(host_id, scheduled_query_id, system_time, user_time, executions) VALUES %s`
	scheduledQueryStatsCount := 100 // 1000000
	for i := 0; i < scheduledQueryStatsCount; i++ {
		if len(args) > batchSize {
			values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?),", len(args)/5), ",")
			_, err := ds.writer.Exec(fmt.Sprintf(insertScheduledQuerySQL, values), args...)
			require.NoError(t, err)
			args = []interface{}{}
		}
		args = append(args, rand.Intn(hostCount)+1, rand.Intn(scheduledQueryCount)+1, rand.Intn(10000)+100, rand.Intn(10000)+100, rand.Intn(10000)+100)
	}
	if len(args) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?),", len(args)/5), ",")
		_, err := ds.writer.Exec(fmt.Sprintf(insertScheduledQuerySQL, values), args...)
		require.NoError(t, err)
	}

	// Make sure we have some queries and scheduled queries that don't have stats
	for i := queryCount; i < queryCount+4; i++ {
		_, err := ds.writer.Exec(`INSERT INTO queries(name, query, description) VALUES (?,?,?)`, fmt.Sprint(i), fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	for i := scheduledQueryCount; i < scheduledQueryCount+4; i++ {
		_, err := ds.writer.Exec(`INSERT INTO scheduled_queries(query_id,name,query_name) VALUES (?,?,?)`, rand.Intn(queryCount)+1, fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}

	t.Log("Done inserting dummy data. Took:", time.Since(start))

	testcases := []struct {
		table     string
		aggregate string
		aggFunc   func(ctx context.Context) error
	}{
		{"scheduled_queries", "scheduled_query", ds.UpdateScheduledQueryAggregatedStats},
		{"queries", "query", ds.UpdateQueryAggregatedStats},
	}
	for _, tt := range testcases {
		t.Run(tt.table, func(t *testing.T) {
			start = time.Now()
			require.NoError(t, tt.aggFunc(context.Background()))
			t.Log("Generated stats for ", tt.table, " in:,", time.Since(start))

			var stats []struct {
				ID uint `db:"id"`
				fleet.AggregatedStats
			}
			require.NoError(t,
				ds.writer.Select(&stats,
					`
select 
       id, 
       JSON_EXTRACT(json_value, "$.user_time_p50") as user_time_p50, 
       JSON_EXTRACT(json_value, "$.user_time_p95") as user_time_p95,
       JSON_EXTRACT(json_value, "$.system_time_p50") as system_time_p50, 
       JSON_EXTRACT(json_value, "$.system_time_p95") as system_time_p95,
       JSON_EXTRACT(json_value, "$.total_executions") as total_executions
from aggregated_stats where type=?`, tt.aggregate))

			require.True(t, len(stats) > 0)
			for _, stat := range stats {
				checkAgainstSlowStats(t, ds, stat.ID, 50, tt.table, "user_time", stat.UserTimeP50)
				checkAgainstSlowStats(t, ds, stat.ID, 95, tt.table, "user_time", stat.UserTimeP95)
				checkAgainstSlowStats(t, ds, stat.ID, 50, tt.table, "system_time", stat.SystemTimeP50)
				checkAgainstSlowStats(t, ds, stat.ID, 95, tt.table, "system_time", stat.SystemTimeP95)
				require.NotNil(t, stat.TotalExecutions)
				assert.True(t, *stat.TotalExecutions >= 0)
			}
		})
	}
}

func checkAgainstSlowStats(t *testing.T, ds *Datastore, id uint, percentile int, table, column string, against *float64) {
	slowp := slowStats(t, ds, id, percentile, table, column)
	if against != nil {
		assert.Equal(t, slowp, *against)
	} else {
		assert.Zero(t, slowp)
	}
}
