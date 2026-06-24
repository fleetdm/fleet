package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260617172853(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Table should exist and be empty.
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins`).Scan(&count))
	require.Equal(t, 0, count)

	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('Firefox', 'apps')`)

	// A caret expression round-trips.
	_, err := db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '^147')`, titleID)
	require.NoError(t, err)

	var pinned string
	require.NoError(t, db.QueryRow(`SELECT pinned_version FROM software_title_team_pins WHERE team_id = 0 AND title_id = ?`, titleID).Scan(&pinned))
	require.Equal(t, "^147", pinned)

	// Duplicate (team_id, title_id) should fail.
	_, err = db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, ?, '147.0.5')`, titleID)
	require.Error(t, err)

	// FK: referencing a non-existent title should fail.
	_, err = db.Exec(`INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (0, 99999, '^147')`)
	require.Error(t, err)

	// ON DELETE CASCADE: deleting the title removes the pin row.
	_, err = db.Exec(`DELETE FROM software_titles WHERE id = ?`, titleID)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software_title_team_pins WHERE title_id = ?`, titleID).Scan(&count))
	require.Equal(t, 0, count, "expected ON DELETE CASCADE to remove pin row")
}
