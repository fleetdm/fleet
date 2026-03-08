#!/bin/bash
# Remediation script: Block and remove Clawbot/Moltbot malware
# This script is deployed automatically via Fleet when a policy detects
# the "Clawbot" or "Moltbot" malware on a macOS endpoint.
#
# Actions taken:
#   1. Add Santa path-based block rules for known malware binary locations
#   2. Kill any running clawbot/moltbot processes
#   3. Unload and remove related LaunchAgents and LaunchDaemons
#   4. Remove malware files and directories from disk
#
# All actions are logged with timestamps.

# Define variables
SANTACTL="/usr/local/bin/santactl"
CUSTOM_MSG="This application has been blocked by our security policy. Clawbot/Moltbot malware detected."

# Paths where clawbot/moltbot binaries may be found
BLOCK_PATHS=(
  "/usr/local/bin/clawbot"
  "/usr/local/bin/moltbot"
  "/tmp/clawbot"
  "/tmp/moltbot"
  "/Applications/Clawbot.app"
  "/Applications/Moltbot.app"
)

# Process names to kill
PROCESS_NAMES=(
  "clawbot"
  "moltbot"
)

# LaunchAgent/LaunchDaemon plist patterns (system-level)
SYSTEM_PLIST_DIRS=(
  "/Library/LaunchAgents"
  "/Library/LaunchDaemons"
)

PLIST_PATTERNS=(
  "com.clawbot.*"
  "com.moltbot.*"
)

# Application Support directories to remove
APP_SUPPORT_DIRS=(
  "Clawbot"
  "Moltbot"
)

# --------------------------------------------------------------------------
# Helper: log with timestamp
# --------------------------------------------------------------------------
log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# --------------------------------------------------------------------------
# Pre-flight checks
# --------------------------------------------------------------------------

# Check if running as root/sudo
if [ "$EUID" -ne 0 ]; then
  log "Error: This script must be run as root or with sudo privileges."
  exit 1
fi

# Check if santactl exists at the specified path
if [ ! -x "$SANTACTL" ]; then
  log "Error: santactl not found at $SANTACTL or not executable."
  exit 1
fi

log "Starting Clawbot/Moltbot remediation..."
log "========================================="

# --------------------------------------------------------------------------
# Step 1: Add Santa path-based block rules
# --------------------------------------------------------------------------
log "Step 1: Adding Santa block rules for known malware paths..."

for BLOCK_PATH in "${BLOCK_PATHS[@]}"; do
  log "Adding blocking rule for path: $BLOCK_PATH"
  "$SANTACTL" rule --blacklist --path "$BLOCK_PATH" --message "$CUSTOM_MSG" 2>&1 || true

  # Verify the rule was added
  log "Verifying rule was added..."
  CHECK_OUTPUT=$("$SANTACTL" rule --check --path "$BLOCK_PATH" 2>&1) || true
  log "Rule check output: $CHECK_OUTPUT"

  # Check if the output contains any indication of a rule
  if [ -n "$CHECK_OUTPUT" ]; then
    log "✅ Rule successfully applied for $BLOCK_PATH"
  else
    log "❌ Failed to apply rule for $BLOCK_PATH"
  fi

  echo "---------------------------------"
done

# --------------------------------------------------------------------------
# Step 2: Kill running clawbot/moltbot processes
# --------------------------------------------------------------------------
log "Step 2: Killing any running clawbot/moltbot processes..."

for PROC_NAME in "${PROCESS_NAMES[@]}"; do
  if pgrep -ix "$PROC_NAME" > /dev/null 2>&1; then
    log "Found running process: $PROC_NAME — sending SIGKILL..."
    pkill -9 -ix "$PROC_NAME" 2>&1 || true
    sleep 1

    if pgrep -ix "$PROC_NAME" > /dev/null 2>&1; then
      log "❌ Process $PROC_NAME is still running after kill attempt"
    else
      log "✅ Process $PROC_NAME terminated successfully"
    fi
  else
    log "No running process found for: $PROC_NAME"
  fi

  echo "---------------------------------"
done

# --------------------------------------------------------------------------
# Step 3: Unload and remove LaunchAgents / LaunchDaemons
# --------------------------------------------------------------------------
log "Step 3: Removing LaunchAgents and LaunchDaemons..."

