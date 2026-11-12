#!/bin/bash

# CIS 1.1 - Ensure Apple-provided Software Updates Are Installed
# This script causes the policy to FAIL by deferring all available updates.
# The policy query checks: SELECT 1 FROM software_update WHERE software_update_required = '0';
# It fails when there are required updates pending.
#
# Note: There is no reliable way to force macOS to have pending updates if the
# system is already fully patched. This script ignores available updates by
# resetting the last successful update date, which will cause the fleetd
# software_update table to report updates as required on the next check.

/usr/bin/sudo /usr/bin/defaults delete /Library/Preferences/com.apple.SoftwareUpdate LastFullSuccessfulDate 2>/dev/null

echo "Cleared LastFullSuccessfulDate. The software_update table should report"
echo "updates as required on the next osquery/fleetd check cycle."
echo "If the system is fully patched, this may not be sufficient to trigger a failure."
echo "In that case, wait for Apple to release a new update to test the fail condition."
