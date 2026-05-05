#!/bin/bash

# CIS - Ensure Security Auditing Retention Is Configured
# Sets audit retention to: expire-after:60d OR 5G

AUDIT_CONTROL="/etc/security/audit_control"
tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT

cp "$AUDIT_CONTROL" "$tmpfile"
origExpire=$(grep 'expire-after' "$tmpfile")
sed "s/${origExpire}/expire-after:60d OR 5G/" "$tmpfile" | sudo tee "$AUDIT_CONTROL" > /dev/null

# Explanation:
# In your /etc/security/audit_control, look for a line starting at: expire-after
# SHOULD PASS:   expire-after:60d OR 5G
# SHOULD PASS:   expire-after:61d OR 5G
# SHOULD PASS:   expire-after:60d OR 6G
# SHOULD PASS:   expire-after:61d OR 6G
# SHOULD FAIL:   expire-after:60d (no size component)
# SHOULD FAIL:   expire-after:5G (no time component)
# SHOULD FAIL:   expire-after:59d OR 5G
# SHOULD FAIL:   expire-after:60d OR 4G
