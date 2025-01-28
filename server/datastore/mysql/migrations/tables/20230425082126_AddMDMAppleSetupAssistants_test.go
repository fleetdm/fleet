package tables

import (
	"database/sql"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUp_20230425082126(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// create a row with default team
	r, err := db.Exec(`INSERT INTO mdm_apple_setup_assistants (name, profile) VALUES (?, ?)`, "Test", "{}")
	require.NoError(t, err)
	id, _ := r.LastInsertId()

	// create a row for a non-existing team id
	_, err = db.Exec(`INSERT INTO mdm_apple_setup_assistants (name, profile, team_id, global_or_team_id) VALUES (?, ?, ?, ?)`, "Test2", "{}", 999, 999)
	require.Error(t, err)
	require.ErrorContains(t, err, "foreign key constraint fails")

	type assistant struct {
		ID             uint   `db:"id"`
		Name           string `db:"name"`
		Profile        string `db:"profile"`
		TeamID         *uint  `db:"team_id"`
		GlobalOrTeamID uint   `db:"global_or_team_id"`
	}
	var asst assistant
	err = db.Get(&asst, `SELECT id, name, profile, team_id, global_or_team_id FROM mdm_apple_setup_assistants WHERE id = ?`, id)
	require.NoError(t, err)
	require.Equal(t, assistant{ID: uint(id), Name: "Test", Profile: "{}", TeamID: nil, GlobalOrTeamID: 0}, //nolint:gosec // dismiss G115
		asst)

	// create a team
	r, err = db.Exec(`INSERT INTO teams (name) VALUES (?)`, "Test Team")
	require.NoError(t, err)
	tmID, _ := r.LastInsertId()

	// create a row for that team
	r, err = db.Exec(`INSERT INTO mdm_apple_setup_assistants (name, profile, team_id, global_or_team_id) VALUES (?, ?, ?, ?)`, "Test2", "{}", tmID, tmID)
	require.NoError(t, err)
	id2, _ := r.LastInsertId()

	err = db.Get(&asst, `SELECT id, name, profile, team_id, global_or_team_id FROM mdm_apple_setup_assistants WHERE id = ?`, id2)
	require.NoError(t, err)
	require.Equal(t, assistant{ID: uint(id2), Name: "Test2", Profile: "{}", TeamID: ptr.Uint(uint(tmID)), //nolint:gosec // dismiss G115
		GlobalOrTeamID: uint(tmID)}, asst) //nolint:gosec // dismiss G115

	// delete the team, that deletes the row
	_, err = db.Exec(`DELETE FROM teams WHERE id = ?`, tmID)
	require.NoError(t, err)

	err = db.Get(&asst, `SELECT id FROM mdm_apple_setup_assistants WHERE id = ?`, id2)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
