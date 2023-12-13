package tables

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20231212161121(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
		INSERT INTO scheduled_query_stats (host_id, scheduled_query_id, average_memory, denylisted, executions, schedule_interval, output_size, system_time, user_time, wall_time) VALUES
			(%d,%d,%d,%d,%d,%d,%d,%d,%d,%d);
	`

	setupStmt := fmt.Sprintf(insertStmt, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	_, err := db.Exec(setupStmt)
	require.NoError(t, err)
	// Apply current migration.
	applyNext(t, db)

	stmt := `
		SELECT host_id, query_type FROM scheduled_query_stats WHERE host_id = 1;
	`
	rows, err := db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()
	count := 0
	for rows.Next() {
		count += 1
		var hostId, queryType int
		err := rows.Scan(&hostId, &queryType)
		require.NoError(t, err)
		require.Equal(t, 1, hostId)
		require.Equal(t, 0, queryType)
	}
	require.Equal(t, 1, count)

	insertStmt = `
		INSERT INTO scheduled_query_stats (host_id, scheduled_query_id, average_memory, denylisted, executions, schedule_interval, output_size, system_time, user_time, wall_time, query_type) VALUES
			(%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d);
	`
	stmt = fmt.Sprintf(insertStmt, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1)
	_, err = db.Exec(stmt)
	require.NoError(t, err)

	stmt = `
		SELECT host_id, query_type FROM scheduled_query_stats WHERE host_id = 1 AND query_type = 1;
	`
	rows, err = db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()
	count = 0
	for rows.Next() {
		count += 1
		var hostId, queryType int
		err := rows.Scan(&hostId, &queryType)
		require.NoError(t, err)
		require.Equal(t, 1, hostId)
		require.Equal(t, 1, queryType)
	}
	require.Equal(t, 1, count)

	// Testing unique constraint -- expect error due to duplicate entry for primary key
	stmt = fmt.Sprintf(insertStmt, 1, 2, 30, 40, 50, 60, 70, 80, 90, 100, 1)
	_, err = db.Exec(stmt)
	require.Error(t, err)

}
