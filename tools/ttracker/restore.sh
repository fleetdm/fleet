#!/bin/bash
# ttracker/restore.sh
#
# Restores Claude sessions from the latest snapshot.
# - Reads latest.json to find all Claude sessions that should be open
# - Checks which are already running (by Claude session ID)
# - Opens new iTerm2 windows only for missing sessions
# - Resumes each with `claude --resume <session_id>`
# - Restores the badge on each window
#
# Usage:
#   ./restore.sh              # Interactive - prompts for options
#   ./restore.sh --dangerous  # Use --dangerously-skip-permissions (like cld-dng)
#   ./restore.sh --safe       # Use default permissions
#   ./restore.sh --dry-run    # Show what would be restored without doing it
#   ./restore.sh --file path  # Use a specific snapshot file instead of latest

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SNAPSHOT_DIR="$SCRIPT_DIR/snapshots"
SNAPSHOT_FILE="$SNAPSHOT_DIR/latest.json"
DRY_RUN=false
PERM_MODE=""

# Colors
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
MAGENTA='\033[35m'
RESET='\033[0m'

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --dangerous) PERM_MODE="dangerous"; shift ;;
        --safe)      PERM_MODE="safe"; shift ;;
        --dry-run)   DRY_RUN=true; shift ;;
        --file)      SNAPSHOT_FILE="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: restore.sh [--dangerous|--safe] [--dry-run] [--file SNAPSHOT]"
            echo "  --dangerous   Use --dangerously-skip-permissions (like cld-dng alias)"
            echo "  --safe        Use default Claude permissions"
            echo "  --dry-run     Show what would be restored without doing it"
            echo "  --file PATH   Restore from a specific snapshot file"
            exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Check snapshot exists
if [ ! -f "$SNAPSHOT_FILE" ]; then
    printf "${RED}Error: No snapshot found at %s${RESET}\n" "$SNAPSHOT_FILE"
    printf "Run daemon.sh first to create a snapshot.\n"
    exit 1
fi

# Read snapshot
snapshot_time=$(python3 -c "import json; print(json.load(open('$SNAPSHOT_FILE'))['timestamp'])")
printf "\n${BOLD}Terminal Tracker - Restore${RESET}\n"
printf "Snapshot: ${CYAN}%s${RESET} from ${CYAN}%s${RESET}\n\n" "$(basename "$SNAPSHOT_FILE")" "$snapshot_time"

