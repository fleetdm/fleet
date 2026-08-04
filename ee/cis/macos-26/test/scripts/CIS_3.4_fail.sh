#!/bin/bash
# CIS 3.4 - Ensure Security Auditing Logs Are Retained for 30 Days
# Sets a mixed expire-after whose leading day directive is below 30
# (expire-after:7d OR 30d) so the query returns 0 rows — verifying it rejects a
# config whose effective retention floor is under 30 days.
if [ ! -f /etc/security/audit_control ]; then
  /usr/bin/sudo /bin/cp /etc/security/audit_control.example /etc/security/audit_control
fi
TMP="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)"
/usr/bin/sudo /usr/bin/awk '
  /^expire-after:/ { print "expire-after:7d OR 30d"; found=1; next }
  { print }
  END { if (!found) print "expire-after:7d OR 30d" }
' /etc/security/audit_control > "$TMP" || exit 1
/usr/bin/sudo /bin/mv "$TMP" /etc/security/audit_control || exit 1
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/security/audit_control || exit 1
/usr/bin/sudo /bin/chmod 0440 /etc/security/audit_control || exit 1
