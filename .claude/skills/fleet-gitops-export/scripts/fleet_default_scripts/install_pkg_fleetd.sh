#!/bin/sh

# Fleet-osquery (orbit) specific install script.
# When orbit installs an update to itself (in-band upgrade), the macOS installer
# fails with "An unexpected error occurred while moving files to the final
# destination" because the running orbit process has these files in use.
# Pre-removing the binaries is safe because on macOS the running process keeps
# its file descriptor to the old inode. Orbit continues running with the old
# binary long enough for the installer to write new files and for orbit to
# report the install result. The pkg's postinstall script then schedules a
# delayed restart to pick up the new binary.
if [ -d /opt/orbit/bin ]; then
    rm -rf /opt/orbit/bin/orbit/macos 2>/dev/null
    rm -rf /opt/orbit/bin/osqueryd 2>/dev/null
    rm -rf /opt/orbit/bin/desktop 2>/dev/null
    # Marker file tells the postinstall script this is an in-band upgrade.
    # (The macOS installer command does not propagate env vars to postinstall scripts.)
    touch /opt/orbit/.inband_upgrade
fi

installer -pkg "$INSTALLER_PATH" -target /
exit_code=$?

# Clean up the marker file in case the installer failed before the postinstall
# script had a chance to run and remove it.
rm -f /opt/orbit/.inband_upgrade 2>/dev/null

exit $exit_code
