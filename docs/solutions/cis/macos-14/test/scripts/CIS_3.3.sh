#!/bin/bash
# CIS 3.3 - Ensure install.log Is Retained for 365 or More Days and No Maximum Size
# The query requires:
#   - a line containing ttl=NNN where NNN >= 365
#   - NO line containing "all_max="
#
# The original script only removed all_max= patterns but never added
# ttl=365 when it was missing, so the query could never pass.

INSTALL_FILE="/etc/asl/com.apple.install"
TMP_FILE="$(/usr/bin/mktemp /tmp/com.apple.install.XXXXXX)" || exit 1
trap '/bin/rm -f "$TMP_FILE"' EXIT

if ! /usr/bin/sudo /usr/bin/awk '
    # Remove any all_max= token (query fails on any remaining all_max=)
    { gsub(/[[:space:]]*all_max=[^[:space:]]*/, "", $0) }
    # On the "* file" line, ensure ttl=365 is present
    /^\* file/ {
        if ($0 ~ /ttl=[0-9]+/) {
            gsub(/ttl=[0-9]+/, "ttl=365", $0)
        } else {
            $0 = $0 " ttl=365"
        }
    }
    { print }
' "$INSTALL_FILE" > "$TMP_FILE"; then
    echo "Failed to rewrite $INSTALL_FILE" >&2
    exit 1
fi

/usr/bin/sudo /bin/mv "$TMP_FILE" "$INSTALL_FILE"
/usr/bin/sudo /usr/sbin/chown root:wheel "$INSTALL_FILE"
/usr/bin/sudo /bin/chmod 0644 "$INSTALL_FILE"
