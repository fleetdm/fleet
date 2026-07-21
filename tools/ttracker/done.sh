#!/bin/bash
# ttracker/done.sh
#
# Marks the current terminal's Claude session as "done" so restore.sh
# won't reopen it. Run this after exiting Claude, before closing the terminal.
#
# Usage (typically via alias):
#   ttdone

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SNAPSHOT_DIR="$SCRIPT_DIR/snapshots"
DISMISSED_FILE="$SNAPSHOT_DIR/dismissed.txt"
SNAPSHOT_FILE="$SNAPSHOT_DIR/latest.json"

if [ ! -f "$SNAPSHOT_FILE" ]; then
    echo "No snapshot found. Nothing to dismiss."
    exit 1
fi

# Find which Claude session was running on this TTY
my_tty=$(tty)

# Look up session ID from the latest snapshot by matching TTY
session_info=$(python3 -c "
import json
data = json.load(open('$SNAPSHOT_FILE'))
for s in data['sessions']:
    if s.get('tty') == '$my_tty' and s.get('claude_session_id'):
        print(s['claude_session_id'] + '\t' + s.get('badge', ''))
        break
" 2>/dev/null || echo "")

if [ -z "$session_info" ]; then
    echo "No Claude session found for this terminal ($my_tty)."
    echo "If Claude already exited, the session may not match. Checking by badge..."

    # Fallback: try matching by iTerm2 session UUID via AppleScript
    iterm_uuid=$(osascript -e '
    tell application "iTerm2"
        repeat with w from 1 to (count of windows)
            set win to window w
            repeat with t from 1 to (count of tabs of win)
                repeat with s from 1 to (count of sessions of tab t of win)
                    set sess to session s of tab t of win
                    if (tty of sess) = "'"$my_tty"'" then
                        return unique ID of sess
                    end if
                end repeat
            end repeat
        end repeat
        return ""
    end tell' 2>/dev/null || echo "")

    if [ -n "$iterm_uuid" ]; then
        session_info=$(python3 -c "
import json
data = json.load(open('$SNAPSHOT_FILE'))
for s in data['sessions']:
    if s.get('iterm_uuid') == '$iterm_uuid' and s.get('claude_session_id'):
        print(s['claude_session_id'] + '\t' + s.get('badge', ''))
        break
" 2>/dev/null || echo "")
    fi

    if [ -z "$session_info" ]; then
        echo "Could not find a matching session. Nothing dismissed."
        exit 1
    fi
fi

sid=$(echo "$session_info" | cut -f1)
badge=$(echo "$session_info" | cut -f2)

# Add to dismissed list (if not already there)
mkdir -p "$SNAPSHOT_DIR"
touch "$DISMISSED_FILE"
if grep -q "^${sid}$" "$DISMISSED_FILE" 2>/dev/null; then
    echo "Session already dismissed: $badge ($sid)"
else
    echo "$sid" >> "$DISMISSED_FILE"
    echo "Dismissed: $badge (${sid:0:8}...)"
    echo "This session will not be restored. You can close the terminal."
fi
