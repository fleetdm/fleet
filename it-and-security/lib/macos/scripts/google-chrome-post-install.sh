#!/bin/bash

#
# Chrome restart post-Install script
# Restarts Google Chrome with session restoration
#

set -euo pipefail

# Configuration
readonly CHROME_APP_NAME="Google Chrome"
readonly CHROME_APP_PATH="/Applications/Google Chrome.app"
readonly WAIT_TIME=2
readonly MAX_KILL_WAIT=10
readonly MAX_RESTART_ATTEMPTS=3
readonly VERIFICATION_WAIT=3

# Post-install context - always run as root
readonly PKG_INSTALL_CONTEXT=true

# Verify running as root (required for pkg post-install)
verify_root() {
    if [[ $EUID -ne 0 ]]; then
        echo "ERROR: This script must be run as root (pkg post-install context)" >&2
        exit 1
    fi
}

# Enhanced logging function for pkg post-install context
log() {
    local level="$1"
    shift
    local message="$*"
    
    # In pkg post-install context, log to system log and stdout
    local log_message="$(date '+%Y-%m-%d %H:%M:%S') [Chrome Post-Install] [$level]: $message"
    
    # Always output to stdout for pkg installer visibility
    echo "$log_message"
    
    # Also log to system log for persistence
    logger -t "chrome-post-install" "$message"
}

# Cleanup function for signal handling
cleanup() {
    log "INFO" "Script interrupted, cleaning up..."
    exit 1
}

# Set up signal handlers
setup_signal_handlers() {
    trap cleanup SIGINT SIGTERM
}

# Enhanced Chrome process detection
get_chrome_pids() {
    # Get main Chrome process PIDs, excluding helpers and renderers
    pgrep -i "Google Chrome" 2>/dev/null | while read -r pid; do
        # Check if this is a main Chrome process (not helper/renderer)
        local cmdline
        cmdline=$(ps -p "$pid" -o command= 2>/dev/null || true)
        if [[ "$cmdline" =~ "Google Chrome" && ! "$cmdline" =~ "Helper" && ! "$cmdline" =~ "Renderer" ]]; then
            echo "$pid"
        fi
    done
}

# Check if Chrome is running (enhanced)
is_chrome_running() {
    local pids
    pids=$(get_chrome_pids)
    [[ -n "$pids" ]]
}

# Verify Chrome startup
verify_chrome_started() {
    log "DEBUG" "Verifying Chrome startup..."
    sleep "$VERIFICATION_WAIT"
    
    if is_chrome_running; then
        log "INFO" "Chrome startup verified successfully"
        return 0
    else
        log "WARN" "Chrome may not have started properly"
        return 1
    fi
}

# Restart Chrome with retry logic (optimized for post-install)
restart_chrome_with_retry() {
    local attempt=1
    
    while [[ $attempt -le $MAX_RESTART_ATTEMPTS ]]; do
        log "INFO" "Restarting Chrome (attempt $attempt/$MAX_RESTART_ATTEMPTS)..."
        
        # In post-install context, we need to restart Chrome for the logged-in user
        # Use launchctl to start Chrome for the current console user
        local console_user
        console_user=$(stat -f %Su /dev/console 2>/dev/null || echo "")
        
        if [[ -n "$console_user" && "$console_user" != "root" ]]; then
            # Start Chrome as the console user
            if sudo -u "$console_user" open -a "$CHROME_APP_NAME" --args --restore-last-session; then
                if verify_chrome_started; then
                    log "INFO" "Chrome restart completed successfully for user: $console_user"
                    return 0
                else
                    log "WARN" "Chrome started but verification failed"
                fi
            else
                log "WARN" "Chrome restart attempt $attempt failed for user: $console_user"
            fi
        else
            # Fallback: try to start Chrome directly (may not work without user context)
            log "WARN" "No console user found, attempting direct Chrome start..."
            if open -a "$CHROME_APP_NAME" --args --restore-last-session; then
                if verify_chrome_started; then
                    log "INFO" "Chrome restart completed successfully"
                    return 0
                else
                    log "WARN" "Chrome started but verification failed"
                fi
            else
                log "WARN" "Chrome restart attempt $attempt failed"
            fi
        fi
        
        if [[ $attempt -lt $MAX_RESTART_ATTEMPTS ]]; then
            log "INFO" "Waiting before retry..."
            sleep 2
        fi
        
        ((attempt++))
    done
    
    log "ERROR" "Failed to restart Chrome after $MAX_RESTART_ATTEMPTS attempts"
    return 1
}

# Gracefully terminate Chrome processes (optimized for post-install)
terminate_chrome() {
    log "INFO" "Terminating Chrome processes..."
    
    # Get current Chrome PIDs for logging
    local pids
    pids=$(get_chrome_pids)
    if [[ -n "$pids" ]]; then
        log "INFO" "Found Chrome PIDs: $pids"
    fi
    
    # Send TERM signal to Chrome processes
    pkill -TERM "Google Chrome" 2>/dev/null || true
    
    # Wait for graceful shutdown
    local count=0
    while is_chrome_running && [[ $count -lt $MAX_KILL_WAIT ]]; do
        sleep 1
        ((count++))
        log "INFO" "Waiting for graceful shutdown... ($count/$MAX_KILL_WAIT)"
    done
    
    # Force kill if still running
    if is_chrome_running; then
        log "WARN" "Graceful shutdown failed, force killing Chrome..."
        pkill -KILL "Google Chrome" 2>/dev/null || true
        sleep 1
        
        # Final check
        if is_chrome_running; then
            log "ERROR" "Failed to terminate Chrome processes"
            return 1
        fi
    fi
    
    log "INFO" "Chrome processes terminated successfully"
    return 0
}

# Main execution function (optimized for pkg post-install)
main() {
    # Verify we're running as root (required for pkg post-install)
    verify_root
    
    # Set up signal handlers
    setup_signal_handlers
    
    log "INFO" "Starting Chrome post-install restart process..."
    log "INFO" "Running as root in pkg post-install context"
    
    # Check if Chrome is installed
    if [[ ! -d "$CHROME_APP_PATH" ]]; then
        log "ERROR" "Google Chrome not found at $CHROME_APP_PATH"
        exit 1
    fi
    
    log "INFO" "Chrome installation verified at $CHROME_APP_PATH"
    
    # Check if Chrome is running
    if ! is_chrome_running; then
        log "INFO" "Chrome is not currently running, nothing to restart"
        exit 0
    fi
    
    log "INFO" "Chrome is currently running, proceeding with restart"
    
    # Terminate Chrome
    if ! terminate_chrome; then
        log "ERROR" "Failed to terminate Chrome processes"
        exit 1
    fi
    
    # Wait before restart
    log "INFO" "Waiting ${WAIT_TIME} seconds before restart..."
    sleep "$WAIT_TIME"
    
    # Restart Chrome with retry logic
    if ! restart_chrome_with_retry; then
        log "ERROR" "Failed to restart Chrome"
        exit 1
    fi
    
    log "INFO" "Chrome post-install restart process completed successfully"
}

# Execute main function
main "$@"
