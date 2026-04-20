#!/bin/bash

# CIS - Ensure Security Auditing Flags Are Configured Per Local Organizational Requirements
# Sets audit flags to: -fm,ad,-ex,aa,-fr,lo,-fw

AUDIT_CONTROL="/etc/security/audit_control"
tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT

cp "$AUDIT_CONTROL" "$tmpfile"
origFlags=$(grep 'flags:' "$tmpfile" | grep -v 'naflags')
sed "s/${origFlags}/flags:-fm,ad,-ex,aa,-fr,lo,-fw/" "$tmpfile" | sudo tee "$AUDIT_CONTROL" > /dev/null
