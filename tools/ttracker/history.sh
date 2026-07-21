#!/bin/bash
# ttracker/history.sh
#
# Shows parked Claude sessions and lets you reopen one.
#
# Usage (typically via alias):
#   tthistory

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SNAPSHOT_DIR="$SCRIPT_DIR/snapshots"
HISTORY_FILE="$SNAPSHOT_DIR/history.json"

# Colors
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
MAGENTA='\033[35m'
RESET='\033[0m'

if [ ! -f "$HISTORY_FILE" ]; then
    echo "No parked sessions yet."
    exit 0
fi

# Get current running session IDs to mark any that are already open
current_sessions=""
for f in "$HOME"/.claude/sessions/*.json; do
    [ -f "$f" ] || continue
    pid=$(basename "$f" .json)
    if ps -p "$pid" > /dev/null 2>&1; then
        sid=$(python3 -c "import json; print(json.load(open('$f'))['sessionId'])" 2>/dev/null || echo "")
        if [ -n "$sid" ]; then
            current_sessions+="$sid"$'\n'
        fi
    fi
done

# Display history table
entries=$(python3 -c "
import json
with open('$HISTORY_FILE') as f:
    history = json.load(f)
if not history:
    exit(1)
for i, h in enumerate(history):
    sid = h.get('claude_session_id', '')
    badge = h.get('badge', '')
    name = h.get('session_name', '')
    parked = h.get('parked_at', '?')
    cwd = h.get('cwd', '')
    print(f'{i}\t{sid}\t{badge}\t{name}\t{parked}\t{cwd}')
" 2>/dev/null || echo "")

if [ -z "$entries" ]; then
    echo "No parked sessions."
    exit 0
fi

printf "\n${BOLD}Parked Sessions${RESET}\n\n"
printf "${BOLD}%-4s %-20s %-36s %-18s %s${RESET}\n" "#" "Badge" "Session Name" "Parked At" "Status"
printf "%s\n" "$(printf '%.0s-' {1..110})"

count=0
while IFS=$'\t' read -r idx sid badge name parked cwd; do
    [ -z "$idx" ] && [ -z "$sid" ] && continue
    count=$((count + 1))

    display_name="${name:0:34}"
    display_badge="${badge:0:18}"
    display_num=$((idx + 1))

    if echo "$current_sessions" | grep -q "^${sid}$"; then
        printf "${DIM}%-4s %-20s %-36s %-18s ${GREEN}open${RESET}\n" \
            "$display_num" "$display_badge" "$display_name" "$parked"
    else
        printf "${YELLOW}%-4s${RESET} ${MAGENTA}%-20s${RESET} %-36s ${DIM}%-18s${RESET} parked\n" \
            "$display_num" "$display_badge" "$display_name" "$parked"
    fi
done <<< "$entries"

printf "%s\n" "$(printf '%.0s-' {1..110})"
printf "\n${BOLD}Total:${RESET} %d parked session(s)\n\n" "$count"

# Prompt for selection
read -r -p "Enter # to restore (or q to quit): " choice

if [[ "$choice" == "q" || -z "$choice" ]]; then
    exit 0
fi

# Validate input
if ! [[ "$choice" =~ ^[0-9]+$ ]]; then
    printf "${RED}Invalid selection.${RESET}\n"
    exit 1
fi

# Get session info for the selected entry (convert display # back to 0-indexed)
selected_idx=$((choice - 1))
session_data=$(python3 -c "
import json
with open('$HISTORY_FILE') as f:
    history = json.load(f)
if $selected_idx < 0 or $selected_idx >= len(history):
    exit(1)
h = history[$selected_idx]
print(h.get('claude_session_id', '') + '\t' + h.get('badge', '') + '\t' + h.get('cwd', ''))
" 2>/dev/null || echo "")

if [ -z "$session_data" ]; then
    printf "${RED}Invalid selection.${RESET}\n"
    exit 1
fi

sid=$(echo "$session_data" | cut -f1)
badge=$(echo "$session_data" | cut -f2)
cwd=$(echo "$session_data" | cut -f3)
target_cwd="${cwd:-$HOME}"

# Check if already open
if echo "$current_sessions" | grep -q "^${sid}$"; then
    printf "${YELLOW}That session is already open.${RESET}\n"
    exit 0
fi

# Prompt for permission mode
printf "\nHow should Claude be launched?\n"
printf "  ${BOLD}1${RESET}) ${YELLOW}cld-dng${RESET} - dangerously skip permissions\n"
printf "  ${BOLD}2${RESET}) ${GREEN}safe${RESET}    - default permissions\n\n"
read -r -p "Choose [1/2]: " perm_choice

case "$perm_choice" in
    1) CLAUDE_CMD="claude --dangerously-skip-permissions --resume" ;;
    2) CLAUDE_CMD="claude --resume" ;;
    *)
        printf "${RED}Invalid choice. Aborting.${RESET}\n"
        exit 1 ;;
esac

badge_b64=$(echo -n "$badge" | base64)

printf "\nRestoring: ${MAGENTA}%s${RESET}..." "$badge"

tmp_script=$(mktemp /tmp/tt-history-XXXXXX.applescript)
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

# Remove from history
python3 -c "
import json
with open('$HISTORY_FILE') as f:
    history = json.load(f)
history.pop($selected_idx)
with open('$HISTORY_FILE', 'w') as f:
    json.dump(history, f, indent=2)
"

printf " ${GREEN}done${RESET}\n"
printf "Session removed from history.\n\n"
