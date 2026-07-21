#!/bin/bash
# ttracker/status.sh
#
# Reports all open iTerm2 terminal sessions with:
#   - iTerm2 window/session info (name, window ID, session UUID)
#   - Badge text (the watermark label set via bdg)
#   - The foreground process running in each (e.g. claude, zsh)
#   - For Claude sessions: the Claude session ID (for --resume)

set -euo pipefail

# Colors
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
MAGENTA='\033[35m'
RESET='\033[0m'

# Collect iTerm2 session data + badge via AppleScript (single call for performance)
iterm_data=$(osascript -e '
tell application "iTerm2"
    set output to ""
    repeat with w from 1 to (count of windows)
        set win to window w
        repeat with t from 1 to (count of tabs of win)
            set theTab to tab t of win
            repeat with s from 1 to (count of sessions of theTab)
                set sess to session s of theTab
                set b to ""
                try
                    tell sess
                        set b to (variable named "badge")
                    end tell
                end try
                set output to output & (id of win) & "\t" & t & "\t" & s & "\t" & (unique ID of sess) & "\t" & (tty of sess) & "\t" & b & "\t" & (name of sess) & linefeed
            end repeat
        end repeat
    end repeat
    return output
end tell' 2>/dev/null)

if [ -z "$iterm_data" ]; then
    echo "No iTerm2 sessions found (is iTerm2 running?)."
    exit 1
fi

total=0
claude_count=0

printf "\n${BOLD}%-4s %-20s %-40s %-14s %-12s %s${RESET}\n" "#" "Badge" "Session Name" "TTY" "Process" "Claude Session"
printf "%s\n" "$(printf '%.0s-' {1..140})"

while IFS=$'\t' read -r win_id tab_num sess_num sess_uuid tty badge sess_name; do
    [ -z "$win_id" ] && continue
    total=$((total + 1))

    # Find the foreground process on this TTY
    short_tty=$(basename "$tty")
    fg_proc=$(ps -t "$short_tty" -o stat,command 2>/dev/null \
        | awk '$1 ~ /\+/ {for(i=2;i<=NF;i++) printf "%s ", $i; print ""; exit}' \
        | xargs 2>/dev/null || echo "unknown")

    # Simplify process name
    proc_name=$(echo "$fg_proc" | awk '{print $1}' | xargs basename 2>/dev/null || echo "$fg_proc")

    # Check if a Claude session is running on this TTY (even if foreground is a subprocess like gopls)
    claude_session=""
    claude_pid=$(ps -t "$short_tty" -o pid,command 2>/dev/null \
        | awk '/claude/ && !/awk/ {print $1; exit}')
    if [ -n "$claude_pid" ]; then
        session_file="$HOME/.claude/sessions/${claude_pid}.json"
        if [ -f "$session_file" ]; then
            claude_session=$(python3 -c "import json; print(json.load(open('$session_file'))['sessionId'])" 2>/dev/null || echo "")
            claude_count=$((claude_count + 1))
        fi
    fi

    # Truncate fields for display
    display_name="${sess_name:0:38}"
    display_badge="${badge:0:18}"

    if [ -n "$claude_session" ]; then
        printf "${GREEN}%-4s${RESET} ${MAGENTA}%-20s${RESET} %-40s ${DIM}%-14s${RESET} ${CYAN}%-12s${RESET} ${YELLOW}%s${RESET}\n" \
            "$total" "$display_badge" "$display_name" "$tty" "$proc_name" "$claude_session"
    else
        printf "%-4s ${MAGENTA}%-20s${RESET} %-40s ${DIM}%-14s${RESET} %-12s %s\n" \
            "$total" "$display_badge" "$display_name" "$tty" "$proc_name" "-"
    fi
done <<< "$iterm_data"

printf "%s\n" "$(printf '%.0s-' {1..140})"
printf "\n${BOLD}Total sessions:${RESET} %d  |  ${BOLD}Claude sessions:${RESET} %d\n\n" "$total" "$claude_count"
