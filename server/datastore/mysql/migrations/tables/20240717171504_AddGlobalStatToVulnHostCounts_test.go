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

	// insert team count
	_, err := db.Exec(stmt, "CVE-2024-0717", 1, 1)
	require.NoError(t, err)

	// insert global count
	_, err = db.Exec(stmt, "CVE-2024-0717", 0, 1)
	require.NoError(t, err)

	applyNext(t, db)

	// Check that the global_stat column has 0 for team_id = 1
	selectStmt := `
	SELECT global_stats FROM vulnerability_host_counts
	WHERE cve = ? and team_id = ?
	`
	var globalStat bool
	err = db.QueryRow(selectStmt, "CVE-2024-0717", 1).Scan(&globalStat)
	require.NoError(t, err)
	require.False(t, globalStat)

	// Check that the global_stat column has 1 for team_id = 0
	err = db.QueryRow(selectStmt, "CVE-2024-0717", 0).Scan(&globalStat)
	require.NoError(t, err)
	require.True(t, globalStat)

	// err on unique constraint with global_stat
	_, err = db.Exec(stmt, "CVE-2024-0717", 1, 1, 1)
	require.Error(t, err)
}
