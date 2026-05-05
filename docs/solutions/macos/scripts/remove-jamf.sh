#!/bin/bash
# This script runs one last recon and then removes the Jamf framework.
# Must be run as root. Deploy with Fleet, NOT Jamf Pro.

if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root." >&2
    exit 1
fi

jamf_binary=$(command -v jamf)

if [ -z "$jamf_binary" ]; then
    for path in "/usr/local/bin/jamf" "/usr/local/jamf/bin/jamf" "/usr/sbin/jamf"; do
        if [ -x "$path" ]; then
            jamf_binary="$path"
            break
        fi
    done
fi

if [ -z "$jamf_binary" ]; then
    echo "Jamf binary not found. Exiting." >&2
    exit 1
fi

echo "Jamf binary found at: $jamf_binary"

echo "Running final inventory update..."
$jamf_binary recon || echo "Warning: recon command failed (continuing anyway)"

echo "Removing Jamf framework..."
if $jamf_binary removeFramework; then
    echo "Jamf removal successful!"
else
    echo "Error: removeFramework command failed." >&2
    exit 1
fi
