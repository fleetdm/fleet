package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/require"
)

func TestUp_20250106150150(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...
	contents := `
#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"
# copy to the applications folder
sudo cp -R "$TMPDIR/Figma.app" "$APPDIR"
	`

	tx, err := db.Begin()
	require.NoError(t, err)
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	installScriptID, err := getOrInsertScript(txx, contents)
	require.NoError(t, err)
	uninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
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

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}
