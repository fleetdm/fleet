package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250905090000(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// software title
	query := `
		INSERT INTO software_titles (name, source)
		VALUES (?, ?)
	`
	softwareTitleID := execNoErrLastID(t, db, query, "Banana software title", "apps")

	// team
	query = `
		INSERT INTO teams (name, description)
		VALUES (?, ?)
	`
	teamID := execNoErrLastID(t, db, query, "Banana team", "Bananas are yellow")

	query = `
		INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
		VALUES (?, ?, ?, ?)
	`
	_, err := db.Exec(query, teamID, softwareTitleID, "storage_id_1", "icon_filename_1")
	require.NoError(t, err)

	type SoftwareTitleIconResult struct {
		TeamID          int64
		SoftwareTitleID int64
		StorageID       string
		Filename        string
	}
	var result SoftwareTitleIconResult
	err = db.QueryRow(`
		SELECT team_id, software_title_id, storage_id, filename
		FROM software_title_icons
		WHERE team_id = ? AND software_title_id = ?
	`, teamID, softwareTitleID).Scan(&result.TeamID, &result.SoftwareTitleID, &result.StorageID, &result.Filename)
	require.NoError(t, err)
	require.Equal(t, teamID, result.TeamID)
	require.Equal(t, softwareTitleID, result.SoftwareTitleID)
	require.Equal(t, "storage_id_1", result.StorageID)
	require.Equal(t, "icon_filename_1", result.Filename)

	query = `
		INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
		VALUES (?, ?, ?, ?)
	`
	// unique constraint error
	_, err = db.Exec(query, teamID, softwareTitleID, "storage_id_2", "icon_filename_2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Duplicate entry")
}
