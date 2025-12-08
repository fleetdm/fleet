#!/bin/bash

# =============================================================================
# SLACK PREFERENCES MIGRATION SCRIPT
# =============================================================================
# This script migrates Slack preferences from the App Store (VPP) version
# to the Fleet-maintained app version to ensure a seamless transition.
#
# The App Store version stores preferences in sandboxed containers, while
# the Fleet-maintained version uses standard preference locations.
#
# IMPORTANT: This script ONLY performs migration when needed:
#   - It detects if App Store version exists by checking for sandboxed containers
#   - If no App Store version is detected, migration is skipped
#   - Migration only occurs for users who have App Store version data
#
# After migration, Slack is automatically launched for the console user
# (currently logged-in user) if a migration was performed.
# =============================================================================

# Don't use set -e as we want to handle errors gracefully
set -o pipefail

# Bundle identifiers
# Note: App Store version may use com.toddheasley.Slack or com.tinyspeck.slackmacgap
# The key difference is that App Store apps are sandboxed and store data in Containers
APP_STORE_BUNDLE_IDS=("com.toddheasley.Slack" "com.tinyspeck.slackmacgap")
FLEET_BUNDLE_ID="com.tinyspeck.slackmacgap"

# Logging
LOG_FILE="/var/log/slack-migration.log"
LOG_DIR="$(dirname "$LOG_FILE")"

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info() {
    log "INFO" "$@"
    echo "[INFO] $*"
}

log_success() {
    log "SUCCESS" "$@"
    echo "[SUCCESS] $*"
}

log_warning() {
    log "WARNING" "$@"
    echo "[WARNING] $*"
}

