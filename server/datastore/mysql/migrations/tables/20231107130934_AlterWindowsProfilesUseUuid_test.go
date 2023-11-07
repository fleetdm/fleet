package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231107130934(t *testing.T) {
	db := applyUpToPrev(t)

	// create some profiles
	idA := execNoErrLastID(t, db, `INSERT INTO mdm_windows_configuration_profiles (team_id, name, syncml) VALUES (0, 'A', '<Replace>A</Replace>')`)
	idB := execNoErrLastID(t, db, `INSERT INTO mdm_windows_configuration_profiles (team_id, name, syncml) VALUES (1, 'B', '<Replace>B</Replace>')`)
	idC := execNoErrLastID(t, db, `INSERT INTO mdm_windows_configuration_profiles (team_id, name, syncml) VALUES (0, 'C', '<Replace>C</Replace>')`)
	nonExistingID := idC + 1000

	// create some hosts profiles with one not related to an existing profile
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_id, command_uuid) VALUES ('h1', ?, 'c1')`, idA)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_id, command_uuid) VALUES ('h2', ?, 'c2')`, idB)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_id, command_uuid) VALUES ('h2', ?, 'c3')`, nonExistingID)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_id, command_uuid) VALUES ('h2', ?, 'c4')`, idA)

	// Apply current migration.
	applyNext(t, db)

	var profUUIDs []string
	err := sqlx.Select(db, &profUUIDs, `SELECT profile_uuid FROM mdm_windows_configuration_profiles ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, profUUIDs, 3)
	require.NotEmpty(t, profUUIDs[0])
	require.NotEmpty(t, profUUIDs[1])
	require.NotEmpty(t, profUUIDs[2])

	var hostUUIDs []string
	// get hosts with profile A
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, profUUIDs[0])
	require.NoError(t, err)
	require.Equal(t, []string{"h1", "h2"}, hostUUIDs)

	// get hosts with profile B
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, profUUIDs[1])
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)

	// get hosts with profile C
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, profUUIDs[2])
	require.NoError(t, err)
	require.Empty(t, hostUUIDs)

	// get profile uuid of non-existing profile
	profUUIDs = profUUIDs[:0]
	err = sqlx.Select(db, &profUUIDs, `SELECT profile_uuid FROM host_mdm_windows_profiles WHERE command_uuid = 'c3' ORDER BY profile_uuid`)
	require.NoError(t, err)
	require.Len(t, profUUIDs, 1)
	require.NotEmpty(t, profUUIDs[0])
}
