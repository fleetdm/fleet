package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20231009094543(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	insertStmt := `INSERT INTO queries (
		name, description, query, discard_data
	) VALUES (?, ?, ?, ?)`

	res, err := db.Exec(insertStmt, "test", "test description", "SELECT 1 from hosts", false)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	require.NotNil(t, id)
	require.Equal(t, int64(1), id)

	var query []fleet.Query
	err = db.Select(&query, `SELECT
		id,
		name,
		description,
		query,
		discard_data
	FROM queries WHERE id = ?`, id)
	require.NoError(t, err)
	require.False(t, query[0].DiscardData)

	// Insert without discard_data, verify that default is correct

	insertStmt = `INSERT INTO queries (
		name, description, query
	) VALUES (?, ?, ?)`

	res, err = db.Exec(insertStmt, "test 2", "test description 2", "SELECT 1 from hosts")
	require.NoError(t, err)
	id, _ = res.LastInsertId()
	require.NotNil(t, id)
	require.Equal(t, int64(2), id)

	var queryNoDiscard []fleet.Query
	err = db.Select(&queryNoDiscard, `SELECT
		id,
		name,
		description,
		query,
		discard_data
	FROM queries WHERE id = ?`, id)
	require.NoError(t, err)
	require.True(t, queryNoDiscard[0].DiscardData)
}
