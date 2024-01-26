package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20240126020642(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// create some Windows profiles
	idwA, idwB, idwC := "w"+uuid.New().String(), "w"+uuid.New().String(), "w"+uuid.New().String()
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 0, 'A', '<Replace>A</Replace>')`, idwA)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 1, 'B', '<Replace>B</Replace>')`, idwB)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, 0, 'C', '<Replace>C</Replace>')`, idwC)
	nonExistingWID := "w" + uuid.New().String()

	// create some Apple profiles
	idaA, idaB, idaC := "a"+uuid.New().String(), "a"+uuid.New().String(), "a"+uuid.New().String()
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum) VALUES (?, 0, 'IA', 'NA', '<plist></plist>', '')`, idaA)
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum) VALUES (?, 1, 'IB', 'NB', '<plist></plist>', '')`, idaB)
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum) VALUES (?, 0, 'IC', 'NC', '<plist></plist>', '')`, idaC)
	nonExistingAID := "a" + uuid.New().String()

	// create some labels
	idlA := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LA', 'select 1')`)
	idlB := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LB', 'select 1')`)
	idlC := execNoErrLastID(t, db, `INSERT INTO labels (name, query) VALUES ('LC', 'select 1')`)
	nonExistingLID := idlC + 1

	// apply labels A and B to Windows profile A
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idwA, "LA", idlA)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idwA, "LB", idlB)

	// apply labels B and C to Windows profile B (team 1)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idwB, "LB", idlB)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idwB, "LC", idlC)

	// apply labels A and C to Apple profile A
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idaA, "LA", idlA)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idaA, "LC", idlC)

	// apply label B to Apple profile B (team 1)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idaB, "LB", idlB)

	// apply label A to non-existing Windows profile
	_, err := db.Exec(`INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, nonExistingWID, "LA", idlA)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// apply label A to non-existing Apple profile
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, nonExistingAID, "LA", idlA)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// apply non-existing label to Windows profile A
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idwA, "Lnone", nonExistingLID)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// apply non-existing label to Apple profile A
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idaA, "Lnone", nonExistingLID)
	require.ErrorContains(t, err, "foreign key constraint fails")

	// apply duplicate (label A to Windows profile A)
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (windows_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idwA, "LA", idlA)
	require.ErrorContains(t, err, "Duplicate entry")

	// apply duplicate (label A to Apple profile A)
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)`, idaA, "LA", idlA)
	require.ErrorContains(t, err, "Duplicate entry")
}
