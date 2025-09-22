#!/bin/bash

# =============================================================================
# LOCK SCREEN MESSAGE MANAGER
# =============================================================================

# Configuration
LOCK_MESSAGE="🔴 Empathy 🟠 Ownership 🟢 Results
🔵 Objectivity 🟣 Openness"
SCRIPT_DIR="/usr/local/bin/lockscreen_manager"
LOG_FILE="/var/log/lockscreen_manager.log"

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

check_sudo() {
    if [[ $EUID -ne 0 ]] && [[ "$1" != "install" ]] && [[ "$1" != "uninstall" ]]; then
        echo "Note: This operation requires sudo privileges for modifying system preferences."
    fi
}

create_log_dir() {
    mkdir -p "$(dirname "$LOG_FILE")"
}

# Get the preboot volume UUID dynamically with better error handling
get_preboot_uuid() {
    # Try multiple methods to find the preboot volume
    local system_volume_uuid
    system_volume_uuid=$(diskutil info / 2>/dev/null | grep "Volume UUID" | awk '{print $3}')
    
    if [[ -n "$system_volume_uuid" ]]; then
        local preboot_path="/System/Volumes/Preboot/$system_volume_uuid"
        if [[ -d "$preboot_path" ]]; then
            echo "$system_volume_uuid"
            return 0
        fi
    fi
    
    # Fallback: find any preboot volume
    local preboot_volumes
    preboot_volumes=$(ls /System/Volumes/Preboot/ 2>/dev/null | grep -E '^[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}$' | head -1)
    
    if [[ -n "$preboot_volumes" ]]; then
        echo "$preboot_volumes"
        return 0
    fi
    
    # Final fallback: any directory in Preboot
    local any_preboot
    any_preboot=$(ls /System/Volumes/Preboot/ 2>/dev/null | head -1)
    
    if [[ -n "$any_preboot" ]]; then
        echo "$any_preboot"
        return 0
    fi
    
    # If all else fails, return a default UUID to prevent script crash
    echo "DEFAULT-UUID"
    return 0
}

# Get the full preboot plist path with better error handling
get_preboot_plist_path() {
    local uuid
    uuid=$(get_preboot_uuid)
    
    if [[ $? -eq 0 && -n "$uuid" && "$uuid" != "DEFAULT-UUID" ]]; then
        local preboot_path="/System/Volumes/Preboot/$uuid/Library/Preferences/com.apple.loginwindow.plist"
        if [[ -f "$preboot_path" ]]; then
            echo "$preboot_path"
            return 0
        fi
    fi
    
    # Fallback: find any loginwindow.plist in preboot volumes
    local preboot_plist
    preboot_plist=$(find /System/Volumes/Preboot -name "com.apple.loginwindow.plist" 2>/dev/null | head -1)
    
    if [[ -n "$preboot_plist" ]]; then
        echo "$preboot_plist"
        return 0
    fi
    
    # If all else fails, return a default path to prevent script crash
    echo "/System/Volumes/Preboot/DEFAULT-UUID/Library/Preferences/com.apple.loginwindow.plist"
    return 0
}

# =============================================================================
# SET LOCK SCREEN MESSAGE SCRIPT
# =============================================================================

set_lock_message() {
    create_log_dir
    
    if sudo defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE"; then
        echo "$(date): Lock screen message set" >> "$LOG_FILE"
        echo "Lock screen message set successfully"
        
        # Try to update preboot volume, but don't fail if it doesn't work
        echo "$(date): Attempting to update preboot volume..." >> "$LOG_FILE"
        if sudo diskutil apfs updatePreboot / >> "$LOG_FILE" 2>&1; then
            echo "$(date): Preboot volume updated successfully" >> "$LOG_FILE"
        else
            echo "$(date): Preboot volume update failed, but system preferences updated" >> "$LOG_FILE"
        fi
    else
        echo "Failed to set lock screen message" >&2
        exit 1
    fi
}

# =============================================================================
# CLEAR LOCK SCREEN MESSAGE SCRIPT  
# =============================================================================

