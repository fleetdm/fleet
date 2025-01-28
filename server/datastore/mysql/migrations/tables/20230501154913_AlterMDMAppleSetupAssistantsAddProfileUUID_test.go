package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230501154913(t *testing.T) {
	db := applyUpToPrev(t)

	r, err := db.Exec(`INSERT INTO mdm_apple_setup_assistants (name, profile) VALUES (?, ?)`, "Test", "{}")
	require.NoError(t, err)
	id, _ := r.LastInsertId()

	// Apply current migration.
	applyNext(t, db)

	type assistant struct {
		ID          uint   `db:"id"`
		Name        string `db:"name"`
		ProfileUUID string `db:"profile_uuid"`
	}
	var asst assistant
	err = db.Get(&asst, `SELECT id, name, profile_uuid FROM mdm_apple_setup_assistants WHERE id = ?`, id)
	require.NoError(t, err)
	require.Equal(t, assistant{ID: uint(id), Name: "Test", ProfileUUID: ""}, asst) //nolint:gosec // dismiss G115

	// create a team
	r, err = db.Exec(`INSERT INTO teams (name) VALUES (?)`, "Test Team")
	require.NoError(t, err)
	tmID, _ := r.LastInsertId()

	// create another profile with a UUID for that team
	r, err = db.Exec(`INSERT INTO mdm_apple_setup_assistants (name, profile, profile_uuid, team_id, global_or_team_id) VALUES (?, ?, ?, ?, ?)`, "Test2", "{}", "abc", tmID, tmID)
	require.NoError(t, err)
	id, _ = r.LastInsertId()

	err = db.Get(&asst, `SELECT id, name, profile_uuid FROM mdm_apple_setup_assistants WHERE id = ?`, id)
	require.NoError(t, err)
	require.Equal(t, assistant{ID: uint(id), Name: "Test2", ProfileUUID: "abc"}, asst) //nolint:gosec // dismiss G115
}
