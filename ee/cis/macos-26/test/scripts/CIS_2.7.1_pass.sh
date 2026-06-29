#!/bin/bash
# CIS 2.7.1 - Ensure Screen Saver Corners Are Secure
# Sets all four hot corners to 0 (no action, != 6) for every local user so the query passes.
for userhome in /Users/*; do
  user=$(basename "$userhome")
  case "$user" in
    Shared|Guest|.*) continue ;;
  esac
  if [ ! -d "$userhome/Library/Preferences" ]; then
    continue
  fi
  for corner in wvous-tl-corner wvous-tr-corner wvous-bl-corner wvous-br-corner; do
    /usr/bin/sudo -u "$user" /usr/bin/defaults write com.apple.dock "$corner" -int 0
  done
done
