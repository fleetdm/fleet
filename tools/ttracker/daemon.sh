#!/bin/bash
# ttracker/daemon.sh
#
# Background daemon that snapshots all iTerm2 terminal sessions every N minutes.
# Stores snapshots as JSON so restore.sh can reopen missing Claude sessions.
#
# Usage:
#   ./daemon.sh              # Run in foreground with default 10-minute interval
#   ./daemon.sh --bg         # Run in background (logs to snapshots/daemon.log)
#   ./daemon.sh --interval 5 # Custom interval in minutes
#   ./daemon.sh --once       # Take a single snapshot and exit
#   ./daemon.sh --stop       # Stop a running background daemon

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SNAPSHOT_DIR="$SCRIPT_DIR/snapshots"
PID_FILE="$SNAPSHOT_DIR/daemon.pid"
LOG_FILE="$SNAPSHOT_DIR/daemon.log"
INTERVAL_MINUTES=10
ONCE=false
BACKGROUND=false

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --interval) INTERVAL_MINUTES="$2"; shift 2 ;;
        --once)     ONCE=true; shift ;;
        --bg)       BACKGROUND=true; shift ;;
        --stop)
            if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
                kill "$(cat "$PID_FILE")"
                rm -f "$PID_FILE"
                echo "Daemon stopped."
            else
                echo "No daemon running."
                rm -f "$PID_FILE"
            fi
            exit 0 ;;
        -h|--help)
            echo "Usage: daemon.sh [--bg] [--interval MINUTES] [--once] [--stop]"
            echo "  --bg           Run in background (logs to snapshots/daemon.log)"
            echo "  --interval N   Snapshot every N minutes (default: 10)"
            echo "  --once         Take one snapshot and exit"
            echo "  --stop         Stop a running background daemon"
            exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

mkdir -p "$SNAPSHOT_DIR"

# Colors for terminal output
BOLD='\033[1m'
DIM='\033[2m'
GREEN='\033[32m'
YELLOW='\033[33m'
CYAN='\033[36m'
RESET='\033[0m'

function take_snapshot() {
    local timestamp
    timestamp=$(date +%Y-%m-%dT%H:%M:%S)
    local snapshot_file="$SNAPSHOT_DIR/latest.json"
    local backup_file="$SNAPSHOT_DIR/snapshot-$(date +%Y%m%d-%H%M%S).json"

    # Collect all data from iTerm2 in one AppleScript call
    local iterm_data
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
end tell' 2>/dev/null || echo "")

    if [ -z "$iterm_data" ]; then
        printf "${YELLOW}[%s] Warning: could not reach iTerm2, skipping snapshot${RESET}\n" "$timestamp"
        return 1
    fi

    # Build JSON array of sessions
    local json_sessions="["
    local first=true

    while IFS=$'\t' read -r win_id tab_num sess_num sess_uuid tty badge sess_name; do
        [ -z "$win_id" ] && continue

        local short_tty
        short_tty=$(basename "$tty")

        # Find Claude session ID
        local claude_session=""
        local claude_pid
        claude_pid=$(ps -t "$short_tty" -o pid,command 2>/dev/null \
            | awk '/claude/ && !/awk/ {print $1; exit}' || echo "")
        local cwd=""
        if [ -n "$claude_pid" ]; then
            local session_file="$HOME/.claude/sessions/${claude_pid}.json"
            if [ -f "$session_file" ]; then
                claude_session=$(python3 -c "import json; d=json.load(open('$session_file')); print(d['sessionId'])" 2>/dev/null || echo "")
                cwd=$(python3 -c "import json; d=json.load(open('$session_file')); print(d.get('cwd',''))" 2>/dev/null || echo "")
            fi
        fi

        # Find foreground process
        local proc_name
        proc_name=$(ps -t "$short_tty" -o stat,command 2>/dev/null \
            | awk '$1 ~ /\+/ {print $2; exit}' \
            | xargs basename 2>/dev/null || echo "unknown")

        # Escape strings for JSON
        local json_badge json_name json_cwd
        json_badge=$(python3 -c "import json; print(json.dumps('''$badge'''))" 2>/dev/null || echo "\"$badge\"")
        json_name=$(python3 -c "import json; print(json.dumps('''$sess_name'''))" 2>/dev/null || echo "\"$sess_name\"")
        json_cwd=$(python3 -c "import json; print(json.dumps('''$cwd'''))" 2>/dev/null || echo "\"$cwd\"")

        if [ "$first" = true ]; then
            first=false
        else
            json_sessions+=","
        fi

        json_sessions+=$(cat <<ENTRY

    {
      "window_id": $win_id,
      "tab": $tab_num,
      "session": $sess_num,
      "iterm_uuid": "$sess_uuid",
      "tty": "$tty",
      "badge": $json_badge,
      "session_name": $json_name,
      "process": "$proc_name",
      "claude_session_id": "$claude_session",
      "cwd": $json_cwd
    }
ENTRY
)
    done <<< "$iterm_data"

    json_sessions+=$'\n  ]'

    # Write full snapshot
    local snapshot_json
    snapshot_json=$(cat <<EOF
{
  "timestamp": "$timestamp",
  "session_count": $(echo "$iterm_data" | grep -c '[^\s]' || echo 0),
  "sessions": $json_sessions
}
EOF
)

    echo "$snapshot_json" > "$snapshot_file"
    cp "$snapshot_file" "$backup_file"

    # Prune old backups (keep last 50)
    local count
    count=$(ls -1 "$SNAPSHOT_DIR"/snapshot-*.json 2>/dev/null | wc -l | tr -d ' ')
    if [ "$count" -gt 50 ]; then
        ls -1t "$SNAPSHOT_DIR"/snapshot-*.json | tail -n +51 | xargs rm -f
    fi

    # Count Claude sessions in this snapshot
    local claude_count
    claude_count=$(echo "$snapshot_json" | grep -c '"claude_session_id": "[^"]' || echo 0)

    printf "${GREEN}[%s]${RESET} Snapshot saved: %d sessions (%d Claude) -> %s\n" \
        "$timestamp" \
        "$(echo "$iterm_data" | grep -c '[^\s]' || echo 0)" \
        "$claude_count" \
        "$(basename "$backup_file")"
}

# If --bg, re-exec ourselves in the background
if [ "$BACKGROUND" = true ]; then
    # Check if already running
    if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        echo "Daemon already running (PID $(cat "$PID_FILE")). Use --stop first."
        exit 1
    fi

    # Re-launch without --bg, redirect output to log
    nohup "$0" --interval "$INTERVAL_MINUTES" >> "$LOG_FILE" 2>&1 &
    bg_pid=$!
    echo "$bg_pid" > "$PID_FILE"
    echo "Daemon started in background (PID $bg_pid)."
    echo "  Log: $LOG_FILE"
    echo "  Stop: $0 --stop"
    exit 0
fi

# When running in foreground, write PID file too (for --stop to work)
echo $$ > "$PID_FILE"
trap 'rm -f "$PID_FILE"' EXIT

# Main
printf "\n${BOLD}Terminal Tracker Daemon${RESET}\n"
printf "Snapshot dir: ${CYAN}%s${RESET}\n" "$SNAPSHOT_DIR"

if [ "$ONCE" = true ]; then
    printf "Taking single snapshot...\n\n"
    take_snapshot
else
    printf "Interval: every ${CYAN}%d minutes${RESET}\n" "$INTERVAL_MINUTES"
    if [ -t 1 ]; then
        printf "Press Ctrl+C to stop.\n\n"
    fi
    while true; do
        take_snapshot
        sleep $((INTERVAL_MINUTES * 60))
    done
fi
