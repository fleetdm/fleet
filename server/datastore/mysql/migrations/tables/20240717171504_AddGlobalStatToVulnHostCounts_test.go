package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240717171504(t *testing.T) {
	db := applyUpToPrev(t)

	stmt := `
	INSERT INTO vulnerability_host_counts
	(cve, team_id, host_count)
	VALUES (?, ?, ?)
	`
	_, err := db.Exec(stmt, "CVE-2024-0717", 1, 1)
	require.NoError(t, err)

	applyNext(t, db)

	// Check that the global_stat column was added with default 0
	selectStmt := `
	SELECT global_stats FROM vulnerability_host_counts
	WHERE cve = ?
	`
	var globalStat bool
	err = db.QueryRow(selectStmt, "CVE-2024-0717").Scan(&globalStat)
	require.NoError(t, err)
	require.False(t, globalStat)

	// insert global_stat value for same row
	stmt = `
	INSERT INTO vulnerability_host_counts
	(cve, team_id, host_count, global_stats)
	VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(stmt, "CVE-2024-0717", 1, 1, 1)
	require.NoError(t, err)

	// err on unique constraint with global_stat
	_, err = db.Exec(stmt, "CVE-2024-0717", 1, 1, 1)
	require.Error(t, err)
}
