#!/bin/bash

# CIS 2.3.1.1 - Ensure AirDrop Is Disabled When Not Actively Transferring Files
# This script causes the policy to FAIL by removing the MDM profile that
# disables AirDrop.
# The policy query checks managed_policies for allowAirDrop = false.
# Without the profile installed, the query returns no rows and the policy fails.

PROFILE_IDENTIFIER="com.fleetdm.cis-2.3.1.1"

echo "Removing profile ${PROFILE_IDENTIFIER}..."
/usr/bin/sudo /usr/bin/profiles -R -p "$PROFILE_IDENTIFIER" 2>/dev/null

if [ $? -eq 0 ]; then
    echo "Profile removed. AirDrop is no longer restricted. Policy should now fail."
else
    echo "Profile was not installed or could not be removed."
    echo "If managed by MDM, the profile must be removed through the MDM server."
fi
