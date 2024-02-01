package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240129162819(t *testing.T) {
	db := applyUpToPrev(t)

	prof1UUID := "w" + uuid.NewString()
	prof1UpdatedAt := time.Now().UTC().AddDate(0, 0, -3).Truncate(time.Second)
	updateProfUUID := uuid.NewString()
	updateProfUpdatedAt := time.Now().UTC().AddDate(0, 0, -3).Truncate(time.Second)

	profStmt := `
		INSERT INTO mdm_windows_configuration_profiles (team_id, name, syncml, created_at, updated_at, profile_uuid) VALUES
			(0,'prof1','<Replace></Replace>','2023-11-03 20:32:32',?,?),
			(0,'updateProf','<Replace></Replace>','2023-11-03 21:32:32',?,?);
	`

	hostProfStmt := `
		INSERT INTO host_mdm_windows_profiles (host_uuid, command_uuid, profile_uuid) VALUES
			('1','1',?),
			('2','2',?);
	`

	execNoErr(t, db, profStmt, prof1UpdatedAt, prof1UUID, updateProfUpdatedAt, updateProfUUID)
	execNoErr(t, db, hostProfStmt, prof1UUID, updateProfUUID)

	// Apply current migration.
	applyNext(t, db)

	// Check that both Windows profiles have the prefix and their updated_at value wasn't modified

	type result struct {
		Name      string    `db:"name"`
		UUID      string    `db:"profile_uuid"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	expected := map[string]result{"prof1": {UUID: prof1UUID, UpdatedAt: prof1UpdatedAt}, "updateProf": {UUID: "w" + updateProfUUID, UpdatedAt: updateProfUpdatedAt}}

	var results []result
	err := sqlx.Select(db, &results, `SELECT name, profile_uuid, updated_at FROM mdm_windows_configuration_profiles;`)
	require.NoError(t, err)

	for _, r := range results {
		require.Equal(t, expected[r.Name].UUID, r.UUID)
		require.Equal(t, expected[r.Name].UpdatedAt, r.UpdatedAt)
	}

	// Check that the UUIDs in the mapping table also have the prefix

	var hostProfUUIDs []string
	err = sqlx.Select(db, &hostProfUUIDs, `SELECT profile_uuid FROM host_mdm_windows_profiles;`)
	require.NoError(t, err)

	for _, u := range hostProfUUIDs {
		require.Equal(t, byte('w'), u[0])
	}
}
