#!/bin/bash
# CIS 3.2 - Ensure Security Auditing Flags Are Configured
# Sets the flags line to include aa, ad, -ex, -fm, -fr, -fw, lo.
if [ ! -f /etc/security/audit_control ]; then
  /usr/bin/sudo /bin/cp /etc/security/audit_control.example /etc/security/audit_control
fi
TMP="$(/usr/bin/mktemp /tmp/audit_control.XXXXXX)"
/usr/bin/sudo /usr/bin/awk '
  /^flags:/ { print "flags:aa,ad,-ex,-fm,-fr,-fw,lo"; found=1; next }
  { print }
  END { if (!found) print "flags:aa,ad,-ex,-fm,-fr,-fw,lo" }
' /etc/security/audit_control > "$TMP"
/usr/bin/sudo /bin/mv "$TMP" /etc/security/audit_control
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod 0440 /etc/security/audit_control
