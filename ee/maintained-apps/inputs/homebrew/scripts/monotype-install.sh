#!/bin/bash

# variables
APPDIR="/Applications/"
TMPDIR=$(dirname "$(realpath "$INSTALLER_PATH")")
# functions

quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    eval "export $var_name=0"
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    eval "export $var_name=0"
    return
  fi

  # App was running, mark it for relaunch
  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

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


relaunch_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local was_running

  # Check if the app was running before installation
  eval "was_running=\$$var_name"
  if [[ "$was_running" != "1" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping relaunching application ID '$bundle_id'."
    return
  fi

  echo "Relaunching application '$bundle_id'..."

  # Launch the app in the logged-in user's GUI session. Apps launched by root
  # won't register with the user's Dock/GUI, so run 'open' as the console user.
  # Use 'launchctl asuser' to bootstrap into the console user's Mach namespace
  # and GUI session — 'sudo -u' alone doesn't do this, which can cause
  # LSOpenURLsWithRole() failures even when 'open' exits 0.
  local open_status=0
  if [[ $EUID -eq 0 ]]; then
    local console_uid
    console_uid=$(id -u "$console_user")
    /bin/launchctl asuser "$console_uid" sudo -u "$console_user" open -b "$bundle_id" >/dev/null 2>&1 || open_status=$?
  else
    open -b "$bundle_id" >/dev/null 2>&1 || open_status=$?
  fi

  if [[ $open_status -eq 0 ]]; then
    echo "Application '$bundle_id' relaunched successfully."
  else
    echo "Failed to relaunch application '$bundle_id'."
  fi
}


# extract contents
unzip "$INSTALLER_PATH" -d "$TMPDIR"
# install pkg files
quit_and_track_application 'com.monotype.monotype-fonts'
sudo installer -pkg "$TMPDIR/MTFInstaller.pkg" -target / || exit $?

# The pkg installs the app inside /Applications/Monotype Fonts/Application/,
# a subfolder osquery's apps table only sees through LaunchServices. Register
# the bundle for root (osqueryd's context) and the console user so inventory
# reports it before its first launch.
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
MONOTYPE_APP="/Applications/Monotype Fonts/Application/Monotype Fonts.app"
echo "DIAG whoami=$(id -un) console=$(stat -f %Su /dev/console)"
ls -la "/Applications/Monotype Fonts" "/Applications/Monotype Fonts/Application" 2>&1
plutil -lint "$MONOTYPE_APP/Contents/Info.plist" 2>&1
"$LSREGISTER" -f "$MONOTYPE_APP"; echo "DIAG lsregister(root) exit=$?"
console_user=$(stat -f "%Su" /dev/console)
if [[ -n "$console_user" && "$console_user" != "root" && "$console_user" != "loginwindow" ]]; then
  /bin/launchctl asuser "$(id -u "$console_user")" sudo -u "$console_user" "$LSREGISTER" -f "$MONOTYPE_APP"; echo "DIAG lsregister(user) exit=$?"
fi
echo "DIAG LS dump (root):"; "$LSREGISTER" -dump 2>/dev/null | grep -iE '^\s*path:.*monotype' | head -10
echo "DIAG osqueryi apps (root):"; osqueryi --json "SELECT path, bundle_identifier, bundle_short_version FROM apps WHERE path LIKE '%Monotype%';" 2>&1 | head -30
echo "DIAG osqueryi direct path:"; osqueryi --json "SELECT path, bundle_identifier FROM apps WHERE path = '$MONOTYPE_APP';" 2>&1 | head -8
echo "DIAG osqueryi total apps: $(osqueryi --json 'SELECT count(*) AS n FROM apps;' 2>/dev/null)"
diag_q() { echo "DIAG [$1] main app rows: $(osqueryi --json "SELECT path FROM apps WHERE bundle_identifier = 'com.monotype.monotype-fonts';" 2>/dev/null | tr -d '\n ')"; echo "DIAG [$1] LS dump main: $("$LSREGISTER" -dump 2>/dev/null | grep -cE '^\s*path:.*Application/Monotype Fonts\.app \(')"; }
diag_q "baseline"
"$LSREGISTER" -f "/Applications/Monotype Fonts"; echo "DIAG folder lsregister exit=$?"; sleep 2; diag_q "after-folder-lsregister"
sudo spctl --add "/Applications/Monotype Fonts"; echo "DIAG spctl exit=$?"; diag_q "after-spctl"
sudo xattr -r -d com.apple.quarantine "/Applications/Monotype Fonts" 2>&1 | head -2; diag_q "after-xattr"
sleep 8; diag_q "after-8s"
VQ="SELECT COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app') ) AS name, path, bundle_short_version, bundle_version FROM apps WHERE bundle_identifier LIKE '%com.monotype.monotype-fonts%' OR LOWER(COALESCE(NULLIF(display_name, ''), NULLIF(bundle_name, ''), NULLIF(bundle_executable, ''), TRIM(name, '.app'))) LIKE LOWER('%Monotype Fonts%') OR path LIKE '%/Applications/Monotype Fonts%'"
echo "DIAG env HOME=$HOME USER=$USER SUDO_USER=$SUDO_USER"; env | grep -iE '^(HOME|USER|LOGNAME|SUDO|TMPDIR|__CF)' | head
echo "DIAG [validator query, script env]: $(osqueryi --json "$VQ" 2>/dev/null | grep -c 'Application/Monotype Fonts.app"') main-app rows"
echo "DIAG [validator query, HOME=/Users/runner]: $(HOME=/Users/runner osqueryi --json "$VQ" 2>/dev/null | grep -c 'Application/Monotype Fonts.app"') main-app rows"
echo "DIAG [validator query, HOME=/var/root]: $(HOME=/var/root osqueryi --json "$VQ" 2>/dev/null | grep -c 'Application/Monotype Fonts.app"') main-app rows"
echo "DIAG [validator query, env -i]: $(env -i PATH=/usr/local/bin:/usr/bin:/bin osqueryi --json "$VQ" 2>/dev/null | grep -c 'Application/Monotype Fonts.app"') main-app rows"
echo "DIAG [validator query, as runner]: $(sudo -u runner osqueryi --json "$VQ" 2>/dev/null | grep -c 'Application/Monotype Fonts.app"') main-app rows"
echo "DIAG runner LS dump monotype paths:"; sudo -u runner "$LSREGISTER" -dump 2>/dev/null | grep -iE '^\s*path:.*monotype' | head -10
echo "DIAG [as runner] all monotype rows:"; sudo -u runner osqueryi --json "SELECT path, bundle_identifier FROM apps WHERE path LIKE '%Monotype%';" 2>/dev/null | head -20
echo "DIAG running monotype procs:"; pgrep -fl "Monotype" | head -8

relaunch_application 'com.monotype.monotype-fonts'
