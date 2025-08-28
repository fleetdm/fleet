package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250828122736(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// software title
	query := `
		INSERT INTO software_titles (name, source)
		VALUES (?, ?)
	`
	_, err := db.Exec(query, "Banana software title", "apps")
	require.NoError(t, err)
	var softwareTitleID int
	err = db.QueryRow("SELECT LAST_INSERT_ID()").Scan(&softwareTitleID)
	require.NoError(t, err)

	// team
	query = `
		INSERT INTO teams (name, description)
		VALUES (?, ?)
	`
	_, err = db.Exec(query, "Banana team", "Bananas are yellow")
	require.NoError(t, err)
	var teamID int
	err = db.QueryRow("SELECT LAST_INSERT_ID()").Scan(&teamID)
	require.NoError(t, err)

	query = `
		INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
		VALUES (?, ?, ?, ?)
	`
	_, err = db.Exec(query, teamID, softwareTitleID, "storage_id_1", "icon_filename_1")
	require.NoError(t, err)

	type SoftwareTitleIconResult struct {
		TeamID          int
		SoftwareTitleID int
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
	_, err = db.Exec(query, teamID, softwareTitleID, "storage_id_1", "icon_filename_1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Duplicate entry")
}
