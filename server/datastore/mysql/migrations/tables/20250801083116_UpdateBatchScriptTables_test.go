package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250801083116(t *testing.T) {
	db := applyUpToPrev(t)
	stmt := `INSERT INTO scripts (name) VALUES ('Test Script')`
	r, err := db.Exec(stmt)
	if err != nil {
		t.Fatalf("failed to insert script: %v", err)
	}
	scriptID, err := r.LastInsertId()
	require.NoError(t, err)

	stmt = `INSERT INTO batch_script_executions (script_id, execution_id, user_id) VALUES (?, ?, ?)`
	r, err = db.Exec(stmt, scriptID, "abc123", 1)
	require.NoError(t, err)
	batchID, _ := r.LastInsertId()

	// Apply current migration.
	applyNext(t, db)

	var (
		jobID        sql.NullInt64
		status       sql.NullString
		activityType sql.NullString
	)

	stmt = "SELECT job_id, status, activity_type FROM batch_activities WHERE id = ?"
	err = db.QueryRow(stmt, batchID).Scan(&jobID, &status, &activityType)
	require.NoError(t, err)
	require.Equal(t, sql.NullInt64{Int64: 0, Valid: false}, jobID)
	require.Equal(t, sql.NullString{String: "started", Valid: true}, status)
	require.Equal(t, sql.NullString{String: "script", Valid: true}, activityType)
}
