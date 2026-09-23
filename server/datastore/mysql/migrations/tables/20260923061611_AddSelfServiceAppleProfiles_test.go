package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260923061611(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
		INSERT INTO mdm_apple_configuration_profiles (profile_uuid, identifier, name, mobileconfig, checksum)
		VALUES ('a_test-profile', 'com.example.test', 'TestAppleProfile', 'x', UNHEX('00'))`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json)
		VALUES ('d_test-declaration', 'com.example.test', 'TestDeclaration', '{}')`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO mdm_windows_configuration_profiles (profile_uuid, name, syncml)
		VALUES ('w_test-profile', 'TestWindowsProfile', 'x')`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO mdm_android_configuration_profiles (profile_uuid, name, raw_json)
		VALUES ('g_test-profile', 'TestAndroidProfile', '{}')`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	var selfService, hidden bool
	err = db.QueryRowContext(t.Context(), `SELECT self_service, hidden FROM mdm_apple_configuration_profiles WHERE profile_uuid = 'a_test-profile'`).Scan(&selfService, &hidden)
	require.NoError(t, err)
	require.False(t, selfService)
	require.False(t, hidden)

	err = db.GetContext(t.Context(), &hidden, `SELECT hidden FROM mdm_apple_declarations WHERE declaration_uuid = 'd_test-declaration'`)
	require.NoError(t, err)
	require.False(t, hidden)

	err = db.GetContext(t.Context(), &hidden, `SELECT hidden FROM mdm_windows_configuration_profiles WHERE profile_uuid = 'w_test-profile'`)
	require.NoError(t, err)
	require.False(t, hidden)

	err = db.GetContext(t.Context(), &hidden, `SELECT hidden FROM mdm_android_configuration_profiles WHERE profile_uuid = 'g_test-profile'`)
	require.NoError(t, err)
	require.False(t, hidden)
}
