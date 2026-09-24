package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260924191956(t *testing.T) {
	db := applyUpToPrev(t)

	teamA := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team-a')`)
	teamB := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team-b')`)

	hostInTeamA := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, team_id) VALUES ('host-a', 'host-a', 'host-a', 'host-a', ?)`, teamA)
	hostInNoTeam := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid) VALUES ('host-none', 'host-none', 'host-none', 'host-none')`)

	iosInTeamATitle := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('ios-team-a', 'ios_apps')`)
	iosInNoTeamTitle := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('ios-no-team', 'ios_apps')`)
	iosOtherFleetTitle := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('ios-other-fleet', 'ios_apps')`)

	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id) VALUES ('ios-team-a', 'ios', ?)`, iosInTeamATitle)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id) VALUES ('ios-no-team', 'ios', ?)`, iosInNoTeamTitle)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id) VALUES ('ios-other-fleet', 'ios', ?)`, iosOtherFleetTitle)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('ios-and-ipados', 'ios'), ('ios-and-ipados', 'ipados')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('ios-nothing', 'ios')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('ios-many-fleets', 'ios'), ('ios-team-b-only', 'ios')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('android-team-a', 'android'), ('android-no-team', 'android'), ('android-other-fleet', 'android'), ('android-global-config-only', 'android')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('macos-team-a', 'darwin')`)

	iosInTeamA := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-team-a', 'ios', ?, ?)`, teamA, teamA)
	iosInNoTeam := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id) VALUES ('ios-no-team', 'ios', 0)`)
	iosOtherFleet := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-other-fleet', 'ios', ?, ?)`, teamA, teamA)
	iosOfIOSAndIPadOS := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-and-ipados', 'ios', ?, ?)`, teamA, teamA)
	ipadosOfIOSAndIPadOS := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-and-ipados', 'ipados', ?, ?)`, teamA, teamA)
	iosNothing := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-nothing', 'ios', ?, ?)`, teamA, teamA)
	manyFleetsInTeamA := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-many-fleets', 'ios', ?, ?)`, teamA, teamA)
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-many-fleets', 'ios', ?, ?)`, teamB, teamB)
	manyFleetsInNoTeam := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id) VALUES ('ios-many-fleets', 'ios', 0)`)
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-team-b-only', 'ios', ?, ?)`, teamB, teamB)
	androidInTeamA := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('android-team-a', 'android', ?, ?)`, teamA, teamA)
	androidInNoTeam := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id) VALUES ('android-no-team', 'android', 0)`)
	androidOtherFleet := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('android-other-fleet', 'android', ?, ?)`, teamA, teamA)
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('android-global-config-only', 'android', ?, ?)`, teamA, teamA)
	macosInTeamA := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('macos-team-a', 'darwin', ?, ?)`, teamA, teamA)

	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES ('ios-team-a', ?, 'ios', '<dict>team-a</dict>')`, teamA)
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES ('ios-no-team', 0, 'ios', '<dict>no-team</dict>')`)
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES ('ios-other-fleet', ?, 'ios', '<dict>team-b</dict>')`, teamB)
	execNoErr(t, db, `INSERT INTO vpp_app_configurations (application_id, team_id, platform, configuration) VALUES ('ios-and-ipados', ?, 'ipados', '<dict>ipados</dict>')`, teamA)

	androidTeamAConfig := execNoErrLastID(t, db, `INSERT INTO android_app_configurations (application_id, team_id, global_or_team_id, configuration) VALUES ('android-team-a', ?, ?, '{"managedConfiguration": {"key": "team-a"}}')`, teamA, teamA)
	androidNoTeamConfig := execNoErrLastID(t, db, `INSERT INTO android_app_configurations (application_id, global_or_team_id, configuration) VALUES ('android-no-team', 0, '{"managedConfiguration": {"key": "no-team"}}')`)
	androidOtherFleetConfig := execNoErrLastID(t, db, `INSERT INTO android_app_configurations (application_id, team_id, global_or_team_id, configuration) VALUES ('android-other-fleet', ?, ?, '{"managedConfiguration": {"key": "team-b"}}')`, teamB, teamB)
	androidGlobalOnlyConfig := execNoErrLastID(t, db, `INSERT INTO android_app_configurations (application_id, global_or_team_id, configuration) VALUES ('android-global-config-only', 0, '{"managedConfiguration": {"key": "global"}}')`)

	execNoErr(t, db, `INSERT INTO software_update_schedules (team_id, title_id, enabled, start_time, end_time) VALUES (?, ?, 1, '01:00', '03:00')`, teamA, iosInTeamATitle)
	execNoErr(t, db, `INSERT INTO software_update_schedules (team_id, title_id, enabled, start_time, end_time) VALUES (0, ?, 1, '04:00', '06:00')`, iosInNoTeamTitle)
	execNoErr(t, db, `INSERT INTO software_update_schedules (team_id, title_id, enabled, start_time, end_time) VALUES (?, ?, 1, '07:00', '09:00')`, teamB, iosOtherFleetTitle)

	installOnlyInHostFleet := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-team-a', 'ios', 'install-1')`, hostInTeamA)
	installInHostAndOtherFleets := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-many-fleets', 'ios', 'install-2')`, hostInTeamA)
	installOnlyInOtherFleet := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-team-b-only', 'ios', 'install-3')`, hostInTeamA)
	noTeamInstallOnlyInHostFleet := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-no-team', 'ios', 'install-4')`, hostInNoTeam)
	noTeamInstallInHostAndOtherFleets := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-many-fleets', 'ios', 'install-5')`, hostInNoTeam)
	noTeamInstallOnlyInOtherFleet := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid) VALUES (?, 'ios-team-b-only', 'ios', 'install-6')`, hostInNoTeam)

	queuedOnlyInHostFleet := execNoErrLastID(t, db, `INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload) VALUES (?, 'vpp_app_install', 'queued-1', '{}')`, hostInTeamA)
	execNoErr(t, db, `INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform) VALUES (?, 'ios-team-a', 'ios')`, queuedOnlyInHostFleet)
	noTeamQueuedOnlyInHostFleet := execNoErrLastID(t, db, `INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload) VALUES (?, 'vpp_app_install', 'queued-2', '{}')`, hostInNoTeam)
	execNoErr(t, db, `INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform) VALUES (?, 'ios-no-team', 'ios')`, noTeamQueuedOnlyInHostFleet)
	queuedOnlyInOtherFleet := execNoErrLastID(t, db, `INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload) VALUES (?, 'vpp_app_install', 'queued-3', '{}')`, hostInTeamA)
	execNoErr(t, db, `INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform) VALUES (?, 'ios-team-b-only', 'ios')`, queuedOnlyInOtherFleet)

	var hostUUIDVariableID int64
	err := db.Get(&hostUUIDVariableID, `SELECT id FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_UUID'`)
	require.NoError(t, err)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, fleet_variable_id) VALUES (?, ?)`, androidTeamAConfig, hostUUIDVariableID)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, fleet_variable_id) VALUES (?, ?)`, androidNoTeamConfig, hostUUIDVariableID)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, fleet_variable_id) VALUES (?, ?)`, androidOtherFleetConfig, hostUUIDVariableID)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, fleet_variable_id) VALUES (?, ?)`, androidGlobalOnlyConfig, hostUUIDVariableID)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES ('w-profile', ?, 'windows-profile', '<SyncML/>')`, teamA)
	windowsProfileLink := execNoErrLastID(t, db, `INSERT INTO mdm_configuration_profile_variables (windows_profile_uuid, fleet_variable_id) VALUES ('w-profile', ?)`, hostUUIDVariableID)

	applyNext(t, db)

	type appStoreAppRow struct {
		InstanceName          string  `db:"instance_name"`
		Configuration         *string `db:"configuration"`
		UpdateScheduleEnabled bool    `db:"update_schedule_enabled"`
		StartTime             string  `db:"start_time"`
		EndTime               string  `db:"end_time"`
	}
	const selectAppStoreApp = `SELECT instance_name, configuration, update_schedule_enabled, start_time, end_time FROM vpp_apps_teams WHERE id = ?`

	// read every existing app, it should be named Default version
	var instanceNames []string
	err = db.Select(&instanceNames, `SELECT DISTINCT instance_name FROM vpp_apps_teams`)
	require.NoError(t, err)
	require.Equal(t, []string{"Default version"}, instanceNames)

	// read the iOS app in team A, its configuration and update schedule from team A should be copied
	var app appStoreAppRow
	err = db.Get(&app, selectAppStoreApp, iosInTeamA)
	require.NoError(t, err)
	require.NotNil(t, app.Configuration)
	require.Equal(t, "<dict>team-a</dict>", *app.Configuration)
	require.True(t, app.UpdateScheduleEnabled)
	require.Equal(t, "01:00", app.StartTime)
	require.Equal(t, "03:00", app.EndTime)

	// read the iOS app in no team, its configuration and update schedule from no team should be copied
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, iosInNoTeam)
	require.NoError(t, err)
	require.NotNil(t, app.Configuration)
	require.Equal(t, "<dict>no-team</dict>", *app.Configuration)
	require.True(t, app.UpdateScheduleEnabled)
	require.Equal(t, "04:00", app.StartTime)
	require.Equal(t, "06:00", app.EndTime)

	// read the iOS app in team A whose configuration and schedule belong to team B, nothing should be copied
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, iosOtherFleet)
	require.NoError(t, err)
	require.Nil(t, app.Configuration)
	require.False(t, app.UpdateScheduleEnabled)
	require.Empty(t, app.StartTime)
	require.Empty(t, app.EndTime)

	// read the iOS and iPadOS rows of one app with only an iPadOS configuration, only the iPadOS row should get it
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, iosOfIOSAndIPadOS)
	require.NoError(t, err)
	require.Nil(t, app.Configuration)
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, ipadosOfIOSAndIPadOS)
	require.NoError(t, err)
	require.NotNil(t, app.Configuration)
	require.Equal(t, "<dict>ipados</dict>", *app.Configuration)

	// read the iOS app with no configuration or schedule, the columns should hold their defaults
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, iosNothing)
	require.NoError(t, err)
	require.Nil(t, app.Configuration)
	require.False(t, app.UpdateScheduleEnabled)
	require.Empty(t, app.StartTime)
	require.Empty(t, app.EndTime)

	// read the Android apps, the configuration from the same fleet should be copied as JSON text
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, androidInTeamA)
	require.NoError(t, err)
	require.NotNil(t, app.Configuration)
	require.JSONEq(t, `{"managedConfiguration": {"key": "team-a"}}`, *app.Configuration)
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, androidInNoTeam)
	require.NoError(t, err)
	require.NotNil(t, app.Configuration)
	require.JSONEq(t, `{"managedConfiguration": {"key": "no-team"}}`, *app.Configuration)
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, androidOtherFleet)
	require.NoError(t, err)
	require.Nil(t, app.Configuration)

	// read the macOS app, only the instance name should be set
	app = appStoreAppRow{}
	err = db.Get(&app, selectAppStoreApp, macosInTeamA)
	require.NoError(t, err)
	require.Equal(t, "Default version", app.InstanceName)
	require.Nil(t, app.Configuration)
	require.False(t, app.UpdateScheduleEnabled)

	// read the installs, each should link to the app in the host's current fleet or to nothing
	const selectInstallAppTeamID = `SELECT vpp_app_team_id FROM host_vpp_software_installs WHERE id = ?`
	var linkedAppTeamID *int64
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, installOnlyInHostFleet)
	require.NoError(t, err)
	require.NotNil(t, linkedAppTeamID)
	require.Equal(t, iosInTeamA, *linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, installInHostAndOtherFleets)
	require.NoError(t, err)
	require.NotNil(t, linkedAppTeamID)
	require.Equal(t, manyFleetsInTeamA, *linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, installOnlyInOtherFleet)
	require.NoError(t, err)
	require.Nil(t, linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, noTeamInstallOnlyInHostFleet)
	require.NoError(t, err)
	require.NotNil(t, linkedAppTeamID)
	require.Equal(t, iosInNoTeam, *linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, noTeamInstallInHostAndOtherFleets)
	require.NoError(t, err)
	require.NotNil(t, linkedAppTeamID)
	require.Equal(t, manyFleetsInNoTeam, *linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, noTeamInstallOnlyInOtherFleet)
	require.NoError(t, err)
	require.Nil(t, linkedAppTeamID)

	// read the queued installs, each should link to the app in the host's current fleet or to nothing
	const selectQueuedAppTeamID = `SELECT vpp_app_team_id FROM vpp_app_upcoming_activities WHERE upcoming_activity_id = ?`
	err = db.Get(&linkedAppTeamID, selectQueuedAppTeamID, queuedOnlyInHostFleet)
	require.NoError(t, err)
	require.NotNil(t, linkedAppTeamID)
	require.Equal(t, iosInTeamA, *linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectQueuedAppTeamID, noTeamQueuedOnlyInHostFleet)
	require.NoError(t, err)
	require.NotNil(t, linkedAppTeamID)
	require.Equal(t, iosInNoTeam, *linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectQueuedAppTeamID, queuedOnlyInOtherFleet)
	require.NoError(t, err)
	require.Nil(t, linkedAppTeamID)

	// read the Android variable links, the ones with an app in the configuration's fleet should point at that app and the rest should be deleted
	var variableLinkAppTeamIDs []int64
	err = db.Select(&variableLinkAppTeamIDs, `SELECT vpp_app_team_id FROM mdm_configuration_profile_variables WHERE vpp_app_team_id IS NOT NULL ORDER BY vpp_app_team_id`)
	require.NoError(t, err)
	require.Equal(t, []int64{androidInTeamA, androidInNoTeam}, variableLinkAppTeamIDs)

	// read the Windows profile variable link, it should be unchanged
	var windowsLinkCount int
	err = db.Get(&windowsLinkCount, `SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE id = ? AND windows_profile_uuid = 'w-profile' AND vpp_app_team_id IS NULL`, windowsProfileLink)
	require.NoError(t, err)
	require.Equal(t, 1, windowsLinkCount)

	// look up the dropped tables and column, none should exist
	var droppedTableCount int
	err = db.Get(&droppedTableCount, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN ('vpp_app_configurations', 'android_app_configurations', 'software_update_schedules')`)
	require.NoError(t, err)
	require.Zero(t, droppedTableCount)
	var droppedColumnCount int
	err = db.Get(&droppedColumnCount, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'mdm_configuration_profile_variables' AND column_name = 'android_app_configuration_id'`)
	require.NoError(t, err)
	require.Zero(t, droppedColumnCount)

	// insert a second instance of an app on the same fleet with a new name, it should be allowed
	execNoErr(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id, instance_name) VALUES ('ios-team-a', 'ios', ?, ?, 'Second')`, teamA, teamA)

	// insert the same instance name again for the same app and fleet, it should be rejected
	_, err = db.Exec(`INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id, instance_name) VALUES ('ios-team-a', 'ios', ?, ?, 'Second')`, teamA, teamA)
	require.ErrorContains(t, err, "Duplicate entry")

	// insert an app without an instance name, it should be rejected
	_, err = db.Exec(`INSERT INTO vpp_apps_teams (adam_id, platform, team_id, global_or_team_id) VALUES ('ios-nothing', 'ios', ?, ?)`, teamB, teamB)
	require.ErrorContains(t, err, "instance_name")

	// insert a variable link that points at both an app and a Windows profile, it should be rejected by the check
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (vpp_app_team_id, windows_profile_uuid, fleet_variable_id) VALUES (?, 'w-profile', ?)`, iosInTeamA, hostUUIDVariableID)
	require.Error(t, err)

	// insert the same app variable link twice, the second should be rejected
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (vpp_app_team_id, fleet_variable_id) VALUES (?, ?)`, androidInTeamA, hostUUIDVariableID)
	require.ErrorContains(t, err, "Duplicate entry")

	// delete the Android app in team A, its variable link should be deleted
	execNoErr(t, db, `DELETE FROM vpp_apps_teams WHERE id = ?`, androidInTeamA)
	var deletedAppLinkCount int
	err = db.Get(&deletedAppLinkCount, `SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE vpp_app_team_id = ?`, androidInTeamA)
	require.NoError(t, err)
	require.Zero(t, deletedAppLinkCount)

	// delete the iOS app in team A, its install and queued install should keep their rows with no linked app
	execNoErr(t, db, `DELETE FROM vpp_apps_teams WHERE id = ?`, iosInTeamA)
	err = db.Get(&linkedAppTeamID, selectInstallAppTeamID, installOnlyInHostFleet)
	require.NoError(t, err)
	require.Nil(t, linkedAppTeamID)
	err = db.Get(&linkedAppTeamID, selectQueuedAppTeamID, queuedOnlyInHostFleet)
	require.NoError(t, err)
	require.Nil(t, linkedAppTeamID)
}
