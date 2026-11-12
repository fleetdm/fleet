#!/bin/bash

# CIS 2.3.1.1 - Ensure AirDrop Is Disabled When Not Actively Transferring Files
# This script causes the policy to PASS by installing the MDM profile that
# disables AirDrop (allowAirDrop = false).
# The policy query checks managed_policies for allowAirDrop = false.
#
# Requires the corresponding profile: ../profiles/2.3.1.1.mobileconfig

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROFILE="${SCRIPT_DIR}/../profiles/2.3.1.1.mobileconfig"

if [ ! -f "$PROFILE" ]; then
    echo "Error: Profile not found at ${PROFILE}"
    exit 1
fi

echo "Installing profile 2.3.1.1.mobileconfig..."
echo "Open the profile manually:"
echo "  open '${PROFILE}'"
echo "Then go to System Settings > Privacy & Security > Profiles and install it."

open "$PROFILE"
