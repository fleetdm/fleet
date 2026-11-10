#!/bin/bash

app_process_running() {
  local bundle_id="$1" app
  while IFS= read -r app; do
    grep -qixF -- "$bundle_id" <<< "$(defaults read "$app/Contents/Info" CFBundleIdentifier 2>/dev/null)" && return 0
  done < <(ps -axo comm= | sed -n 's|^ *\(.*\.app\)/Contents/MacOS/.*|\1|p' | sort -u)
  return 1
}

quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  local app_running
  app_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null)
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

  local quit_success=false still_running quit_error
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    quit_error=$(osascript -e 'on run argv' -e 'tell application id (item 1 of argv) to quit' -e 'end run' -- "$bundle_id" 2>&1 >/dev/null)
    if [[ "$quit_error" == *"(-600)" ]] && ! app_process_running "$bundle_id"; then
      echo "Application '$bundle_id' isn't running."
      quit_success=true
      break
    fi
    sleep 1
    if still_running=$(osascript -e 'on run argv' -e 'return application id (item 1 of argv) is running' -e 'end run' -- "$bundle_id" 2>/dev/null) && [[ "$still_running" == "false" ]]; then
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
