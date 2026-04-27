#!/bin/bash

# Get the serial number of the host
serial_number=""
for serial_path in /sys/class/dmi/id/product_serial /sys/class/dmi/id/board_serial; do
    if [ -r "$serial_path" ]; then
        candidate=$(tr -d '\n' < "$serial_path")
        if [ -n "$candidate" ]; then
            serial_number="$candidate"
            break
        fi
    fi
done

# Check if the serial number was retrieved successfully
if [ -z "$serial_number" ]; then
    echo "Error: Could not retrieve the serial number."
    exit 1
fi

# Define the new hostname
new_hostname="COMPANY-LNX-$serial_number"

# Capture old hostname before changing it
old_hostname="$(hostname)"

# Set the new hostname
sudo hostnamectl set-hostname "$new_hostname"

# Update the /etc/hosts file to reflect the new hostname
tmp_hosts="$(mktemp)"
awk -v old="$old_hostname" -v new="$new_hostname" '{
  for (i = 1; i <= NF; i++) if ($i == old) $i = new
  print
}' /etc/hosts > "$tmp_hosts"
sudo cp "$tmp_hosts" /etc/hosts
rm -f "$tmp_hosts"

echo "Hostname updated to $new_hostname"