# System-level plists (/Library/LaunchAgents, /Library/LaunchDaemons)
for PLIST_DIR in "${SYSTEM_PLIST_DIRS[@]}"; do
  for PATTERN in "${PLIST_PATTERNS[@]}"; do
    for PLIST_FILE in "$PLIST_DIR"/$PATTERN; do
      # Skip if glob didn't match anything
      [ -e "$PLIST_FILE" ] || continue

      log "Found plist: $PLIST_FILE"

      # Attempt to unload before removing
      LABEL=$(defaults read "$PLIST_FILE" Label 2>/dev/null) || true
      if [ -n "$LABEL" ]; then
        log "Unloading service: $LABEL"
        launchctl bootout system/"$LABEL" 2>&1 || true
      fi

      log "Removing $PLIST_FILE..."
      rm -f "$PLIST_FILE" 2>&1 || true

      if [ ! -e "$PLIST_FILE" ]; then
        log "✅ Removed $PLIST_FILE"
      else
        log "❌ Failed to remove $PLIST_FILE"
      fi

      echo "---------------------------------"
    done
  done
done

# Per-user LaunchAgents (~/Library/LaunchAgents)
for USER_HOME in /Users/*; do
  [ -d "$USER_HOME" ] || continue
  USERNAME=$(basename "$USER_HOME")

  for PATTERN in "${PLIST_PATTERNS[@]}"; do
    for PLIST_FILE in "$USER_HOME/Library/LaunchAgents"/$PATTERN; do
      # Skip if glob didn't match anything
      [ -e "$PLIST_FILE" ] || continue

      log "Found per-user plist ($USERNAME): $PLIST_FILE"

      # Attempt to unload for the user
      LABEL=$(defaults read "$PLIST_FILE" Label 2>/dev/null) || true
      USER_UID=$(id -u "$USERNAME" 2>/dev/null) || true
      if [ -n "$LABEL" ] && [ -n "$USER_UID" ]; then
        log "Unloading service for user $USERNAME (uid $USER_UID): $LABEL"
        launchctl bootout gui/"$USER_UID"/"$LABEL" 2>&1 || true
      fi

      log "Removing $PLIST_FILE..."
      rm -f "$PLIST_FILE" 2>&1 || true

      if [ ! -e "$PLIST_FILE" ]; then
        log "✅ Removed $PLIST_FILE"
      else
        log "❌ Failed to remove $PLIST_FILE"
      fi

      echo "---------------------------------"
    done
  done
done

# --------------------------------------------------------------------------
# Step 4: Remove malware files and directories from disk
# --------------------------------------------------------------------------
log "Step 4: Removing malware files and directories..."

# Remove known binary/app paths
for BLOCK_PATH in "${BLOCK_PATHS[@]}"; do
  if [ -e "$BLOCK_PATH" ]; then
    log "Removing $BLOCK_PATH..."
    rm -rf "$BLOCK_PATH" 2>&1 || true

    if [ ! -e "$BLOCK_PATH" ]; then
      log "✅ Removed $BLOCK_PATH"
    else
      log "❌ Failed to remove $BLOCK_PATH"
    fi
  else
    log "Not found (already absent): $BLOCK_PATH"
  fi

  echo "---------------------------------"
done

# Remove per-user Application Support directories
for USER_HOME in /Users/*; do
  [ -d "$USER_HOME" ] || continue
  USERNAME=$(basename "$USER_HOME")

  for APP_DIR in "${APP_SUPPORT_DIRS[@]}"; do
    TARGET="$USER_HOME/Library/Application Support/$APP_DIR"

    if [ -e "$TARGET" ]; then
      log "Removing Application Support directory ($USERNAME): $TARGET"
      rm -rf "$TARGET" 2>&1 || true

      if [ ! -e "$TARGET" ]; then
        log "✅ Removed $TARGET"
      else
        log "❌ Failed to remove $TARGET"
      fi
    else
      log "Not found (already absent, $USERNAME): $TARGET"
    fi

    echo "---------------------------------"
  done
done

# --------------------------------------------------------------------------
# Done
# --------------------------------------------------------------------------
log "========================================="
log "Clawbot/Moltbot remediation completed."
log "All rule and cleanup operations finished."
