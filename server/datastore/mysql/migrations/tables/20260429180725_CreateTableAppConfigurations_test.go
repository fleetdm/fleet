package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260429180725(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed parent rows: same VPP adam_id on both platforms, and a separate
	// in_house_apps row per platform (in-house apps are platform-specific).
	const adamID = "1234567890"
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, adamID, "ios")
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, adamID, "ipados")
	iosAppID := execNoErrLastID(t, db, `INSERT INTO in_house_apps (filename, storage_id, platform) VALUES (?, ?, ?)`, "test.ipa", "abc123", "ios")
	ipadosAppID := execNoErrLastID(t, db, `INSERT INTO in_house_apps (filename, storage_id, platform) VALUES (?, ?, ?)`, "test.ipa", "def456", "ipados")

	// Apply current migration.
	applyNext(t, db)

	// Realistic plist managed-config payload (multi-line raw string).
	const iosPlist = `<dict>
	<key>ServerURL</key>
	<string>https://fleetdm.com</string>
	<key>EnableTelemetry</key>
	<true/>
	<key>MaxRetries</key>
	<integer>5</integer>
</dict>`

	// VPP: same adam_id can be configured for ios and ipados independently on the same team.
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		adamID, 1, "ios", iosPlist)
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		adamID, 1, "ipados", "<dict><key>platform</key><string>ipados</string></dict>")

	// Round-trip the plist back out to make sure MEDIUMTEXT preserves it byte-for-byte
	// (whitespace, newlines, and angle brackets all intact).
	var got string
	require.NoError(t, db.Get(&got, `SELECT configuration FROM vpp_app_configurations WHERE application_id = ? AND team_id = ? AND platform = ?`,
		adamID, 1, "ios"))
	assert.Equal(t, iosPlist, got)

	// VPP: same adam_id+platform on a different team is allowed.
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		adamID, 2, "ios", "<dict/>")

	// In-house: each in_house_apps row (one per platform, already team-scoped) gets its own config.
	execNoErr(t, db, `INSERT INTO in_house_app_configurations (in_house_app_id, configuration) VALUES (?, ?)`,
		iosAppID, "<dict><key>platform</key><string>ios</string></dict>")
	execNoErr(t, db, `INSERT INTO in_house_app_configurations (in_house_app_id, configuration) VALUES (?, ?)`,
		ipadosAppID, "<dict><key>platform</key><string>ipados</string></dict>")

	// VPP duplicate (team_id, application_id, platform) — must fail.
	_, err := db.Exec(`INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		adamID, 1, "ios", "<dict/>")
	require.Error(t, err)

	// In-house duplicate in_house_app_id — must fail.
	_, err = db.Exec(`INSERT INTO in_house_app_configurations (in_house_app_id, configuration) VALUES (?, ?)`,
		iosAppID, "<dict/>")
	require.Error(t, err)

	// VPP composite FK rejects an adam_id that exists only for the other platform.
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, "iosonly", "ios")
	_, err = db.Exec(`INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		"iosonly", 1, "ipados", "<dict/>")
	require.Error(t, err)

	// VPP composite FK rejects an unknown adam_id.
	_, err = db.Exec(`INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		"9999999999", 1, "ios", "<dict/>")
	require.Error(t, err)

	// In-house FK rejects an unknown id.
	_, err = db.Exec(`INSERT INTO in_house_app_configurations (in_house_app_id, configuration) VALUES (?, ?)`,
		999999, "<dict/>")
	require.Error(t, err)

	// Cascade: deleting only the iOS row from vpp_apps drops its config but leaves the iPadOS config intact.
	execNoErr(t, db, `DELETE FROM vpp_apps WHERE adam_id = ? AND platform = ?`, adamID, "ios")
	var vppCount int
	require.NoError(t, db.Get(&vppCount, `SELECT COUNT(*) FROM vpp_app_configurations WHERE application_id = ? AND platform = ?`, adamID, "ios"))
	assert.Equal(t, 0, vppCount)
	require.NoError(t, db.Get(&vppCount, `SELECT COUNT(*) FROM vpp_app_configurations WHERE application_id = ? AND platform = ?`, adamID, "ipados"))
	assert.Equal(t, 1, vppCount)

	// Cascade: deleting only the iOS in_house_apps row drops its config but leaves iPadOS intact.
	execNoErr(t, db, `DELETE FROM in_house_apps WHERE id = ?`, iosAppID)
	var inHouseCount int
	require.NoError(t, db.Get(&inHouseCount, `SELECT COUNT(*) FROM in_house_app_configurations WHERE in_house_app_id = ?`, iosAppID))
	assert.Equal(t, 0, inHouseCount)
	require.NoError(t, db.Get(&inHouseCount, `SELECT COUNT(*) FROM in_house_app_configurations WHERE in_house_app_id = ?`, ipadosAppID))
	assert.Equal(t, 1, inHouseCount)
}
