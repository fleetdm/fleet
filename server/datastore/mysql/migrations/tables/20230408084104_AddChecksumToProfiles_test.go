package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230408084104(t *testing.T) {
	db := applyUpToPrev(t)
	stmt := `
INSERT INTO
    mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig)
VALUES (?, ?, ?, ?)`

	mcBytes := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>TestPayloadName</string>
	<key>PayloadIdentifier</key>
	<string>TestPayloadIdentifier</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>TestPayloadUUID</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`)

	r, err := db.Exec(stmt, 0, "TestPayloadIdentifier", "TestPayloadName", mcBytes)
	profileID, _ := r.LastInsertId()
	require.NoError(t, err)

	var (
		identifier   string
		mobileconfig []byte
	)
	err = db.QueryRow(`SELECT identifier, mobileconfig FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?`, "TestPayloadName", 0).Scan(&identifier, &mobileconfig)
	require.NoError(t, err)
	require.Equal(t, "TestPayloadIdentifier", identifier)
	require.Equal(t, mcBytes, mobileconfig)

	var status []string
	err = db.Select(&status, `SELECT status FROM mdm_apple_delivery_status`)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"failed", "applied", "pending"}, status)

	var opTypes []string
	err = db.Select(&opTypes, `SELECT operation_type FROM mdm_apple_operation_types`)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"install", "remove"}, opTypes)

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

	// apply migration
	applyNext(t, db)

	var checksum []byte
	err = db.QueryRow(`SELECT checksum FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?`, "TestPayloadName", 0).Scan(&checksum)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(mcBytes)), fmt.Sprintf("%x", checksum)) // nolint:gosec // used only to hash for efficient comparisons

	err = db.QueryRow(`SELECT checksum FROM host_mdm_apple_profiles WHERE profile_id = ?`, profileID).Scan(&checksum)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%x", md5.Sum(mcBytes)), fmt.Sprintf("%x", checksum)) // nolint:gosec // used only to hash for efficient comparisons

}