clear_lock_message() {
    echo "$(date): Clearing lock screen message..." >> "$LOG_FILE"
    
    # Clear the message from system preferences
    if defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null; then
        echo "$(date): Message cleared from system preferences" >> "$LOG_FILE"
    else
        echo "$(date): Lock screen message was already cleared or not set" >> "$LOG_FILE"
    fi
    
    # Try to clear from preboot volume if possible
    local preboot_path
    preboot_path=$(get_preboot_plist_path)
    
    if [[ $? -eq 0 && -f "$preboot_path" ]]; then
        echo "$(date): Attempting to clear preboot volume..." >> "$LOG_FILE"
        
        # Clear from preboot plist
        if plutil -remove LoginwindowText "$preboot_path" 2>/dev/null; then
            echo "$(date): Successfully removed LoginwindowText from preboot volume plist" >> "$LOG_FILE"
        fi
        
        # Try to update preboot volume
        if diskutil apfs updatePreboot / 2>/dev/null; then
            echo "$(date): Preboot volume updated successfully" >> "$LOG_FILE"
        else
            echo "$(date): Preboot volume update failed, but plist was cleared" >> "$LOG_FILE"
        fi
    else
        echo "$(date): Could not access preboot volume, system preferences cleared" >> "$LOG_FILE"
    fi
    
    echo "$(date): Lock screen message clearing completed" >> "$LOG_FILE"
}

# =============================================================================
# INSTALLATION SCRIPT
# =============================================================================

