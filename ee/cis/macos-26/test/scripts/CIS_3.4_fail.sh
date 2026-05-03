#!/bin/bash
# CIS 3.4 - Ensure Security Auditing Retention Is Enabled
# Sets expire-after to a value below threshold so the query fails.
if [ ! -f /etc/security/audit_control ]; then
  /usr/bin/sudo /bin/cp /etc/security/audit_control.example /etc/security/audit_control
fi
TMP="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)"
/usr/bin/sudo /usr/bin/awk '
  /^expire-after:/ { print "expire-after:7d OR 1G"; found=1; next }
  { print }
  END { if (!found) print "expire-after:7d OR 1G" }
' /etc/security/audit_control > "$TMP"
/usr/bin/sudo /bin/mv "$TMP" /etc/security/audit_control
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod 0440 /etc/security/audit_control
