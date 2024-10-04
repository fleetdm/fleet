#!/bin/sh

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath $INSTALLER_PATH)")

# extract contents
MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"
# copy to the applications folder
sudo cp -R "$TMPDIR/Docker.app" "$APPDIR"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker" "/usr/local/bin/docker"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-desktop" "/usr/local/bin/docker-credential-desktop"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-ecr-login" "/usr/local/bin/docker-credential-ecr-login"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain" "/usr/local/bin/docker-credential-osxkeychain"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/docker-index" "/usr/local/bin/docker-index"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/kubectl" "/usr/local/bin/kubectl.docker"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/cli-plugins/docker-compose" "/usr/local/cli-plugins/docker-compose"
/bin/ln -h -f -s -- "$APPDIR/Docker.app/Contents/Resources/bin/hub-tool" "/usr/local/bin/hub-tool"
