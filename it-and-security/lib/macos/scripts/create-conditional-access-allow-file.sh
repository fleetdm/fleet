#!/bin/bash

# Script to check for entra-conditional-access-allow file and create it if needed
# Target location: /var/fleet/entra-conditional-access-allow

FILE_PATH="/var/fleet/entra-conditional-access-allow"
DIR_PATH="/var/fleet"

# Check if the directory exists, create if it doesn't
if [ ! -d "$DIR_PATH" ]; then
    echo "Directory $DIR_PATH does not exist. Creating it..."
    sudo mkdir -p "$DIR_PATH"
fi

# Check if the file exists
if [ -f "$FILE_PATH" ]; then
    echo "File $FILE_PATH already exists."
    
    # Check current permissions
    CURRENT_PERMS=$(stat -f "%Lp" "$FILE_PATH" 2>/dev/null)
    if [ "$CURRENT_PERMS" != "644" ]; then
        echo "Current permissions: $CURRENT_PERMS. Updating to 644..."
        sudo chmod 644 "$FILE_PATH"
        echo "Permissions updated to 644."
    else
        echo "Permissions are already set to 644."
    fi
else
    echo "File $FILE_PATH does not exist. Creating it..."
    
    # Create the file using touch
    sudo touch "$FILE_PATH"
    
    # Set permissions to 644
    sudo chmod 644 "$FILE_PATH"
    
    echo "File created with permissions 644."
fi

# Verify the file and permissions
if [ -f "$FILE_PATH" ]; then
    FINAL_PERMS=$(stat -f "%Lp" "$FILE_PATH" 2>/dev/null)
    echo "✓ File exists at: $FILE_PATH"
    echo "✓ Permissions: $FINAL_PERMS"
else
    echo "✗ Error: Failed to create file"
    exit 1
fi
