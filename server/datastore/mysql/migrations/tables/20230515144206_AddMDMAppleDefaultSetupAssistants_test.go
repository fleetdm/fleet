package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230515144206(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	type assistant struct {
		ID          uint   `db:"id"`
		ProfileUUID string `db:"profile_uuid"`
	}
	var asst assistant

	r, err := db.Exec(`INSERT INTO mdm_apple_default_setup_assistants (profile_uuid) VALUES (?)`, "abc")
	require.NoError(t, err)
	id, _ := r.LastInsertId()

	err = db.Get(&asst, `SELECT id, profile_uuid FROM mdm_apple_default_setup_assistants WHERE id = ?`, id)
	require.NoError(t, err)
	require.Equal(t, assistant{ID: uint(id), ProfileUUID: "abc"}, asst) //nolint:gosec // dismiss G115

	// create a team
	r, err = db.Exec(`INSERT INTO teams (name) VALUES (?)`, "Test Team")
	require.NoError(t, err)
	tmID, _ := r.LastInsertId()

	// create another profile with a UUID for that team
	r, err = db.Exec(`INSERT INTO mdm_apple_default_setup_assistants (profile_uuid, team_id, global_or_team_id) VALUES (?, ?, ?)`, "def", tmID, tmID)
	require.NoError(t, err)
	id, _ = r.LastInsertId()

	err = db.Get(&asst, `SELECT id, profile_uuid FROM mdm_apple_default_setup_assistants WHERE id = ?`, id)
	require.NoError(t, err)
	require.Equal(t, assistant{ID: uint(id), ProfileUUID: "def"}, asst) //nolint:gosec // dismiss G115
}
