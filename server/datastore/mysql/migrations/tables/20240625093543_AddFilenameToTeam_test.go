package tables

import (
	"database/sql"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20240625093543(t *testing.T) {
	db := applyUpToPrev(t)

	// create a team
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "Test Team")

	applyNext(t, db)

	// Check that the filename column is added and NULL
	selectStmt := `SELECT filename from teams WHERE id = ?`
	var filename sql.NullString
	require.NoError(t, db.Get(&filename, selectStmt, teamID))
	require.False(t, filename.Valid)

	// Insert a filename
	goldenFilename := "goldenFilename.yml"
	teamID2 := execNoErrLastID(t, db, `INSERT INTO teams (name, filename) VALUES (?, ?)`, "Test Team 2", goldenFilename)
	require.NoError(t, db.Get(&filename, selectStmt, teamID2))
	require.True(t, filename.Valid)
	require.Equal(t, goldenFilename, filename.String)

	// Insert a duplicate filename, which is not allowed.
	_, err := db.Exec(`INSERT INTO teams (name, filename) VALUES (?, ?)`, "Test Team 3", goldenFilename)
	require.Error(t, err)
}
