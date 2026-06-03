#!/bin/bash

# CIS 1.1 - Ensure Apple-provided Software Updates Are Installed
# This script installs all available Apple software updates to cause the policy to PASS.
# The policy query checks: SELECT 1 FROM software_update WHERE software_update_required = '0';
# It passes when there are no required updates pending.

/usr/bin/sudo /usr/sbin/softwareupdate -i -a

echo "All available software updates have been installed."
echo "If a restart is required, run: /usr/bin/sudo /usr/sbin/softwareupdate -i -a -R"
