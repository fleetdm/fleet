#!/bin/bash

# Gyazo's pkg postinstall ends with two un-try'd `open gyazo://grantaccess` calls
# that need a GUI session, so `installer` reports failure as root even though the
# payload installed. Tolerate that only when the app is present at the version the
# package declares; anything else is a real failure.

APPDIR="/Applications"
BUNDLE_ID="com.gyazo.menu"
APP_PATH="$APPDIR/Gyazo Menu.app"

quit_application() {
  local bundle_id="$1"
  local console_user="$2"
  local timeout_duration=10

  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

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

pkg_declared_version() {
  local workdir
  workdir=$(mktemp -d) || return 1

  local version=""
  if (cd "$workdir" && xar -xf "$INSTALLER_PATH" Gyazo.pkg/PackageInfo >/dev/null 2>&1); then
    # Require whitespace before version= so format-version= and
    # generator-version= on the same element aren't picked up.
    version=$(sed -n 's/.*<pkg-info[^>]*[[:space:]]version="\([^"]*\)".*/\1/p' \
      "$workdir/Gyazo.pkg/PackageInfo" | head -1)
  fi

  rm -rf "$workdir"
  [[ -n "$version" ]] || return 1
  echo "$version"
}

installed_app_version() {
  [[ -d "$APP_PATH" ]] || return 1
  /usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" \
    "$APP_PATH/Contents/Info.plist" 2>/dev/null
}

CONSOLE_USER=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")

APP_WAS_RUNNING=false
if [[ "$(osascript -e "application id \"$BUNDLE_ID\" is running" 2>/dev/null)" == "true" ]]; then
  APP_WAS_RUNNING=true
  quit_application "$BUNDLE_ID" "$CONSOLE_USER"
fi

installer -pkg "$INSTALLER_PATH" -target /
INSTALLER_STATUS=$?

if [[ $INSTALLER_STATUS -ne 0 ]]; then
  echo "installer exited with status $INSTALLER_STATUS; checking whether the payload installed anyway."

  EXPECTED_VERSION=$(pkg_declared_version)
  if [[ -z "$EXPECTED_VERSION" ]]; then
    echo "Could not read the version declared by the package; treating this as a failed install."
    exit $INSTALLER_STATUS
  fi

  INSTALLED_VERSION=$(installed_app_version)
  if [[ "$INSTALLED_VERSION" != "$EXPECTED_VERSION" ]]; then
    echo "'$APP_PATH' is at version '${INSTALLED_VERSION:-<not installed>}', expected '$EXPECTED_VERSION'; the payload did not install."
    exit $INSTALLER_STATUS
  fi

  echo "'$APP_PATH' installed at '$EXPECTED_VERSION'; only the postinstall script failed. Treating as successful."
fi

if [[ "$APP_WAS_RUNNING" == "true" ]]; then
  sleep 2
  echo "Relaunching application '$BUNDLE_ID'..."
  # launchctl asuser bootstraps the console user's GUI session; sudo -u alone
  # doesn't, which can fail LSOpenURLsWithRole() even when open exits 0.
  if [[ $EUID -eq 0 && -n "$CONSOLE_USER" && "$CONSOLE_USER" != "root" ]]; then
    CONSOLE_UID=$(id -u "$CONSOLE_USER")
    /bin/launchctl asuser "$CONSOLE_UID" sudo -u "$CONSOLE_USER" open -b "$BUNDLE_ID" || true
  else
    open -b "$BUNDLE_ID" || true
  fi
fi
