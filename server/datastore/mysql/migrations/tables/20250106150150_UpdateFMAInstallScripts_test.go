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
		"Figma",
		"figma",
		"124.7.4",
		"darwin",
		"https://desktop.figma.com/mac-arm/Figma-124.7.4.zip",
		"3160c0cac00b8b81b7b62375f04b9598b11cbd9e5d42a5ad532e8b98fecc6b15",
		"com.figma.Desktop",
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
