package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231122155427(t *testing.T) {
	db := applyUpToPrev(t)

	// create some Windows profiles
	idwA, idwB, idwC := uuid.New().String(), uuid.New().String(), uuid.New().String()
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 0, 'A', '<Replace>A</Replace>')`, idwA)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 1, 'B', '<Replace>B</Replace>')`, idwB)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 0, 'C', '<Replace>C</Replace>')`, idwC)
	nonExistingWID := uuid.New().String()

	// create some Windows hosts profiles with one not related to an existing profile
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h1', ?, 'c1')`, idwA)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h2', ?, 'c2')`, idwB)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h2', ?, 'c3')`, nonExistingWID)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h2', ?, 'c4')`, idwA)

	// create some Apple profiles
	idaA := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum) VALUES (0, 'IA', 'NA', '<plist></plist>', '')`)
	idaB := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum) VALUES (1, 'IB', 'NB', '<plist></plist>', '')`)
	idaC := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum) VALUES (0, 'IC', 'NC', '<plist></plist>', '')`)
	nonExistingAID := idaC + 1000

	// create some Apple hosts profiles with one not related to an existing profile
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h1', ?, 'c1', 'IA', '')`, idaA)
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h2', ?, 'c2', 'IB', '')`, idaB)
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h2', ?, 'c3', 'IZ', '')`, nonExistingAID)
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h2', ?, 'c4', 'IA', '')`, idaA)

	// Apply current migration.
	applyNext(t, db)

	// Windows profile uuids were updated with the prefix
	var wprofUUIDs []string
	err := sqlx.Select(db, &wprofUUIDs, `SELECT profile_uuid FROM mdm_windows_configuration_profiles ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, wprofUUIDs, 3)
	require.Equal(t, "w"+idwA, wprofUUIDs[0])
	require.Equal(t, "w"+idwB, wprofUUIDs[1])
	require.Equal(t, "w"+idwC, wprofUUIDs[2])

	// Apple profiles were assigned uuids in addition to identifier
	var aprofUUIDs []string
	err = sqlx.Select(db, &aprofUUIDs, `SELECT profile_uuid FROM mdm_apple_configuration_profiles ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, aprofUUIDs, 3)
	require.Len(t, aprofUUIDs[0], 37)
	require.Len(t, aprofUUIDs[1], 37)
	require.Len(t, aprofUUIDs[2], 37)
	require.Equal(t, "a", string(aprofUUIDs[0][0]))
	require.Equal(t, "a", string(aprofUUIDs[1][0]))
	require.Equal(t, "a", string(aprofUUIDs[2][0]))

	var hostUUIDs []string
	// get Windows hosts with profile A
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, wprofUUIDs[0])
	require.NoError(t, err)
	require.Equal(t, []string{"h1", "h2"}, hostUUIDs)

	// get Windows hosts with profile B
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, wprofUUIDs[1])
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)

	// get Windows hosts with profile C
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, wprofUUIDs[2])
	require.NoError(t, err)
	require.Empty(t, hostUUIDs)

	// get Windows hosts with unknown profile
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, "w"+nonExistingWID)
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)

	// get profile uuid of non-existing profile
	var nonExistingProfUUIDs []string
	err = sqlx.Select(db, &nonExistingProfUUIDs, `SELECT profile_uuid FROM host_mdm_windows_profiles WHERE command_uuid = 'c3' ORDER BY profile_uuid`)
	require.NoError(t, err)
	require.Len(t, nonExistingProfUUIDs, 1)
	require.Len(t, nonExistingProfUUIDs[0], 37)
	require.Equal(t, "w", string(nonExistingProfUUIDs[0][0]))

	// get Apple hosts with profile NA
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, aprofUUIDs[0])
	require.NoError(t, err)
	require.Equal(t, []string{"h1", "h2"}, hostUUIDs)

	// get Apple hosts with profile NB
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, aprofUUIDs[1])
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)

	// get Apple hosts with profile C
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, aprofUUIDs[2])
	require.NoError(t, err)
	require.Empty(t, hostUUIDs)

	// get Apple hosts with unknown profile, it was assigned an apple uuid
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_identifier = 'IZ' ORDER BY host_uuid`)
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)
	nonExistingProfUUIDs = nonExistingProfUUIDs[:0]
	err = sqlx.Select(db, &nonExistingProfUUIDs, `SELECT profile_uuid FROM host_mdm_apple_profiles WHERE profile_identifier = 'IZ' ORDER BY host_uuid`)
	require.NoError(t, err)
	require.Len(t, nonExistingProfUUIDs, 1)
	require.Len(t, nonExistingProfUUIDs[0], 37)
	require.Equal(t, "a", string(nonExistingProfUUIDs[0][0]))

	// creating a new Apple profile still generates a unique numerical id
	idaD := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum) VALUES (CONCAT('a', CONVERT(uuid() USING utf8mb4)), 0, 'ID', 'ND', '<plist></plist>', '')`)
	require.NotZero(t, idaD)
	require.Greater(t, idaD, idaC)

	// batch-creating new Apple profiles also generates unique numerical ids
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles
		(profile_uuid, team_id, identifier, name, mobileconfig, checksum)
	VALUES
		(CONCAT('a', CONVERT(uuid() USING utf8mb4)), 0, 'IE', 'NE', '<plist></plist>', ''),
		(CONCAT('a', CONVERT(uuid() USING utf8mb4)), 0, 'IF', 'NF', '<plist></plist>', ''),
		(CONCAT('a', CONVERT(uuid() USING utf8mb4)), 0, 'IG', 'NG', '<plist></plist>', '')
`)
	var profIDs []int64
	err = sqlx.Select(db, &profIDs, `SELECT profile_id FROM mdm_apple_configuration_profiles ORDER BY name`)
	require.NoError(t, err)
	require.Equal(t, []int64{idaA, idaB, idaC, idaD, idaD + 1, idaD + 2, idaD + 3}, profIDs)
}