install_lockscreen_manager() {
    echo "Installing Lock Screen Message Manager..."

    # Clean up any existing installation
    if [[ $EUID -eq 0 ]]; then
        launchctl bootout system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null || true
        rm -f /Library/LaunchDaemons/com.lockscreen.coordinator.plist
        rm -rf "$SCRIPT_DIR"
        defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || true
    else
        sudo launchctl bootout system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null || true
        sudo rm -f /Library/LaunchDaemons/com.lockscreen.coordinator.plist
        sudo rm -rf "$SCRIPT_DIR"
        sudo defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || true
    fi

    # Create directories
    if [[ $EUID -eq 0 ]]; then
        mkdir -p "$SCRIPT_DIR"
        mkdir -p "$(dirname "$LOG_FILE")"
    else
        sudo mkdir -p "$SCRIPT_DIR"
        sudo mkdir -p "$(dirname "$LOG_FILE")"
    fi

    # Create the coordinator script
    if [[ $EUID -eq 0 ]]; then
        tee "$SCRIPT_DIR/lockscreen_coordinator.sh" > /dev/null << 'EOF'
#!/bin/bash

LOG_FILE="/var/log/lockscreen_manager.log"
LOCK_MESSAGE="🔴 Empathy 🟠 Ownership 🟢 Results
🔵 Objectivity 🟣 Openness"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Get the preboot plist path with error handling
get_preboot_plist_path() {
    # Try to find any loginwindow.plist in preboot volumes
    local preboot_plist
    preboot_plist=$(find /System/Volumes/Preboot -name "com.apple.loginwindow.plist" 2>/dev/null | head -1)
    
    if [[ -n "$preboot_plist" ]]; then
        echo "$preboot_plist"
        return 0
    fi
    
    # If not found, return empty to prevent script crash
    echo ""
    return 1
}

# Function to set lock screen message
set_lock_message() {
    if defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE" 2>/dev/null; then
        echo "$(date): Lock screen message set in system preferences" >> "$LOG_FILE"
        
        # Clear the message from system preferences to prepare for preboot sync
        echo "$(date): Clearing message from system preferences to prepare for preboot sync..." >> "$LOG_FILE"
        defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null
        
        # Update preboot volume with cleared state
        echo "$(date): Updating preboot volume with cleared state..." >> "$LOG_FILE"
        UPDATE_OUTPUT=$(diskutil apfs updatePreboot / >> "$LOG_FILE" 2>&1)
        UPDATE_EXIT_CODE=$?
        echo "$(date): updatePreboot exit code: $UPDATE_EXIT_CODE" >> "$LOG_FILE"
        echo "$(date): updatePreboot output: $UPDATE_OUTPUT" >> "$LOG_FILE"
        
        if [ $UPDATE_EXIT_CODE -eq 0 ]; then
            echo "$(date): Preboot volume updated successfully - FileVault message should be cleared" >> "$LOG_FILE"
        else
            echo "$(date): Failed to update preboot volume: $UPDATE_OUTPUT" >> "$LOG_FILE"
        fi
        
        # Set the message again in system preferences for lock screen display
        echo "$(date): Setting message again in system preferences for lock screen display..." >> "$LOG_FILE"
        defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE"
        echo "$(date): Lock screen message restored for display" >> "$LOG_FILE"
        
        return 0
    else
        echo "$(date): Failed to set lock screen message" >> "$LOG_FILE"
        return 1
    fi
}

# Function to clear lock screen message
clear_lock_message() {
    echo "$(date): Clearing lock screen message..." >> "$LOG_FILE"
    
    # Log current state before clearing
    CURRENT_MESSAGE=$(defaults read /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || echo "NOT_SET")
    echo "$(date): Current message in system preferences: $CURRENT_MESSAGE" >> "$LOG_FILE"
    
    # Clear the message
    if defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null; then
        echo "$(date): Message cleared from system preferences" >> "$LOG_FILE"
    else
        echo "$(date): Lock screen message was already cleared or not set" >> "$LOG_FILE"
    fi
    
    # Verify message was cleared from system preferences
    VERIFIED_MESSAGE=$(defaults read /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || echo "NOT_SET")
    echo "$(date): Verified system preferences message: $VERIFIED_MESSAGE" >> "$LOG_FILE"
    
    # Update preboot volume for FileVault compatibility
    echo "$(date): Updating preboot volume for FileVault screen..." >> "$LOG_FILE"
    
    # Log preboot volume state before update
    PREBOOT_PLIST_PATH=$(get_preboot_plist_path)
    if [[ $? -ne 0 || -z "$PREBOOT_PLIST_PATH" ]]; then
        echo "$(date): WARNING: Could not determine preboot plist path, skipping preboot operations" >> "$LOG_FILE"
        echo "$(date): Lock screen message clearing completed (system preferences only)" >> "$LOG_FILE"
        return 0
    fi
    PREBOOT_MESSAGE_BEFORE=$(defaults read "$PREBOOT_PLIST_PATH" LoginwindowText 2>/dev/null || echo "NOT_SET")
    echo "$(date): Preboot volume message before update: $PREBOOT_MESSAGE_BEFORE" >> "$LOG_FILE"
    
    # Directly clear the message from preboot volume plist file
    echo "$(date): Directly clearing message from preboot volume plist file..." >> "$LOG_FILE"
    if plutil -remove LoginwindowText "$PREBOOT_PLIST_PATH" 2>/dev/null; then
        echo "$(date): Successfully removed LoginwindowText from preboot volume plist" >> "$LOG_FILE"
    else
        echo "$(date): Failed to remove LoginwindowText from preboot volume plist (may not exist)" >> "$LOG_FILE"
    fi
    
    # Try to create a minimal plist file without LoginwindowText to ensure it's cleared
    echo "$(date): Creating minimal preboot plist file to ensure message is cleared..." >> "$LOG_FILE"
    printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0">\n<dict>\n</dict>\n</plist>\n' > "$PREBOOT_PLIST_PATH"
    if [ $? -eq 0 ]; then
        echo "$(date): Successfully created minimal preboot plist file" >> "$LOG_FILE"
    else
        echo "$(date): Failed to create minimal preboot plist file" >> "$LOG_FILE"
    fi
    
    # Run updatePreboot with detailed logging
    echo "$(date): Running: diskutil apfs updatePreboot /" >> "$LOG_FILE"
    UPDATE_OUTPUT=$(diskutil apfs updatePreboot / >> "$LOG_FILE" 2>&1)
    UPDATE_EXIT_CODE=$?
    echo "$(date): updatePreboot exit code: $UPDATE_EXIT_CODE" >> "$LOG_FILE"
    echo "$(date): updatePreboot output: $UPDATE_OUTPUT" >> "$LOG_FILE"
    
    if [ $UPDATE_EXIT_CODE -eq 0 ]; then
        echo "$(date): Preboot volume updated successfully" >> "$LOG_FILE"
    else
        echo "$(date): ERROR: Failed to update preboot volume (exit code: $UPDATE_EXIT_CODE)" >> "$LOG_FILE"
        echo "$(date): WARNING: updatePreboot failed, but direct plist modification should still work" >> "$LOG_FILE"
    fi
    
    # Log preboot volume state after update
    PREBOOT_MESSAGE_AFTER=$(defaults read "$PREBOOT_PLIST_PATH" LoginwindowText 2>/dev/null || echo "NOT_SET")
    echo "$(date): Preboot volume message after update: $PREBOOT_MESSAGE_AFTER" >> "$LOG_FILE"
    
    # Wait for changes to take effect
    sleep 2
    
    # Final verification
    if [ "$PREBOOT_MESSAGE_AFTER" = "NOT_SET" ]; then
        echo "$(date): SUCCESS: Message cleared from preboot volume" >> "$LOG_FILE"
    else
        echo "$(date): WARNING: Message still exists in preboot volume: $PREBOOT_MESSAGE_AFTER" >> "$LOG_FILE"
    fi
    
    echo "$(date): Lock screen message clearing completed" >> "$LOG_FILE"
}

# Signal handler for shutdown
cleanup_and_exit() {
    echo "$(date): Shutdown signal received, clearing message..." >> "$LOG_FILE"
    clear_lock_message
    exit 0
}

# Set up signal handlers
trap cleanup_and_exit SIGTERM SIGINT SIGQUIT SIGUSR1 SIGUSR2 SIGHUP EXIT

# Log startup
echo "$(date): Lock screen coordinator started (PID: $$)" >> "$LOG_FILE"

# Check if we're in shutdown mode (don't set message if system is shutting down)
if [[ -f "/private/var/run/com.apple.shutdown.started" ]] || \
   [[ -f "/private/var/run/com.apple.reboot.started" ]] || \
   [[ -f "/private/var/run/com.apple.logout.started" ]]; then
    echo "$(date): System is shutting down, not setting message" >> "$LOG_FILE"
    exit 0
fi

# Also check if system processes are already stopped (indicates shutdown in progress)
# Wait up to 30 seconds for system processes to start during boot
STARTUP_WAIT=0
while [[ $STARTUP_WAIT -lt 30 ]]; do
    if pgrep -i WindowServer > /dev/null 2>&1 && pgrep -i loginwindow > /dev/null 2>&1; then
        echo "$(date): System processes detected after ${STARTUP_WAIT} seconds" >> "$LOG_FILE"
        break
    fi
    echo "$(date): Waiting for system processes to start... (${STARTUP_WAIT}/30 seconds)" >> "$LOG_FILE"
    sleep 2
    STARTUP_WAIT=$((STARTUP_WAIT + 2))
done

# If processes still aren't running after 30 seconds, assume shutdown
if ! pgrep -i WindowServer > /dev/null 2>&1 || ! pgrep -i loginwindow > /dev/null 2>&1; then
    echo "$(date): System processes not running after 30 seconds, likely shutting down - not setting message" >> "$LOG_FILE"
    exit 0
fi

# Set message on startup
set_lock_message

# Immediately clear the message from preboot volume to ensure FileVault screen stays clear
echo "$(date): Proactively clearing message from preboot volume after startup..." >> "$LOG_FILE"
# Only clear preboot volume, not system preferences
echo "$(date): Clearing preboot volume only (keeping system preferences message)..." >> "$LOG_FILE"

# Log current state
PREBOOT_PLIST_PATH=$(get_preboot_plist_path)
if [[ $? -ne 0 || -z "$PREBOOT_PLIST_PATH" ]]; then
    echo "$(date): WARNING: Could not determine preboot plist path, skipping proactive clearing" >> "$LOG_FILE"
else
    PREBOOT_MESSAGE_BEFORE=$(defaults read "$PREBOOT_PLIST_PATH" LoginwindowText 2>/dev/null || echo "NOT_SET")
    SYSTEM_MESSAGE_BEFORE=$(defaults read /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || echo "NOT_SET")
    echo "$(date): Preboot volume message before proactive clearing: $PREBOOT_MESSAGE_BEFORE" >> "$LOG_FILE"
    echo "$(date): System preferences message before proactive clearing: $SYSTEM_MESSAGE_BEFORE" >> "$LOG_FILE"

    # Temporarily clear system preferences to prevent updatePreboot from re-syncing the message
    echo "$(date): Temporarily clearing system preferences to prevent updatePreboot from re-syncing..." >> "$LOG_FILE"
    defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null

    # Directly clear the message from preboot volume plist file
    echo "$(date): Directly clearing message from preboot volume plist file..." >> "$LOG_FILE"
    if plutil -remove LoginwindowText "$PREBOOT_PLIST_PATH" 2>/dev/null; then
        echo "$(date): Successfully removed LoginwindowText from preboot volume plist" >> "$LOG_FILE"
    else
        echo "$(date): Failed to remove LoginwindowText from preboot volume plist (may not exist)" >> "$LOG_FILE"
    fi

    # Try to create a minimal plist file without LoginwindowText to ensure it's cleared
    echo "$(date): Creating minimal preboot plist file to ensure message is cleared..." >> "$LOG_FILE"
    printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0">\n<dict>\n</dict>\n</plist>\n' > "$PREBOOT_PLIST_PATH"
    if [ $? -eq 0 ]; then
        echo "$(date): Successfully created minimal preboot plist file" >> "$LOG_FILE"
    else
        echo "$(date): Failed to create minimal preboot plist file" >> "$LOG_FILE"
    fi

    # Run updatePreboot with detailed logging (now with cleared system preferences)
    echo "$(date): Running: diskutil apfs updatePreboot /" >> "$LOG_FILE"
    UPDATE_OUTPUT=$(diskutil apfs updatePreboot / >> "$LOG_FILE" 2>&1)
    UPDATE_EXIT_CODE=$?
    echo "$(date): updatePreboot exit code: $UPDATE_EXIT_CODE" >> "$LOG_FILE"
    echo "$(date): updatePreboot output: $UPDATE_OUTPUT" >> "$LOG_FILE"

    if [ $UPDATE_EXIT_CODE -eq 0 ]; then
        echo "$(date): Preboot volume updated successfully" >> "$LOG_FILE"
    else
        echo "$(date): ERROR: Failed to update preboot volume (exit code: $UPDATE_EXIT_CODE)" >> "$LOG_FILE"
        echo "$(date): WARNING: updatePreboot failed, but direct plist modification should still work" >> "$LOG_FILE"
    fi

    # Restore system preferences message
    echo "$(date): Restoring system preferences message..." >> "$LOG_FILE"
    defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE"

    # Log final state
    PREBOOT_MESSAGE_AFTER=$(defaults read "$PREBOOT_PLIST_PATH" LoginwindowText 2>/dev/null || echo "NOT_SET")
    SYSTEM_MESSAGE_AFTER=$(defaults read /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null || echo "NOT_SET")
    echo "$(date): Preboot volume message after proactive clearing: $PREBOOT_MESSAGE_AFTER" >> "$LOG_FILE"
    echo "$(date): System preferences message after proactive clearing: $SYSTEM_MESSAGE_AFTER" >> "$LOG_FILE"

    # Final verification
    if [ "$PREBOOT_MESSAGE_AFTER" = "NOT_SET" ] && [ "$SYSTEM_MESSAGE_AFTER" != "NOT_SET" ]; then
        echo "$(date): SUCCESS: Proactive clearing completed - FileVault screen clear, lock screen shows message" >> "$LOG_FILE"
    else
        echo "$(date): WARNING: Proactive clearing may not have worked as expected" >> "$LOG_FILE"
    fi
fi

echo "$(date): Proactive preboot clearing completed" >> "$LOG_FILE"

# Record startup time to avoid false shutdown detection during startup
STARTUP_TIME=$(date +%s)

# Main monitoring loop
while true; do
    # Check for shutdown indicators - clear message early when system is still operational
    if [[ -f "/private/var/run/com.apple.shutdown.started" ]] || \
       [[ -f "/private/var/run/com.apple.reboot.started" ]] || \
       [[ -f "/private/var/run/com.apple.logout.started" ]]; then
        echo "$(date): Shutdown/reboot/logout detected, clearing message..." >> "$LOG_FILE"
        echo "$(date): Shutdown files found:" >> "$LOG_FILE"
        [[ -f "/private/var/run/com.apple.shutdown.started" ]] && echo "$(date): - /private/var/run/com.apple.shutdown.started" >> "$LOG_FILE"
        [[ -f "/private/var/run/com.apple.reboot.started" ]] && echo "$(date): - /private/var/run/com.apple.reboot.started" >> "$LOG_FILE"
        [[ -f "/private/var/run/com.apple.logout.started" ]] && echo "$(date): - /private/var/run/com.apple.logout.started" >> "$LOG_FILE"
        
        # Clear message immediately while system is still operational
        echo "$(date): Clearing message early while DiskManagement framework is available..." >> "$LOG_FILE"
        clear_lock_message
        
        echo "$(date): Lock screen coordinator stopped" >> "$LOG_FILE"
        exit 0
    fi
    
    # Check if system processes are stopping (shutdown indicator)
    # Only check this if we've been running for at least 30 seconds to avoid false positives during startup
    CURRENT_TIME=$(date +%s)
    if [[ $((CURRENT_TIME - STARTUP_TIME)) -gt 30 ]]; then
        if ! pgrep -i WindowServer > /dev/null 2>&1 || ! pgrep -i loginwindow > /dev/null 2>&1; then
            echo "$(date): System processes stopped, clearing message..." >> "$LOG_FILE"
            WINDOWSERVER_PID=$(pgrep -i WindowServer 2>/dev/null || echo "NOT_FOUND")
            LOGINWINDOW_PID=$(pgrep -i loginwindow 2>/dev/null || echo "NOT_FOUND")
            echo "$(date): WindowServer PID: $WINDOWSERVER_PID" >> "$LOG_FILE"
            echo "$(date): loginwindow PID: $LOGINWINDOW_PID" >> "$LOG_FILE"
            clear_lock_message
            echo "$(date): Lock screen coordinator stopped" >> "$LOG_FILE"
            exit 0
        fi
    fi
    
    # Run proactive clearing every 30 seconds to ensure preboot volume stays clear
    if [[ $((CURRENT_TIME - STARTUP_TIME)) -gt 30 ]] && [[ $((CURRENT_TIME % 30)) -lt 2 ]]; then
        echo "$(date): Running periodic proactive clearing..." >> "$LOG_FILE"
        
        # Check current state
        PREBOOT_PLIST_PATH=$(get_preboot_plist_path)
        if [[ $? -ne 0 || -z "$PREBOOT_PLIST_PATH" ]]; then
            echo "$(date): WARNING: Could not determine preboot plist path, skipping periodic clearing" >> "$LOG_FILE"
            continue
        fi
        PREBOOT_MESSAGE=$(defaults read "$PREBOOT_PLIST_PATH" LoginwindowText 2>/dev/null || echo "NOT_SET")
        if [ "$PREBOOT_MESSAGE" != "NOT_SET" ]; then
            echo "$(date): Preboot volume has message, clearing it..." >> "$LOG_FILE"
            
            # Temporarily clear system preferences
            defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null
            
            # Clear preboot volume
            plutil -remove LoginwindowText "$PREBOOT_PLIST_PATH" 2>/dev/null
            printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0">\n<dict>\n</dict>\n</plist>\n' > "$PREBOOT_PLIST_PATH"
            
            # Run updatePreboot
            diskutil apfs updatePreboot / >/dev/null 2>&1
            
            # Restore system preferences
            defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE"
            
            echo "$(date): Periodic proactive clearing completed" >> "$LOG_FILE"
        else
            echo "$(date): Preboot volume already clear, no action needed" >> "$LOG_FILE"
        fi
    fi
    
    sleep 2
done

echo "$(date): Lock screen coordinator stopped" >> "$LOG_FILE"
EOF
    fi

    # Make script executable
    if [[ $EUID -eq 0 ]]; then
        chmod +x "$SCRIPT_DIR/lockscreen_coordinator.sh"
    else
        sudo chmod +x "$SCRIPT_DIR/lockscreen_coordinator.sh"
    fi

    # Create LaunchDaemon
    if [[ $EUID -eq 0 ]]; then
        tee /Library/LaunchDaemons/com.lockscreen.coordinator.plist > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.lockscreen.coordinator</string>
    <key>ProgramArguments</key>
    <array>
        <string>$SCRIPT_DIR/lockscreen_coordinator.sh</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/lockscreen_coordinator.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/lockscreen_coordinator.log</string>
    <key>ProcessType</key>
    <string>Background</string>
    <key>ThrottleInterval</key>
    <integer>10</integer>
    <key>ExitTimeOut</key>
    <integer>5</integer>
    <key>WorkingDirectory</key>
    <string>$SCRIPT_DIR</string>
    <key>UserName</key>
    <string>root</string>
    <key>GroupName</key>
    <string>wheel</string>
</dict>
</plist>
EOF
    else
        sudo tee /Library/LaunchDaemons/com.lockscreen.coordinator.plist > /dev/null << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.lockscreen.coordinator</string>
    <key>ProgramArguments</key>
    <array>
        <string>$SCRIPT_DIR/lockscreen_coordinator.sh</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/lockscreen_coordinator.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/lockscreen_coordinator.log</string>
    <key>ProcessType</key>
    <string>Background</string>
    <key>ThrottleInterval</key>
    <integer>10</integer>
    <key>ExitTimeOut</key>
    <integer>5</integer>
    <key>WorkingDirectory</key>
    <string>$SCRIPT_DIR</string>
    <key>UserName</key>
    <string>root</string>
    <key>GroupName</key>
    <string>wheel</string>
</dict>
</plist>
EOF
    fi

    # Set proper permissions
    if [[ $EUID -eq 0 ]]; then
        chown root:wheel /Library/LaunchDaemons/com.lockscreen.coordinator.plist
        chmod 644 /Library/LaunchDaemons/com.lockscreen.coordinator.plist
    else
        sudo chown root:wheel /Library/LaunchDaemons/com.lockscreen.coordinator.plist
        sudo chmod 644 /Library/LaunchDaemons/com.lockscreen.coordinator.plist
    fi

    # Load the LaunchDaemon
    if [[ $EUID -eq 0 ]]; then
        launchctl bootstrap system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null || true
    else
        sudo launchctl bootstrap system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null || true
    fi

    # Set the message immediately
    if [[ $EUID -eq 0 ]]; then
        defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE" 2>/dev/null || true
        diskutil apfs updatePreboot / >> "$LOG_FILE" 2>&1 || true
    else
        sudo defaults write /Library/Preferences/com.apple.loginwindow LoginwindowText "$LOCK_MESSAGE" 2>/dev/null || true
        sudo diskutil apfs updatePreboot / >> "$LOG_FILE" 2>&1 || true
    fi

    echo "✅ Installation complete!"
    echo "📋 Configuration:"
    echo "   Message: $LOCK_MESSAGE"
    echo "   Log file: $LOG_FILE"
    echo ""
    echo "🔄 The system will now:"
    echo "   • Set the message immediately (and on system startup)"
    echo "   • Monitor for shutdown/reboot events and clear the message"
    echo "   • Update preboot volume for FileVault compatibility"
    echo "   • Log all actions to $LOG_FILE"
}

