package tables

import (
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20240209110212(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(
		t, db,
		`INSERT INTO scheduled_query_stats (host_id, scheduled_query_id, query_type, average_memory, denylisted, executions, schedule_interval, output_size, system_time, user_time, wall_time)
				VALUES (0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1)`,
	)

	// Apply current migration.
	applyNext(t, db)

	var wallTime uint64
	err := sqlx.Get(
		db, &wallTime, `SELECT wall_time from scheduled_query_stats where host_id = 0 and scheduled_query_id = 0 and query_type = 0`,
	)
	require.NoError(t, err)
	require.Equal(t, uint64(1000), wallTime)
}
