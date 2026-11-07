#!/bin/bash

# CIS 1.6 - Ensure Install Security Responses and System Files Is Enabled
# This script causes the policy to PASS by installing the MDM profile that
# enables CriticalUpdateInstall.
# The policy query checks managed_policies for CriticalUpdateInstall = true.
#
# Requires the corresponding profile: ../profiles/1.6.mobileconfig
# Install the profile by double-clicking it or using the command below.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROFILE="${SCRIPT_DIR}/../profiles/1.6.mobileconfig"

if [ ! -f "$PROFILE" ]; then
    echo "Error: Profile not found at ${PROFILE}"
    exit 1
fi

echo "Installing profile 1.6.mobileconfig..."
echo "Open the profile manually:"
echo "  open '${PROFILE}'"
echo "Then go to System Settings > Privacy & Security > Profiles and install it."

open "$PROFILE"
