package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20231207102321(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)"
	_, err := db.Exec(insertStmt, "test-name", "test-source", "")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source", "")
	require.NoError(t, err)

	applyNext(t, db)

	// unique constraint applies to name+source+browser
	_, err = db.Exec(insertStmt, "test-name", "test-source", "")
	require.ErrorContains(t, err, "Duplicate entry")

	_, err = db.Exec(insertStmt, "test-name", "test-source", "test-browser")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source", "test-browser")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source2", "test-browser")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source2", "test-browser2")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "test-name2", "test-source2", "test-browser2")
	require.ErrorContains(t, err, "Duplicate entry")
}
