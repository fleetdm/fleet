package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261020200751(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid) VALUES ('host', 'host', 'host', 'host')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('ios-app', 'ios')`)
	configuredVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('ios-app', 'ios', 0, 'Configured', '<dict></dict>')`)
	installID := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-app', 'ios', 'install')`, hostID)
	variablesVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('ios-app', 'ios', 0, 'Variables', '<dict><key>a</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART</string><key>b</key><string>${FLEET_VAR_HOST_UUID}</string></dict>')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('android-app', 'android')`)
	androidVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('android-app', 'android', 0, 'Android', '{"managedConfiguration": {"a": "$FLEET_VAR_HOST_UUID"}}')`)

	applyNext(t, db)

	// read the configured version, it should have no configuration time so its existing installs aren't sent again
	var configurationUpdatedAt sql.NullTime
	err := db.Get(&configurationUpdatedAt, `SELECT configuration_updated_at FROM vpp_apps_teams WHERE id = ?`, configuredVersionID)
	require.NoError(t, err)
	require.False(t, configurationUpdatedAt.Valid)

	// read the existing install, it should not need a configuration re-send
	var needsConfigurationResend bool
	err = db.Get(&needsConfigurationResend, `SELECT needs_configuration_resend FROM host_vpp_software_installs WHERE id = ?`, installID)
	require.NoError(t, err)
	require.False(t, needsConfigurationResend)

	// read the variables linked to the iOS configuration, the local part variable and the UUID variable in braces should be linked but not the shorter username variable
	var linkedVariables []string
	err = db.Select(&linkedVariables, `SELECT fv.name FROM mdm_configuration_profile_variables mcpv JOIN fleet_variables fv ON fv.id = mcpv.fleet_variable_id WHERE mcpv.vpp_app_team_id = ? ORDER BY fv.name`, variablesVersionID)
	require.NoError(t, err)
	require.Equal(t, []string{"FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART", "FLEET_VAR_HOST_UUID"}, linkedVariables)

	// read the variables linked to the Android configuration, the migration should not link any
	var androidLinks int
	err = db.Get(&androidLinks, `SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE vpp_app_team_id = ?`, androidVersionID)
	require.NoError(t, err)
	require.Zero(t, androidLinks)
}
