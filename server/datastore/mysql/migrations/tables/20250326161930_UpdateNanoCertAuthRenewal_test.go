package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20250326161930(t *testing.T) {
	db := applyUpToPrev(t)

	notValidAfter := time.Now().Add(time.Hour).Truncate(time.Second).UTC()

	authMsg := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>BuildVersion</key>
	<string>20H350</string>
	<key>MessageType</key>
	<string>Authenticate</string>
	<key>OSVersion</key>
	<string>16.7.10</string>
	<key>ProductName</key>
	<string>iPhone10,6</string>
	<key>SerialNumber</key>
	<string>AAABBBCCCDDD</string>
	<key>Topic</key>
	<string>com.apple.mgmt.External.aaabbbcccdddeee</string>
	<key>UDID</key>
	<string>device-%d</string>
</dict>
</plist>`

	// create a few commands
	for i := 0; i < 3; i++ {
		execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command)
		VALUES (?, ?, ?)`, fmt.Sprintf("renew-command_uuid-%d", i), "InstallProfile", "some-command")
	}

	// create a few devices/enrollments:
	for i := 0; i < 3; i++ {
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate)
		VALUES (?, ?)`, fmt.Sprintf("device-%d", i), fmt.Sprintf(authMsg, i))
		execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, NOW())`, fmt.Sprintf("device-%d", i), fmt.Sprintf("device-%d", i), "Device", "topic", "push_magic", "token_hex")
	}

	// create some command results
	execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result)
		VALUES (?, ?, ?, ?)`, "device-1", "renew-command_uuid-1", "Acknowledged", "<?xml><string>some-result</string>")
	execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result)
		VALUES (?, ?, ?, ?)`, "device-2", "renew-command_uuid-1", "Error", "<?xml><string>some-result</string>")

	// create cert auth association with no renew command
	execNoErr(t, db, `INSERT INTO nano_cert_auth_associations (id, sha256, cert_not_valid_after)
		VALUES (?, ?, ?)`, "device-1", "hash-1", notValidAfter)

	// create cert auth association with renew command that has been acknowledged, should be cleared
	execNoErr(t, db, `INSERT INTO nano_cert_auth_associations (id, sha256, cert_not_valid_after, renew_command_uuid)
		VALUES (?, ?, ?, ?)`, "device-1", "hash-2", notValidAfter, "renew-command_uuid-1")

	// create cert auth association with renew command that has not been acknowledged, should not be cleared
	execNoErr(t, db, `INSERT INTO nano_cert_auth_associations (id, sha256, cert_not_valid_after, renew_command_uuid)
		VALUES (?, ?, ?, ?)`, "device-2", "hash-3", notValidAfter, "renew-command_uuid-1")

	applyNext(t, db)

	// check that the cert auth association with the acknowledged renew command was cleared
	var rows []struct {
		ID                string    `db:"id"`
		Sha256            string    `db:"sha256"`
		CertNotValidAfter time.Time `db:"cert_not_valid_after"`
		RenewCommandUUID  *string   `db:"renew_command_uuid"`
	}
	require.NoError(t, db.Select(&rows, `SELECT id, sha256, cert_not_valid_after, renew_command_uuid FROM nano_cert_auth_associations ORDER BY id, sha256`))
	require.Len(t, rows, 3)
	require.Equal(t, "device-1", rows[0].ID)
	require.Equal(t, "hash-1", rows[0].Sha256)
	require.Equal(t, notValidAfter, rows[0].CertNotValidAfter)
	require.Nil(t, rows[0].RenewCommandUUID)

	require.Equal(t, "device-1", rows[1].ID)
	require.Equal(t, "hash-2", rows[1].Sha256)
	require.Equal(t, notValidAfter, rows[1].CertNotValidAfter)
	require.Nil(t, rows[1].RenewCommandUUID)

	require.Equal(t, "device-2", rows[2].ID)
	require.Equal(t, "hash-3", rows[2].Sha256)
	require.Equal(t, notValidAfter, rows[2].CertNotValidAfter)
	require.NotNil(t, rows[2].RenewCommandUUID)
	require.Equal(t, "renew-command_uuid-1", *rows[2].RenewCommandUUID)
}
