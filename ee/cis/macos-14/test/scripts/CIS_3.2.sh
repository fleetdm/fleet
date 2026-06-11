#!/bin/bash
# CIS 3.2 - Security Auditing Flags For User-Attributable Events
# Sets the flags line in /etc/security/audit_control to include the
# required audit classes (-fm, ad, -ex, aa, -fr, lo, -fw).
# If no flags line exists (file corrupted by previous test runs), it
# is appended.

AUDIT_FILE="/etc/security/audit_control"
TMP_FILE="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)" || exit 1
trap '/bin/rm -f "$TMP_FILE"' EXIT

# Replace existing flags: line, or append it if missing.
if ! /usr/bin/sudo /usr/bin/awk '
    /^flags:/ { print "flags:-fm,ad,-ex,aa,-fr,lo,-fw"; seen=1; next }
    { print }
    END { if (!seen) print "flags:-fm,ad,-ex,aa,-fr,lo,-fw" }
' "$AUDIT_FILE" > "$TMP_FILE"; then
    echo "Failed to rewrite $AUDIT_FILE" >&2
    exit 1
fi

/usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE"
/usr/bin/sudo /usr/sbin/chown root:wheel "$AUDIT_FILE"
/usr/bin/sudo /bin/chmod 0400 "$AUDIT_FILE"
