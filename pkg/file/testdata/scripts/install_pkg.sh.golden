#!/bin/sh

# For fleet-osquery (orbit) packages, pre-remove binaries that may be in use
# by the running orbit process. Without this, the macOS installer fails with
# "An unexpected error occurred while moving files to the final destination."
# On macOS, removing these files is safe because the running process keeps its
# file descriptor to the old inode. Orbit continues running with the old binary
# long enough for the installer to write new files at the same paths and for
# orbit to report the install result. The pkg's postinstall script then
# schedules a delayed restart to pick up the new binary.
if installer -pkginfo -pkg "$INSTALLER_PATH" 2>/dev/null | grep -qi "fleet osquery"; then
    rm -rf /opt/orbit/bin/orbit/macos 2>/dev/null
    rm -rf /opt/orbit/bin/osqueryd 2>/dev/null
    rm -rf /opt/orbit/bin/fleet-desktop 2>/dev/null
    # Marker file tells the postinstall script this is an in-band upgrade
    # (INSTALLER_PATH env var is not propagated by the macOS installer command)
    touch /opt/orbit/.inband_upgrade
fi

installer -pkg "$INSTALLER_PATH" -target /
