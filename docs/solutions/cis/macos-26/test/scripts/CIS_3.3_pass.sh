#!/bin/bash
# CIS 3.3 - Ensure install.log Is Retained for 365 or More Days and No Maximum Size
# Updates ttl to 365 and strips all_max= from the file-level line in /etc/asl/com.apple.install.
TMP="$(/usr/bin/mktemp /tmp/com.apple.install.XXXXXX)"
/usr/bin/sudo /usr/bin/awk '
  /^\>[[:space:]]*\/var\/log\/install\.log/ || /file .*\/var\/log\/install\.log/ {
    gsub(/all_max=[0-9KMGkmg]+/, "")
    if (match($0, /ttl=[0-9]+/)) {
      sub(/ttl=[0-9]+/, "ttl=365")
    } else {
      sub(/$/, " ttl=365")
    }
    gsub(/[[:space:]]+/, " ")
    print
    next
  }
  { print }
' /etc/asl/com.apple.install > "$TMP"
/usr/bin/sudo /bin/mv "$TMP" /etc/asl/com.apple.install
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/asl/com.apple.install
/usr/bin/sudo /bin/chmod 0644 /etc/asl/com.apple.install
