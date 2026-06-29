#!/bin/bash
# CIS 3.5 - Ensure Access to Audit Records Is Controlled
# The query requires exact permissions:
#   - /etc/security/audit_control: mode 0400, owned by root:wheel
#   - Entries in /var/audit/ AND in the `dir:` configured inside
#     audit_control: mode 0440, owned by root:wheel
# The original script only covered /var/audit; hosts with a
# customized `dir:` setting would still fail the query.

# /etc/security/audit_control must be exactly 0400
/usr/bin/sudo /usr/sbin/chown root:wheel /etc/security/audit_control
/usr/bin/sudo /bin/chmod 0400 /etc/security/audit_control

# Collect audit directories: always /var/audit, plus the `dir:` line
# from /etc/security/audit_control if it's configured to something
# different.
AUDIT_DIRS=("/var/audit")
CONFIGURED_DIR="$(/usr/bin/sudo /usr/bin/awk -F: '/^dir:/ { print $2; exit }' /etc/security/audit_control | /usr/bin/tr -d '[:space:]')"
if [ -n "$CONFIGURED_DIR" ] && [ "$CONFIGURED_DIR" != "/var/audit" ]; then
    AUDIT_DIRS+=("$CONFIGURED_DIR")
fi

for dir in "${AUDIT_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        /usr/bin/sudo /usr/sbin/chown -R root:wheel "$dir"
        # The query uses `path LIKE '/var/audit/%'` which also matches
        # subdirectories. Chmod every entry under the dir (file or
        # directory) to 0440 to keep the query satisfied.
        /usr/bin/sudo /usr/bin/find "$dir" -mindepth 1 -exec /bin/chmod 0440 {} \;
    fi
done
