package tables

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261021195658(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid) VALUES ('host', 'host', 'host', 'host')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('ios-app', 'ios')`)
	configuredVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('ios-app', 'ios', 0, 'Configured', '<dict></dict>')`)
	installID := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-app', 'ios', 'install')`, hostID)
	variablesVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('ios-app', 'ios', 0, 'Variables', '<dict><key>a</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART</string><key>b</key><string>${FLEET_VAR_HOST_UUID}</string></dict>')`)
	notVariablesVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('ios-app', 'ios', 0, 'Not variables', '<dict><key>a</key><string>$fleet_var_host_uuid</string><key>b</key><string>$FLEET_VAR_host_uuid</string><key>c</key><string>${FLEET_VAR_HOST_UUID</string></dict>')`)
	// Configurations with Fleet variables written in ways the save path does and doesn't treat as variables, with the variables it links
	configurationCases := []struct {
		configuration     string
		expectedVariables []string
	}{
		{`<string>$FLEET_VAR_HOST_UUID</string>`, []string{"FLEET_VAR_HOST_UUID"}},
		{`$FLEET_VAR_HOST_UUID`, []string{"FLEET_VAR_HOST_UUID"}},
		{`<string>${FLEET_VAR_HOST_UUID}x</string>`, []string{"FLEET_VAR_HOST_UUID"}},
		{`<string>${FLEET_VAR_HOST_UUID}}</string>`, []string{"FLEET_VAR_HOST_UUID"}},
		{`<string>$$FLEET_VAR_HOST_UUID</string>`, []string{"FLEET_VAR_HOST_UUID"}},
		{`<string>$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT-x</string>`, []string{"FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT"}},
		{`<key>$FLEET_VAR_HOST_PLATFORM</key><string>a</string>`, []string{"FLEET_VAR_HOST_PLATFORM"}},
		{`<!-- $FLEET_VAR_HOST_HARDWARE_SERIAL --><dict/>`, []string{"FLEET_VAR_HOST_HARDWARE_SERIAL"}},
		{`<string><![CDATA[${FLEET_VAR_HOST_END_USER_IDP_GROUPS}]]></string>`, []string{"FLEET_VAR_HOST_END_USER_IDP_GROUPS"}},
		{"<string>café\n$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME\n</string>", []string{"FLEET_VAR_HOST_END_USER_IDP_FULL_NAME"}},
		{`<string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP ${FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID}</string>`, []string{"FLEET_VAR_HOST_END_USER_EMAIL_IDP", "FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID"}},
		{`<string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME $FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART</string>`, []string{"FLEET_VAR_HOST_END_USER_IDP_USERNAME", "FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART"}},
		{`<string>$FLEET_VAR_HOST_UUIDX</string>`, nil},
		{`<string>$FLEET_VAR_HOST_UUID1</string>`, nil},
		{`<string>$FLEET_VAR_HOST_UUID_LOCAL</string>`, nil},
		{`<string>${FLEET_VAR_HOST_UUID_X}</string>`, nil},
		{`<string>${ FLEET_VAR_HOST_UUID}</string>`, nil},
		{`<string>&#36;FLEET_VAR_HOST_UUID</string>`, nil},
		{`<string>$FLEET_VAR_NOT_REAL</string>`, nil},
		{`<string>$Fleet_Var_Host_Uuid</string>`, nil},
	}
	caseVersionIDs := make([]int64, len(configurationCases))
	for i, c := range configurationCases {
		caseVersionIDs[i] = execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('ios-app', 'ios', 0, ?, ?)`, fmt.Sprintf("Case %d", i), c.configuration)
	}
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('android-app', 'android')`)
	androidVersionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name, configuration) VALUES ('android-app', 'android', 0, 'Android', '{"managedConfiguration": {"a": "$FLEET_VAR_HOST_UUID"}}')`)
	const backdated = "2026-01-02 03:04:05"
	execNoErr(t, db, `UPDATE vpp_apps_teams SET updated_at = ? WHERE id = ?`, backdated, configuredVersionID)
	execNoErr(t, db, `UPDATE host_vpp_software_installs SET updated_at = ? WHERE id = ?`, backdated, installID)

	applyNext(t, db)

	// read the updated_at of the configured version and the existing install, adding the columns should not change them
	var versionUpdatedAtUnchanged bool
	err := db.Get(&versionUpdatedAtUnchanged, `SELECT updated_at = ? FROM vpp_apps_teams WHERE id = ?`, backdated, configuredVersionID)
	require.NoError(t, err)
	require.True(t, versionUpdatedAtUnchanged)
	var installUpdatedAtUnchanged bool
	err = db.Get(&installUpdatedAtUnchanged, `SELECT updated_at = ? FROM host_vpp_software_installs WHERE id = ?`, backdated, installID)
	require.NoError(t, err)
	require.True(t, installUpdatedAtUnchanged)

	// read configuration_updated_at of the configured version, it should be NULL so the cron doesn't send the version's existing installs again
	var configurationUpdatedAt sql.NullTime
	err = db.Get(&configurationUpdatedAt, `SELECT configuration_updated_at FROM vpp_apps_teams WHERE id = ?`, configuredVersionID)
	require.NoError(t, err)
	require.False(t, configurationUpdatedAt.Valid)

	// read needs_resend of the existing install, it should be 0 so the cron doesn't send the install again
	var needsResend bool
	err = db.Get(&needsResend, `SELECT needs_resend FROM host_vpp_software_installs WHERE id = ?`, installID)
	require.NoError(t, err)
	require.False(t, needsResend)

	// read the variables linked to the iOS configuration, the local part variable and the UUID variable in braces should be linked but not the shorter username variable
	var linkedVariables []string
	err = db.Select(&linkedVariables, `SELECT fv.name FROM mdm_configuration_profile_variables mcpv JOIN fleet_variables fv ON fv.id = mcpv.fleet_variable_id WHERE mcpv.vpp_app_team_id = ? ORDER BY fv.name`, variablesVersionID)
	require.NoError(t, err)
	require.Equal(t, []string{"FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART", "FLEET_VAR_HOST_UUID"}, linkedVariables)

	// read the variables linked to the iOS configuration with lowercase names and a brace that isn't closed, the migration should not link any
	var notVariablesLinks int
	err = db.Get(&notVariablesLinks, `SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE vpp_app_team_id = ?`, notVariablesVersionID)
	require.NoError(t, err)
	require.Zero(t, notVariablesLinks)

	// read the variables linked to each case's configuration, they should be the variables the save path links
	for i, c := range configurationCases {
		var caseVariables []string
		err = db.Select(&caseVariables, `SELECT fv.name FROM mdm_configuration_profile_variables mcpv JOIN fleet_variables fv ON fv.id = mcpv.fleet_variable_id WHERE mcpv.vpp_app_team_id = ? ORDER BY fv.name`, caseVersionIDs[i])
		require.NoError(t, err)
		require.ElementsMatch(t, c.expectedVariables, caseVariables, c.configuration)
	}

	// read the variables linked to the Android configuration, the migration should not link any
	var androidLinks int
	err = db.Get(&androidLinks, `SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE vpp_app_team_id = ?`, androidVersionID)
	require.NoError(t, err)
	require.Zero(t, androidLinks)
}
