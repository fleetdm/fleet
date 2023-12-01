package tables

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231130132931(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := "INSERT INTO software (name, source, version) VALUES (?, ?, ?)"

	_, err := db.Exec(insertStmt, "test-name", "test-source", "test-version")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source", "test-version")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name", "test-source2", "test-version")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name", "test-source", "test-version2")
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Check that the title_id column was added.
	selectStmt := `
SELECT
	id,
	name,
	version,
	source,
	title_id
FROM software
WHERE name IN ('test-name', 'test-name2') AND title_id IS NULL`

	var rows []fleet.Software
	err = sqlx.SelectContext(context.Background(), db, &rows, selectStmt)
	require.NoError(t, err)
	require.Len(t, rows, 4)

	for _, row := range rows {
		require.Contains(t, []string{"test-name", "test-name2"}, row.Name)
		require.Contains(t, []string{"test-source", "test-source2"}, row.Source)
		require.Contains(t, []string{"test-version", "test-version2"}, row.Version)
		require.Nil(t, row.TitleID)
	}

	// add a row without the title_id set
	_, err = db.Exec(insertStmt, "test-name", "test-source", "test-version3")
	require.NoError(t, err)

	// add a row with the title_id set
	insertStmt = "INSERT INTO software (name, source, version, title_id) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(insertStmt, "test-name", "test-source", "test-version4", 1)
	require.NoError(t, err)

	selectStmt = `
SELECT
	id,
	name,
	version,
	source,
	title_id
FROM software
WHERE title_id = ?`

	rows = []fleet.Software{}
	err = sqlx.SelectContext(context.Background(), db, &rows, selectStmt, 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	updateStmt := "UPDATE software SET title_id = ? WHERE name = ? AND source = ?"

	_, err = db.Exec(updateStmt, 1, "test-name", "test-source")
	require.NoError(t, err)

	rows = []fleet.Software{}
	err = sqlx.SelectContext(context.Background(), db, &rows, selectStmt, 1)
	require.NoError(t, err)
	require.Len(t, rows, 4)

	for _, row := range rows {
		require.NotNil(t, row.TitleID)
		require.Equal(t, uint(1), *row.TitleID)
		require.Equal(t, "test-name", row.Name)
		require.Equal(t, "test-source", row.Source)
		require.Contains(t, []string{"test-version", "test-version2", "test-version3", "test-version4"}, row.Version)
	}
}
