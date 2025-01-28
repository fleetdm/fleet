#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")
# functions

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


# extract contents
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"
# copy to the applications folder
quit_application 'com.docker.docker'
sudo [ -d "$APPDIR/Docker.app" ] && sudo mv "$APPDIR/Docker.app" "$TMPDIR/Docker.app.bkp"
sudo cp -R "$TMPDIR/Docker.app" "$APPDIR"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker" "/usr/local/bin/docker"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-desktop" "/usr/local/bin/docker-credential-desktop"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-ecr-login" "/usr/local/bin/docker-credential-ecr-login"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain" "/usr/local/bin/docker-credential-osxkeychain"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-index" "/usr/local/bin/docker-index"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/kubectl" "/usr/local/bin/kubectl.docker"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/cli-plugins/docker-compose" "/usr/local/cli-plugins/docker-compose"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/hub-tool" "/usr/local/bin/hub-tool"