# =============================================================================
# UNINSTALL SCRIPT
# =============================================================================

uninstall_lockscreen_manager() {
    echo "Uninstalling Lock Screen Message Manager..."

    # Unload services
    if [[ $EUID -eq 0 ]]; then
        launchctl bootout system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null
        rm -f /Library/LaunchDaemons/com.lockscreen.coordinator.plist
        rm -rf "$SCRIPT_DIR"
        defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null
    else
        sudo launchctl bootout system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null
        sudo rm -f /Library/LaunchDaemons/com.lockscreen.coordinator.plist
        sudo rm -rf "$SCRIPT_DIR"
        sudo defaults delete /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null
    fi

    echo "✅ Uninstallation complete!"
}

# =============================================================================
# STATUS CHECK
# =============================================================================

show_status() {
    echo "Lock Screen Message Manager Status"
    echo "=================================="
    
    # Check if files exist
    if [[ -f "/Library/LaunchDaemons/com.lockscreen.coordinator.plist" ]]; then
        echo "✅ Coordinator LaunchDaemon: Installed"
    else
        echo "❌ Coordinator LaunchDaemon: Not installed"
    fi
    
    # Check if script directory exists
    if [[ -d "$SCRIPT_DIR" ]]; then
        echo "✅ Script directory: $SCRIPT_DIR"
    else
        echo "❌ Script directory: Missing"
    fi
    
    # Check if LaunchDaemon is loaded
    if [[ $EUID -eq 0 ]]; then
        if launchctl list | grep -q "com.lockscreen.coordinator"; then
            echo "✅ Service: Running"
        else
            echo "❌ Service: Not running"
        fi
    else
        if sudo launchctl list | grep -q "com.lockscreen.coordinator"; then
            echo "✅ Service: Running"
        else
            echo "❌ Service: Not running"
        fi
    fi
    
    # Check current lock screen message
    if [[ $EUID -eq 0 ]]; then
        CURRENT_MESSAGE=$(defaults read /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null)
    else
        CURRENT_MESSAGE=$(sudo defaults read /Library/Preferences/com.apple.loginwindow LoginwindowText 2>/dev/null)
    fi
    
    if [[ -n "$CURRENT_MESSAGE" ]]; then
        echo "🔒 Current message: $CURRENT_MESSAGE"
    else
        echo "🔓 No lock screen message currently set"
    fi
    
    # Check log file
    if [[ -f "$LOG_FILE" ]]; then
        echo "📝 Recent log entries:"
        tail -5 "$LOG_FILE" | sed 's/^/   /'
    else
        echo "📝 No log file found"
    fi
}

