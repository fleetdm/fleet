#!/bin/bash
# ttracker/done.sh
#
# Parks the current terminal's Claude session into history.
# It won't be auto-restored, but you can bring it back via history.sh.
#
# Usage (typically via alias):
#   ttdone

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SNAPSHOT_DIR="$SCRIPT_DIR/snapshots"
HISTORY_FILE="$SNAPSHOT_DIR/history.json"
SNAPSHOT_FILE="$SNAPSHOT_DIR/latest.json"

if [ ! -f "$SNAPSHOT_FILE" ]; then
    echo "No snapshot found. Nothing to park."
    exit 1
fi

# Find which Claude session was running on this TTY
my_tty=$(tty)

# Look up full session info from the latest snapshot by matching TTY
session_json=$(python3 -c "
import json
data = json.load(open('$SNAPSHOT_FILE'))
for s in data['sessions']:
    if s.get('tty') == '$my_tty' and s.get('claude_session_id'):
        print(json.dumps(s))
        break
" 2>/dev/null || echo "")

if [ -z "$session_json" ]; then
    echo "No Claude session found for this terminal ($my_tty)."
    echo "Checking by iTerm2 session..."

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
        session_json=$(python3 -c "
import json
data = json.load(open('$SNAPSHOT_FILE'))
for s in data['sessions']:
    if s.get('iterm_uuid') == '$iterm_uuid' and s.get('claude_session_id'):
        print(json.dumps(s))
        break
" 2>/dev/null || echo "")
    fi

    if [ -z "$session_json" ]; then
        echo "Could not find a matching session. Nothing parked."
        exit 1
    fi
fi

# Add parked_at timestamp and append to history
python3 -c "
import json, os
from datetime import datetime

entry = json.loads('$session_json')
entry['parked_at'] = datetime.now().strftime('%Y-%m-%d %H:%M')

history_file = '$HISTORY_FILE'
history = []
if os.path.exists(history_file):
    with open(history_file) as f:
        history = json.load(f)

# Skip if already in history
if any(h['claude_session_id'] == entry['claude_session_id'] for h in history):
    badge = entry.get('badge', '')
    print(f'Already parked: {badge}')
else:
    history.append(entry)
    with open(history_file, 'w') as f:
        json.dump(history, f, indent=2)
    badge = entry.get('badge', '')
    sid_short = entry['claude_session_id'][:8]
    print(f'Parked: {badge} ({sid_short}...)')
    print('Retrieve it anytime with: tthistory')
"
