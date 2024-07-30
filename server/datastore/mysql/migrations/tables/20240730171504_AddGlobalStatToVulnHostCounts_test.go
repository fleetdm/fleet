package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240730171504(t *testing.T) {
	db := applyUpToPrev(t)

	stmt := `
	INSERT INTO vulnerability_host_counts
	(cve, team_id, host_count)
	VALUES (?, ?, ?)
	`

	// Insert team 1 counts
	_, err := db.Exec(stmt, "CVE-2024-1000", 1, 5)
	require.NoError(t, err)
	_, err = db.Exec(stmt, "CVE-2024-2000", 1, 10)
	require.NoError(t, err)
	_, err = db.Exec(stmt, "CVE-2024-3000", 1, 1)
	require.NoError(t, err)

	// Insert team 2 count
	_, err = db.Exec(stmt, "CVE-2024-1000", 2, 2)
	require.NoError(t, err)
	_, err = db.Exec(stmt, "CVE-2024-2000", 2, 0) // 0 count
	require.NoError(t, err)
	_, err = db.Exec(stmt, "CVE-2024-3000", 2, 2)
	require.NoError(t, err)

	// Insert global count
	_, err = db.Exec(stmt, "CVE-2024-1000", 0, 10)
	require.NoError(t, err)
	_, err = db.Exec(stmt, "CVE-2024-2000", 0, 90)
	require.NoError(t, err)
	_, err = db.Exec(stmt, "CVE-2024-3000", 0, 0) // edge case, wrong count
	require.NoError(t, err)

	applyNext(t, db)

	assertHostCount := func(cve string, teamID, expectedCount int, globalStat bool) {
		t.Helper()
		selectStmt := `
	SELECT host_count FROM vulnerability_host_counts
	WHERE cve = ? and team_id = ? and global_stats = ?
	`

		var count int
		err = db.QueryRow(selectStmt, cve, teamID, globalStat).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, expectedCount, count)
	}

	// Check team 1 counts
	assertHostCount("CVE-2024-1000", 1, 5, false)
	assertHostCount("CVE-2024-2000", 1, 10, false)
	assertHostCount("CVE-2024-3000", 1, 1, false)

	// Check team 2 counts
	assertHostCount("CVE-2024-1000", 2, 2, false)
	assertHostCount("CVE-2024-2000", 2, 0, false)
	assertHostCount("CVE-2024-3000", 2, 2, false)

	// Check global counts
	assertHostCount("CVE-2024-1000", 0, 10, true)
	assertHostCount("CVE-2024-2000", 0, 90, true)
	assertHostCount("CVE-2024-3000", 0, 0, true)

	// Check no team counts
	assertHostCount("CVE-2024-1000", 0, 3, false)
	assertHostCount("CVE-2024-2000", 0, 80, false)
	assertHostCount("CVE-2024-3000", 0, 0, false) // edge case, wrong count should not result in negative count

	// Check unique constraint violation
	_, err = db.Exec(stmt, "CVE-2024-0717", 1, 1, 1)
	require.Error(t, err)
}
