#!/bin/bash
# CIS 3.5 - Ensure Access to Audit Records Is Controlled
# Sets audit_control and the dir: target to root:wheel mode 440.
if [ ! -f /etc/security/audit_control ]; then
  /usr/bin/sudo /bin/cp /etc/security/audit_control.example /etc/security/audit_control
fi
/usr/bin/sudo /usr/sbin/chown -R root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod -R 440 /etc/security/audit_control

auditdir=$(/usr/bin/sudo /usr/bin/grep '^dir' /etc/security/audit_control | /usr/bin/awk -F: '{print $2}')
if [ -n "$auditdir" ] && [ -d "$auditdir" ]; then
  /usr/bin/sudo /usr/sbin/chown -R root:wheel "$auditdir"
  /usr/bin/sudo /bin/chmod -R 440 "$auditdir"
fi
