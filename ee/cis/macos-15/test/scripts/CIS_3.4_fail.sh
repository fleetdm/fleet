#!/bin/bash
# CIS 3.4 - Ensure Security Auditing Logs Are Retained for 30 Days
# Sets a mixed expire-after whose leading day directive is below 30
# (expire-after:7d OR 30d) so the query returns 0 rows — verifying it rejects a
# config whose effective retention floor is under 30 days. CIS_3.4_pass.sh
# restores a compliant value.
AUDIT_FILE="/etc/security/audit_control"
if [ ! -f "$AUDIT_FILE" ]; then
  /usr/bin/sudo /bin/cp "${AUDIT_FILE}.example" "$AUDIT_FILE"
fi
TMP_FILE="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)" || exit 1
trap '/bin/rm -f "$TMP_FILE"' EXIT
/usr/bin/sudo /usr/bin/awk '
  /^expire-after:/ { print "expire-after:7d OR 30d"; found=1; next }
  { print }
  END { if (!found) print "expire-after:7d OR 30d" }
' "$AUDIT_FILE" > "$TMP_FILE" || exit 1
/usr/bin/sudo /bin/mv "$TMP_FILE" "$AUDIT_FILE" || exit 1
/usr/bin/sudo /usr/sbin/chown root:wheel "$AUDIT_FILE" || exit 1
/usr/bin/sudo /bin/chmod 0400 "$AUDIT_FILE" || exit 1
