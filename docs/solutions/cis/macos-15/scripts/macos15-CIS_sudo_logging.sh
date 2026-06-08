#!/bin/bash

# CIS macOS 15 Sequoia Benchmark
# Ensure Logging Is Enabled for Sudo
#
# Remediation: Creates or updates the sudoers drop-in configuration file
# /etc/sudoers.d/10_cissudoconfiguration with "Defaults log_allowed" so that
# all allowed sudo invocations are captured in the unified log.
#
# Note: macOS ignores sudoers.d files that contain a period in the filename,
# so the file must NOT have an extension.

SUDOERS_FILE="/etc/sudoers.d/10_cissudoconfiguration"

# Create the file if it doesn't exist
if [ ! -f "$SUDOERS_FILE" ]; then
    /usr/bin/sudo /usr/bin/touch "$SUDOERS_FILE"
    /usr/bin/sudo /bin/chmod 0440 "$SUDOERS_FILE"
    /usr/bin/sudo /usr/sbin/chown root:wheel "$SUDOERS_FILE"
fi

# Add "Defaults log_allowed" if not already present
if ! /usr/bin/sudo /usr/bin/grep -q "Defaults log_allowed" "$SUDOERS_FILE"; then
    echo "Defaults log_allowed" | /usr/bin/sudo /usr/bin/tee -a "$SUDOERS_FILE" > /dev/null
fi

# Validate the sudoers file syntax
/usr/bin/sudo /usr/sbin/visudo -cf "$SUDOERS_FILE"
if [ $? -ne 0 ]; then
    echo "ERROR: sudoers file syntax check failed. Removing invalid configuration."
    /usr/bin/sudo /bin/rm -f "$SUDOERS_FILE"
    exit 1
fi

echo "Sudo logging enabled: $SUDOERS_FILE configured with 'Defaults log_allowed'"
