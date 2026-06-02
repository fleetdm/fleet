#!/bin/bash
set -euo pipefail

# Get the serial number of the host
serial_number=$(ioreg -l | grep IOPlatformSerialNumber | awk '{print $4}' | tr -d '"')

# Check if the serial number was retrieved successfully
if [ -z "$serial_number" ]; then
    echo "Error: Could not retrieve the serial number."
    exit 1
fi

# Define the new hostname
new_hostname="COMPANY-MAC-$serial_number"

# Set the new hostname
sudo scutil --set ComputerName "$new_hostname"
sudo scutil --set LocalHostName "$new_hostname"
sudo scutil --set HostName "$new_hostname"

# Print the new hostname
echo "Hostname updated to $new_hostname"
