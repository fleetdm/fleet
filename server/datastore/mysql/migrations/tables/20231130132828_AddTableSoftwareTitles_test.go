package tables

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231130132828(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	insertStmt := "INSERT INTO software_titles (name, source) VALUES (?, ?)"

	_, err := db.Exec(insertStmt, "test-name", "test-source")
	require.NoError(t, err)

	// unique constraint applies to name+source
	_, err = db.Exec(insertStmt, "test-name", "test-source")
	require.ErrorContains(t, err, "Duplicate entry")

	_, err = db.Exec(insertStmt, "test-name", "test-source2")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source2")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name", "test-name")
	require.NoError(t, err)

	selectStmt := "SELECT id, name, source FROM software_titles"
	var rows []struct {
		ID     uint   `db:"id"`
		Name   string `db:"name"`
		Source string `db:"source"`
	}
	err = sqlx.SelectContext(context.Background(), db, &rows, selectStmt)
	require.NoError(t, err)
	require.Len(t, rows, 5)
}
