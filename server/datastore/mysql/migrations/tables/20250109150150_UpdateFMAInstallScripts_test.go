package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/require"
)

func TestUp_20250109150150(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...
	originalContents := `
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
	installScriptID, err := getOrInsertScript(txx, originalContents)
	require.NoError(t, err)
	uninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)
	boxInstallScriptID, err := getOrInsertScript(txx, "echo install")
	require.NoError(t, err)
	boxUninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Insert Figma (one of our target FMAs)
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

	// Insert Box Drive, should be unaffected
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
		boxInstallScriptID,
		boxUninstallScriptID,
	)

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
	var scriptContents struct {
		InstallScriptContents string `db:"contents"`
		Checksum              string `db:"md5_checksum"`
	}

	selectStmt := `
SELECT 
	sc.contents AS contents,
	HEX(sc.md5_checksum) AS md5_checksum
FROM 
	fleet_library_apps fla 
	JOIN script_contents sc 
	ON fla.install_script_content_id = sc.id
WHERE fla.token = ?`

	err = sqlx.Get(db, &scriptContents, selectStmt, "figma")
	require.NoError(t, err)

	expectedContents := `
#!/bin/sh


quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
      if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
        echo "Application '$bundle_id' quit successfully."
        quit_success=true
        break
      fi
    fi
    sleep 1
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
  fi
}


# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"
# copy to the applications folder
quit_application com.figma.Desktop
sudo [ -d "$APPDIR/Figma.app" ] && sudo mv "$APPDIR/Figma.app" "$TMPDIR/Figma.app.bkp"
sudo cp -R "$TMPDIR/Figma.app" "$APPDIR"
	`

	expectedChecksum := md5ChecksumScriptContent(expectedContents)

	require.Equal(t, expectedContents, scriptContents.InstallScriptContents)
	require.Equal(t, expectedChecksum, scriptContents.Checksum)

	err = sqlx.Get(db, &scriptContents, selectStmt, "box-drive")
	require.NoError(t, err)
	require.Equal(t, "echo install", scriptContents.InstallScriptContents)
	require.Equal(t, md5ChecksumScriptContent("echo install"), scriptContents.Checksum)
}
