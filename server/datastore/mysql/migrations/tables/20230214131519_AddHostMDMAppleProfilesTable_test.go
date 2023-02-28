package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230214131519(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	var status []string
	err := db.Select(&status, `SELECT status FROM mdm_apple_delivery_status`)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"failed", "applied", "pending"}, status)

	var opTypes []string
	err = db.Select(&opTypes, `SELECT operation_type FROM mdm_apple_operation_types`)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"install", "remove"}, opTypes)

	r, err := db.Exec(`
	  INSERT INTO
	      mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig)
	  VALUES (?, ?, ?, ?)`, 0, "TestPayloadIdentifier", "TestPayloadName", `<?xml version="1.0"`)
	require.NoError(t, err)
	profileID, _ := r.LastInsertId()

	_, err = db.Exec(`
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('command-uuid', 'foo', '<?xml')
	`)
	require.NoError(t, err)

	insertStmt := `
          INSERT INTO host_mdm_apple_profiles
            (profile_id, profile_identifier, host_uuid, command_uuid, status, operation_type, detail)
          VALUES
            (?, 'com.foo.bar', ?, 'command-uuid', ?, ?, ?)
        `
	execNoErr(t, db, insertStmt, profileID, "ABC", "pending", "install", "")
	execNoErr(t, db, insertStmt, profileID, "DEF", "failed", "remove", "error message")

	_, err = db.Exec(insertStmt, profileID, "XYZ", "foo", "install", "")
	require.ErrorContains(t, err, "Error 1452")

	_, err = db.Exec(insertStmt, profileID, "LMN", "failed", "foo", "")
	require.ErrorContains(t, err, "Error 1452")
}
