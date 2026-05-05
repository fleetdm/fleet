package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/require"
)

func TestUp_20250320200000(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a scheduled and a triggered job run for maintained_apps
	execNoErr(t, db, `INSERT INTO cron_stats (name, instance, stats_type, status) VALUES (?, 'foo', ?, ?)`, fleet.CronMaintainedApps, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusCompleted)
	execNoErr(t, db, `INSERT INTO cron_stats (name, instance, stats_type, status) VALUES (?, 'foo', ?, ?)`, fleet.CronMaintainedApps, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusCompleted)

	// Add the old Zoom, Zoom for IT Admins, and Box Drive FMAs
	tx, err := db.Begin()
	require.NoError(t, err)
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	installScriptID, err := getOrInsertScript(txx, "echo install")
	require.NoError(t, err)
	uninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)

	installScriptID2, err := getOrInsertScript(txx, "echo install2")
	require.NoError(t, err)
	uninstallScriptID2, err := getOrInsertScript(txx, "echo uninstall2")
	require.NoError(t, err)

	installScriptID3, err := getOrInsertScript(txx, "echo install different")
	require.NoError(t, err)

	otherScriptID, err := getOrInsertScript(txx, "just a lil scripty boi")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	execNoErr(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Zoom",
		"zoom",
		"6.2.11.43613",
		"darwin",
		"https://cdn.zoom.us/prod/6.2.11.43613/arm64/zoomusInstallerFull.pkg",
		"dd6d28853eb6be7eaf7731aae1855c68cd6411ef6847158e6af18fffed5f8597",
		"us.zoom.xos",
		installScriptID,
		uninstallScriptID,
	)

	execNoErr(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Zoom for IT Admins",
		"zoom-for-it-admins",
		"6.2.11.43613",
		"darwin",
		"https://cdn.zoom.us/prod/6.2.11.43613/arm64/zoomusInstallerFull.pkg",
		"dd6d28853eb6be7eaf7731aae1855c68cd6411ef6847158e6af18fffed5f8597",
		"us.zoom.xos",
		installScriptID,
		uninstallScriptID,
	)

	boxFMAID := execNoErrLastID(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Box Drive",
		"box-drive",
		"2.42.212",
		"darwin",
		"https://e3.boxcdn.net/desktop/releases/mac/BoxDrive-2.42.212.pkg",
		"93550756150c434bc058c30b82352c294a21e978caf436ac99e0a5f431adfb6e",
		"com.box.desktop",
		installScriptID2,
		uninstallScriptID2,
	)

	// add a software installer for Box to No team, same install scripts
	noTeamBox := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id, fleet_library_app_id)
		VALUES
		(?,?,?,?,?,?,?,?)`, "box.pkg", "2.42.212", "darwin", installScriptID2, "sha-is-not-president", "", uninstallScriptID2, boxFMAID)

	// add a software installer for Box to another team, different install script
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ("Foo")`)
	otherTeamBox := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id)
		VALUES
		(?,?,?,?,?,?,?,?,?)`, teamID, teamID, "box.pkg", "2.42.212", "darwin", installScriptID3, "sha-is-not-president", "", uninstallScriptID2)

	// add a separate script to No team
	execNoErr(t, db, `INSERT INTO scripts (
		team_id, global_or_team_id, name, script_content_id
	) VALUES (?, ?, ?, ?)`, nil, 0, "myscript.sh", otherScriptID)

	// Apply current migration.
	applyNext(t, db)

	// install/uninstall scripts for Zoom should be gone
	// Box script should remain, other script should remain
	var scriptContentsIDs []int64
	err = db.Select(&scriptContentsIDs, `SELECT id FROM script_contents ORDER BY id`)
	require.NoError(t, err)
	require.Equal(t, []int64{installScriptID2, uninstallScriptID2, installScriptID3, otherScriptID}, scriptContentsIDs)

	// Should only have one Zoom plus Box
	var fmas []fleet.MaintainedApp
	err = db.Select(&fmas, `SELECT id, name, slug, unique_identifier FROM fleet_maintained_apps ORDER BY name`)
	require.NoError(t, err)
	require.Len(t, fmas, 2)
	require.Equal(t, "Box Drive", fmas[0].Name)
	require.Equal(t, "box-drive/darwin", fmas[0].Slug)
	require.Equal(t, "com.box.desktop", fmas[0].UniqueIdentifier)
	require.Equal(t, "Zoom", fmas[1].Name)
	require.Equal(t, "zoom/darwin", fmas[1].Slug)
	require.Equal(t, "us.zoom.xos", fmas[1].UniqueIdentifier)

	var linkedFMAID *int64

	// FMA ID for Box software installer on No team should match ID of Box FMA
	err = db.Get(&linkedFMAID, `SELECT fleet_maintained_app_id FROM software_installers WHERE id = ?`, noTeamBox)
	require.NoError(t, err)
	require.Equal(t, boxFMAID, *linkedFMAID)

	// FMA ID for Box software installer on other team should be null
	err = db.Get(&linkedFMAID, `SELECT fleet_maintained_app_id FROM software_installers WHERE id = ?`, otherTeamBox)
	require.NoError(t, err)
	require.Nil(t, linkedFMAID)

	// Only the triggered job record should remain in the cron_stats table
	var stats []fleet.CronStats
	err = db.Select(&stats, `SELECT name, instance, stats_type, status FROM cron_stats`)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, string(fleet.CronMaintainedApps), stats[0].Name)
	require.Equal(t, fleet.CronStatsTypeTriggered, stats[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusCompleted, stats[0].Status)
}