log_error() {
    log "ERROR" "$@"
    echo "[ERROR] $*" >&2
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Ensure log directory exists
ensure_log_dir() {
    mkdir -p "$LOG_DIR"
    touch "$LOG_FILE"
}

# Get all user home directories
get_user_homes() {
    # Get logged-in users
    local users
    users=$(dscl . -list /Users | grep -v "^_\|^root\|^daemon\|^nobody" || true)
    
    # Also check for home directories in /Users
    for user in $users; do
        local home_dir
        home_dir=$(dscl . -read "/Users/$user" NFSHomeDirectory 2>/dev/null | awk '{print $2}' || echo "")
        if [[ -n "$home_dir" && -d "$home_dir" ]]; then
            echo "$home_dir"
        fi
    done
}

# Check if App Store version is installed
# App Store apps are sandboxed and store data in ~/Library/Containers/
check_app_store_version() {
    local home_dir="$1"
    
    # Check for any Slack container (App Store versions are sandboxed)
    for bundle_id in "${APP_STORE_BUNDLE_IDS[@]}"; do
        local container_path="$home_dir/Library/Containers/$bundle_id"
        if [[ -d "$container_path" ]]; then
            echo "$bundle_id"
            return 0  # App Store version found
        fi
    done
    
    # Also check for com.tinyspeck.slackmacgap in Containers (some App Store versions use this)
    local slack_container="$home_dir/Library/Containers/com.tinyspeck.slackmacgap"
    if [[ -d "$slack_container" ]] && [[ -d "$slack_container/Data" ]]; then
        # If it has a Data subdirectory, it's likely sandboxed (App Store version)
        echo "com.tinyspeck.slackmacgap"
        return 0
    fi
    
    return 1  # App Store version not found
}

# Check if Fleet-maintained version is installed
check_fleet_version() {
    # Check if the app exists in /Applications
    if [[ -d "/Applications/Slack.app" ]]; then
        # Check bundle identifier
        local bundle_id
        bundle_id=$(mdls -name kMDItemCFBundleIdentifier "/Applications/Slack.app" 2>/dev/null | awk -F'"' '{print $2}' || echo "")
        if [[ "$bundle_id" == "$FLEET_BUNDLE_ID" ]]; then
            return 0  # Fleet version found
        fi
    fi
    return 1  # Fleet version not found
}

# Migrate preferences for a specific user
migrate_user_preferences() {
    local home_dir="$1"
    local username=$(basename "$home_dir")
    
    log_info "Processing user: $username"
    
    # Detect App Store bundle ID
    local detected_bundle_id
    detected_bundle_id=$(check_app_store_version "$home_dir")
    if [[ -z "$detected_bundle_id" ]]; then
        log_info "No App Store version detected for $username, skipping"
        return 0
    fi
    
    log_info "Detected App Store version with bundle ID: $detected_bundle_id"
    
    # App Store version paths (sandboxed container)
    local app_store_container="$home_dir/Library/Containers/$detected_bundle_id"
    local app_store_prefs="$app_store_container/Data/Library/Preferences"
    local app_store_app_support="$app_store_container/Data/Library/Application Support/Slack"
    
    # Fleet-maintained version paths
    local fleet_prefs_dir="$home_dir/Library/Preferences"
    local fleet_app_support="$home_dir/Library/Application Support/Slack"
    
    log_info "App Store version data found for $username, starting migration..."
    
    # Migrate preference plists
    if [[ -d "$app_store_prefs" ]]; then
        log_info "Migrating preference plists..."
        
        # Find all plist files related to Slack
        while IFS= read -r -d '' plist_file; do
            local plist_name=$(basename "$plist_file")
            local target_plist="$fleet_prefs_dir/$plist_name"
            
            # Skip if target already exists (unless it's empty or older)
            if [[ -f "$target_plist" ]]; then
                if [[ "$plist_file" -nt "$target_plist" ]]; then
                    log_warning "Target plist exists but source is newer: $plist_name"
                    log_info "Backing up existing plist and migrating..."
                    mv "$target_plist" "${target_plist}.backup.$(date +%Y%m%d%H%M%S)"
                else
                    log_info "Target plist already exists and is up to date: $plist_name"
                    continue
                fi
            fi
            
            # Copy the plist file
            if cp "$plist_file" "$target_plist" 2>/dev/null; then
                # Set correct ownership
                chown "$username:staff" "$target_plist" 2>/dev/null || true
                log_success "Migrated preference: $plist_name"
            else
                log_error "Failed to migrate preference: $plist_name"
            fi
        done < <(find "$app_store_prefs" -name "*.plist" -type f -print0 2>/dev/null || true)
    fi
    
    # Migrate Application Support data
    if [[ -d "$app_store_app_support" ]]; then
        log_info "Migrating Application Support data..."
        
        # Create target directory if it doesn't exist
        mkdir -p "$fleet_app_support"
        chown "$username:staff" "$fleet_app_support" 2>/dev/null || true
        
        # Check if target directory already has data
        if [[ -d "$fleet_app_support" ]] && [[ -n "$(ls -A "$fleet_app_support" 2>/dev/null)" ]]; then
            log_warning "Application Support directory already exists for $username"
            log_info "Merging data (App Store data will take precedence for conflicts)..."
            
            # Use rsync to merge, with App Store version taking precedence
            if rsync -a --ignore-existing "$app_store_app_support/" "$fleet_app_support/" 2>/dev/null; then
                log_success "Merged Application Support data for $username"
            else
                log_error "Failed to merge Application Support data for $username"
            fi
        else
            # Target is empty or doesn't exist, safe to copy
            if cp -R "$app_store_app_support"/* "$fleet_app_support/" 2>/dev/null; then
                # Set correct ownership recursively
                chown -R "$username:staff" "$fleet_app_support" 2>/dev/null || true
                log_success "Migrated Application Support data for $username"
            else
                log_error "Failed to migrate Application Support data for $username"
            fi
        fi
    fi
    
    # Migrate other potential locations
    # Caches
    local app_store_caches="$app_store_container/Data/Library/Caches/com.tinyspeck.slackmacgap*"
    if ls $app_store_caches 1>/dev/null 2>&1; then
        log_info "Found cache files, but skipping (caches will be regenerated)"
    fi
    
    # Cookies
    local app_store_cookies="$app_store_container/Data/Library/Cookies/com.tinyspeck.slackmacgap.binarycookies"
    if [[ -f "$app_store_cookies" ]]; then
        local fleet_cookies_dir="$home_dir/Library/Cookies"
        mkdir -p "$fleet_cookies_dir"
        if cp "$app_store_cookies" "$fleet_cookies_dir/" 2>/dev/null; then
            chown "$username:staff" "$fleet_cookies_dir/$(basename "$app_store_cookies")" 2>/dev/null || true
            log_success "Migrated cookies for $username"
        fi
    fi
    
    # Saved Application State
    local app_store_saved_state="$app_store_container/Data/Library/Saved Application State/com.tinyspeck.slackmacgap.savedState"
    if [[ -d "$app_store_saved_state" ]]; then
        local fleet_saved_state_dir="$home_dir/Library/Saved Application State"
        mkdir -p "$fleet_saved_state_dir"
        local saved_state_target="$fleet_saved_state_dir/com.tinyspeck.slackmacgap.savedState"
        if [[ -d "$saved_state_target" ]]; then
            rm -rf "$saved_state_target"
        fi
        if cp -R "$app_store_saved_state" "$saved_state_target" 2>/dev/null; then
            chown -R "$username:staff" "$saved_state_target" 2>/dev/null || true
            log_success "Migrated saved application state for $username"
        fi
    fi
    
    log_success "Migration completed for user: $username"
    
    # Return success status
    return 0
}

# Quit Slack application (following Google Chrome pattern)
quit_slack() {
    local bundle_id="$1"
    local console_user="$2"
    local timeout_duration=10

    if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
        log_info "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
        return
    fi

    log_info "Quitting application '$bundle_id'..."

    # try to quit the application within the timeout period
    local quit_success=false
    SECONDS=0
    while (( SECONDS < timeout_duration )); do
        if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
            if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
                log_success "Application '$bundle_id' quit successfully."
                quit_success=true
                break
            fi
        fi
        sleep 1
    done

    if [[ "$quit_success" = false ]]; then
        log_warning "Application '$bundle_id' did not quit."
    fi
}

# Launch Slack for the console user (following Google Chrome pattern)
# Only launches if Slack was running before migration
launch_slack() {
    local console_user="$1"
    local was_running="$2"
    
    if [[ -n "$console_user" && "$console_user" != "root" ]]; then
        # Only launch if Slack was running before
        if [[ "$was_running" != "true" ]]; then
            log_info "Slack was not running before migration, skipping launch"
            return 0
        fi
        
        log_info "Slack was running before migration, relaunching for console user: $console_user"
        
        # Check if Slack is already running and quit it first
        if osascript -e "application id \"com.tinyspeck.slackmacgap\" is running" 2>/dev/null; then
            log_info "Slack is already running, quitting first..."
            quit_slack "com.tinyspeck.slackmacgap" "$console_user"
            sleep 2
        fi
        
        if [[ -d "/Applications/Slack.app" ]]; then
            sudo -u "$console_user" open -a "Slack" 2>/dev/null || {
                log_warning "Failed to launch Slack for user $console_user"
                return 1
            }
            log_success "Slack launched successfully for user $console_user"
            return 0
        else
            log_warning "Slack.app not found in /Applications"
            return 1
        fi
    else
        log_info "No console user found or user is root, skipping Slack launch"
        return 0
    fi
}

# Clean up App Store container after successful migration
cleanup_app_store_container() {
    local home_dir="$1"
    local username="$2"
    local bundle_id="$3"
    
    local app_store_container="$home_dir/Library/Containers/$bundle_id"
    
    if [[ -d "$app_store_container" ]]; then
        log_info "Cleaning up App Store container for user $username..."
        
        # Remove the container directory
        if rm -rf "$app_store_container" 2>/dev/null; then
            log_success "Removed App Store container for user $username"
            return 0
        else
            log_warning "Failed to remove App Store container for user $username (may require user logout)"
            return 1
        fi
    else
        log_info "App Store container not found for user $username, nothing to clean up"
        return 0
    fi
}

# Main migration function
main() {
    log_info "Starting Slack preferences migration script..."
    log_info "================================================"
    
    ensure_log_dir
    
    # Check if Fleet-maintained version is installed
    if ! check_fleet_version; then
        log_warning "Fleet-maintained version of Slack not found in /Applications"
        log_info "This script can still migrate preferences in preparation for installation"
    else
        log_success "Fleet-maintained version of Slack detected"
    fi
    
    # Get all user home directories
    local user_homes
    user_homes=$(get_user_homes)
    
    if [[ -z "$user_homes" ]]; then
        log_info "No user home directories found (likely during Setup experience with no users created yet)"
        log_info "Skipping preferences migration - no migration needed at this time"
        log_success "Script completed successfully (no users to migrate)"
        exit 0
    fi
    
    # Get console user once (used for launching Slack)
    local console_user
    console_user=$(stat -f "%Su" /dev/console 2>/dev/null || echo "")
    
    # Check if Slack is running before migration (only relaunch if it was running)
    local slack_was_running=false
    if osascript -e "application id \"com.tinyspeck.slackmacgap\" is running" 2>/dev/null; then
        slack_was_running=true
        log_info "Slack is currently running - will relaunch after migration"
    else
        log_info "Slack is not currently running - will not launch after migration"
    fi
    
    local migration_count=0
    local skipped_count=0
    local migration_needed=false
    local containers_to_cleanup=()
    
    # Process each user
    while IFS= read -r home_dir; do
        if [[ -z "$home_dir" ]]; then
            continue
        fi
        
        local username=$(basename "$home_dir")
        local detected_bundle_id
        detected_bundle_id=$(check_app_store_version "$home_dir")
        if [[ -n "$detected_bundle_id" ]]; then
            # Migration is needed if App Store version is detected
            migration_needed=true
            if migrate_user_preferences "$home_dir"; then
                ((migration_count++))
                # Store container info for cleanup (bundle_id|home_dir|username)
                containers_to_cleanup+=("$detected_bundle_id|$home_dir|$username")
            else
                log_error "Migration failed for user: $username"
                ((skipped_count++))
            fi
        else
            log_info "No App Store version data found for $username, skipping"
            ((skipped_count++))
        fi
    done <<< "$user_homes"
    
    log_info "================================================"
    log_success "Migration completed!"
    log_info "Users migrated: $migration_count"
    log_info "Users skipped: $skipped_count"
    log_info "Log file: $LOG_FILE"
    
    # Clean up App Store containers if migration was performed
    if [[ "$migration_needed" == "true" && $migration_count -gt 0 ]]; then
        # Clean up App Store containers after successful migration
        log_info "Cleaning up App Store containers..."
        for container_info in "${containers_to_cleanup[@]}"; do
            IFS='|' read -r bundle_id home_dir username <<< "$container_info"
            cleanup_app_store_container "$home_dir" "$username" "$bundle_id"
        done
    fi
    
    # Launch Slack for console user (only if it was running before and Fleet version is installed)
    if check_fleet_version; then
        if [[ "$slack_was_running" == "true" ]]; then
            sleep 2
            launch_slack "$console_user" "$slack_was_running" || true
        fi
        
        if [[ "$migration_needed" == "true" && $migration_count -gt 0 ]]; then
            echo ""
            echo "✓ Preferences have been migrated successfully."
            if [[ "$slack_was_running" == "true" ]]; then
                echo "✓ Slack has been relaunched for the console user."
            fi
            echo "✓ App Store version containers have been cleaned up."
        elif [[ "$slack_was_running" == "true" ]]; then
            echo ""
            echo "✓ Slack has been relaunched for the console user."
        else
            echo ""
            echo "No App Store version data found to migrate."
        fi
    elif [[ "$migration_needed" == "false" ]]; then
        echo ""
        echo "No App Store version data found to migrate."
    else
        echo ""
        echo "Migration attempted but encountered errors. Check log file for details."
    fi
}

# Run main function
main "$@"

