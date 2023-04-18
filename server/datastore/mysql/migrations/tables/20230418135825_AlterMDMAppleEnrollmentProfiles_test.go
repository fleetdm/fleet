package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230418135825(t *testing.T) {
	db := applyUpToPrev(t)

	// insert an automatic profile
	_, err := db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type) VALUES ("tok1", "automatic")`)
	require.NoError(t, err)

	// insert a manual profile
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type) VALUES ("tok2", "manual")`)
	require.NoError(t, err)

	// insert another manual profile fails
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type) VALUES ("tok3", "manual")`)
	require.Error(t, err)
	require.ErrorContains(t, err, "Duplicate entry 'manual' for key 'idx_type'")

	// insert a duplicate token fails
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type) VALUES ("tok2", "other")`)
	require.Error(t, err)
	require.ErrorContains(t, err, "Duplicate entry 'tok2' for key 'idx_token'")

	// Apply current migration.
	applyNext(t, db)

	// team id is 0 for existing profiles
	var teamIDs []uint
	err = db.Select(&teamIDs, `SELECT team_id FROM mdm_apple_enrollment_profiles`)
	require.NoError(t, err)
	require.Equal(t, []uint{0, 0}, teamIDs)

	// insert an automatic profile for a team
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type, team_id) VALUES ("tok3", "automatic", 1)`)
	require.NoError(t, err)

	// insert a manual profile for a team
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type, team_id) VALUES ("tok4", "manual", 1)`)
	require.NoError(t, err)

	// insert another team automatic profile fails, due to team-type uniqueness
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type, team_id) VALUES ("tok5", "automatic", 1)`)
	require.Error(t, err)
	require.ErrorContains(t, err, "Duplicate entry '1-automatic' for key 'unq_enrollment_profiles_team_id_type'")

	// insert a duplicate token still fails
	_, err = db.Exec(`INSERT INTO mdm_apple_enrollment_profiles (token, type, team_id) VALUES ("tok2", "other", 2)`)
	require.Error(t, err)
	require.ErrorContains(t, err, "Duplicate entry 'tok2' for key 'idx_token'")
}
