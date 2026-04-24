#!/bin/bash

# CIS 1.6 - Ensure Install Security Responses and System Files Is Enabled
# This script causes the policy to FAIL by removing the MDM profile that
# enables CriticalUpdateInstall.
# The policy query checks managed_policies for CriticalUpdateInstall = true.
# Without the profile installed, the query returns no rows and the policy fails.

PROFILE_IDENTIFIER="com.fleetdm.cis-1.6"

echo "Removing profile ${PROFILE_IDENTIFIER}..."
/usr/bin/sudo /usr/bin/profiles -R -p "$PROFILE_IDENTIFIER" 2>/dev/null

if [ $? -eq 0 ]; then
    echo "Profile removed. Policy should now fail."
else
    echo "Profile was not installed or could not be removed."
    echo "If managed by MDM, the profile must be removed through the MDM server."
fi
