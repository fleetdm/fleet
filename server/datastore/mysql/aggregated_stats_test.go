package mysql

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func slowStats(t *testing.T, ds *Datastore, id uint, percentile int, column string) float64 {
	queriesSQL := fmt.Sprintf(
		`
		SELECT SUM(d.%[1]s) / SUM(d.executions)
		FROM scheduled_query_stats d
			JOIN queries q ON (d.scheduled_query_id=q.id)
		WHERE q.id=? AND d.executions > 0
		GROUP BY d.host_id
		ORDER BY (SUM(d.%[1]s) / SUM(d.executions))`, column,
	)
	rows, err := ds.writer(context.Background()).Queryx(queriesSQL, id)
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
	hostCount := 10  // 2000
	queryCount := 30 // 1000

	start := time.Now()
	for i := 0; i < queryCount; i++ {
		_, err := ds.writer(context.Background()).Exec(`INSERT INTO queries(name, query, description) VALUES (?,?,?)`, fmt.Sprint(i), fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}
	insertScheduledQuerySQL := `INSERT IGNORE INTO scheduled_query_stats(host_id, scheduled_query_id, system_time, user_time, executions, query_type) VALUES %s`
	scheduledQueryStatsCount := 100 // 1000000
	for i := 0; i < scheduledQueryStatsCount; i++ {
		if len(args) > batchSize {
			values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(args)/6), ",")
			_, err := ds.writer(context.Background()).Exec(fmt.Sprintf(insertScheduledQuerySQL, values), args...)
			require.NoError(t, err)
			args = []interface{}{}
		}
		// Occasionally set 0 executions
		executions := rand.Intn(10000) + 100
		if rand.Intn(100) < 5 {
			executions = 0
		}
		args = append(
			args, rand.Intn(hostCount)+1, rand.Intn(queryCount)+1, rand.Intn(10000)+100, rand.Intn(10000)+100, executions,
			rand.Intn(2),
		)
	}
	if len(args) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?),", len(args)/6), ",")
		_, err := ds.writer(context.Background()).Exec(fmt.Sprintf(insertScheduledQuerySQL, values), args...)
		require.NoError(t, err)
	}

	// Make sure we have some queries that don't have stats
	for i := queryCount; i < queryCount+4; i++ {
		_, err := ds.writer(context.Background()).Exec(`INSERT INTO queries(name, query, description) VALUES (?,?,?)`, fmt.Sprint(i), fmt.Sprint(i), fmt.Sprint(i))
		require.NoError(t, err)
	}

	t.Log("Done inserting dummy data. Took:", time.Since(start))

	testcases := []struct {
		table     string
		aggregate fleet.AggregatedStatsType
		aggFunc   func(ctx context.Context) error
	}{
		{"queries", fleet.AggregatedStatsTypeScheduledQuery, ds.UpdateQueryAggregatedStats},
	}
	for _, tt := range testcases {
		t.Run(tt.table, func(t *testing.T) {
			start = time.Now()
			require.NoError(t, tt.aggFunc(context.Background()))
			t.Log("Generated stats for ", tt.table, " in:,", time.Since(start))

			var stats []struct {
				ID          uint `db:"id"`
				GlobalStats bool `db:"global_stats"`
				fleet.AggregatedStats
			}
			require.NoError(t,
				ds.writer(context.Background()).Select(&stats,
					`
select
       id,
	   global_stats,
       JSON_EXTRACT(json_value, '$.user_time_p50') as user_time_p50,
       JSON_EXTRACT(json_value, '$.user_time_p95') as user_time_p95,
       JSON_EXTRACT(json_value, '$.system_time_p50') as system_time_p50,
       JSON_EXTRACT(json_value, '$.system_time_p95') as system_time_p95,
       JSON_EXTRACT(json_value, '$.total_executions') as total_executions
from aggregated_stats where type=?`, tt.aggregate))

			require.True(t, len(stats) > 0)
			for _, stat := range stats {
				require.False(t, stat.GlobalStats)
				checkAgainstSlowStats(t, ds, stat.ID, 50, "user_time", stat.UserTimeP50)
				checkAgainstSlowStats(t, ds, stat.ID, 95, "user_time", stat.UserTimeP95)
				checkAgainstSlowStats(t, ds, stat.ID, 50, "system_time", stat.SystemTimeP50)
				checkAgainstSlowStats(t, ds, stat.ID, 95, "system_time", stat.SystemTimeP95)
				require.NotNil(t, stat.TotalExecutions)
				assert.True(t, *stat.TotalExecutions >= 0)
			}
		})
	}
}

