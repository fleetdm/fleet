#!/bin/bash
# CIS 2.3.3.10 - Ensure Bluetooth Sharing Is Disabled
# Writes PrefKeyServicesEnabled=true for every console user so the policy query fails.
for user_home in /Users/*; do
  username=$(/usr/bin/basename "$user_home")
  case "$username" in Shared|Guest|.*) continue ;; esac
  [ ! -d "$user_home" ] && continue
  /usr/bin/sudo -u "$username" /usr/bin/defaults -currentHost write com.apple.Bluetooth PrefKeyServicesEnabled -bool true
done
