package tables

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/require"
)

func TestUp_20250121094600(t *testing.T) {
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
sudo cp -R "$TMPDIR/%s" "$APPDIR"
	`

	tx, err := db.Begin()
	require.NoError(t, err)
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	installScriptID, err := getOrInsertScript(txx, fmt.Sprintf(originalContents, "Figma.app"))
	require.NoError(t, err)
	uninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)
	firefoxInstallScriptID, err := getOrInsertScript(txx, fmt.Sprintf(originalContents, "Firefox.app"))
	require.NoError(t, err)
	firefoxUninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)
	vsCodeInstallScriptID, err := getOrInsertScript(txx, fmt.Sprintf(originalContents, "Visual Studio Code.app"))
	require.NoError(t, err)
	vsCodeUninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)
	braveInstallScriptID, err := getOrInsertScript(txx, fmt.Sprintf(originalContents, "Brave Browser.app"))
	require.NoError(t, err)
	braveUninstallScriptID, err := getOrInsertScript(txx, "echo uninstall")
	require.NoError(t, err)
	dockerSymLink := `/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker" "/usr/local/bin/docker"`
	dockerInstallScriptID, err := getOrInsertScript(txx, fmt.Sprintf(originalContents, "Docker.app")+dockerSymLink)
	require.NoError(t, err)
	dockerUninstallScriptID, err := getOrInsertScript(txx, `
		remove_launchctl_service 'com.docker.helper'
		remove_launchctl_service 'com.docker.socket'
		remove_launchctl_service 'com.docker.vmnetd'
		quit_application 'com.docker.docker'
		sudo rm -rf '/Library/PrivilegedHelperTools/com.docker.socket'
		sudo rm -rf '/Library/PrivilegedHelperTools/com.docker.vmnetd'
		sudo rmdir '~/.docker/bin'
		sudo rm -rf "$APPDIR/Docker.app"
		sudo rm -rf '/usr/local/bin/docker'
		sudo rm -rf '/usr/local/bin/docker-credential-desktop'
		sudo rm -rf '/usr/local/bin/docker-credential-ecr-login'
		sudo rm -rf '/usr/local/bin/docker-credential-osxkeychain'
		sudo rm -rf '/usr/local/bin/hub-tool'
		sudo rm -rf '/usr/local/cli-plugins/docker-compose'
		sudo rm -rf '/usr/local/bin/kubectl.docker'
		sudo rmdir '~/Library/Caches/com.plausiblelabs.crashreporter.data'
		sudo rmdir '~/Library/Caches/KSCrashReports'
	`)
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

	// Insert Firefox (one of our target apps)
	execNoErr(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Mozilla Firefox",
		"firefox",
		"134.0.1",
		"darwin",
		"https://download-installer.cdn.mozilla.net/pub/firefox/releases/134.0.1/mac/en-US/Firefox%20134.0.1.dmg",
		"b3342c12bb44b7c78351fb32442a0775c15fb2ac809c24447fd8f8d1e2a42c62",
		"org.mozilla.firefox",
		firefoxInstallScriptID,
		firefoxUninstallScriptID,
	)

	// Insert VSCode (one of our target apps)
	execNoErr(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Microsoft Visual Studio Code",
		"visual-studio-code",
		"1.96.4",
		"darwin",
		"https://update.code.visualstudio.com/1.96.4/darwin-arm64/stable",
		"331a1969ee128b251917ae76c58ac65eb1c81deb90aad277d6466f0531dffd8b",
		"com.microsoft.VSCode",
		vsCodeInstallScriptID,
		vsCodeUninstallScriptID,
	)

	// Insert Brave (one of our target apps)
	execNoErr(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Brave",
		"brave-browser",
		"1.74.48.0",
		"darwin",
		"https://updates-cdn.bravesoftware.com/sparkle/Brave-Browser/stable-arm64/174.48/Brave-Browser-arm64.dmg",
		"c49b8d7e7029ed665bacafaf93a36b96b0889338f713ded62ed60c0306cf22af",
		"com.brave.Browser",
		braveInstallScriptID,
		braveUninstallScriptID,
	)

	execNoErr(
		t,
		db,
		`INSERT INTO fleet_library_apps (name, token, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"Docker Desktop",
		"docker",
		"4.37.2,179585",
		"darwin",
		"https://desktop.docker.com/mac/main/arm64/179585/Docker.dmg",
		"624dec2ae9fc2269e07533921f5905c53514d698858dde25ab10f28f80e333c7",
		"com.docker.docker",
		dockerInstallScriptID,
		dockerUninstallScriptID,
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
		InstallScriptContents   string `db:"contents"`
		UninstallScriptContents string `db:"uninstall_contents"`
		Checksum                string `db:"md5_checksum"`
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

	uninstallSelectStmt := `
SELECT 
	sc.contents AS uninstall_contents,
	HEX(sc.md5_checksum) AS md5_checksum
FROM 
	fleet_library_apps fla 
	JOIN script_contents sc 
	ON fla.uninstall_script_content_id = sc.id
WHERE fla.token = ?`

	expectedContentsTmpl := `
#!/bin/sh

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%%Su" /dev/console)
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
quit_application '%[1]s'
sudo [ -d "$APPDIR/%[2]s" ] && sudo mv "$APPDIR/%[2]s" "$TMPDIR/%[2]s.bkp"
sudo cp -R "$TMPDIR/%[2]s" "$APPDIR"
	`

	err = sqlx.Get(db, &scriptContents, selectStmt, "figma")
	require.NoError(t, err)

	expectedContents := fmt.Sprintf(expectedContentsTmpl, "com.figma.Desktop", "Figma.app")
	expectedChecksum := md5ChecksumScriptContent(expectedContents)
	require.Equal(t, expectedContents, scriptContents.InstallScriptContents)
	require.Equal(t, expectedChecksum, scriptContents.Checksum)

	err = sqlx.Get(db, &scriptContents, selectStmt, "firefox")
	require.NoError(t, err)

	expectedContents = fmt.Sprintf(expectedContentsTmpl, "org.mozilla.firefox", "Firefox.app")
	expectedChecksum = md5ChecksumScriptContent(expectedContents)
	require.Equal(t, expectedContents, scriptContents.InstallScriptContents)
	require.Equal(t, expectedChecksum, scriptContents.Checksum)

	err = sqlx.Get(db, &scriptContents, selectStmt, "visual-studio-code")
	require.NoError(t, err)

	expectedContents = fmt.Sprintf(expectedContentsTmpl, "com.microsoft.VSCode", "Visual Studio Code.app")
	expectedChecksum = md5ChecksumScriptContent(expectedContents)
	require.Equal(t, expectedContents, scriptContents.InstallScriptContents)
	require.Equal(t, expectedChecksum, scriptContents.Checksum)

	err = sqlx.Get(db, &scriptContents, selectStmt, "brave-browser")
	require.NoError(t, err)

	expectedContents = fmt.Sprintf(expectedContentsTmpl, "com.brave.Browser", "Brave Browser.app")
	expectedChecksum = md5ChecksumScriptContent(expectedContents)
	require.Equal(t, expectedContents, scriptContents.InstallScriptContents)
	require.Equal(t, expectedChecksum, scriptContents.Checksum)

	err = sqlx.Get(db, &scriptContents, selectStmt, "docker")
	require.NoError(t, err)
	require.Contains(t, scriptContents.InstallScriptContents, "quit_application 'com.electron.dockerdesktop'")
	require.Contains(t, scriptContents.InstallScriptContents, fmt.Sprintf(`[ -d "/usr/local/bin" ] && %s`, dockerSymLink))

	err = sqlx.Get(db, &scriptContents, uninstallSelectStmt, "docker")
	require.NoError(t, err)
	require.Contains(t, scriptContents.UninstallScriptContents, "quit_application 'com.electron.dockerdesktop'")

	err = sqlx.Get(db, &scriptContents, selectStmt, "box-drive")
	require.NoError(t, err)
	require.Equal(t, "echo install", scriptContents.InstallScriptContents)
	require.Equal(t, md5ChecksumScriptContent("echo install"), scriptContents.Checksum)
}
