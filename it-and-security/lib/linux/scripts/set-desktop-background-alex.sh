#!/bin/bash

# Download alex-mitchell.png from Google Drive and set as desktop background
FILE_ID="1wFcjAGgP8LJXqeT6j76dTcAb8OMhFlIF"
DOWNLOAD_URL="https://drive.google.com/uc?export=download&id=${FILE_ID}"
DEST_DIR="/usr/share/backgrounds"
DEST_PATH="${DEST_DIR}/alex-mitchell.png"

# Create directory if it doesn't exist
mkdir -p "${DEST_DIR}"

# Download the image
curl -sL "${DOWNLOAD_URL}" -o "${DEST_PATH}"

if [ ! -f "${DEST_PATH}" ]; then
  echo "Failed to download alex-mitchell.png"
  exit 1
fi

# Get the currently logged-in user
CURRENT_USER=$(logname 2>/dev/null || whoami)

if [ -z "${CURRENT_USER}" ] || [ "${CURRENT_USER}" = "root" ]; then
  echo "No user logged in, skipping wallpaper set"
  exit 1
fi

# Set desktop background for GNOME
su - "${CURRENT_USER}" -c "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u ${CURRENT_USER})/bus gsettings set org.gnome.desktop.background picture-uri \"file://${DEST_PATH}\""
su - "${CURRENT_USER}" -c "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u ${CURRENT_USER})/bus gsettings set org.gnome.desktop.background picture-uri-dark \"file://${DEST_PATH}\""

echo "Desktop background set to alex-mitchell.png"
exit 0
