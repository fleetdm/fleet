#!/bin/bash
# CIS 2.6.8 - Ensure an Administrator Password Is Required to Access System-Wide Preferences
# Rewrites each relevant authorizationdb right so shared=false, group=admin,
# authenticate-user=true, session-owner=false. Mirrors the CIS remediation script.
authDBs=(
  "system.preferences"
  "system.preferences.energysaver"
  "system.preferences.network"
  "system.preferences.printing"
  "system.preferences.sharing"
  "system.preferences.softwareupdate"
  "system.preferences.startupdisk"
  "system.preferences.timemachine"
)

for section in "${authDBs[@]}"; do
  plist="/tmp/$section.plist"
  /usr/bin/sudo /usr/bin/security -q authorizationdb read "$section" > "$plist"

  class_key_value=$(/usr/libexec/PlistBuddy -c "Print :class" "$plist" 2>&1)
  if [[ "$class_key_value" == *"Does Not Exist"* ]]; then
    /usr/libexec/PlistBuddy -c "Add :class string user" "$plist"
  else
    /usr/libexec/PlistBuddy -c "Set :class user" "$plist"
  fi

  shared_value=$(/usr/libexec/PlistBuddy -c "Print :shared" "$plist" 2>&1)
  if [[ "$shared_value" == *"Does Not Exist"* ]]; then
    /usr/libexec/PlistBuddy -c "Add :shared bool false" "$plist"
  else
    /usr/libexec/PlistBuddy -c "Set :shared false" "$plist"
  fi

  auth_user_value=$(/usr/libexec/PlistBuddy -c "Print :authenticate-user" "$plist" 2>&1)
  if [[ "$auth_user_value" == *"Does Not Exist"* ]]; then
    /usr/libexec/PlistBuddy -c "Add :authenticate-user bool true" "$plist"
  else
    /usr/libexec/PlistBuddy -c "Set :authenticate-user true" "$plist"
  fi

  session_owner_value=$(/usr/libexec/PlistBuddy -c "Print :session-owner" "$plist" 2>&1)
  if [[ "$session_owner_value" == *"Does Not Exist"* ]]; then
    /usr/libexec/PlistBuddy -c "Add :session-owner bool false" "$plist"
  else
    /usr/libexec/PlistBuddy -c "Set :session-owner false" "$plist"
  fi

  group_value=$(/usr/libexec/PlistBuddy -c "Print :group" "$plist" 2>&1)
  if [[ "$group_value" == *"Does Not Exist"* ]]; then
    /usr/libexec/PlistBuddy -c "Add :group string admin" "$plist"
  else
    /usr/libexec/PlistBuddy -c "Set :group admin" "$plist"
  fi

  /usr/bin/sudo /usr/bin/security -q authorizationdb write "$section" < "$plist"
  /bin/rm -f "$plist"
done
