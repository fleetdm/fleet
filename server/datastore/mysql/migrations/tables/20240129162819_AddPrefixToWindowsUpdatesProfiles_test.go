package tables

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20240129162819(t *testing.T) {
	db := applyUpToPrev(t)

	prof1UUID := "w" + uuid.NewString()
	updateProfUUID := uuid.NewString()
	setupStmt := `
		INSERT INTO mdm_windows_configuration_profiles VALUES
			(0,'prof1','<Replace></Replace>','2023-11-03 20:32:32','2023-11-03 20:32:32', '%s'),
			(0,'updateProf','<Replace></Replace>','2023-11-03 21:32:32','2023-11-03 21:32:32', '%s');
		INSERT INTO host_mdm_windows_profiles (host_uuid, command_uuid, profile_uuid) VALUES
			('1', '1', '%s'),
			('2', '2', '%s');
	`

	_, err := db.Exec(fmt.Sprintf(setupStmt, prof1UUID, updateProfUUID, prof1UUID, updateProfUUID))
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	stmt := `SELECT profile_uuid FROM mdm_windows_configuration_profiles;`
	rows, err := db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var uuid string
		err := rows.Scan(&uuid)
		require.NoError(t, err)
		require.Equal(t, byte('w'), uuid[0])
	}

	stmt = `SELECT profile_uuid FROM host_mdm_windows_profiles;`
	hostRows, err := db.Query(stmt)
	require.NoError(t, hostRows.Err())
	require.NoError(t, err)
	defer hostRows.Close()

	for hostRows.Next() {
		var uuid string
		err := hostRows.Scan(&uuid)
		require.NoError(t, err)
		require.Equal(t, byte('w'), uuid[0])
	}
}
