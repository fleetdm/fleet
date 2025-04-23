package tables

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20250326161931(t *testing.T) {
	db := applyUpToPrev(t)

	// create a few devices/enrollments:
	// - one that is disabled
	// - one that is enabled and has valid authenticate message
	// - one that is enabled and has invalid authenticate message
	// Using a 1-letter prefix to ensure consistent ordering

	disabledDevID := "a" + uuid.NewString()
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate)
		VALUES (?, ?)`, disabledDevID, validAuthenticateMessage(disabledDevID))
	execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, NOW())`, disabledDevID, disabledDevID, "Device", "topic", "push_magic", "token_hex")

	validDevID := "b" + uuid.NewString()
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate)
		VALUES (?, ?)`, validDevID, validAuthenticateMessage(validDevID))
	execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, NOW())`, validDevID, validDevID, "Device", "topic", "push_magic", "token_hex")

	invalidDevID := "c" + uuid.NewString()
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate)
		VALUES (?, ?)`, invalidDevID, uuid.NewString())
	execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, NOW())`, invalidDevID, invalidDevID, "Device", "topic", "push_magic", "token_hex")

	tmID := execNoErrLastID(t, db, `INSERT INTO teams (name)
		VALUES (?)`, uuid.NewString())

	applyNext(t, db)

	// existing devices are still there
	var rows []struct {
		ID           string `db:"id"`
		Platform     string `db:"platform"`
		EnrollTeamID *uint  `db:"enroll_team_id"`
	}
	require.NoError(t, db.Select(&rows, `SELECT id, platform, enroll_team_id FROM nano_devices ORDER BY id`))
	require.Len(t, rows, 3)
	// disabled device
	require.Empty(t, rows[0].Platform)
	require.Nil(t, rows[0].EnrollTeamID)
	// valid device got the platform update
	require.Equal(t, "ios", rows[1].Platform)
	require.Nil(t, rows[1].EnrollTeamID)
	// invalid device still there
	require.Empty(t, rows[2].Platform)
	require.Nil(t, rows[2].EnrollTeamID)

	// on the valid device, set the enroll team to the existing team
	execNoErr(t, db, `UPDATE nano_devices SET enroll_team_id = ? WHERE id = ?`, tmID, validDevID)
	rows = rows[:0]
	require.NoError(t, db.Select(&rows, `SELECT id, platform, enroll_team_id FROM nano_devices WHERE id = ?`, validDevID))
	require.Len(t, rows, 1)
	require.Equal(t, "ios", rows[0].Platform)
	require.NotNil(t, rows[0].EnrollTeamID)
	require.EqualValues(t, tmID, *rows[0].EnrollTeamID)

	// deleting the team nulls the enroll team id field
	execNoErr(t, db, `DELETE FROM teams WHERE id = ?`, tmID)
	rows = rows[:0]
	require.NoError(t, db.Select(&rows, `SELECT id, platform, enroll_team_id FROM nano_devices WHERE id = ?`, validDevID))
	require.Len(t, rows, 1)
	require.Equal(t, "ios", rows[0].Platform)
	require.Nil(t, rows[0].EnrollTeamID)
}

func validAuthenticateMessage(devID string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
	<string>%s</string>
</dict>
</plist>`, devID)
}
