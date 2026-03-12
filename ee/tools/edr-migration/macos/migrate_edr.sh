#!/bin/bash
# =============================================================================
# Grubhub EDR Migration Script — macOS
# Migrates endpoints from SentinelOne to CrowdStrike Falcon
#
# Designed for deployment via Fleet (or any MDM).
# Uses SwiftDialog (/usr/local/bin/dialog) for end-user UI.
#
# Copyright (c) Grubhub IT
# =============================================================================

set -euo pipefail

# =============================================================================
# CONFIGURATION — Set these variables before deployment
# =============================================================================

# Path to the CrowdStrike Falcon sensor installer PKG
CROWDSTRIKE_PKG_PATH=""

# CrowdStrike Customer ID (CID) — obtain from your Falcon console
CROWDSTRIKE_CID=""

# SentinelOne management passphrase — required for uninstall on managed endpoints
SENTINELONE_PASSPHRASE=""

# Support contact info shown to end users on failure
SUPPORT_EMAIL="it-security@grubhub.com"
SUPPORT_URL="https://grubhub.service-now.com"

# SwiftDialog icon — can be an SF Symbol name (e.g., "shield.checkerboard") or
# an absolute path to an image file (e.g., "/path/to/grubhub_logo.png")
DIALOG_ICON="shield.checkerboard"

# Optional banner image for the welcome dialog (leave empty to skip)
DIALOG_BANNER_IMAGE=""

# SwiftDialog binary path
DIALOG_BIN="/usr/local/bin/dialog"

# Log file
LOG_FILE="/var/log/grubhub_edr_migration.log"

# CrowdStrike paths
FALCONCTL="/Applications/Falcon.app/Contents/Resources/falconctl"

# SentinelOne paths
SENTINELONE_APP_DIR="/Applications/SentinelOne"
SENTINELONE_UNINSTALL_SCRIPT="/Library/Sentinel/uninstall.sh"
SENTINELCTL="/usr/local/bin/sentinelctl"

# =============================================================================
# INTERNAL VARIABLES — Do not modify
# =============================================================================

DIALOG_CMD_FILE=$(mktemp /tmp/grubhub_dialog_cmd.XXXXXX)
MIGRATION_SUCCESS=false
SCRIPT_NAME="$(basename "$0")"
TIMESTAMP_FMT="+%Y-%m-%d %H:%M:%S"

# =============================================================================
# FUNCTIONS
# =============================================================================

# --- Logging ---

log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp
    timestamp=$(date "$TIMESTAMP_FMT")
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info()  { log "INFO"  "$@"; }
log_warn()  { log "WARN"  "$@"; }
log_error() { log "ERROR" "$@"; }

# --- Cleanup ---

cleanup() {
    local exit_code=$?
    log_info "Cleanup: removing temporary files"
    rm -f "$DIALOG_CMD_FILE" 2>/dev/null || true

    if [[ "$MIGRATION_SUCCESS" != "true" && $exit_code -ne 0 ]]; then
        log_error "Migration did not complete successfully (exit code: $exit_code)"
        show_failure_dialog "The migration encountered an unexpected error (code: $exit_code). Please contact Grubhub IT for assistance."
    fi

    exit "$exit_code"
}

trap cleanup EXIT

# --- SwiftDialog helpers ---

# Update the progress dialog via the command file
dialog_update() {
    echo "$@" >> "$DIALOG_CMD_FILE"
}

# Show a simple informational dialog (blocking)
show_info_dialog() {
    local title="$1"
    local message="$2"
    local button_text="${3:-OK}"

    local cmd=("$DIALOG_BIN"
        --title "$title"
        --message "$message"
        --button1text "$button_text"
        --icon "$DIALOG_ICON"
        --messagefont "size=14"
        --infobuttontext "Grubhub IT Support"
        --infobuttonaction "$SUPPORT_URL"
    )

    if [[ -n "$DIALOG_BANNER_IMAGE" ]]; then
        cmd+=(--bannerimage "$DIALOG_BANNER_IMAGE")
    fi

    "${cmd[@]}" 2>/dev/null || true
}

# Show the failure dialog with error details
show_failure_dialog() {
    local error_message="$1"

    "$DIALOG_BIN" \
        --title "Migration Failed — Grubhub IT" \
        --message "## Endpoint Security Migration Failed\n\n$error_message\n\n**Please contact Grubhub IT for assistance:**\n\n- **Email:** $SUPPORT_EMAIL\n- **Portal:** [$SUPPORT_URL]($SUPPORT_URL)\n\nPlease include your computer name and a description of the issue." \
        --button1text "Close" \
        --icon "xmark.shield.fill" \
        --iconcolor "#F63440" \
        --messagefont "size=14" \
        2>/dev/null || true
}

