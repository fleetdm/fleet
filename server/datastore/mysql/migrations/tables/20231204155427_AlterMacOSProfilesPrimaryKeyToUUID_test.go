package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20231204155427(t *testing.T) {
	db := applyUpToPrev(t)

	threeDayAgo := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Second)

	// create some Windows profiles
	idwA, idwB, idwC := uuid.New().String(), uuid.New().String(), uuid.New().String()
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, created_at, updated_at) VALUES (?, 0, 'A', '<Replace>A</Replace>', ?, ?)`, idwA, threeDayAgo, threeDayAgo)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, created_at, updated_at) VALUES (?, 1, 'B', '<Replace>B</Replace>', ?, ?)`, idwB, threeDayAgo, threeDayAgo)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, created_at, updated_at) VALUES (?, 0, 'C', '<Replace>C</Replace>', ?, ?)`, idwC, threeDayAgo, threeDayAgo)
	nonExistingWID := uuid.New().String()

	// create some Windows hosts profiles with one not related to an existing profile
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h1', ?, 'c1')`, idwA)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h2', ?, 'c2')`, idwB)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h2', ?, 'c3')`, nonExistingWID)
	execNoErr(t, db, `INSERT INTO host_mdm_windows_profiles (host_uuid, profile_uuid, command_uuid) VALUES ('h2', ?, 'c4')`, idwA)

	// create some Apple profiles
	idaA := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, created_at, updated_at) VALUES (0, 'IA', 'NA', '<plist></plist>', '', ?, ?)`, threeDayAgo, threeDayAgo)
	idaB := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, created_at, updated_at) VALUES (1, 'IB', 'NB', '<plist></plist>', '', ?, ?)`, threeDayAgo, threeDayAgo)
	idaC := execNoErrLastID(t, db, `INSERT INTO mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, created_at, updated_at) VALUES (0, 'IC', 'NC', '<plist></plist>', '', ?, ?)`, threeDayAgo, threeDayAgo)
	nonExistingAID := idaC + 1000

	// create some Apple hosts profiles with one not related to an existing profile
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h1', ?, 'c1', 'IA', '')`, idaA)
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h2', ?, 'c2', 'IB', '')`, idaB)
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h2', ?, 'c3', 'IZ', '')`, nonExistingAID)
	execNoErr(t, db, `INSERT INTO host_mdm_apple_profiles (host_uuid, profile_id, command_uuid, profile_identifier, checksum) VALUES ('h2', ?, 'c4', 'IA', '')`, idaA)

	// Apply current migration.
	applyNext(t, db)

	// Windows profile uuids were updated with the prefix
	var wprofs []struct {
		ProfileUUID string    `db:"profile_uuid"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := sqlx.Select(db, &wprofs, `SELECT profile_uuid, updated_at FROM mdm_windows_configuration_profiles ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, wprofs, 3)
	require.Equal(t, "w"+idwA, wprofs[0].ProfileUUID)
	require.Equal(t, "w"+idwB, wprofs[1].ProfileUUID)
	require.Equal(t, "w"+idwC, wprofs[2].ProfileUUID)
	for _, wprof := range wprofs {
		// updated_at did not change
		require.Equal(t, threeDayAgo, wprof.UpdatedAt)
	}

	// Apple profiles were assigned uuids in addition to identifier
	var aprofs []struct {
		ProfileUUID string    `db:"profile_uuid"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err = sqlx.Select(db, &aprofs, `SELECT profile_uuid, updated_at FROM mdm_apple_configuration_profiles ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, aprofs, 3)
	require.Len(t, aprofs[0].ProfileUUID, 37)
	require.Len(t, aprofs[1].ProfileUUID, 37)
	require.Len(t, aprofs[2].ProfileUUID, 37)
	require.Equal(t, "a", string(aprofs[0].ProfileUUID[0]))
	require.Equal(t, "a", string(aprofs[1].ProfileUUID[0]))
	require.Equal(t, "a", string(aprofs[2].ProfileUUID[0]))
	for _, aprof := range aprofs {
		// updated_at did not change
		require.Equal(t, threeDayAgo, aprof.UpdatedAt)
	}

	var hostUUIDs []string
	// get Windows hosts with profile A
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, wprofs[0].ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, []string{"h1", "h2"}, hostUUIDs)

	// get Windows hosts with profile B
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, wprofs[1].ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)

	// get Windows hosts with profile C
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_windows_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, wprofs[2].ProfileUUID)
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
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, aprofs[0].ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, []string{"h1", "h2"}, hostUUIDs)

	// get Apple hosts with profile NB
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, aprofs[1].ProfileUUID)
	require.NoError(t, err)
	require.Equal(t, []string{"h2"}, hostUUIDs)

	// get Apple hosts with profile C
	hostUUIDs = hostUUIDs[:0]
	err = sqlx.Select(db, &hostUUIDs, `SELECT host_uuid FROM host_mdm_apple_profiles WHERE profile_uuid = ? ORDER BY host_uuid`, aprofs[2].ProfileUUID)
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
