package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240829165930_None(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// nothing in the default setup assistant
	var count int
	err := sqlx.Get(db, &count, `SELECT COUNT(*) FROM mdm_apple_default_setup_assistants`)
	require.NoError(t, err)
	require.Zero(t, count)

	// nothing in the custom setup assistants
	err = sqlx.Get(db, &count, `SELECT COUNT(*) FROM mdm_apple_setup_assistants`)
	require.NoError(t, err)
	require.Zero(t, count)

	// nothing in the new custom setup table
	err = sqlx.Get(db, &count, `SELECT COUNT(*) FROM mdm_apple_setup_assistant_profiles`)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestUp_20240829165930_Existing(t *testing.T) {
	db := applyUpToPrev(t)

	// create the single ABM token (can only have 1 when this migration runs)
	abmTokID := execNoErrLastID(t, db, "INSERT INTO abm_tokens (organization_name, apple_id, renew_at, token) VALUES (?, ?, ?, ?)", "org", "apple", time.Now(), uuid.NewString())

	// create a couple teams
	tm1 := execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES ('team1')")
	tm2 := execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES ('team2')")

	// setup the default assistant
	execNoErr(t, db, `INSERT INTO mdm_apple_enrollment_profiles (token, type, dep_profile) VALUES (?, ?, ?)`, uuid.NewString(), "automatic", "{}")
	defProfUUID := uuid.NewString()
	execNoErr(t, db, `INSERT INTO mdm_apple_default_setup_assistants (team_id, global_or_team_id, profile_uuid) VALUES (?, ?, ?)`, nil, 0, defProfUUID)
	execNoErr(t, db, `INSERT INTO mdm_apple_default_setup_assistants (team_id, global_or_team_id, profile_uuid) VALUES (?, ?, ?)`, tm1, tm1, defProfUUID)
	// no profile registered for team 2
	execNoErr(t, db, `INSERT INTO mdm_apple_default_setup_assistants (team_id, global_or_team_id, profile_uuid) VALUES (?, ?, ?)`, tm2, tm2, "")

	// load the default assistant timestamps (ordered by global or team id)
	var defTs []time.Time
	err := sqlx.Select(db, &defTs, `SELECT updated_at FROM mdm_apple_default_setup_assistants ORDER BY global_or_team_id`)
	require.NoError(t, err)

	// create a custom setup assistant for tm1
	asst1ProfUUID := uuid.NewString()
	asst1 := execNoErrLastID(t, db, `INSERT INTO mdm_apple_setup_assistants (team_id, global_or_team_id, name, profile, profile_uuid) VALUES (?, ?, ?, ?, ?)`, tm1, tm1, "asst1", "{}", asst1ProfUUID)

	// load the custom assistant timestamp
	var asst1Ts time.Time
	err = sqlx.Get(db, &asst1Ts, `SELECT updated_at FROM mdm_apple_setup_assistants WHERE id = ?`, asst1)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// the default assistants have the ABM token stored, otherwise unchanged
	var postDefTs []time.Time
	err = sqlx.Select(db, &postDefTs, `SELECT updated_at FROM mdm_apple_default_setup_assistants ORDER BY global_or_team_id`)
	require.NoError(t, err)
	require.ElementsMatch(t, defTs, postDefTs)

	var count int
	err = sqlx.Get(db, &count, `SELECT COUNT(*) FROM mdm_apple_default_setup_assistants WHERE abm_token_id = ?`, abmTokID)
	require.NoError(t, err)
	require.Equal(t, len(postDefTs), count)
	require.Equal(t, 3, count)

	// inserting another default assistant with an existing team+token fails (new unique constraint)
	_, err = db.Exec(`INSERT INTO mdm_apple_default_setup_assistants (team_id, global_or_team_id, abm_token_id) VALUES (?, ?, ?)`, nil, 0, abmTokID)
	require.Error(t, err)
	require.ErrorContains(t, err, "Duplicate entry")

	// the custom assistant entry has been created in the new table with the same timestamp and correct token ID
	var customAssts []struct {
		SetupAssistantID uint      `db:"setup_assistant_id"`
		ABMTokenID       uint      `db:"abm_token_id"`
		ProfileUUID      string    `db:"profile_uuid"`
		UpdatedAt        time.Time `db:"updated_at"`
	}
	err = sqlx.Select(db, &customAssts, `SELECT setup_assistant_id, abm_token_id, profile_uuid, updated_at FROM mdm_apple_setup_assistant_profiles`)
	require.NoError(t, err)
	require.Len(t, customAssts, 1)
	require.EqualValues(t, asst1, customAssts[0].SetupAssistantID)
	require.EqualValues(t, abmTokID, customAssts[0].ABMTokenID)
	require.Equal(t, asst1ProfUUID, customAssts[0].ProfileUUID)
	require.Equal(t, asst1Ts, customAssts[0].UpdatedAt)
}