# Get Claude sessions from the snapshot
saved_sessions=$(python3 -c "
import json
data = json.load(open('$SNAPSHOT_FILE'))
for s in data['sessions']:
    sid = s.get('claude_session_id', '')
    if sid:
        badge = s.get('badge', '')
        cwd = s.get('cwd', '')
        name = s.get('session_name', '')
        print(f'{sid}\t{badge}\t{cwd}\t{name}')
")

if [ -z "$saved_sessions" ]; then
    printf "${YELLOW}No Claude sessions found in snapshot.${RESET}\n"
    exit 0
fi

# Get currently running Claude session IDs
current_sessions=""
for f in "$HOME"/.claude/sessions/*.json; do
    [ -f "$f" ] || continue
    pid=$(basename "$f" .json)
    # Check if the process is actually running
    if ps -p "$pid" > /dev/null 2>&1; then
        sid=$(python3 -c "import json; print(json.load(open('$f'))['sessionId'])" 2>/dev/null || echo "")
        if [ -n "$sid" ]; then
            current_sessions+="$sid"$'\n'
        fi
    fi
done

# Load parked (done) sessions from history
HISTORY_FILE="$SNAPSHOT_DIR/history.json"
dismissed_sessions=""
if [ -f "$HISTORY_FILE" ]; then
    dismissed_sessions=$(python3 -c "
import json
with open('$HISTORY_FILE') as f:
    history = json.load(f)
for h in history:
    sid = h.get('claude_session_id', '')
    if sid:
        print(sid)
" 2>/dev/null || echo "")
fi

# Compare and find missing sessions
missing_count=0
already_count=0
dismissed_count=0
missing_list=""

printf "${BOLD}%-4s %-20s %-40s %s${RESET}\n" "#" "Badge" "Session Name" "Status"
printf "%s\n" "$(printf '%.0s-' {1..100})"

idx=0
while IFS=$'\t' read -r sid badge cwd name; do
    [ -z "$sid" ] && continue
    idx=$((idx + 1))

    display_name="${name:0:38}"
    display_badge="${badge:0:18}"

    if echo "$dismissed_sessions" | grep -q "^${sid}$"; then
        dismissed_count=$((dismissed_count + 1))
        printf "${DIM}%-4s %-20s %-40s parked${RESET}\n" "$idx" "$display_badge" "$display_name"
    elif echo "$current_sessions" | grep -q "^${sid}$"; then
        already_count=$((already_count + 1))
        printf "${DIM}%-4s %-20s %-40s ${GREEN}already open${RESET}\n" "$idx" "$display_badge" "$display_name"
    else
        missing_count=$((missing_count + 1))
        missing_list+="${sid}"$'\t'"${badge}"$'\t'"${cwd}"$'\t'"${name}"$'\n'
        printf "${YELLOW}%-4s${RESET} ${MAGENTA}%-20s${RESET} %-40s ${RED}MISSING${RESET}\n" "$idx" "$display_badge" "$display_name"
    fi
done <<< "$saved_sessions"

printf "%s\n" "$(printf '%.0s-' {1..100})"
printf "\n${BOLD}Already open:${RESET} %d  |  ${BOLD}Missing:${RESET} %d  |  ${BOLD}Parked:${RESET} %d\n\n" "$already_count" "$missing_count" "$dismissed_count"

if [ "$missing_count" -eq 0 ]; then
    printf "${GREEN}All sessions are already open. Nothing to restore.${RESET}\n\n"
    exit 0
fi

if [ "$DRY_RUN" = true ]; then
    printf "${YELLOW}Dry run - no terminals will be opened.${RESET}\n\n"
    exit 0
fi

# Prompt for permission mode if not specified
if [ -z "$PERM_MODE" ]; then
    printf "How should Claude be launched?\n"
    printf "  ${BOLD}1${RESET}) ${YELLOW}cld-dng${RESET} - dangerously skip permissions (like your alias)\n"
    printf "  ${BOLD}2${RESET}) ${GREEN}safe${RESET}    - default Claude permissions\n"
    printf "\n"
    read -r -p "Choose [1/2]: " choice
    case "$choice" in
        1) PERM_MODE="dangerous" ;;
        2) PERM_MODE="safe" ;;
        *)
            printf "${RED}Invalid choice. Aborting.${RESET}\n"
            exit 1 ;;
    esac
    printf "\n"
fi

if [ "$PERM_MODE" = "dangerous" ]; then
    CLAUDE_CMD="claude --dangerously-skip-permissions --resume"
    printf "Mode: ${YELLOW}dangerously-skip-permissions${RESET}\n\n"
else
    CLAUDE_CMD="claude --resume"
    printf "Mode: ${GREEN}safe (default permissions)${RESET}\n\n"
fi

# Restore missing sessions
restored=0
while IFS=$'\t' read -r sid badge cwd name; do
    [ -z "$sid" ] && continue

    # Default cwd to home if empty
    target_cwd="${cwd:-$HOME}"

    printf "Restoring: ${MAGENTA}%s${RESET} (%s)..." "$badge" "${sid:0:8}..."

    # Encode badge for the SetBadgeFormat escape sequence
    badge_b64=$(echo -n "$badge" | base64)

    # Open a new iTerm2 window, set badge, then launch claude --resume.
    # Each step is a separate "write text" with delays so the terminal processes
    # the badge escape sequence before claude's TUI takes over.
    # We write a temp AppleScript file to avoid heredoc escaping issues.
    tmp_script=$(mktemp /tmp/tt-restore-XXXXXX.applescript)
    cat > "$tmp_script" << EOF
tell application "iTerm2"
    set newWindow to (create window with default profile)
    tell current session of current tab of newWindow
        write text "cd $target_cwd"
        delay 1
        write text "printf '\\\\e]1337;SetBadgeFormat=%s\\\\a' '$badge_b64'"
        delay 2
        write text "$CLAUDE_CMD $sid"
    end tell
end tell
EOF
    osascript "$tmp_script" 2>/dev/null
    rm -f "$tmp_script"

    restored=$((restored + 1))
    printf " ${GREEN}done${RESET}\n"

    # Small delay between opens to avoid overwhelming iTerm2
    sleep 1
done <<< "$missing_list"

printf "\n${GREEN}${BOLD}Restored %d session(s).${RESET}\n\n" "$restored"
