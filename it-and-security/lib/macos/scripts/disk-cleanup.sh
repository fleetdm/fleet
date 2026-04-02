#!/bin/bash

# Disk cleanup script for macOS
# Clears common space-consuming caches and temporary files

set -e

echo "Starting disk cleanup..."

# Get current logged-in user
LOGGED_IN_USER=$(stat -f%Su /dev/console)
USER_HOME="/Users/$LOGGED_IN_USER"

# Clear system caches (safe to remove)
echo "Clearing system caches..."
rm -rf /Library/Caches/* 2>/dev/null || true

# Clear user caches
echo "Clearing user caches..."
rm -rf "$USER_HOME/Library/Caches/*" 2>/dev/null || true

# Clear system log files older than 7 days
echo "Clearing old log files..."
find /private/var/log -type f -mtime +7 -delete 2>/dev/null || true
find "$USER_HOME/Library/Logs" -type f -mtime +7 -delete 2>/dev/null || true

# Empty Trash for current user
echo "Emptying Trash..."
rm -rf "$USER_HOME/.Trash/*" 2>/dev/null || true

# Clear Xcode derived data (common space hog for developers)
if [ -d "$USER_HOME/Library/Developer/Xcode/DerivedData" ]; then
  echo "Clearing Xcode derived data..."
  rm -rf "$USER_HOME/Library/Developer/Xcode/DerivedData/*" 2>/dev/null || true
fi

# Clear old Software Update downloads
echo "Clearing old Software Update downloads..."
rm -rf /Library/Updates/* 2>/dev/null || true

# Clear temporary files
echo "Clearing temporary files..."
rm -rf /private/tmp/* 2>/dev/null || true
rm -rf /private/var/tmp/* 2>/dev/null || true

# Clear old Time Machine local snapshots (if space is critically low)
DISK_USAGE=$(df -H / | tail -1 | awk '{print $5}' | tr -d '%')
if [ "$DISK_USAGE" -gt 90 ]; then
  echo "Disk usage above 90%, pruning Time Machine local snapshots..."
  tmutil thinlocalsnapshots / 10000000000 4 2>/dev/null || true
fi

echo "Disk cleanup complete."
