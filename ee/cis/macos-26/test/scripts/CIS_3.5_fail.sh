#!/bin/bash
# CIS 3.5 - Ensure Access to Audit Records Is Controlled
# Loosens audit_control permissions so the query fails.
if [ -f /etc/security/audit_control ]; then
  /usr/bin/sudo /bin/chmod 644 /etc/security/audit_control
fi
