#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# install pkg files
INSTALLER_EXIT_CODE=0
sudo installer -pkg "$TMPDIR/ZoomInstallerIT.pkg" -target / || INSTALLER_EXIT_CODE=$?

# Check if Zoom is running
ZOOM_RUNNING=false
if osascript -e "application id \"us.zoom.xos\" is running" 2>/dev/null; then
	ZOOM_RUNNING=true
fi

# If installer failed and Zoom is running, check if package was actually installed
if [ $INSTALLER_EXIT_CODE -ne 0 ] && [ "$ZOOM_RUNNING" = "true" ]; then
	# Check if the Zoom app exists (package was installed)
	if [ -d "/Applications/zoom.us.app" ]; then
		echo "Zoom package installed successfully. The app is currently running and will update automatically when Zoom is quit and relaunched."
		exit 0
	fi
fi

# If we get here and installer failed, exit with the error
if [ $INSTALLER_EXIT_CODE -ne 0 ]; then
	exit $INSTALLER_EXIT_CODE
fi

