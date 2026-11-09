#!/bin/bash

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false still_running
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1
    sleep 1
    if still_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
      echo "Application '$bundle_id' quit successfully."
      quit_success=true
      break
    fi
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
    return 1
  fi
}

for id in "com.perforce.p4v" "com.perforce.p4merge" "com.perforce.p4admin"; do
  quit_application "$id" || exit 1
done

rm -rf "/Applications/p4v.app" "/Applications/p4merge.app" "/Applications/p4admin.app" >/dev/null 2>&1 || true

# Remove p4vc command line binary
rm -f /usr/local/bin/p4vc >/dev/null 2>&1 || true

for udir in /Users/* /var/root; do
  [[ -d "$udir/Library" ]] || continue
  rm -f  "$udir/Library/Preferences/com.perforce.p4v.plist" \
         "$udir/Library/Preferences/com.perforce.p4merge.plist" \
         "$udir/Library/Preferences/com.perforce.p4admin.plist" >/dev/null 2>&1 || true
  rm -rf "$udir/Library/Caches/com.perforce.p4v" \
         "$udir/Library/Caches/com.perforce.p4merge" \
         "$udir/Library/Caches/com.perforce.p4admin" >/dev/null 2>&1 || true
  rm -rf "$udir/Library/Application Support/Perforce" \
         "$udir/Library/Application Support/P4V" >/dev/null 2>&1 || true
done

echo "p4v suite uninstalled"
