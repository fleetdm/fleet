#!/bin/bash

# CIS - Ensure Sending Diagnostic and Usage Data to Apple Is Disabled
# Part 1: System-level diagnostic settings
sudo /usr/bin/defaults write "/Library/Application Support/CrashReporter/DiagnosticMessagesHistory.plist" AutoSubmit -bool false
sudo /usr/bin/defaults write "/Library/Application Support/CrashReporter/DiagnosticMessagesHistory.plist" ThirdPartyDataSubmit -bool false
sudo /bin/chmod 644 "/Library/Application Support/CrashReporter/DiagnosticMessagesHistory.plist"
sudo /usr/sbin/chgrp admin "/Library/Application Support/CrashReporter/DiagnosticMessagesHistory.plist"

# Part 2: Per-user Siri data sharing opt-out
for username in $(dscl . -list /Users UniqueID | awk '$2 >= 500 {print $1}'); do
  home_dir=$(dscl . -read "/Users/$username" NFSHomeDirectory 2>/dev/null | awk '{print $2}')
  if [ -d "$home_dir" ]; then
    sudo -u "$username" /usr/bin/defaults write "$home_dir/Library/Preferences/com.apple.assistant.support" "Siri Data Sharing Opt-In Status" -int 2
  fi
done
