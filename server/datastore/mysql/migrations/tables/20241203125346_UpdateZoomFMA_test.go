package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/require"
)

func TestUp_20241203125346(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a scheduled and a triggered job run for maintained_apps
	execNoErr(t, db, `INSERT INTO cron_stats (name, instance, stats_type, status) VALUES (?, 'foo', ?, ?)`, fleet.CronMaintainedApps, fleet.CronStatsTypeScheduled, fleet.CronStatsStatusCompleted)
	execNoErr(t, db, `INSERT INTO cron_stats (name, instance, stats_type, status) VALUES (?, 'foo', ?, ?)`, fleet.CronMaintainedApps, fleet.CronStatsTypeTriggered, fleet.CronStatsStatusCompleted)

	// Add the old Zoom and Box Drive FMAs
	tx, err := db.Begin()
	require.NoError(t, err)
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	installScriptID, err := getOrInsertScript(txx, "echo install")
	require.NoError(t, err)
	uninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
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
		"Box Drive",
		"box-drive",
		"2.42.212",
		"darwin",
		"https://e3.boxcdn.net/desktop/releases/mac/BoxDrive-2.42.212.pkg",
		"93550756150c434bc058c30b82352c294a21e978caf436ac99e0a5f431adfb6e",
		"com.box.desktop",
		installScriptID,
		uninstallScriptID,
	)

	// Apply current migration.
	applyNext(t, db)

	// Zoom should be deleted, only the Box Drive FMA should remain
	var fmas []fleet.MaintainedApp
	err = db.Select(&fmas, `SELECT name, token FROM fleet_library_apps`)
	require.NoError(t, err)
	require.Len(t, fmas, 1)
	require.Equal(t, "Box Drive", fmas[0].Name)
	require.Equal(t, "box-drive", fmas[0].Token)

	// Only the triggered job record should remain in the cron_stats table
	var stats []fleet.CronStats
	err = db.Select(&stats, `SELECT name, instance, stats_type, status FROM cron_stats`)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, string(fleet.CronMaintainedApps), stats[0].Name)
	require.Equal(t, fleet.CronStatsTypeTriggered, stats[0].StatsType)
	require.Equal(t, fleet.CronStatsStatusCompleted, stats[0].Status)
}
