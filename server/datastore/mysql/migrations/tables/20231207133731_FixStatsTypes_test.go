package tables

import (
	"github.com/stretchr/testify/require"
	"math"
	"testing"
)

func TestUp_20231207133731(t *testing.T) {
	db := applyUpToPrev(t)

	setupStmt := `
		INSERT INTO scheduled_query_stats (host_id, scheduled_query_id, average_memory, denylisted, executions, schedule_interval, output_size, system_time, user_time, wall_time, last_executed) VALUES
			(?,?,?,?,?,?,?,?,?,?,?);
	`

	_, err := db.Exec(setupStmt, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, "2023-12-07 13:17:17")
	require.NoError(t, err)
	// Apply current migration.
	applyNext(t, db)

	stmt := `
		SELECT host_id, average_memory FROM scheduled_query_stats WHERE host_id = 1;
	`
	rows, err := db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()
	count := 0
	for rows.Next() {
		count += 1
		var hostId int
		var avgMem uint64
		err := rows.Scan(&hostId, &avgMem)
		require.NoError(t, err)
		require.Equal(t, 1, hostId)
		require.Equal(t, uint64(3), avgMem)
	}
	require.Equal(t, 1, count)

	_, err = db.Exec(setupStmt, 2, 2, uint64(math.MaxUint64), 4, uint64(math.MaxUint64-1), 6, uint64(math.MaxUint64-2), uint64(math.MaxUint64-3), uint64(math.MaxUint64-4), uint64(math.MaxUint64-5), "2023-12-07 13:17:17")
	require.NoError(t, err)

	stmt = `
		SELECT host_id, average_memory, executions, output_size, system_time, user_time, wall_time FROM scheduled_query_stats WHERE host_id = 2;
	`
	rows, err = db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()
	count = 0
	for rows.Next() {
		count += 1
		var hostId int
		var avgMem, executions, outputSize, systemTime, userTime, wallTime uint64
		err := rows.Scan(&hostId, &avgMem, &executions, &outputSize, &systemTime, &userTime, &wallTime)
		require.NoError(t, err)
		require.Equal(t, 2, hostId)
		require.Equal(t, uint64(math.MaxUint64), avgMem)
		require.Equal(t, uint64(math.MaxUint64-1), executions)
		require.Equal(t, uint64(math.MaxUint64-2), outputSize)
		require.Equal(t, uint64(math.MaxUint64-3), systemTime)
		require.Equal(t, uint64(math.MaxUint64-4), userTime)
		require.Equal(t, uint64(math.MaxUint64-5), wallTime)
	}
	require.Equal(t, 1, count)

}
