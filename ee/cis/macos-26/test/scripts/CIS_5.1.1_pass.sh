#!/bin/bash
# CIS 5.1.1 - Ensure Home Folders Are Secure
# Tightens permissions on each user home folder to 700 so the query passes.
for userhome in /Users/*; do
  user=$(basename "$userhome")
  case "$user" in
    Shared|Guest|.*) continue ;;
  esac
  [ -d "$userhome" ] || continue
  /usr/bin/sudo /bin/chmod 700 "$userhome"
done
