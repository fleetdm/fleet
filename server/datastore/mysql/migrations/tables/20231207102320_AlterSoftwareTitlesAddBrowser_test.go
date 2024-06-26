package tables

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231207102320(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := "INSERT INTO software_titles (name, source) VALUES (?, ?)"

	_, err := db.Exec(insertStmt, "test-name", "test-source")
	require.NoError(t, err)

	selectStmt := "SELECT id, name, source FROM software_titles"
	var rows []struct {
		ID     uint   `db:"id"`
		Name   string `db:"name"`
		Source string `db:"source"`
	}
	err = sqlx.SelectContext(context.Background(), db, &rows, selectStmt)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	applyNext(t, db)

	selectStmt = "SELECT id, name, source, browser FROM software_titles"
	type newRow struct {
		ID      uint   `db:"id"`
		Name    string `db:"name"`
		Source  string `db:"source"`
		Browser string `db:"browser"`
	}
	var newRows []newRow

	// migration should delete all rows
	err = sqlx.SelectContext(context.Background(), db, &newRows, selectStmt)
	require.NoError(t, err)
	require.Len(t, newRows, 0)

	// re-insert the old row
	_, err = db.Exec(insertStmt, "test-name", "test-source")
	require.NoError(t, err)
	err = sqlx.SelectContext(context.Background(), db, &newRows, selectStmt)
	require.NoError(t, err)
	require.Len(t, newRows, 1)
	require.Equal(t, "test-name", newRows[0].Name)
	require.Equal(t, "test-source", newRows[0].Source)
	require.Equal(t, "", newRows[0].Browser) // default browser is empty string

	insertStmt = "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)"

	_, err = db.Exec(insertStmt, "test-name", "test-source", "test-browser")
	require.NoError(t, err)

	newRows = []newRow{}
	err = sqlx.SelectContext(context.Background(), db, &newRows, selectStmt)
	require.NoError(t, err)
	require.Len(t, newRows, 2)
	var found bool
	for _, row := range newRows {
		if row.Browser == "test-browser" {
			require.False(t, found)
			found = true
		} else {
			// browser should be empty for existing rows
			require.Equal(t, "", row.Browser)
		}
	}
}