func checkAgainstSlowStats(t *testing.T, ds *Datastore, id uint, percentile int, column string, against *float64) {
	slowp := slowStats(t, ds, id, percentile, column)
	if against != nil {
		assert.Equal(t, slowp, *against)
	} else {
		assert.Zero(t, slowp)
	}
}

func TestAggregatedStatsSkipsDeletedQueries(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	live := test.NewQuery(t, ds, nil, "live", "select 1", 0, true)
	deleted := test.NewQuery(t, ds, nil, "deleted", "select 2", 0, true)
	notRun := test.NewQuery(t, ds, nil, "not run", "select 3", 0, true)
	test.NewQuery(t, ds, nil, "no stats", "select 4", 0, true)
	for id, executions := range map[uint]int{live.ID: 1, deleted.ID: 1, notRun.ID: 0} {
		_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO scheduled_query_stats
			(host_id, scheduled_query_id, average_memory, executions, output_size, system_time, user_time, wall_time)
			VALUES (1, ?, 1, ?, 1, 1, 1, 1)`, id, executions)
		require.NoError(t, err)
	}
	// Delete only the report row, leaving its stats behind like a failed deleteQueryStats.
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM queries WHERE id = ?`, deleted.ID)
	require.NoError(t, err)

	require.NoError(t, ds.UpdateQueryAggregatedStats(ctx))

	var ids []uint
	require.NoError(t, ds.writer(ctx).SelectContext(ctx, &ids,
		`SELECT id FROM aggregated_stats WHERE type = ?`, fleet.AggregatedStatsTypeScheduledQuery))
	require.Equal(t, []uint{live.ID}, ids)
}

func TestDeleteOrphanedQueryStats(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	orig := deleteQueryStatsBatchSize
	deleteQueryStatsBatchSize = 2
	t.Cleanup(func() { deleteQueryStatsBatchSize = orig })

	live := test.NewQuery(t, ds, nil, "live", "select 1", 0, true)
	deleted := test.NewQuery(t, ds, nil, "deleted", "select 2", 0, true)
	for _, id := range []uint{live.ID, deleted.ID} {
		for hostID := 1; hostID <= 3; hostID++ {
			_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO scheduled_query_stats
				(host_id, scheduled_query_id, average_memory, executions, output_size, system_time, user_time, wall_time)
				VALUES (?, ?, 1, 1, 1, 1, 1, 1)`, hostID, id)
			require.NoError(t, err)
		}
	}
	require.NoError(t, ds.UpdateQueryAggregatedStats(ctx))
	// Another type's row with the same id must stay.
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO aggregated_stats (id, type, global_stats, json_value) VALUES (?, ?, 0, '{}')`,
		deleted.ID, aggregatedStatsTypeMunkiVersions)
	require.NoError(t, err)
	// A deleted report with an aggregated row but no stats rows left.
	aggOnly := test.NewQuery(t, ds, nil, "aggregated only", "select 3", 0, true)
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO aggregated_stats (id, type, global_stats, json_value) VALUES (?, ?, 0, '{}')`,
		aggOnly.ID, fleet.AggregatedStatsTypeScheduledQuery)
	require.NoError(t, err)
	// Delete only the report rows, leaving their stats behind like a failed deleteQueryStats.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM queries WHERE id IN (?, ?)`, deleted.ID, aggOnly.ID)
	require.NoError(t, err)

	require.NoError(t, ds.DeleteOrphanedQueryStats(ctx))

	var statIDs []uint
	require.NoError(t, ds.writer(ctx).SelectContext(ctx, &statIDs, `SELECT scheduled_query_id FROM scheduled_query_stats`))
	require.Equal(t, []uint{live.ID, live.ID, live.ID}, statIDs)
	var aggregated []string
	require.NoError(t, ds.writer(ctx).SelectContext(ctx, &aggregated, `SELECT CONCAT(type, ':', id) FROM aggregated_stats ORDER BY type`))
	require.Equal(t, []string{
		fmt.Sprintf("%s:%d", aggregatedStatsTypeMunkiVersions, deleted.ID),
		fmt.Sprintf("%s:%d", fleet.AggregatedStatsTypeScheduledQuery, live.ID),
	}, aggregated)
}