# =============================================================================
# RESTART SERVICE
# =============================================================================

restart_service() {
    echo "Restarting lock screen manager service..."
    
    if [[ $EUID -eq 0 ]]; then
        launchctl bootout system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null
        sleep 1
        launchctl bootstrap system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null || true
    else
        sudo launchctl bootout system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null
        sleep 1
        sudo launchctl bootstrap system /Library/LaunchDaemons/com.lockscreen.coordinator.plist 2>/dev/null || true
    fi
    
    echo "✅ Service restarted"
}

# =============================================================================
# MAIN SCRIPT LOGIC
# =============================================================================

case "${1:-install}" in
    "install")
        install_lockscreen_manager
        ;;
    "uninstall")
        uninstall_lockscreen_manager
        ;;
    "set")
        check_sudo "$1"
        set_lock_message
        ;;
    "clear")
        check_sudo "$1"
        clear_lock_message
        ;;
    "restart")
        check_sudo "$1"
        restart_service
        ;;
    "status")
        show_status
        ;;
    *)
        echo "Lock Screen Message Manager"
        echo "=========================="
        echo ""
        echo "Usage: $0 {install|uninstall|set|clear|restart|status}"
        echo ""
        echo "Commands:"
        echo "  install    - Install the lock screen message manager (default)"
        echo "  uninstall  - Remove the lock screen message manager"
        echo "  set        - Set the lock screen message immediately"
        echo "  clear      - Clear the lock screen message immediately"
        echo "  restart    - Restart the lock screen manager service"
        echo "  status     - Show current status and configuration"
        echo ""
        echo "Features:"
        echo "  • Automatically sets message after login"
        echo "  • Monitors for shutdown/reboot events"
        echo "  • Automatically clears message before shutdown/restart"
        echo "  • FileVault compatible with preboot volume updates"
        echo "  • Comprehensive logging and status monitoring"
        echo ""
        echo "Example: $0 (runs install by default)"
        exit 1
        ;;
esac
