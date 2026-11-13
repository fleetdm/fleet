package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (ds *Datastore) AsyncBatchSaveHostsScheduledQueryStats(ctx context.Context, stats map[uint][]fleet.ScheduledQueryStats, batchSize int) (int, error) {
	return saveHostsScheduledQueryStatsDB(ctx, ds.writer(ctx), stats, batchSize)
}

// saveHostsScheduledQueryStatsDB upserts the stats of one or more hosts in
// batches of batchSize rows and returns the number of statements executed.
// Callers resolve each stat's report ID (ScheduledQueryID) up front; stats
// without one are skipped.
func saveHostsScheduledQueryStatsDB(ctx context.Context, db sqlx.ExecerContext, stats map[uint][]fleet.ScheduledQueryStats, batchSize int) (int, error) {
	stmt := `
		INSERT IGNORE INTO scheduled_query_stats (
			scheduled_query_id,
			host_id,
			average_memory,
			denylisted,
			executions,
			schedule_interval,
			last_executed,
			output_size,
			system_time,
			user_time,
			wall_time
		)
		VALUES %s ON DUPLICATE KEY UPDATE
			scheduled_query_id = VALUES(scheduled_query_id),
			host_id = VALUES(host_id),
			average_memory = VALUES(average_memory),
			denylisted = VALUES(denylisted),
			executions = VALUES(executions),
			schedule_interval = VALUES(schedule_interval),
			last_executed = VALUES(last_executed),
			output_size = VALUES(output_size),
			system_time = VALUES(system_time),
			user_time = VALUES(user_time),
			wall_time = VALUES(wall_time);
	`
	var countExecs int

	// inserting sorted by host id (the first key in the PK) apparently helps
	// mysql with locking
	hostIDs := make([]uint, 0, len(stats))
	for k := range stats {
		hostIDs = append(hostIDs, k)
	}
	sort.Slice(hostIDs, func(i, j int) bool {
		return hostIDs[i] < hostIDs[j]
	})

	var (
		batchCount int
		batchArgs  []any
	)

	execBatch := func() error {
		values := strings.TrimSuffix(strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?),", batchCount), ",")
		stmt := fmt.Sprintf(stmt, values)
		if _, err := db.ExecContext(ctx, stmt, batchArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert batch of scheduled query stats")
		}
		countExecs++
		batchArgs = batchArgs[:0]
		batchCount = 0
		return nil
	}

	for _, hostID := range hostIDs {
		hostStats := stats[hostID]

		for _, stat := range hostStats {
			if stat.ScheduledQueryID == 0 {
				continue
			}
			// osquery < 5.3.0 only reports wall_time (seconds).
			if stat.WallTimeMs == 0 && stat.WallTime != 0 {
				stat.WallTimeMs = stat.WallTime * 1000
			}

			batchArgs = append(batchArgs,
				stat.ScheduledQueryID,
				hostID,
				stat.AverageMemory,
				stat.Denylisted,
				stat.Executions,
				stat.Interval,
				stat.LastExecuted,
				stat.OutputSize,
				stat.SystemTime,
				stat.UserTime,
				stat.WallTimeMs,
			)

			batchCount++
			if batchCount >= batchSize {
				if err := execBatch(); err != nil {
					return countExecs, err
				}
			}
		}
	}

	if batchCount > 0 {
		if err := execBatch(); err != nil {
			return countExecs, err
		}
	}

	return countExecs, nil
}