# Show the success dialog
show_success_dialog() {
    "$DIALOG_BIN" \
        --title "Migration Complete — Grubhub IT" \
        --message "## Your Mac is now protected by CrowdStrike Falcon\n\nThe endpoint security migration has completed successfully. No further action is needed on your part.\n\nThank you for your patience!\n\n— **Grubhub IT**" \
        --button1text "Done" \
        --icon "checkmark.shield.fill" \
        --iconcolor "#00A651" \
        --messagefont "size=14" \
        2>/dev/null || true
}

# Start the progress dialog in the background
start_progress_dialog() {
    "$DIALOG_BIN" \
        --title "Grubhub IT — Endpoint Security Migration" \
        --message "Preparing migration..." \
        --icon "$DIALOG_ICON" \
        --progress 100 \
        --progresstext "Initializing..." \
        --button1disabled \
        --commandfile "$DIALOG_CMD_FILE" \
        --messagefont "size=14" \
        --position center \
        --ontop \
        2>/dev/null &
    DIALOG_PID=$!
    # Give the dialog a moment to initialize and read the command file
    sleep 1
}

# Close the progress dialog
close_progress_dialog() {
    dialog_update "quit:"
    sleep 1
    # Ensure the dialog process is terminated
    kill "$DIALOG_PID" 2>/dev/null || true
    wait "$DIALOG_PID" 2>/dev/null || true
}

# --- Migration step helpers ---

update_progress() {
    local percent="$1"
    local status_text="$2"
    dialog_update "progress: $percent"
    dialog_update "progresstext: $status_text"
    dialog_update "message: $status_text"
    log_info "[$percent%] $status_text"
    sleep 1
}

# Run a command and handle failure
run_step() {
    local description="$1"
    shift
    log_info "Executing: $description"
    log_info "Command: $*"

    local output
    if output=$("$@" 2>&1); then
        log_info "Success: $description"
        [[ -n "$output" ]] && log_info "Output: $output"
        return 0
    else
        local rc=$?
        log_error "Failed: $description (exit code: $rc)"
        [[ -n "$output" ]] && log_error "Output: $output"
        return $rc
    fi
}

# =============================================================================
# PRE-FLIGHT CHECKS
# =============================================================================

preflight_checks() {
    log_info "========================================="
    log_info "Grubhub EDR Migration — Starting"
    log_info "========================================="
    log_info "Script: $SCRIPT_NAME"
    log_info "Hostname: $(hostname)"
    log_info "macOS Version: $(sw_vers -productVersion)"
    log_info "Running as: $(whoami)"

    local preflight_failed=false

    # Must run as root
    if [[ "$(id -u)" -ne 0 ]]; then
        log_error "This script must be run as root (via MDM or sudo)"
        echo "ERROR: This script must be run as root." >&2
        exit 1
    fi

    # Check SwiftDialog is installed
    # NOTE: If SwiftDialog is not present, install it first via your MDM.
    #   You can install it with:
    #     curl -L "https://github.com/swiftDialog/swiftDialog/releases/latest/download/dialog-2.5.2-4777.pkg" \
    #       -o /tmp/dialog.pkg && installer -pkg /tmp/dialog.pkg -target /
    if [[ ! -x "$DIALOG_BIN" ]]; then
        log_error "SwiftDialog not found at $DIALOG_BIN"
        log_error "Install SwiftDialog before running this migration."
        echo "ERROR: SwiftDialog is not installed. Deploy it first via MDM." >&2
        exit 1
    fi
    log_info "Preflight: SwiftDialog found at $DIALOG_BIN"

    # Check SentinelOne is installed
    if [[ ! -d "$SENTINELONE_APP_DIR" ]] && ! command -v sentinelctl &>/dev/null; then
        log_warn "SentinelOne does not appear to be installed. Skipping uninstall."
        log_warn "Will proceed with CrowdStrike installation only."
    else
        log_info "Preflight: SentinelOne installation detected"
    fi

    # Check CrowdStrike installer is accessible
    if [[ -z "$CROWDSTRIKE_PKG_PATH" ]]; then
        log_error "CROWDSTRIKE_PKG_PATH is not configured. Set this variable before running."
        preflight_failed=true
    elif [[ ! -f "$CROWDSTRIKE_PKG_PATH" ]]; then
        log_error "CrowdStrike installer not found at: $CROWDSTRIKE_PKG_PATH"
        preflight_failed=true
    else
        log_info "Preflight: CrowdStrike installer found at $CROWDSTRIKE_PKG_PATH"
    fi

    # Check CrowdStrike CID is configured
    if [[ -z "$CROWDSTRIKE_CID" ]]; then
        log_error "CROWDSTRIKE_CID is not configured. Set this variable before running."
        preflight_failed=true
    fi

    if [[ "$preflight_failed" == "true" ]]; then
        log_error "Preflight checks failed. Aborting migration."
        exit 1
    fi

    # Check if CrowdStrike is already installed
    if [[ -x "$FALCONCTL" ]]; then
        local cs_status
        if cs_status=$("$FALCONCTL" stats 2>&1) && echo "$cs_status" | grep -qi "running"; then
            log_info "CrowdStrike Falcon is already installed and running. Nothing to do."
            show_info_dialog \
                "Already Protected — Grubhub IT" \
                "CrowdStrike Falcon is already installed and running on this Mac. No migration is needed.\n\n— **Grubhub IT**"
            exit 0
        fi
    fi

    log_info "Preflight checks passed"
}

