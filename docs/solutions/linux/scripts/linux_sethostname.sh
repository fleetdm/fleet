#!/bin/bash

# Get the serial number of the host
serial_number=$(cat /sys/class/dmi/id/product_serial 2>/dev/null || cat /sys/class/dmi/id/board_serial 2>/dev/null)

# Check if the serial number was retrieved successfully
if [ -z "$serial_number" ]; then
    echo "Error: Could not retrieve the serial number."
    exit 1
fi

# Define the new hostname
new_hostname="COMPANY-LNX-$serial_number"

# Set the new hostname
sudo hostnamectl set-hostname "$new_hostname"

# Update the /etc/hosts file to reflect the new hostname
sudo sed -i "s/$(hostname)/$new_hostname/g" /etc/hosts

echo "Hostname updated to $new_hostname"
