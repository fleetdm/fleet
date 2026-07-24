#!/bin/bash
# CIS 3.4 - Ensure Security Auditing Logs Are Retained for 30 Days
# Sets expire-after to a value below 30 days so the query returns 0 rows
# (verifies the query detects non-compliant retention). CIS_3.4_pass.sh
# restores a compliant value.

AUDIT_FILE="/etc/security/audit_control"
TMP_FILE="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)" || exit 1
trap '/bin/rm -f "$TMP_FILE"' EXIT

# If expire-after exists, replace it; otherwise append it.
if /usr/bin/sudo /usr/bin/grep -q "^expire-after:" "$AUDIT_FILE"; then
    if ! /usr/bin/sudo /usr/bin/awk '
        /^expire-after:/ { print "expire-after:10d"; next }
        { print }
    ' "$AUDIT_FILE" > "$TMP_FILE"; then
        echo "Failed to rewrite $AUDIT_FILE" >&2
        exit 1
    fi
    /usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE"
else
    /usr/bin/sudo /usr/bin/cp "$AUDIT_FILE" "$TMP_FILE" || exit 1
    echo "expire-after:10d" | /usr/bin/sudo /usr/bin/tee -a "$TMP_FILE" > /dev/null
    /usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE"
fi

/usr/bin/sudo /usr/sbin/chown root:wheel "$AUDIT_FILE"
/usr/bin/sudo /bin/chmod 0400 "$AUDIT_FILE"