# =============================================================================
# WELCOME DIALOG
# =============================================================================

show_welcome() {
    log_info "Displaying welcome dialog to user"

    local welcome_message="## Endpoint Security Upgrade\n\n"
    welcome_message+="**Grubhub IT** is upgrading your endpoint security from **SentinelOne** to **CrowdStrike Falcon**.\n\n"
    welcome_message+="This is a routine security upgrade that will:\n\n"
    welcome_message+="1. Safely remove SentinelOne\n"
    welcome_message+="2. Install CrowdStrike Falcon\n"
    welcome_message+="3. Verify the new protection is active\n\n"
    welcome_message+="**Estimated time:** ~5 minutes\n\n"
    welcome_message+="You can continue working during the migration. A notification will appear when it's complete.\n\n"
    welcome_message+="*If you have questions, contact Grubhub IT at $SUPPORT_EMAIL*"

    local cmd=("$DIALOG_BIN"
        --title "Grubhub IT — Endpoint Security Migration"
        --message "$welcome_message"
        --button1text "Begin Migration"
        --button2text "Postpone"
        --icon "$DIALOG_ICON"
        --iconcolor "#F63440"
        --messagefont "size=14"
        --infobuttontext "Grubhub IT Support"
        --infobuttonaction "$SUPPORT_URL"
        --timer 300
        --hidetimerbar
    )

    if [[ -n "$DIALOG_BANNER_IMAGE" ]]; then
        cmd+=(--bannerimage "$DIALOG_BANNER_IMAGE")
    fi

    local dialog_result=0
    "${cmd[@]}" 2>/dev/null || dialog_result=$?

    # SwiftDialog exit codes: 0 = button1, 2 = button2, 4 = timer expired
    case $dialog_result in
        0|4)
            log_info "User accepted the migration (exit code: $dialog_result)"
            ;;
        2)
            log_info "User postponed the migration"
            exit 0
            ;;
        *)
            log_warn "Unexpected dialog exit code: $dialog_result — proceeding with migration"
            ;;
    esac
}

# =============================================================================
# MIGRATION STEPS
# =============================================================================

