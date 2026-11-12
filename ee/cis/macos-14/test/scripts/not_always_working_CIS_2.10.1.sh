#!/bin/bash
# CIS 2.10.1 - Ensure an Inactivity Interval of 15 Minutes Or Less
# not_always_working: 2.10.1 is a managed_policies (profile) check, so a local
# `defaults write` does not satisfy the query; the runner skips this fixture.
user=$(/usr/bin/stat -f "%Su" /dev/console 2>/dev/null)
if [ -n "$user" ] && [ "$user" != "root" ]; then
  /usr/bin/sudo -u "$user" /usr/bin/defaults -currentHost write com.apple.screensaver idleTime -int 900
fi
