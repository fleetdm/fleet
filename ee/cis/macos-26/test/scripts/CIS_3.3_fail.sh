#!/bin/bash
# CIS 3.3 - Ensure install.log Is Retained for 365 or More Days and No Maximum Size
# Sets ttl to 30 (too low) AND adds an all_max= value so the query fails.
TMP="$(/usr/bin/mktemp /tmp/com.apple.install.XXXXXX)"
/usr/bin/sudo /usr/bin/awk '
  /^\>[[:space:]]*(\/var\/log\/)?install\.log/ || /[[:space:]]file[[:space:]]+(\/var\/log\/)?install\.log/ {
    if (match($0, /ttl=[0-9]+/)) {
      sub(/ttl=[0-9]+/, "ttl=30")
    } else {
      sub(/$/, " ttl=30")
    }
    if (!match($0, /all_max=/)) {
      sub(/$/, " all_max=50M")
    }
    print
    next
  }
  { print }
' /etc/asl/com.apple.install > "$TMP"
/usr/bin/sudo /bin/mv "$TMP" /etc/asl/com.apple.install
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/asl/com.apple.install
/usr/bin/sudo /bin/chmod 0644 /etc/asl/com.apple.install
