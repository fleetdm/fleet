#!/bin/bash
# CIS 3.4 - Ensure Security Auditing Logs Are Retained for 30 Days
# Ensures /etc/security/audit_control has expire-after with a day value of
# at least 30 (e.g. "expire-after:30d") so the query returns rows.
AUDIT_FILE="/etc/security/audit_control"
# The base image may not ship an audit_control; seed it from the template.
if [ ! -f "$AUDIT_FILE" ]; then
  /usr/bin/sudo /bin/cp "${AUDIT_FILE}.example" "$AUDIT_FILE"
fi
TMP_FILE="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)" || exit 1
trap '/bin/rm -f "$TMP_FILE"' EXIT
# Replace an existing expire-after line, or append one if absent (single awk pass).
/usr/bin/sudo /usr/bin/awk '
  /^expire-after:/ { print "expire-after:30d"; found=1; next }
  { print }
  END { if (!found) print "expire-after:30d" }
' "$AUDIT_FILE" > "$TMP_FILE" || exit 1
/usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE" || exit 1
/usr/bin/sudo /usr/sbin/chown root:wheel "$AUDIT_FILE" || exit 1
/usr/bin/sudo /bin/chmod 0400 "$AUDIT_FILE" || exit 1
