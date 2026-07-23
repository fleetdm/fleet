#!/bin/bash
# CIS 3.4 - Ensure Security Auditing Logs Are Retained for 30 Days
# The query requires an expire-after line with a day value of at least 30,
# e.g. "expire-after:30d" (an optional size clause such as "OR 5G" is fine).
#
# The original script wrote to /etc/security/audit_control using sudo
# with shell redirection — the redirect happens as the current user,
# not root, so the write silently failed.

AUDIT_FILE="/etc/security/audit_control"
TMP_FILE="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)" || exit 1
trap '/bin/rm -f "$TMP_FILE"' EXIT

# If expire-after exists, replace it; otherwise append it.
if /usr/bin/sudo /usr/bin/grep -q "^expire-after:" "$AUDIT_FILE"; then
    if ! /usr/bin/sudo /usr/bin/awk '
        /^expire-after:/ { print "expire-after:30d"; next }
        { print }
    ' "$AUDIT_FILE" > "$TMP_FILE"; then
        echo "Failed to rewrite $AUDIT_FILE" >&2
        exit 1
    fi
    /usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE"
else
    /usr/bin/sudo /usr/bin/cp "$AUDIT_FILE" "$TMP_FILE" || exit 1
    echo "expire-after:30d" | /usr/bin/sudo /usr/bin/tee -a "$TMP_FILE" > /dev/null
    /usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE"
fi

/usr/bin/sudo /usr/sbin/chown root:wheel "$AUDIT_FILE"
/usr/bin/sudo /bin/chmod 0400 "$AUDIT_FILE"
