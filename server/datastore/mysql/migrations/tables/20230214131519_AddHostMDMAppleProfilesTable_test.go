package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230214131519(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM mdm_apple_profile_status`)
	require.NoError(t, err)
	require.Equal(t, 4, count)

	r, err := db.Exec(`
	  INSERT INTO
	      mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig)
	  VALUES (?, ?, ?, ?)`, 0, "TestPayloadIdentifier", "TestPayloadName", `<?xml version="1.0"`)
	require.NoError(t, err)
	profileID, _ := r.LastInsertId()

	insertStmt := `INSERT INTO host_mdm_apple_profiles (profile_id, host_uuid, status, error) VALUES (?, ?, ?, ?)`
	execNoErr(t, db, insertStmt, profileID, "ABC", "INSTALLING", "")
	execNoErr(t, db, insertStmt, profileID, "DEF", "FAILED", "error message")

	_, err = db.Exec(insertStmt, profileID, "XYZ", "FOO", "")
	require.ErrorContains(t, err, "Error 1452")

	_, err = db.Exec(insertStmt, 12345, "LMN", "INSTALLING", "")
	require.ErrorContains(t, err, "Error 1452")
}