uninstall_sentinelone() {
    # Check if SentinelOne is installed before attempting uninstall
    if [[ ! -d "$SENTINELONE_APP_DIR" ]] && ! command -v sentinelctl &>/dev/null; then
        log_info "SentinelOne is not installed — skipping uninstall"
        update_progress 50 "SentinelOne not detected — skipping removal..."
        return 0
    fi

    # Stop SentinelOne services
    update_progress 25 "Stopping SentinelOne services..."
    if [[ -x "$SENTINELCTL" ]]; then
        run_step "Unload SentinelOne" "$SENTINELCTL" unload || {
            log_warn "sentinelctl unload returned non-zero — continuing anyway"
        }
    fi
    sleep 2

    # Uninstall SentinelOne
    update_progress 40 "Uninstalling SentinelOne..."

    local uninstall_success=false

    # Method 1: Use the SentinelOne uninstall script (preferred)
    if [[ -x "$SENTINELONE_UNINSTALL_SCRIPT" ]]; then
        log_info "Using SentinelOne uninstall script: $SENTINELONE_UNINSTALL_SCRIPT"
        if [[ -n "$SENTINELONE_PASSPHRASE" ]]; then
            run_step "SentinelOne uninstall (script + passphrase)" \
                "$SENTINELONE_UNINSTALL_SCRIPT" --passphrase="$SENTINELONE_PASSPHRASE" && uninstall_success=true
        else
            run_step "SentinelOne uninstall (script)" \
                "$SENTINELONE_UNINSTALL_SCRIPT" && uninstall_success=true
        fi
    fi

    # Method 2: Use sentinelctl uninstall as fallback
    if [[ "$uninstall_success" != "true" && -x "$SENTINELCTL" ]]; then
        log_info "Falling back to sentinelctl uninstall"
        if [[ -n "$SENTINELONE_PASSPHRASE" ]]; then
            run_step "SentinelOne uninstall (sentinelctl + passphrase)" \
                "$SENTINELCTL" uninstall --passphrase "$SENTINELONE_PASSPHRASE" && uninstall_success=true
        else
            run_step "SentinelOne uninstall (sentinelctl)" \
                "$SENTINELCTL" uninstall && uninstall_success=true
        fi
    fi

    if [[ "$uninstall_success" != "true" ]]; then
        log_error "Failed to uninstall SentinelOne using all available methods"
        return 1
    fi

    # Verify removal
    update_progress 50 "Verifying SentinelOne removal..."
    sleep 3

    if [[ -d "$SENTINELONE_APP_DIR" ]]; then
        log_warn "SentinelOne application directory still exists — may require reboot to complete removal"
    else
        log_info "SentinelOne successfully removed"
    fi

    return 0
}

install_crowdstrike() {
    # Install CrowdStrike Falcon sensor
    update_progress 60 "Installing CrowdStrike Falcon..."

    run_step "Install CrowdStrike Falcon PKG" \
        installer -pkg "$CROWDSTRIKE_PKG_PATH" -target / || {
        log_error "CrowdStrike Falcon installation failed"
        return 1
    }

    sleep 2

    # Configure CrowdStrike Falcon with CID
    update_progress 75 "Configuring CrowdStrike Falcon..."

    if [[ ! -x "$FALCONCTL" ]]; then
        # Check alternative path
        if [[ -x "/Library/CS/falconctl" ]]; then
            FALCONCTL="/Library/CS/falconctl"
        else
            log_error "falconctl not found after installation"
            return 1
        fi
    fi

    run_step "Set CrowdStrike CID" \
        "$FALCONCTL" license "$CROWDSTRIKE_CID" || {
        log_error "Failed to set CrowdStrike CID"
        return 1
    }

    # Start CrowdStrike services
    update_progress 85 "Starting CrowdStrike services..."

    run_step "Load CrowdStrike Falcon" \
        "$FALCONCTL" load || {
        log_warn "falconctl load returned non-zero — service may already be running"
    }

    sleep 3

    # Verify CrowdStrike installation
    update_progress 90 "Verifying CrowdStrike installation..."

    local cs_stats
    if cs_stats=$("$FALCONCTL" stats 2>&1); then
        log_info "CrowdStrike Falcon stats:"
        log_info "$cs_stats"

        if echo "$cs_stats" | grep -qi "sensor"; then
            log_info "CrowdStrike Falcon sensor is operational"
        else
            log_warn "CrowdStrike Falcon stats returned but sensor status unclear"
        fi
    else
        log_warn "Could not retrieve CrowdStrike Falcon stats — sensor may still be initializing"
    fi

    return 0
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

main() {
    # Initialize log
    mkdir -p "$(dirname "$LOG_FILE")"
    touch "$LOG_FILE"

    # Pre-flight checks (no UI yet — exits on failure)
    preflight_checks

    # Show welcome dialog — user can postpone
    show_welcome

    # Start progress dialog
    start_progress_dialog
    update_progress 10 "Preparing migration..."

    # Uninstall SentinelOne
    if ! uninstall_sentinelone; then
        close_progress_dialog
        show_failure_dialog "Failed to uninstall SentinelOne. The previous security software could not be removed.\n\nPlease contact Grubhub IT for manual assistance."
        exit 1
    fi

    # Install and configure CrowdStrike
    if ! install_crowdstrike; then
        close_progress_dialog
        show_failure_dialog "Failed to install CrowdStrike Falcon. The new security software could not be installed.\n\nPlease contact Grubhub IT for manual assistance."
        exit 1
    fi

    # Migration complete
    update_progress 100 "Migration complete!"
    sleep 2
    close_progress_dialog

    # Show success dialog
    MIGRATION_SUCCESS=true
    show_success_dialog

    log_info "========================================="
    log_info "Grubhub EDR Migration — Completed Successfully"
    log_info "========================================="

    exit 0
}

# Run
main "$@"
