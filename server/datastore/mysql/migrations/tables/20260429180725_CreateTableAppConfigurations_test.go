package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260429180725(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a VPP app and an in-house app to use as FK targets.
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, "1234567890", "ios")
	inHouseAppID := execNoErrLastID(t, db, `INSERT INTO in_house_apps (filename, storage_id, platform) VALUES (?, ?, ?)`, "test.ipa", "abc123", "ios")

	// Apply current migration.
	applyNext(t, db)

	// Insert a VPP config and an in-house config — both should succeed.
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		"1234567890", 1, "ios", "<dict><key>foo</key><string>bar</string></dict>")
	execNoErr(t, db, `INSERT INTO in_house_app_configurations (in_house_app_id, team_id, configuration) VALUES (?, ?, ?)`,
		inHouseAppID, 1, "<dict><key>foo</key><string>bar</string></dict>")

	// Same VPP app on a different team — allowed.
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		"1234567890", 2, "ios", "<dict/>")

	// Duplicate (team_id, application_id, platform) — must fail.
	_, err := db.Exec(`INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		"1234567890", 1, "ios", "<dict/>")
	require.Error(t, err)

	// Duplicate (team_id, in_house_app_id) — must fail.
	_, err = db.Exec(`INSERT INTO in_house_app_configurations (in_house_app_id, team_id, configuration) VALUES (?, ?, ?)`,
		inHouseAppID, 1, "<dict/>")
	require.Error(t, err)

	// Unknown (application_id, platform) — composite FK must reject.
	_, err = db.Exec(`INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES (?, ?, ?, ?)`,
		"9999999999", 1, "ios", "<dict/>")
	require.Error(t, err)

	// Unknown in_house_app_id — FK must reject.
	_, err = db.Exec(`INSERT INTO in_house_app_configurations (in_house_app_id, team_id, configuration) VALUES (?, ?, ?)`,
		999999, 1, "<dict/>")
	require.Error(t, err)

	// Cascade delete: removing the parent VPP app drops its configuration rows.
	execNoErr(t, db, `DELETE FROM vpp_apps WHERE adam_id = ? AND platform = ?`, "1234567890", "ios")
	var vppCount int
	require.NoError(t, db.Get(&vppCount, `SELECT COUNT(*) FROM vpp_app_configurations WHERE application_id = ?`, "1234567890"))
	assert.Equal(t, 0, vppCount)

	// Cascade delete: removing the parent in-house app drops its configuration rows.
	execNoErr(t, db, `DELETE FROM in_house_apps WHERE id = ?`, inHouseAppID)
	var inHouseCount int
	require.NoError(t, db.Get(&inHouseCount, `SELECT COUNT(*) FROM in_house_app_configurations WHERE in_house_app_id = ?`, inHouseAppID))
	assert.Equal(t, 0, inHouseCount)
}
