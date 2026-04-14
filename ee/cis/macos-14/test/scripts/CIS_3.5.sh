#!/bin/bash
# CIS 3.5 - Ensure Access to Audit Records Is Controlled
# The query requires exact permissions:
#   - /etc/security/audit_control: mode 0400, owned by root:wheel
#   - Files in /var/audit/: mode 0440, owned by root:wheel
# The original script used `chmod -R o-rw` which only strips other
# read+write but doesn't guarantee the exact modes the query demands.

# /etc/security/audit_control must be exactly 0400
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod 0400 /etc/security/audit_control

# Files under /var/audit/ must be exactly 0440, owned by root:wheel
/usr/bin/sudo /usr/sbin/chown -R root:wheel /var/audit/
/usr/bin/sudo /usr/bin/find /var/audit -type f -exec /bin/chmod 0440 {} \;
