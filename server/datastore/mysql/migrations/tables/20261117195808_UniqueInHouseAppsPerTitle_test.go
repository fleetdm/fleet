package tables

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261117195808(t *testing.T) {
	db := applyUpToPrev(t)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1')`)

	iosTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier, extension_for) VALUES ('HelloWorld', 'ios_apps', 'com.example.helloworld', '')`)
	ipadosTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier, extension_for) VALUES ('HelloWorld', 'ipados_apps', 'com.example.helloworld', '')`)

	insertInHouseApp := func(teamID *int64, titleID *int64, filename string, version string, platform string) int64 {
		var globalOrTeamID int64
		if teamID != nil {
			globalOrTeamID = *teamID
		}
		return execNoErrLastID(t, db, `
			INSERT INTO in_house_apps (title_id, team_id, global_or_team_id, filename, version, storage_id, platform, bundle_identifier)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'com.example.helloworld')`,
			titleID, teamID, globalOrTeamID, filename, version, "storage-"+filename+"-"+platform, platform)
	}

	// Team iOS title: the GitOps row has a higher id but its filename sorts first, so it is the row
	// the UI shows and the one kept.
	teamUploadedApp := insertInHouseApp(&teamID, &iosTitleID, "HelloWorld.ipa", "1.0", "ios")
	teamGitOpsApp := insertInHouseApp(&teamID, &iosTitleID, "AHelloWorld.ipa", "1.1", "ios")
	// Team iPadOS title with a single row, which is not changed.
	teamIpadOSApp := insertInHouseApp(&teamID, &ipadosTitleID, "HelloWorld.ipa", "1.0", "ipados")
	// Team rows without a title, which are not changed.
	teamNoTitleApp1 := insertInHouseApp(&teamID, nil, "NoTitle1.ipa", "1.0", "ios")
	teamNoTitleApp2 := insertInHouseApp(&teamID, nil, "NoTitle2.ipa", "1.0", "ios")
	// Unassigned iOS title: the first uploaded row has the filename that sorts first, so it is kept.
	unassignedUploadedApp := insertInHouseApp(nil, &iosTitleID, "HelloWorld.ipa", "1.0", "ios")
	unassignedGitOpsApp := insertInHouseApp(nil, &iosTitleID, "HelloWorld1.1.ipa", "1.1", "ios")

	// Add more installs than one batch to the team row that is deleted, and one to the kept row.
	installValues := make([]string, 0, hostInHouseInstallsBatchSize+1)
	installArgs := make([]any, 0, 2*(hostInHouseInstallsBatchSize+1))
	for i := range hostInHouseInstallsBatchSize + 1 {
		installValues = append(installValues, "(?, ?, ?, 'ios')")
		installArgs = append(installArgs, i+1, teamUploadedApp, fmt.Sprintf("team-uploaded-install-%d", i))
	}
	execNoErr(t, db,
		`INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, command_uuid, platform) VALUES `+strings.Join(installValues, ","),
		installArgs...)
	execNoErr(t, db,
		`INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, command_uuid, platform) VALUES (1, ?, 'team-gitops-install', 'ios')`,
		teamGitOpsApp)
	execNoErr(t, db,
		`INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, command_uuid, platform) VALUES (2, ?, 'unassigned-gitops-install', 'ios')`,
		unassignedGitOpsApp)

	// Add an upcoming install to each row that is deleted.
	teamUpcomingActivity := execNoErrLastID(t, db, `
		INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
		VALUES (3, 'in_house_app_install', 'team-upcoming-install', '{}')`)
	execNoErr(t, db,
		`INSERT INTO in_house_app_upcoming_activities (upcoming_activity_id, in_house_app_id, software_title_id) VALUES (?, ?, ?)`,
		teamUpcomingActivity, teamUploadedApp, iosTitleID)
	unassignedUpcomingActivity := execNoErrLastID(t, db, `
		INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
		VALUES (4, 'in_house_app_install', 'unassigned-upcoming-install', '{}')`)
	execNoErr(t, db,
		`INSERT INTO in_house_app_upcoming_activities (upcoming_activity_id, in_house_app_id, software_title_id) VALUES (?, ?, ?)`,
		unassignedUpcomingActivity, unassignedGitOpsApp, iosTitleID)

	// Add a running and a finished setup experience result for the team row that is deleted.
	execNoErr(t, db, `
		INSERT INTO setup_experience_status_results (host_uuid, name, status, in_house_app_id, nano_command_uuid)
		VALUES ('host-5-uuid', 'HelloWorld', 'running', ?, 'setup-running-command'), ('host-6-uuid', 'HelloWorld', 'success', ?, 'setup-success-command')`,
		teamUploadedApp, teamUploadedApp)

	// Add a configuration to both team rows, only the kept row's configuration should remain.
	execNoErr(t, db, `INSERT INTO in_house_app_configurations (in_house_app_id, configuration) VALUES (?, 'uploaded-config'), (?, 'gitops-config')`,
		teamUploadedApp, teamGitOpsApp)

	// Second team iOS title where both rows have the same file, with a verified install on the row that is deleted.
	sameFileTeamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team2')`)
	sameFileKeptApp := execNoErrLastID(t, db, `
		INSERT INTO in_house_apps (title_id, team_id, global_or_team_id, filename, version, storage_id, platform, bundle_identifier)
		VALUES (?, ?, ?, 'AHelloWorld.ipa', '1.0', 'storage-same-file', 'ios', 'com.example.helloworld')`,
		iosTitleID, sameFileTeamID, sameFileTeamID)
	sameFileDuplicateApp := execNoErrLastID(t, db, `
		INSERT INTO in_house_apps (title_id, team_id, global_or_team_id, filename, version, storage_id, platform, bundle_identifier)
		VALUES (?, ?, ?, 'HelloWorld.ipa', '1.0', 'storage-same-file', 'ios', 'com.example.helloworld')`,
		iosTitleID, sameFileTeamID, sameFileTeamID)
	execNoErr(t, db, `
		INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, command_uuid, platform, verification_at)
		VALUES (7, ?, 'same-file-verified-install', 'ios', NOW(6))`,
		sameFileDuplicateApp)

	applyNext(t, db)

	var remainingInHouseAppIDs []int64
	err := db.Select(&remainingInHouseAppIDs, `SELECT id FROM in_house_apps ORDER BY id`)
	require.NoError(t, err)
	expectedRemainingInHouseAppIDs := []int64{teamGitOpsApp, teamIpadOSApp, teamNoTitleApp1, teamNoTitleApp2, unassignedUploadedApp, sameFileKeptApp}
	require.Equal(t, expectedRemainingInHouseAppIDs, remainingInHouseAppIDs)

	// Every install of the team title is on the kept team row, and the unassigned install is on the kept unassigned row.
	var teamInstallCount int
	err = db.Get(&teamInstallCount, `SELECT COUNT(*) FROM host_in_house_software_installs WHERE in_house_app_id = ?`, teamGitOpsApp)
	require.NoError(t, err)
	require.Equal(t, hostInHouseInstallsBatchSize+2, teamInstallCount)
	var unassignedInstallCommandUUIDs []string
	err = db.Select(&unassignedInstallCommandUUIDs, `SELECT command_uuid FROM host_in_house_software_installs WHERE in_house_app_id = ?`, unassignedUploadedApp)
	require.NoError(t, err)
	require.Equal(t, []string{"unassigned-gitops-install"}, unassignedInstallCommandUUIDs)
	var totalInstallCount int
	err = db.Get(&totalInstallCount, `SELECT COUNT(*) FROM host_in_house_software_installs`)
	require.NoError(t, err)
	require.Equal(t, hostInHouseInstallsBatchSize+4, totalInstallCount)

	// The installs moved from a row with a different file are marked removed, and the install
	// already on the kept team row is not.
	var teamRemovedInstallCount int
	err = db.Get(&teamRemovedInstallCount, `SELECT COUNT(*) FROM host_in_house_software_installs WHERE in_house_app_id = ? AND removed = 1`, teamGitOpsApp)
	require.NoError(t, err)
	require.Equal(t, hostInHouseInstallsBatchSize+1, teamRemovedInstallCount)
	var teamNotRemovedInstallCommandUUIDs []string
	err = db.Select(&teamNotRemovedInstallCommandUUIDs, `SELECT command_uuid FROM host_in_house_software_installs WHERE in_house_app_id = ? AND removed = 0`, teamGitOpsApp)
	require.NoError(t, err)
	require.Equal(t, []string{"team-gitops-install"}, teamNotRemovedInstallCommandUUIDs)
	var unassignedInstallRemoved bool
	err = db.Get(&unassignedInstallRemoved, `SELECT removed FROM host_in_house_software_installs WHERE command_uuid = 'unassigned-gitops-install'`)
	require.NoError(t, err)
	require.True(t, unassignedInstallRemoved)

	// The verified install moved from a row with the same file is on the kept row and not marked removed.
	var sameFileInstall struct {
		InHouseAppID int64 `db:"in_house_app_id"`
		Removed      bool  `db:"removed"`
	}
	err = db.Get(&sameFileInstall, `SELECT in_house_app_id, removed FROM host_in_house_software_installs WHERE command_uuid = 'same-file-verified-install'`)
	require.NoError(t, err)
	require.Equal(t, sameFileKeptApp, sameFileInstall.InHouseAppID)
	require.False(t, sameFileInstall.Removed)

	// The upcoming installs point to the kept rows.
	var teamUpcomingInHouseAppID int64
	err = db.Get(&teamUpcomingInHouseAppID, `SELECT in_house_app_id FROM in_house_app_upcoming_activities WHERE upcoming_activity_id = ?`, teamUpcomingActivity)
	require.NoError(t, err)
	require.Equal(t, teamGitOpsApp, teamUpcomingInHouseAppID)
	var unassignedUpcomingInHouseAppID int64
	err = db.Get(&unassignedUpcomingInHouseAppID, `SELECT in_house_app_id FROM in_house_app_upcoming_activities WHERE upcoming_activity_id = ?`, unassignedUpcomingActivity)
	require.NoError(t, err)
	require.Equal(t, unassignedUploadedApp, unassignedUpcomingInHouseAppID)

	// The setup experience results point to the kept team row and keep their status and command.
	var setupResults []struct {
		Status          string `db:"status"`
		InHouseAppID    int64  `db:"in_house_app_id"`
		NanoCommandUUID string `db:"nano_command_uuid"`
	}
	err = db.Select(&setupResults, `SELECT status, in_house_app_id, nano_command_uuid FROM setup_experience_status_results ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, setupResults, 2)
	require.Equal(t, "running", setupResults[0].Status)
	require.Equal(t, teamGitOpsApp, setupResults[0].InHouseAppID)
	require.Equal(t, "setup-running-command", setupResults[0].NanoCommandUUID)
	require.Equal(t, "success", setupResults[1].Status)
	require.Equal(t, teamGitOpsApp, setupResults[1].InHouseAppID)
	require.Equal(t, "setup-success-command", setupResults[1].NanoCommandUUID)

	// The deleted row's configuration is deleted with it.
	var configurations []string
	err = db.Select(&configurations, `SELECT configuration FROM in_house_app_configurations`)
	require.NoError(t, err)
	require.Equal(t, []string{"gitops-config"}, configurations)

	// Insert a second row for the same title and fleet, it should fail on the new unique index.
	_, err = db.Exec(`
		INSERT INTO in_house_apps (title_id, team_id, global_or_team_id, filename, version, storage_id, platform)
		VALUES (?, ?, ?, 'Another.ipa', '2.0', 'storage-another', 'ios')`,
		iosTitleID, teamID, teamID)
	require.ErrorContains(t, err, "Duplicate entry")

	// Insert a row for another title with a filename already used in the fleet, it should succeed.
	otherTitleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, bundle_identifier, extension_for) VALUES ('Other', 'ios_apps', 'com.example.other', '')`)
	insertInHouseApp(&teamID, &otherTitleID, "AHelloWorld.ipa", "1.0", "ios")
}
