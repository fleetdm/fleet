package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20250922083056(t *testing.T) {
	db := applyUpToPrev(t)

	// create a Windows profile
	win := "w" + uuid.NewString()
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 0, 'A', '<Replace>A</Replace>')`, win)

	// create an Apple profile
	apple := "a" + uuid.NewString()
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum) VALUES (?, 0, 'IA', 'NA', '<plist></plist>', '')`, apple)

	// create some labels
	idA := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LA', 'select 1')`)
	idB := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LB', 'select 1')`)
	idC := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LC', 'select 1')`)

	// apply labels A and B to Windows profile
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, win, "LA", idA)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, win, "LB", idB)

	// apply labels A and C to Apple profile
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, apple, "LA", idA)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, apple, "LC", idC)

	// create a couple Android hosts
	hostAndroidNoUUID := insertHost(t, db, nil)
	hostAndroidWithUUID := insertHost(t, db, nil)
	hostMacNoUUID := insertHost(t, db, nil)
	execNoErr(t, db, `UPDATE hosts SET platform = ?, uuid = '' WHERE id = ?`, "android", hostAndroidNoUUID)
	execNoErr(t, db, `UPDATE hosts SET platform = ?, uuid = 'got-one' WHERE id = ?`, "android", hostAndroidWithUUID)
	execNoErr(t, db, `UPDATE hosts SET platform = ?, uuid = '' WHERE id = ?`, "darwin", hostMacNoUUID)
	execNoErr(t, db, `INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, ?, ?)`, hostAndroidNoUUID, "d1", "from-enterprise")
	execNoErr(t, db, `INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, ?, ?)`, hostAndroidWithUUID, "d2", "from-enterprise2")

	// Apply current migration.
	applyNext(t, db)

	// create an Android profile
	andro := "g" + uuid.NewString()
	execNoErr(t, db, `INSERT INTO mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json) VALUES (?, 0, 'A', '{}')`, andro)

	// apply labels B and C to Android profile
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (android_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, andro, "LB", idB)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (android_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, andro, "LC", idC)

	// windows profile still has labels A and B
	var ids []int64
	err := db.Select(&ids, `SELECT label_id FROM mdm_configuration_profile_labels WHERE windows_profile_uuid = ? ORDER BY label_id`, win)
	require.NoError(t, err)
	require.Equal(t, []int64{idA, idB}, ids)

	// apple profile still has labels A and C
	err = db.Select(&ids, `SELECT label_id FROM mdm_configuration_profile_labels WHERE apple_profile_uuid = ? ORDER BY label_id`, apple)
	require.NoError(t, err)
	require.Equal(t, []int64{idA, idC}, ids)

	// android profile still has labels B and C
	err = db.Select(&ids, `SELECT label_id FROM mdm_configuration_profile_labels WHERE android_profile_uuid = ? ORDER BY label_id`, andro)
	require.NoError(t, err)
	require.Equal(t, []int64{idB, idC}, ids)

	// try to insert with all profile fields
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (android_profile_uuid, windows_profile_uuid, apple_profile_uuid, label_name, label_id)
		VALUES (?, ?, ?, ?, ?)`, andro, win, apple, "LB", idB)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'ck_mdm_configuration_profile_labels_profile_uuid' is violated.")

	// try to insert with android+apple
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (android_profile_uuid, apple_profile_uuid, label_name, label_id)
		VALUES (?, ?, ?, ?)`, andro, apple, "LB", idB)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'ck_mdm_configuration_profile_labels_profile_uuid' is violated.")

	// try to insert with android+windows
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (android_profile_uuid, windows_profile_uuid, label_name, label_id)
		VALUES (?, ?, ?, ?)`, andro, win, "LB", idB)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'ck_mdm_configuration_profile_labels_profile_uuid' is violated.")

	// try to insert with windows+apple
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, apple_profile_uuid, label_name, label_id)
		VALUES (?, ?, ?, ?)`, win, apple, "LB", idB)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'ck_mdm_configuration_profile_labels_profile_uuid' is violated.")

	// try to insert without any profile uuid
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (label_name, label_id)
		VALUES (?, ?)`, "LB", idB)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'ck_mdm_configuration_profile_labels_profile_uuid' is violated.")

	var got []string
	err = db.Select(&got, `SELECT uuid FROM hosts WHERE id IN (?, ?, ?) ORDER BY id`, hostAndroidNoUUID, hostAndroidWithUUID, hostMacNoUUID)
	require.NoError(t, err)
	// empty android host got updated, non-empty stayed the same, mac host not updated
	require.Equal(t, []string{"from-enterprise", "got-one", ""}, got)
}
