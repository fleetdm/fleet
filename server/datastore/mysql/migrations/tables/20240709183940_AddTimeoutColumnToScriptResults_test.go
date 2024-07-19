package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240709183940(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
		INSERT INTO host_script_results 
		(host_id, execution_id, output)
		VALUES (?, ?, ?)
		`
	_, err := db.Exec(insertStmt, 1, 1, "output")
	require.NoError(t, err)

	applyNext(t, db)

	selectStmt := `
		SELECT timeout FROM host_script_results
		WHERE host_id = ?
		`
	var timeout int
	err = db.QueryRow(selectStmt, 1).Scan(&timeout)
	require.NoError(t, err)
	require.Equal(t, 300, timeout)

	// inserting no timeout succeeds
	_, err = db.Exec(insertStmt, 2, 2, "output")
	require.NoError(t, err)
}
