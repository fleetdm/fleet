#!/bin/bash
# CIS 2.6.8 - Ensure an Administrator Password Is Required to Access System-Wide Preferences
# Flips shared back to true on system.preferences so the query fails.
plist="/tmp/system.preferences.plist"
/usr/bin/sudo /usr/bin/security -q authorizationdb read system.preferences \
  | /usr/bin/sudo /usr/bin/tee "$plist" > /dev/null
if [ ! -s "$plist" ]; then
  echo "Failed to read system.preferences authorizationdb; aborting." >&2
  /bin/rm -f "$plist"
  exit 1
fi

shared_value=$(/usr/libexec/PlistBuddy -c "Print :shared" "$plist" 2>&1)
if [[ "$shared_value" == *"Does Not Exist"* ]]; then
  /usr/libexec/PlistBuddy -c "Add :shared bool true" "$plist"
else
  /usr/libexec/PlistBuddy -c "Set :shared true" "$plist"
fi

/usr/bin/sudo /usr/bin/security -q authorizationdb write system.preferences < "$plist"
/bin/rm -f "$plist"
