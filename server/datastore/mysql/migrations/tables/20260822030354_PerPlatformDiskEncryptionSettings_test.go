package tables

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260822030354(t *testing.T) {
	db := applyUpToPrev(t)

	// global config: legacy toggle on, with an existing windows_settings object
	// whose sibling keys must survive the rewrite
	execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(
		json_value,
		'$.mdm.enable_disk_encryption', true,
		'$.mdm.windows_settings', CAST('{"custom_settings": [{"path": "a.xml"}]}' AS JSON)
	) WHERE id = 1`)

	// one team per legacy state: true, false, absent, and no mdm object at
	// all; the true team also carries the legacy macos_settings alias the old
	// code persisted, plus a large integer that must survive the rewrite
	execNoErr(t, db, `INSERT INTO teams (name, config) VALUES
		('enabled',  '{"agent_options": {"big": 9007199254740993}, "mdm": {"enable_disk_encryption": true, "macos_settings": {"custom_settings": [], "enable_disk_encryption": true}}}'),
		('disabled', '{"mdm": {"enable_disk_encryption": false}}'),
		('absent',   '{"mdm": {}}'),
		('nomdm',    '{}')`)

	// hand-edited shapes Fleet's marshaler can never produce (struct values
	// always serialize as objects): JSON_SET below a non-object parent is a
	// silent no-op, so the migration must tolerate these rows rather than fail
	// and block the upgrade; the server heals them from the flat toggle on read
	execNoErr(t, db, `INSERT INTO teams (name, config) VALUES
		('nullmdm',   '{"mdm": null}'),
		('nullmacos', '{"mdm": {"enable_disk_encryption": true, "macos_settings": null}}')`)

	// a FileVault profile that must come out of the migration byte-identical
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles
		(profile_uuid, team_id, identifier, name, mobileconfig, checksum, uploaded_at)
		VALUES ('test-fv-uuid', 0, 'com.fleetdm.fleet.mdm.filevault', 'Disk encryption', '<plist>fv-bytes</plist>', UNHEX(MD5('<plist>fv-bytes</plist>')), '2026-01-02 03:04:05')`)

	applyNext(t, db)

	assertDiskEncryptionKeys := func(t *testing.T, raw json.RawMessage, want bool) {
		t.Helper()
		var cfg map[string]any
		require.NoError(t, json.Unmarshal(raw, &cfg))
		mdm, ok := cfg["mdm"].(map[string]any)
		require.True(t, ok, "mdm object should be present")
		for objKey, key := range map[string]string{
			"macos_settings":   "enable_disk_encryption",
			"windows_settings": "enable_disk_encryption",
			"linux_settings":   "enable_escrow_disk_encryption_key",
		} {
			obj, ok := mdm[objKey].(map[string]any)
			require.True(t, ok, "%s object should be present", objKey)
			require.Equal(t, want, obj[key], "%s.%s", objKey, key)
		}
		macos := mdm["macos_settings"].(map[string]any)
		require.Equal(t, want, macos["enable_escrow_disk_encryption_key"], "macos_settings.enable_escrow_disk_encryption_key")
		// the flat toggle is left in place untouched
		if _, present := mdm["enable_disk_encryption"]; present {
			require.Equal(t, want, mdm["enable_disk_encryption"])
		}
	}

	var raw json.RawMessage
	require.NoError(t, sqlx.Get(db, &raw, `SELECT json_value FROM app_config_json WHERE id = 1`))
	assertDiskEncryptionKeys(t, raw, true)

	// sibling keys under windows_settings survive
	var customSettings string
	require.NoError(t, sqlx.Get(db, &customSettings, `SELECT json_value->>'$.mdm.windows_settings.custom_settings[0].path' FROM app_config_json WHERE id = 1`))
	require.Equal(t, "a.xml", customSettings)

	for _, tc := range []struct {
		team string
		want bool
	}{
		{"enabled", true},
		{"disabled", false},
		{"absent", false},
		{"nomdm", false},
	} {
		t.Run(fmt.Sprintf("team %s", tc.team), func(t *testing.T) {
			var raw json.RawMessage
			require.NoError(t, sqlx.Get(db, &raw, `SELECT config FROM teams WHERE name = ?`, tc.team))
			assertDiskEncryptionKeys(t, raw, tc.want)
		})
	}

	// a JSON-null mdm parent comes out byte-identical
	var nullMDMConfig string
	require.NoError(t, sqlx.Get(db, &nullMDMConfig, `SELECT config FROM teams WHERE name = 'nullmdm'`))
	require.JSONEq(t, `{"mdm": null}`, nullMDMConfig)

	// a JSON-null macos_settings stays null, while the sibling sections and
	// the flat toggle are still written correctly
	var nullMacOSConfig string
	require.NoError(t, sqlx.Get(db, &nullMacOSConfig, `SELECT config FROM teams WHERE name = 'nullmacos'`))
	require.JSONEq(t, `{"mdm": {
		"enable_disk_encryption": true,
		"macos_settings": null,
		"windows_settings": {"enable_disk_encryption": true},
		"linux_settings": {"enable_escrow_disk_encryption_key": true}
	}}`, nullMacOSConfig)

	// large integers elsewhere in the config survive the rewrite
	var bigNumber string
	require.NoError(t, sqlx.Get(db, &bigNumber, `SELECT config->>'$.agent_options.big' FROM teams WHERE name = 'enabled'`))
	require.Equal(t, "9007199254740993", bigNumber)

	// the FileVault profile row is byte-identical: no content, checksum, or
	// uploaded_at change means nothing is re-sent to hosts
	var prof struct {
		Mobileconfig string `db:"mobileconfig"`
		ChecksumOK   bool   `db:"checksum_ok"`
		UploadedAt   string `db:"uploaded_at"`
	}
	require.NoError(t, sqlx.Get(db, &prof, `SELECT mobileconfig,
		checksum = UNHEX(MD5('<plist>fv-bytes</plist>')) AS checksum_ok,
		uploaded_at FROM mdm_apple_configuration_profiles WHERE profile_uuid = 'test-fv-uuid'`))
	require.Equal(t, "<plist>fv-bytes</plist>", prof.Mobileconfig)
	require.True(t, prof.ChecksumOK)
	require.Equal(t, "2026-01-02T03:04:05Z", prof.UploadedAt)
}
