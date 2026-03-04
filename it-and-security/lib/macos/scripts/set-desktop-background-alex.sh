#!/bin/bash

# Download alex-mitchell.png from Google Drive and set as desktop background
FILE_ID="1wFcjAGgP8LJXqeT6j76dTcAb8OMhFlIF"
DOWNLOAD_URL="https://drive.google.com/uc?export=download&id=${FILE_ID}"
DEST_DIR="/Library/Desktop Pictures"
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
CURRENT_USER=$(stat -f%Su /dev/console)

if [ -z "${CURRENT_USER}" ] || [ "${CURRENT_USER}" = "root" ]; then
  echo "No user logged in, skipping wallpaper set"
  exit 1
fi

# Set desktop background using osascript as the logged-in user
su - "${CURRENT_USER}" -c "osascript -e 'tell application \"Finder\" to set desktop picture to POSIX file \"${DEST_PATH}\"'"

echo "Desktop background set to alex-mitchell.png"
exit 0
